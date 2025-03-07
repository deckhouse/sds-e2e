package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	vmList, err := clr.ListVM(VmFilter{NameSpace: nsName})
	if err != nil || len(vmList) < len(vms) {
		Infof("Create VM (2-4m)")
		vmCreate(clr, vms, nsName)
	}

	if err := RetrySec(360, func() error {
		vmList, err = clr.ListVM(VmFilter{NameSpace: nsName, Phase: string(virt.MachineRunning)})
		if err != nil {
			return err
		}
		if len(vmList) < len(vms) {
			return fmt.Errorf("VMs ready: %d of %d", len(vmList), len(vms))
		}
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

	Infof("Master dhctl bootstrap config (6-9m)")
	cmd := fmt.Sprintf(DhInstallCommand, dhImg, masterIp)
	Debugf(cmd)
	cmd = "sudo -i timeout 720 " + cmd + " > /tmp/bootstrap.out || {(tail -30 /tmp/bootstrap.out; exit 124)}"
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
		return fmt.Errorf("dhctl bootstrap resources error")
	}

	return nil
}

func initVmDh(hvClient sshClient, masterVm, bootstrapVm VmConfig, vmKeyPath string) {
	masterClient := hvClient.GetFwdClient("user", masterVm.ip+":22", vmKeyPath)
	defer masterClient.Close()

	out, _ := masterClient.Exec("ls /opt/deckhouse")
	if strings.Contains(out, "cannot access '/opt/deckhouse'") {
		Infof("Install VM DeckHouse (8-12m)")
		mkConfig()
		mkResources()

		client := hvClient.GetFwdClient("user", bootstrapVm.ip+":22", vmKeyPath)
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

	Infof("NS '%s'", nsName)
	if err := clr.CreateNs(nsName); err != nil {
		Fatalf(err.Error())
	}

	Infof("VM check")
	vmSync(clr, VmCluster, nsName)

	masterVm, bootstrapVm := VmCluster[0], VmCluster[1]
	initVmDh(hvClient, masterVm, bootstrapVm, vmKeyPath)
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

	Infof("Check Cluster ready (8-10m)")
	if err := RetrySec(720, func() error {
		dsNodeConfigurator, err := clr.GetDaemonSet("d8-sds-node-configurator", "sds-node-configurator")
		if err != nil {
			return err
		}
		if int(dsNodeConfigurator.Status.NumberReady) < len(VmCluster) {
			return fmt.Errorf("sds-node-configurator ready: %d of %d", dsNodeConfigurator.Status.NumberReady, len(VmCluster))
		}
		return nil
	}); err != nil {
		Fatalf(err.Error())
	}
}
