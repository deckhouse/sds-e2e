package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	DhDevImg                  = "dev-registry.deckhouse.io/sys/deckhouse-oss/install:main"
	DhCeImg                   = "registry.deckhouse.io/deckhouse/ce/install:stable"
	DhInstallCommand          = "sudo -i docker run --network=host -t -v '/home/user/config.yml:/config.yml' -v '/home/user/:/tmp/' %s dhctl bootstrap --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --config=/config.yml"
	DhResourcesInstallCommand = "sudo -i docker run --network=host -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/:/tmp/' %s dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --resources=/resources.yml"
	RegistryLoginCmd          = "sudo docker login -u license-token -p %s dev-registry.deckhouse.io"
)

type VmConfig struct {
	name      string
	ip        string
	cpu       int
	memory    string
	imageName string
	diskSize  int
}

func vmCreate(clr *KCluster, vms []VmConfig, nsName string) {
	sshPubKeyString := CheckAndGetSSHKeys(KubePath, PrivKeyName, PubKeyName)

	for _, vmItem := range vms {
		err := clr.CreateVM(nsName, vmItem.name, vmItem.ip, vmItem.cpu, vmItem.memory, "linstor-r1", vmItem.imageName, sshPubKeyString, vmItem.diskSize)
		if err != nil {
			Fatalf(err.Error())
		}
	}
}

func vmSync(clr *KCluster, vms []VmConfig, nsName string) {
	vmList, err := clr.ListVM(nsName)
	if err != nil || len(vmList) < len(vms) {
		vmCreate(clr, vms, nsName)
	}

	// Check all machines running
	for i := 0; ; i++ {
		allVMUp := true
		vmList, err = clr.ListVM(nsName)
		if err != nil {
			Fatalf("Get vm list error: %s", err.Error())
		}
		for _, vm := range vmList {
			if vm.Status.Phase != virt.MachineRunning {
				allVMUp = false
				break
			}
		}

		if allVMUp && len(vmList) >= len(vms) {
			break
		}

		if i >= retries {
			Fatalf("Timeout waiting for all VMs to be ready")
		}

		time.Sleep(10 * time.Second)
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
	Infof("Apt update docker")
	out := "Unable to lock directory"
	for strings.Contains(out, "Unable to lock directory") {
		out, _ = client.Exec("sudo apt update && sudo apt -y install docker.io")
	}

	dhImg := DhCeImg
	if licenseKey != "" {
		_ = client.ExecFatal(fmt.Sprintf(RegistryLoginCmd, licenseKey))
		dhImg = DhDevImg
	}

	Infof("Master dhctl bootstrap config (5-7m)")
	Debugf(DhInstallCommand, dhImg, masterIp)
	_ = client.ExecFatal(fmt.Sprintf(DhInstallCommand, dhImg, masterIp))

	Infof("Master dhctl bootstrap resources")
	Debugf(DhResourcesInstallCommand, dhImg, masterIp)
	_ = client.ExecFatal(fmt.Sprintf(DhResourcesInstallCommand, dhImg, masterIp))

	return nil
}

func initVmDh(hvClient sshClient, masterVm, workerVm VmConfig, vmKeyPath string) {
	masterClient := hvClient.GetFwdClient("user", masterVm.ip+":22", vmKeyPath)
	defer masterClient.Close()

	out := masterClient.ExecFatal("ls -1 /opt/deckhouse | wc -l")
	if strings.Contains(out, "cannot access '/opt/deckhouse'") {
		mkConfig()
		mkResources()

		client := hvClient.GetFwdClient("user", workerVm.ip+":22", vmKeyPath)
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

		_ = installVmDh(client, masterVm.ip)
	}

	Infof("Get vm kube config")
	out = masterClient.ExecFatal("sudo cat /root/.kube/config")
	out = strings.Replace(out, "127.0.0.1:6445", "127.0.0.1:"+NestedDhPort, -1)
	err := os.WriteFile(NestedClusterKubeConfig, []byte(out), 0600)
	if err != nil {
		Fatalf(err.Error())
	}
}

func ClusterCreate() {
	nsName := TestNS

	hvClient := GetSshClient(HvSshUser, HvHost+":22", HvSshKey)
	go hvClient.NewTunnel("127.0.0.1:"+HvDhPort, "127.0.0.1:"+HvDhPort)

	clr, err := InitKCluster(HypervisorKubeConfig, "")
	if err != nil {
		Critf("Kubeclient '%s' problem", HypervisorKubeConfig)
		Fatalf(err.Error())
	}

	//Infof("Clean old NS")
	// cleanUpNs(clr)

	Infof("Make RSA key")
	vmKeyPath := filepath.Join(KubePath, PrivKeyName)
	GenerateRSAKeys(vmKeyPath, filepath.Join(KubePath, PubKeyName))

	Infof("Create NS '%s'", nsName)
	if err := clr.CreateNs(nsName); err != nil {
		Fatalf(err.Error())
	}

	Infof("Create VM (2-4m)")
	vmSync(clr, VmCluster, nsName)

	Infof("Install VM DeckHouse (7-10m)")
	initVmDh(hvClient, VmCluster[0], VmCluster[1], vmKeyPath)
	go hvClient.NewTunnel("127.0.0.1:"+NestedDhPort, VmCluster[0].ip+":"+NestedDhPort)

	clr, err = InitKCluster("", "")
	if err != nil {
		Critf("Kubeclient '%s' problem", NestedClusterKubeConfig)
		Fatalf(err.Error())
	}

	nodeIps := make([]string, len(VmCluster)-1)
	for i, vm := range VmCluster[1:] {
		nodeIps[i] = vm.ip
	}
	if err := clr.AddStaticNodes("e2e", "user", nodeIps); err != nil {
		Fatalf(err.Error())
	}

	Infof("Check Cluster ready")
	for i := 0; ; i++ {
		dsNodeConfigurator, err := clr.GetDaemonSet("d8-sds-node-configurator", "sds-node-configurator")
		if err != nil {
			if !apierrors.IsNotFound(err) {
				Fatalf(err.Error())
			}
		} else if int(dsNodeConfigurator.Status.NumberReady) >= len(VmCluster) {
			break
		} else {
			Debugf("sds-node-configurator ready: %d", dsNodeConfigurator.Status.NumberReady)
		}

		if i >= retries {
			Fatalf("Timeout waiting all DS sds-node-configurator ready")
		}

		time.Sleep(10 * time.Second)
	}
}
