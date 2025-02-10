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
	KubePath      = "../../../config"
	RemoteAppPath = "/home/user"

	PrivKeyName          = "id_rsa_test"
	PubKeyName           = "id_rsa_test.pub"
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
	retries               = 100

	UbuntuCloudImage = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
)

var (
	HypervisorKubeConfig    = ""
	NestedClusterKubeConfig = "kube-nested.config"

	verboseFlag           = flag.Bool("verbose", false, "Output with Info messages")
	debugFlag             = flag.Bool("debug", false, "Output with Debug messages")
	kconfigFlag           = flag.String("kconfig", NestedClusterKubeConfig, "The k8s config path for test")
	hypervisorkconfigFlag = flag.String("hypervisorkconfig", "", "The k8s config path for vm creation")
	clusterNameFlag       = flag.String("kcluster", "", "The context of cluster to use for test")
	standFlag             = flag.String("stand", "", "Test stand name")
	nsFlag                = flag.String("namespace", "", "Test name space")

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

	if *hypervisorkconfigFlag != "" {
		if strings.HasPrefix(*hypervisorkconfigFlag, "/") {
			HypervisorKubeConfig = *hypervisorkconfigFlag
		} else {
			HypervisorKubeConfig = filepath.Join(KubePath, *hypervisorkconfigFlag)
		}
	}
	if strings.HasPrefix(*kconfigFlag, "/") {
		NestedClusterKubeConfig = *kconfigFlag
	} else {
		NestedClusterKubeConfig = filepath.Join(KubePath, *kconfigFlag)
	}
}
