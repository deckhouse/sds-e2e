package integration

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	AppTmpPath    = "/app/tmp"
	DataPath      = "../data"
	KubePath      = "../../../sds-e2e-cfg"
	RemoteAppPath = "/home/user"

	PrivKeyName      = "id_rsa_test"
	PubKeyName       = "id_rsa_test.pub"
	ConfigTplName    = "config.yml.tpl"
	ConfigName       = "config.yml"
	ResourcesTplName = "resources.yml.tpl"
	ResourcesName    = "resources.yml"

	PVCWaitInterval       = 1
	PVCWaitIterationCount = 20
	PVCDeletedStatus      = "Deleted"
	nsCleanUpSeconds      = 30 * 60
	retries               = 100

	UbuntuCloudImage = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
)

var (
	HvHost               = ""
	HvSshUser            = ""
	HvSshKey             = ""
	HvDhPort             = "6445"
	HypervisorKubeConfig = ""

	NestedHost              = "127.0.0.1"
	NestedSshUser           = "user"
	NestedSshKey            = ""
	NestedDhPort            = "6445"
	NestedClusterKubeConfig = "kube-nested.config"

	VerboseOut = false

	verboseFlag           = flag.Bool("verbose", false, "Output with Info messages")
	debugFlag             = flag.Bool("debug", false, "Output with Debug messages")
	kconfigFlag           = flag.String("kconfig", NestedClusterKubeConfig, "The k8s config path for test")
	hypervisorkconfigFlag = flag.String("hypervisorkconfig", "", "The k8s config path for vm creation")
	clusterNameFlag       = flag.String("kcluster", "", "The context of cluster to use for test")
	standFlag             = flag.String("stand", "", "Test stand name")
	nsFlag                = flag.String("namespace", "", "Test name space")
	sshhostFlag           = flag.String("sshhost", "127.0.0.1", "Test ssh host")
	sshkeyFlag            = flag.String("sshkey", os.Getenv("HOME")+"/.ssh/id_rsa", "Test ssh key")

	NodeRequired = map[string]NodeFilter{
		"Ubu22": {
			Name: Cond{NotContains: []string{"-master-"}},
			Os:   Cond{Contains: []string{"Ubuntu 22.04"}},
		},
		"Ubu24_ultra": {
			Name:    Cond{NotContains: []string{"-master-"}},
			Os:      Cond{Contains: []string{"Ubuntu 24"}},
			Kernel:  Cond{Contains: []string{"5.15.0-122", "5.15.0-128", "5.15.0-127"}},
			Kubelet: Cond{Contains: []string{"v1.28.15"}},
		},
		"Deb11": {
			Name:   Cond{NotContains: []string{"-master-"}},
			Os:     Cond{Contains: []string{"Debian 11", "Debian GNU/Linux 11"}},
			Kernel: Cond{Contains: []string{"5.10.0-33-cloud-amd64", "5.10.0-19-amd64"}},
		},
		"Red7": {
			Name:   Cond{NotContains: []string{"-master-"}},
			Os:     Cond{Contains: []string{"RedOS 7.3", "RED OS MUROM (7.3"}},
			Kernel: Cond{Contains: []string{"6.1.52-1.el7.3.x86_64"}},
		},
		"Alt10": {
			Name: Cond{NotContains: []string{"-master-"}},
			Os:   Cond{Contains: []string{"Alt 10"}},
		},
		"Astra": {
			Name: Cond{NotContains: []string{"-master-"}},
			Os:   Cond{Contains: []string{"Astra Linux"}},
		},
	}

	SkipOptional      = true
	startTime         = time.Now()
	TestNS            = fmt.Sprintf("te2est-%d%d", startTime.Minute(), startTime.Second())
	licenseKey        = os.Getenv("licensekey")
	registryDockerCfg = "e30="
)

func envInit() {
	VerboseOut = *verboseFlag

	if *nsFlag != "" {
		TestNS = *nsFlag
	}

	if licenseKey != "" {
		registryAuthToken := base64Encode("license-token:" + licenseKey)
		registryDockerCfg = base64Encode(fmt.Sprintf("{\"auths\":{\"dev-registry.deckhouse.io\":{\"auth\":\"%s\"}}}", registryAuthToken))
	}

	if *standFlag == "stage" || *standFlag == "ci" || *standFlag == "metal" {
		SkipOptional = false
	}

	sshList := strings.Split(*sshhostFlag, "@")
	if *hypervisorkconfigFlag != "" {
		if strings.HasPrefix(*hypervisorkconfigFlag, "/") {
			HypervisorKubeConfig = *hypervisorkconfigFlag
		} else {
			HypervisorKubeConfig = filepath.Join(KubePath, *hypervisorkconfigFlag)
		}

		if len(sshList) >= 2 {
			HvHost = sshList[1]
			HvSshUser = sshList[0]
		} else {
			HvHost = sshList[0]
		}
		HvSshKey = *sshkeyFlag
		NestedDhPort = "6443"
	} else {
		if len(sshList) >= 2 {
			NestedHost = sshList[1]
			NestedSshUser = sshList[0]
		} else {
			NestedHost = sshList[0]
		}
		NestedSshKey = *sshkeyFlag
	}
	if strings.HasPrefix(*kconfigFlag, "/") {
		NestedClusterKubeConfig = *kconfigFlag
	} else {
		NestedClusterKubeConfig = filepath.Join(KubePath, *kconfigFlag)
	}
}
