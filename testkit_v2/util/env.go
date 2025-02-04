package integration

import (
	"fmt"
	"flag"
	"time"
	"os"
	"path/filepath"
)

const (
	AppTmpPath    = "/app/tmp"
    DataPath      = "../data"
	RemoteAppPath = "/home/user"

	PrivKeyName          = "id_rsa_test"
	PubKeyName           = "id_rsa_test.pub"
	VmKubeConfigName     = "kube-metal.config"
	ConfigTplName        = "config.yml.tpl"
	ConfigName           = "config.yml"
	ResourcesTplName     = "resources.yml.tpl"
	ResourcesName        = "resources.yml"
	UserCreateScriptName = "createuser.sh"

	PVCKind               = "PersistentVolumeClaim"
	PVCAPIVersion         = "v1"
	PVCWaitInterval       = 1
	PVCWaitIterationCount = 20
	PVCDeletedStatus      = "Deleted"
	nsCleanUpSeconds      = 30 * 60
	retries               = 50

	UbuntuCloudImage     = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"

	// VVV to remove VVV
	ImageCloudUbuntu2204 = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	ImageCloudDebian11   = "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-genericcloud-amd64.raw"
	//ImageCloudRedOS73 = "https://files.red-soft.ru/redos/7.3/x86_64/iso/redos-MUROM-7.3.4-20231220.0-Everything-x86_64-DVD1.iso"
	ImageCloudRedOS73 = "https://static.storage-e2e.virtlab.flant.com/media/redos733.qcow2"

	ImageYaCloudUbuntu2204 = "b1gbp6lurl0smp6ci3js" // folderID: b1g1oe1s72nr8b95qkgn
	ImageYaCloudDebian11   = "fd81j47dsud5nvq3498i" // family_id: debian-11-oslogin
	ImageYaCloudRedOS73    = "fd8s4p1p4od29db5u8mi" // family_id: redsoft-red-os-standart-server-7-3
)

var (
	verboseFlag     = flag.Bool("verbose", false, "Output with Info messages")
	debugFlag       = flag.Bool("debug", false, "Output with Debug messages")
	//fakepubsubNodePort = flag.Int("fakepubsub-node-port", 30303, "The port to use for connecting sub tests with the fakepubsub service (for configuring PUBSUB_EMULATOR_HOST)")
	clusterPathFlag = flag.String("kconfig", "", "The k8s config path for test")
	clusterNameFlag = flag.String("kcluster", "", "The context of cluster to use for test")
	standFlag       = flag.String("stand", "", "Test stand name")
	nsFlag          = flag.String("namespace", "", "Test name space")
	SkipFlag        = false // TODO not on Prod/Ci

	//vmOS            = flag.String("virtos", "", "Deploy virtual machine with specified OS")
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

	startTime  = time.Now()
	TestNS     = fmt.Sprintf("te2est-%d%d", startTime.Minute(), startTime.Second())
	licenseKey = os.Getenv("licensekey")
)

func envInit() {
	if *nsFlag != "" {
		TestNS = *nsFlag
	}
}

func envClusterName(clusterName string) string {
	//flag.Parse()
	if clusterName == "default" {
		return ""
	}
	if clusterName == "" {
		clusterName = *clusterNameFlag
	}
	return clusterName
}

func envConfigPath(configPath string) string {
	if configPath != "" {
		if configPath[0] != '/' {
        	wd, _ := os.Getwd()
			return filepath.Join(wd, configPath)
		}
		return configPath
	}

	if *clusterPathFlag != "" {
		return *clusterPathFlag
	}

	return os.Getenv("kubeconfig")
}
