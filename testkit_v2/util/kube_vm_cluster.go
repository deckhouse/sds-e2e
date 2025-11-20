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
	coreapi "k8s.io/api/core/v1"
)

const (
	DhDevImg                  = "dev-registry.deckhouse.io/sys/deckhouse-oss/install:main"
	DhCeImg                   = "registry.deckhouse.io/deckhouse/ce/install:stable"
	DhInstallCommand          = "docker run --network=host -t -v '/home/user/config.yml:/config.yml' -v '/home/user/:/tmp/' %s dhctl bootstrap --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --config=/config.yml"
	DhResourcesInstallCommand = "docker run --network=host -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/:/tmp/' %s dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --resources=/resources.yml"
	RegistryLoginCmd          = "sudo docker login -u license-token -p %s dev-registry.deckhouse.io"

	NodesReadyTimeout = 600 // Timeout for nodes to be ready (in seconds) - 10*60
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
			Fatalf("creating: %w", err)
		}
	}
}

func vmSync(cluster *KCluster, vms []VmConfig, nsName string) {
	vmList, err := cluster.ListVM(VmFilter{NameSpace: nsName})
	if err != nil || len(vmList) < len(vms) {
		Infof("Creating VMs")
		vmCreate(cluster, vms, nsName)
	}

	if err := RetrySec(8*60, func() error {
		vmList, err = cluster.ListVM(VmFilter{NameSpace: nsName, Phase: string(virt.MachineRunning)})
		if err != nil {
			return err
		}
		if len(vmList) < len(vms) {
			return fmt.Errorf("VMs are ready: %d of %d", len(vmList), len(vms))
		}
		Debugf("VMs are ready: %d", len(vmList))
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
	if err := ensureDockerInstalled(client); err != nil {
		return err
	}

	dhImg, err := authenticateRegistry(client)
	if err != nil {
		return err
	}

	if err := bootstrapConfig(client, dhImg, masterIp); err != nil {
		return err
	}

	if err := bootstrapResources(client, dhImg, masterIp); err != nil {
		return err
	}

	return nil
}

func ensureDockerInstalled(client sshClient) error {
	// Check if docker is already installed
	out, err := client.Exec("docker --version")
	if err == nil && strings.Contains(out, "Docker version") {
		Debugf("Docker is already installed: %s", strings.TrimSpace(out))
		return nil
	}

	Infof("Installing Docker")

	// Retry apt installation to handle lock conflicts
	if err := RetrySec(120, func() error {
		out, err := client.Exec("sudo apt update && sudo apt install -y docker.io")
		if err != nil {
			// Check if it's an apt lock error
			if strings.Contains(out, "Could not get lock") || strings.Contains(out, "Unable to lock directory") {
				return fmt.Errorf("apt is locked, retrying: %w\nOutput: %s", err, out)
			}
			// For other errors, return immediately
			return fmt.Errorf("failed to install docker.io: %w\nOutput: %s", err, out)
		}
		return nil
	}); err != nil {
		return err
	}

	// Verify docker installation
	out, err = client.Exec("docker --version")
	if err != nil {
		return fmt.Errorf("docker installation completed but docker command failed: %w\nOutput: %s", err, out)
	}
	if !strings.Contains(out, "Docker version") {
		return fmt.Errorf("docker installation verification failed: expected 'Docker version' in output, got: %s", out)
	}

	Infof("Docker successfully installed: %s", strings.TrimSpace(out))
	return nil
}

// TODO find out why we should remove docker at all!
func ensureDockerRemoved(bootstrapIp string) error {
	bootstrapClient := HvSshClient.GetFwdClient("user", bootstrapIp+":22", NestedSshKey)
	defer bootstrapClient.Close()

	out, _ := bootstrapClient.Exec("docker --version")
	if !strings.Contains(out, "Docker version") {
		Infof("Docker already removed from bootstrap node %s", bootstrapIp)
		return nil
	}

	Infof("Removing Docker from bootstrap node %s", bootstrapIp)
	if out, err := bootstrapClient.Exec("sudo apt-get purge docker.io -y --autoremove"); err != nil {
		return fmt.Errorf("failed to purge packages on bootstrap node %s: %v: %s", bootstrapIp, err, out)
	}

	if out, err := bootstrapClient.Exec("sudo rm -rf /var/lib/docker"); err != nil {
		return fmt.Errorf("failed to remove /var/lib/docker on bootstrap node %s: %v: %s", bootstrapIp, err, out)
	}

	if out, err := bootstrapClient.Exec("sudo rm -rf /etc/docker"); err != nil {
		return fmt.Errorf("failed to remove /etc/docker on bootstrap node %s: %v: %s", bootstrapIp, err, out)
	}

	Infof("Docker successfully removed from bootstrap node %s", bootstrapIp)
	return nil
}

func authenticateRegistry(client sshClient) (string, error) {
	dhImg := DhCeImg
	if licenseKey != "" {
		_ = client.ExecFatal(fmt.Sprintf(RegistryLoginCmd, licenseKey))
		dhImg = DhDevImg
	}
	return dhImg, nil
}

func bootstrapConfig(client sshClient, dhImg, masterIp string) error {
	Infof("Master: running dhctl bootstrap phase 'config'")
	cmd := fmt.Sprintf(DhInstallCommand, dhImg, masterIp)
	Debugf(cmd)
	cmd = "sudo -i timeout 900 " + cmd + " > /tmp/bootstrap.out || {(tail -30 /tmp/bootstrap.out; exit 124)}"
	if out, err := client.Exec(cmd); err != nil {
		Critf(out)
		return fmt.Errorf("dhctl bootstrap config error: %w", err)
	}
	return nil
}

func bootstrapResources(client sshClient, dhImg, masterIp string) error {
	Infof("Master: running dhctl bootstrap phase 'resources'")
	cmd := fmt.Sprintf(DhResourcesInstallCommand, dhImg, masterIp)
	Debugf(cmd)
	cmd = "sudo -i timeout 600 " + cmd + " > /tmp/bootstrap.out || {(tail -30 /tmp/bootstrap.out; exit 124)}"
	if out, err := client.Exec(cmd); err != nil {
		Critf(out)
		return fmt.Errorf("dhctl bootstrap resources error: %w", err)
	}
	return nil
}

// TODO - check if Deckhouse is installed by checking if pods are running in d8-system namespace
func checkDeckhouseInstalled() bool {
	out, _ := NestedSshClient.Exec("ls /opt/deckhouse")
	return !strings.Contains(out, "cannot access '/opt/deckhouse'")
}

func uploadBootstrapFiles(client sshClient) error {
	for _, f := range []string{ConfigName, ResourcesName} {
		err := client.Upload(filepath.Join(DataPath, f), filepath.Join(RemoteAppPath, f))
		if err != nil {
			return err
		}
	}

	for _, f := range []string{PrivKeyName} {
		err := client.Upload(filepath.Join(KubePath, f), filepath.Join(RemoteAppPath, f))
		if err != nil {
			return err
		}
	}
	return nil
}

func getKubeconfig() error {
	out := NestedSshClient.ExecFatal("sudo cat /root/.kube/config")
	out = strings.ReplaceAll(out, "127.0.0.1:6445", "127.0.0.1:"+NestedK8sPort)
	err := os.WriteFile(NestedClusterKubeConfig, []byte(out), 0600)
	if err != nil {
		return err
	}
	return nil
}

// Installs Deckhouse on virtual machines
func initVmD8(masterVm, bootstrapVm *VmConfig, vmKeyPath string) {
	if !checkDeckhouseInstalled() {
		if licenseKey == "" {
			Fatalf("Deckhouse EE license key is required: export licensekey=\"<license key>\"")
		}

		Infof("Deploying Deckhouse on the test cluster")
		mkConfig()
		mkResources()
		// TODO  - add error handling for mkConfig and mkResources

		client := HvSshClient.GetFwdClient("user", bootstrapVm.ip+":22", vmKeyPath)
		defer client.Close()

		if err := uploadBootstrapFiles(client); err != nil {
			Fatalf("failed to upload bootstrap files: %w", err)
		}

		if err := installVmDh(client, masterVm.ip); err != nil {
			Fatalf("failed to install Deckhouse on the test cluster: %w", err)
		}
	}

	if err := getKubeconfig(); err != nil {
		Fatalf("failed to get kubeconfig: %w", err)
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
			Debugf("Deleting old namespace %s", ns.Name)
			if err := cluster.DeleteNs(NsFilter{Name: ns.Name}); err != nil {
				Fatalf("Can't delete old namespace %s: %v", ns.Name, err)
			}
		}
	}
}

func setupHypervisorConnection() (*KCluster, error) {
	HvSshClient = GetSshClient(HvSshUser, HvHost+":22", HvSshKey)
	go HvSshClient.NewTunnel("127.0.0.1:"+HvK8sPort, "127.0.0.1:"+HvK8sPort)

	cluster, err := InitKCluster(HypervisorKubeConfig, "")
	if err != nil {
		Critf("Kubeclient '%s' problem: %w", HypervisorKubeConfig, err)
		return nil, err
	}

	return cluster, nil
}

func prepareNamespace(cluster *KCluster, nsName string) error {
	switch TestNSCleanUp {
	case "reinit":
		Debugf("Deleting old namespace %s", nsName)
		// TODO add NS exists check
		if err := cluster.DeleteNsAndWait(NsFilter{Name: nsName}); err != nil {
			Fatalf("failed to delete old namespace %s: %w", nsName, err)
			return err
		}
	case "free tmp":
		cleanUpNs(cluster)
	}

	GenerateRSAKeys(NestedSshKey, filepath.Join(KubePath, PubKeyName))

	if err := cluster.CreateNs(nsName); err != nil {
		Fatalf("failed to create namespace %s: %w", nsName, err)
		return err
	}

	return nil
}

func identifyVmRoles(vms []VmConfig) ([]*VmConfig, []*VmConfig, *VmConfig, error) {
	var vmMasters, vmWorkers []*VmConfig
	var vmBootstrap *VmConfig

	for _, vm := range vms {
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
		return nil, nil, nil, fmt.Errorf("VmCluster: 1 master and 1 setup record is required")
	}

	return vmMasters, vmWorkers, vmBootstrap, nil
}

// ensureNodesReady checks if all nodes are ready after being added
func ensureNodesReady(cluster *KCluster, expectedNodeCount int) error {
	Infof("Check if nodes are ready")
	return RetrySec(NodesReadyTimeout, func() error {
		nodes, err := cluster.ListNode()
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err)
		}

		readyCount := 0
		for _, node := range nodes {
			// Check if node has Ready condition with status True
			for _, condition := range node.Status.Conditions {
				if condition.Type == coreapi.NodeReady && condition.Status == coreapi.ConditionTrue {
					readyCount++
					break
				}
			}
		}

		if readyCount < expectedNodeCount {
			return fmt.Errorf("nodes ready: %d of %d", readyCount, expectedNodeCount)
		}

		Debugf("All %d nodes are ready", readyCount)
		return nil
	})
}

func ensureClusterReady(cluster *KCluster) error {
	Infof("Check if cluster is ready")

	// Check snapshot-controller module first (required by sds-local-volume)
	if err := cluster.WaitUntilDeploymentReady(SnapshotControllerModuleNamespace, SnapshotControllerDeploymentName, ModuleReadyTimeout); err != nil {
		return fmt.Errorf("snapshot-controller module is not ready: %w", err)
	}

	// Check sds-local-volume module (required by sds-node-configurator)
	// Check required deployment
	if err := cluster.WaitUntilDeploymentReady(SDSLocalVolumeModuleNamespace, SDSLocalVolumeCSIControllerDeploymentName, ModuleReadyTimeout); err != nil {
		return fmt.Errorf("sds-local-volume module deployment is not ready: %w", err)
	}

	// Check daemonset
	if err := cluster.WaitUntilDaemonSetReady(SDSLocalVolumeModuleNamespace, SDSLocalVolumeCSINodeDaemonSetName, ModuleReadyTimeout); err != nil {
		return fmt.Errorf("sds-local-volume module daemonset is not ready: %w", err)
	}

	Debugf("sds-local-volume ready")

	// Check sds-node-configurator module last
	if err := cluster.WaitUntilDaemonSetReady(SDSNodeConfiguratorModuleNamespace, SDSNodeConfiguratorDaemonSetName, ModuleReadyTimeout); err != nil {
		return fmt.Errorf("sds-node-configurator module is not ready: %w", err)
	}

	Debugf("sds-node-configurator ready")

	return nil
}

func ClusterCreate() {
	nsName := TestNS
	Infof("NS '%s'", nsName)

	cluster, err := setupHypervisorConnection()
	if err != nil {
		Fatalf(err.Error())
	}

	if err := prepareNamespace(cluster, nsName); err != nil {
		Fatalf(err.Error())
	}

	vmSync(cluster, VmCluster, nsName)

	vmMasters, vmWorkers, vmBootstrap, err := identifyVmRoles(VmCluster)
	if err != nil {
		Fatalf(err.Error())
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

	if vmBootstrap != nil {
		if err := ensureDockerRemoved(vmBootstrap.ip); err != nil {
			Fatalf("Failed to remove Docker from bootstrap node: %v", err)
		}
	}

	if err := cluster.AddStaticNodes("e2e", "user", nodeIps); err != nil {
		Fatalf(err.Error())
	}

	// Wait for nodes to be ready before checking module readiness
	if err := ensureNodesReady(cluster, len(nodeIps)); err != nil {
		Fatalf("Nodes are not ready: %v", err)
	}

	if err := ensureClusterReady(cluster); err != nil {
		Fatalf(err.Error())
	}
}
