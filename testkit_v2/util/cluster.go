package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
	"github.com/melbahja/goph"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	DhDevImg                  = "dev-registry.deckhouse.io/sys/deckhouse-oss/install:main"
	DhCeImg                   = "registry.deckhouse.io/deckhouse/ce/install:stable"
	DhInstallCommand          = "sudo -i docker run --network=host -t -v '/home/user/config.yml:/config.yml' -v '/home/user/:/tmp/' %s dhctl bootstrap --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --config=/config.yml"
	DhResourcesInstallCommand = "sudo -i docker run --network=host -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/:/tmp/' %s dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --resources=/resources.yml"
	RegistryLoginCmd          = "sudo docker login -u license-token -p %s dev-registry.deckhouse.io"
)

type vm struct {
	name      string
	ip        string
	cpu       int
	memory    string
	scName    string
	imageName string
	sshPort   uint
	sshCl     *goph.Client
}

var (
	vmCfgs = []vm{
		{"vm1", "10.10.10.80", 4, "8Gi", "linstor-r1", UbuntuCloudImage, 2220, nil},
		{"vm2", "10.10.10.81", 2, "4Gi", "linstor-r1", UbuntuCloudImage, 2221, nil},
		{"vm3", "10.10.10.82", 2, "4Gi", "linstor-r1", UbuntuCloudImage, 2222, nil},
		//		{"vm77", "", 2, "4Gi", "linstor-r1", UbuntuCloudImage, 2223, nil},
	}
)

func vmCreate(clr *KCluster, vms []vm, nsName string) {
	sshPubKeyString := CheckAndGetSSHKeys(DataPath, PrivKeyName, PubKeyName)

	for _, vmItem := range vms {
		err := clr.CreateVM(nsName, vmItem.name, vmItem.ip, vmItem.cpu, vmItem.memory, vmItem.scName, vmItem.imageName, sshPubKeyString, 20, 20)
		if err != nil {
			Fatalf(err.Error())
		}
	}

	// Check all machines running
	for count := 0; ; count++ {
		allVMUp := true
		vmList, err := clr.GetVMs(nsName)
		if err != nil {
			Fatalf(err.Error())
		}
		Debugf("%v", vmList)
		for _, item := range vmList {
			if item.Status != virt.MachineRunning {
				allVMUp = false
				break
			}
		}

		if allVMUp && len(vmList) >= len(vms) {
			break
		}

		if count >= retries {
			Fatalf("Timeout waiting for all VMs to be ready")
		}

		time.Sleep(10 * time.Second)
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

func installVmDh(client *goph.Client, masterIp string) error {
	Infof("Apt update docker")
	out := []byte("Unable to lock directory")
	for strings.Contains(string(out), "Unable to lock directory") {
		out, _ = client.Run("sudo apt update && sudo apt -y install docker.io")
	}

	dhImg := DhCeImg
	if licenseKey != "" {
		_ = ExecSshFatal(client, fmt.Sprintf(RegistryLoginCmd, licenseKey))
		dhImg = DhDevImg
	}

	Infof("Master dhctl bootstrap config (5-7m)")
	Debugf(DhInstallCommand, dhImg, masterIp)
	_ = ExecSshFatal(client, fmt.Sprintf(DhInstallCommand, dhImg, masterIp))

	Infof("Master dhctl bootstrap resources")
	Debugf(DhResourcesInstallCommand, dhImg, masterIp)
	_ = ExecSshFatal(client, fmt.Sprintf(DhResourcesInstallCommand, dhImg, masterIp))

	return nil
}

func initVmDh(masterClient *goph.Client, client *goph.Client) {
	out := ExecSshFatal(masterClient, "ls -1 /opt/deckhouse | wc -l")
	if strings.Contains(out, "cannot access '/opt/deckhouse'") {
		mkConfig()
		mkResources()

		for _, f := range []string{ConfigName, PrivKeyName, ResourcesName} {
			err := client.Upload(filepath.Join(DataPath, f), filepath.Join(RemoteAppPath, f))
			if err != nil {
				Fatalf(err.Error())
			}
		}

		// TODO deprecated
		for _, f := range []string{UserCreateScriptName} {
			err := masterClient.Upload(filepath.Join(DataPath, f), filepath.Join(RemoteAppPath, f))
			if err != nil {
				Fatalf(err.Error())
			}
		}

		_ = installVmDh(client, vmCfgs[0].ip)
	}

	Infof("Get vm kube config")
	out = ExecSshFatal(masterClient, "sudo cat /root/.kube/config")
	out = strings.Replace(out, "127.0.0.1:6445", "127.0.0.1:6443", -1)
	err := os.WriteFile(filepath.Join(KubePath, NestedClusterKubeConfigName), []byte(out), 0600)
	if err != nil {
		Fatalf(err.Error())
	}
}

func cleanUpNs(clr *KCluster) {
	unixNow := time.Now().Unix()
	nsExists, _ := clr.GetNs(NsFilter{Name: Cond{Contains: []string{"te2est-"}}})
	for _, ns := range nsExists {
		if ns.Name == TestNS || !strings.HasPrefix(ns.Name, "te2est-") {
			continue
		}
		if unixNow-ns.GetCreationTimestamp().Unix() > nsCleanUpSeconds {
			Debugf("Dedeting NS %s", ns.Name)
			if err := clr.DeleteNs(ns.Name); err != nil {
				Errf("Can't delete NS %s", ns.Name)
			}
		}
	}
}

func ClusterCreate() {
	nsName := TestNS
	configPath := filepath.Join(KubePath, HypervisorKubeConfig)

	clr, err := InitKCluster(configPath, "")
	if err != nil {
		Critf("Kubeclient '%s' problem", configPath)
		Fatalf(err.Error())
	}

	Infof("Clean old NS")
	// cleanUpNs(clr)

	Infof("Make RSA key")
	vmKeyPath := filepath.Join(DataPath, PrivKeyName)
	GenerateRSAKeys(vmKeyPath, filepath.Join(DataPath, PubKeyName))

	Infof("Create NS '%s'", nsName)
	if err := clr.CreateNs(nsName); err != nil {
		Fatalf(err.Error())
	}

	Infof("Create VM (2-3m)")
	vmCreate(clr, vmCfgs, nsName)

	for i, v := range vmCfgs {
		Infof("Get SSH client '%s'", v.name)
		if v.sshPort == 0 {
			vmCfgs[i].sshCl = NewSSHClient("user", v.ip, 22, vmKeyPath)
		} else {
			vmCfgs[i].sshCl = NewSSHClient("user", "127.0.0.1", v.sshPort, vmKeyPath)
		}
		defer vmCfgs[i].sshCl.Close()
	}

	Infof("Install VM DeckHouse (7-8m)")
	initVmDh(vmCfgs[0].sshCl, vmCfgs[1].sshCl)

	clr, err = InitKCluster("", "")
	if err != nil {
		Critf("Kubeclient '%s' problem", configPath)
		Fatalf(err.Error())
	}
	if err := clr.AddStaticNodes("ubuntu", "user", []string{vmCfgs[1].ip, vmCfgs[2].ip}); err != nil {
		Fatalf(err.Error())
	}

	Infof("Check Cluster ready")
	for i := 0; ; i++ {
		dsNodeConfigurator, err := clr.GetDaemonSet("d8-sds-node-configurator", "sds-node-configurator")
		if err != nil {
			if !apierrors.IsNotFound(err) {
				Fatalf(err.Error())
			}
		} else if int(dsNodeConfigurator.Status.NumberReady) >= len(vmCfgs) {
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
