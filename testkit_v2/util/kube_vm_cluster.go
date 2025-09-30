/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
)

const (
	DhDevImg                  = "dev-registry.deckhouse.io/sys/deckhouse-oss/install:main"
	DhCeImg                   = "registry.deckhouse.io/deckhouse/ce/install:stable"
	DhInstallCommand          = "docker run --network=host -t -v '/home/user/config.yml:/config.yml' -v '/home/user/:/tmp/' %s dhctl bootstrap --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --config=/config.yml"
	DhResourcesInstallCommand = "docker run --network=host -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/:/tmp/' %s dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --resources=/resources.yml"
	RegistryLoginCmd          = "sudo docker login -u license-token -p %s dev-registry.deckhouse.io"
)

type VmConfig struct {
	name     string
	roles    []string
	ip       string
	cpu      int
	ram      int
	diskSize int
	image    string
}

func vmCreate(cluster *KCluster, vms []VmConfig, nsName string) {
	sshPubKeyString := CheckAndGetSSHKeys(KubePath, PrivKeyName, PubKeyName)

	for _, vmItem := range vms {
		err := cluster.CreateVM(nsName, vmItem.name, vmItem.ip, vmItem.cpu, vmItem.ram, HvStorageClass, vmItem.image, sshPubKeyString, vmItem.diskSize)
		if err != nil {
			Fatalf("creating vm: %w", err)
		}
	}
}

func vmSync(cluster *KCluster, vms []VmConfig, nsName string) {
	vmList, err := cluster.ListVM(VmFilter{NameSpace: nsName})
	if err != nil || len(vmList) < len(vms) {
		Infof("Create VM (2-5m)")
		vmCreate(cluster, vms, nsName)
	}

	if err := RetrySec(8*60, func() error {
		vmList, err = cluster.ListVM(VmFilter{NameSpace: nsName, Phase: string(virt.MachineRunning)})
		if err != nil {
			return err
		}
		if len(vmList) < len(vms) {
			return fmt.Errorf("VMs ready: %d of %d", len(vmList), len(vms))
		}
		Debugf("VMs ready: %d", len(vmList))
		return nil
	}); err != nil {
		Fatalf(err.Error())
	}

	for _, vm := range vmList {
		for i, cfg := range vms {
			if vm.Name == cfg.name {
				vms[i].ip = vm.Status.IPAddress
				break
			}
		}
	}
}

func mkTemplateFile(tplPath string, resPath string, a ...any) {
	template, err := os.ReadFile(tplPath)
	if err != nil {
		Fatalf(err.Error())
	}

	renderedTemplateString := fmt.Sprintf(string(template), a...)
	err = os.WriteFile(resPath, []byte(renderedTemplateString), 0644)
	if err != nil {
		Fatalf(err.Error())
	}
}

func mkConfig() {
	mkTemplateFile(filepath.Join(DataPath, ConfigTplName), filepath.Join(DataPath, ConfigName), registryDockerCfg)
}

func mkResources() {
	mkTemplateFile(filepath.Join(DataPath, ResourcesTplName), filepath.Join(DataPath, ResourcesName), registryDockerCfg)
}

func installVmDh(client sshClient, masterIp string) error {
	Infof("Docker install/update")
	// TODO add docker check/install for other OS (Astra, RedOS, Alt, ...) with error "docker not found"
	out := "Unable to lock directory"
	for strings.Contains(out, "Unable to lock directory") {
		out, _ = client.Exec("sudo apt update && sudo apt -y install docker.io")
	}

	dhImg := DhCeImg
	if licenseKey != "" {
		_ = client.ExecFatal(fmt.Sprintf(RegistryLoginCmd, licenseKey))
		dhImg = DhDevImg
	}

	Infof("Master dhctl bootstrap config (6-9m)")
	cmd := fmt.Sprintf(DhInstallCommand, dhImg, masterIp)
	Debugf(cmd)
	cmd = "sudo -i timeout 900 " + cmd + " > /tmp/bootstrap.out || {(tail -30 /tmp/bootstrap.out; exit 124)}"
	if out, err := client.Exec(cmd); err != nil {
		Critf(out)
		return fmt.Errorf("dhctl bootstrap config error")
	}

	Infof("Master dhctl bootstrap resources")
	cmd = fmt.Sprintf(DhResourcesInstallCommand, dhImg, masterIp)
	Debugf(cmd)
	cmd = "sudo -i timeout 600 " + cmd + " > /tmp/bootstrap.out || {(tail -30 /tmp/bootstrap.out; exit 124)}"
	if out, err := client.Exec(cmd); err != nil {
		Critf(out)
		return fmt.Errorf("dhctl bootstrap resources error: %w", err)
	}

	return nil
}

func initVmD8(masterVm, bootstrapVm *VmConfig, vmKeyPath string) {
	out, _ := NestedSshClient.Exec("ls /opt/deckhouse")
	if strings.Contains(out, "cannot access '/opt/deckhouse'") {
		if licenseKey == "" {
			Fatalf("Deckhouse EE license key is required: export licensekey=\"<license key>\"")
		}

		Infof("Setup virtual clustaer (8-12m)")
		mkConfig()
		mkResources()

		client := HvSshClient.GetFwdClient("user", bootstrapVm.ip+":22", vmKeyPath)
		defer client.Close()

		for _, f := range []string{ConfigName, ResourcesName} {
			err := client.Upload(filepath.Join(DataPath, f), filepath.Join(RemoteAppPath, f))
			if err != nil {
				Fatalf(err.Error())
			}
		}

		for _, f := range []string{PrivKeyName} {
			err := client.Upload(filepath.Join(KubePath, f), filepath.Join(RemoteAppPath, f))
			if err != nil {
				Fatalf(err.Error())
			}
		}

		if err := installVmDh(client, masterVm.ip); err != nil {
			Fatalf(err.Error())
		}
	}

	out = NestedSshClient.ExecFatal("sudo cat /root/.kube/config")
	out = strings.ReplaceAll(out, "127.0.0.1:6445", "127.0.0.1:"+NestedK8sPort)
	err := os.WriteFile(NestedClusterKubeConfig, []byte(out), 0600)
	if err != nil {
		Fatalf(err.Error())
	}
}

func cleanUpNs(cluster *KCluster) {
	unixNow := time.Now().Unix()
	nsExists, _ := cluster.ListNs(NsFilter{Name: "%e2e-tmp-%"})
	for _, ns := range nsExists {
		if ns.Name == TestNS || !strings.HasPrefix(ns.Name, "e2e-tmp-") {
			continue
		}
		if unixNow-ns.GetCreationTimestamp().Unix() > nsCleanUpSeconds {
			Debugf("Dedeting namespace %s", ns.Name)
			if err := cluster.DeleteNs(NsFilter{Name: ns.Name}); err != nil {
				Fatalf("Can't delete namespace %s: %v", ns.Name, err)
			}
		}
	}
}

func ClusterCreate() {
	nsName := TestNS
	Infof("NS '%s'", nsName)

	HvSshClient = GetSshClient(HvSshUser, HvHost+":22", HvSshKey)
	go HvSshClient.NewTunnel("127.0.0.1:"+HvK8sPort, "127.0.0.1:"+HvK8sPort)

	cluster, err := InitKCluster(HypervisorKubeConfig, "")
	if err != nil {
		Critf("Kubeclient '%s' problem", HypervisorKubeConfig)
		Fatalf(err.Error())
	}

	switch TestNSCleanUp {
	case "reinit":
		Debugf("Delete old namespace %s", nsName)
		// TODO add NS exists check
		if err := cluster.DeleteNsAndWait(NsFilter{Name: nsName}); err != nil {
			Fatalf("Can't delete namespace %s: %v", nsName, err)
		}
	case "free tmp":
		cleanUpNs(cluster)
	}

	GenerateRSAKeys(NestedSshKey, filepath.Join(KubePath, PubKeyName))

	if err := cluster.CreateNs(nsName); err != nil {
		Fatalf(err.Error())
	}

	Infof("VM check")
	vmSync(cluster, VmCluster, nsName)

	var vmMasters, vmWorkers []*VmConfig
	var vmBootstrap *VmConfig
	for _, vm := range VmCluster {
		if slices.Contains(vm.roles, "master") {
			vmMasters = append(vmMasters, &vm)
			continue
		}
		if slices.Contains(vm.roles, "setup") {
			vmBootstrap = &vm
		}
		if slices.Contains(vm.roles, "worker") {
			vmWorkers = append(vmWorkers, &vm)
		}
	}
	if vmBootstrap == nil && len(vmWorkers) > 0 {
		vmBootstrap = vmWorkers[0]
	}
	if len(vmMasters) != 1 || vmBootstrap == nil {
		Fatalf("VmCluster: 1 master and 1 setup record is required")
	}

	NestedSshClient = HvSshClient.GetFwdClient(NestedSshUser, vmMasters[0].ip+":22", NestedSshKey)

	initVmD8(vmMasters[0], vmBootstrap, NestedSshKey)
	go NestedSshClient.NewTunnel("127.0.0.1:"+NestedK8sPort, vmMasters[0].ip+":"+NestedK8sPort)

	cluster, err = InitKCluster("", "")
	if err != nil {
		Critf("Kubeclient '%s' problem", NestedClusterKubeConfig)
		Fatalf(err.Error())
	}

	nodeIps := make([]string, len(vmWorkers))
	for i, vm := range vmWorkers {
		nodeIps[i] = vm.ip
	}
	if err := cluster.AddStaticNodes("e2e", "user", nodeIps); err != nil {
		Fatalf(err.Error())
	}

	Infof("Check Cluster ready (8-10m)")
	if err := RetrySec(12*60, func() error {
		dsNodeConfigurator, err := cluster.GetDaemonSet("d8-sds-node-configurator", "sds-node-configurator")
		if err != nil {
			return err
		}
		if int(dsNodeConfigurator.Status.NumberReady) < len(vmWorkers) {
			return fmt.Errorf("sds-node-configurator ready: %d of %d", dsNodeConfigurator.Status.NumberReady, len(vmWorkers))
		}
		Debugf("sds-node-configurator ready")
		return nil
	}); err != nil {
		Fatalf(err.Error())
	}
}
