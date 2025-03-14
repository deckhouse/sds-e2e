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
	DataPath      = "../data"
	KubePath      = "../../../sds-e2e-cfg"
	RemoteAppPath = "/home/user"

	PrivKeyName      = "id_rsa_test"
	PubKeyName       = "id_rsa_test.pub"
	ConfigTplName    = "config.yml.tpl"
	ConfigName       = "config.yml"
	ResourcesTplName = "resources.yml.tpl"
	ResourcesName    = "resources.yml"

	pvcWaitInterval       = 1
	pvcWaitIterationCount = 20
	nsCleanUpSeconds      = 30 * 60
	retries               = 100
)

var (
	SkipOptional      = false
	startTime         = time.Now()
	TestNS            = fmt.Sprintf("te2est-%d%d", startTime.Minute(), startTime.Second())
	licenseKey        = os.Getenv("licensekey")
	registryDockerCfg = "e30="
	Parallel          = false
	TreeMode          = false

	HypervisorKubeConfig = ""
	HvHost               = ""
	HvSshUser            = ""
	HvSshKey             = ""
	HvDhPort             = "6445"
	HvSshClient          sshClient

	NestedHost              = "127.0.0.1"
	NestedSshUser           = "user"
	NestedSshKey            = ""
	NestedDhPort            = "6445"
	NestedClusterKubeConfig = "kube-nested.config"
	NestedSshClient         sshClient

	verboseFlag           = flag.Bool("verbose", false, "Output with Info messages")
	debugFlag             = flag.Bool("debug", false, "Output with Debug messages")
	treeFlag              = flag.Bool("tree", false, "Tests output in tree mode")
	kconfigFlag           = flag.String("kconfig", NestedClusterKubeConfig, "The k8s config path for test")
	hypervisorkconfigFlag = flag.String("hypervisorkconfig", "", "The k8s config path for vm creation")
	clusterNameFlag       = flag.String("kcluster", "", "The context of cluster to use for test")
	standFlag             = flag.String("stand", "", "Test stand name")
	nsFlag                = flag.String("namespace", "", "Test name space")
	sshhostFlag           = flag.String("sshhost", "127.0.0.1", "Test ssh host")
	sshkeyFlag            = flag.String("sshkey", os.Getenv("HOME")+"/.ssh/id_rsa", "Test ssh key")
	skipOptionalFlag      = flag.Bool("skipoptional", false, "Skip optional tests (no required resources)")
	notParallelFlag       = flag.Bool("notparallel", false, "Run test groups in single mode")

	NodeRequired = map[string]NodeFilter{
		"Ubu22": {
			Name: "!%-master-%",
			Os:   "%Ubuntu 22.04%",
		},
		"Ubu24": {
			Name:    "!%-master-%",
			Os:      "%Ubuntu 24%",
			Kernel:  WhereLike{"5.15.0-122", "5.15.0-128", "5.15.0-127", "6.8.0-53"},
			Kubelet: WhereLike{"v1.28.15"},
		},
		"Deb11": {
			Name:   "!%-master-%",
			Os:     WhereLike{"Debian 11", "Debian GNU/Linux 11"},
			Kernel: WhereLike{"5.10.0-33-cloud-amd64", "5.10.0-19-amd64"},
		},
		//"Red7": {
		//	Name: "!%-master-%",
		//	Os:     WhereLike{"RedOS 7.3", "RED OS MUROM (7.3"},
		//	Kernel: WhereLike{"6.1.52-1.el7.3.x86_64"},
		//},
		"Red8": {
			Name:   "!%-master-%",
			Os:     WhereLike{"RED OS 8"},
			Kernel: WhereLike{"6.6.6-1.red80.x86_64"},
		},
		"Astra": {
			Name: "!%-master-%",
			Os:   WhereLike{"Astra Linux"},
		},
		//"Alt10": {
		//	Name: "!%-master-%",
		//	Os:   WhereLike{"Alt 10"},
		//},
	}

	//DH supported versions https://deckhouse.ru/products/kubernetes-platform/documentation/v1/supported_versions.html
	Images = map[string]string{ //qcow2, vmdk, vdi, iso, raw, raw.gz, raw.xz
		//https://cloud-images.ubuntu.com/
		"Ubuntu_22":      "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
		"Ubuntu_24":      "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
		"Ubuntu_24_vmdk": "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.vmdk",
		"Ubuntu_24_old":  "https://cloud-images.ubuntu.com/noble/20241128/noble-server-cloudimg-amd64.img",
		//https://cloud.debian.org/images/cloud/
		"Debian_11":     "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-genericcloud-amd64.qcow2",
		"Debian_11_raw": "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-genericcloud-amd64.raw",
		//RedOs
		"RedOS_7_3":       "https://files.red-soft.ru/redos/7.3/x86_64/iso/redos-MUROM-7.3.4-20231220.0-Everything-x86_64-DVD1.iso",
		"RedOS_7_3_flant": "https://static.storage-e2e.virtlab.flant.com/media/redos733.qcow2",
		"RedOS_8_flant":   "https://static.storage-e2e.virtlab.flant.com/media/redos8.qcow2",
		//https://ftp.altlinux.ru/pub/distributions/ALTLinux/
		"Alt_10":        "https://ftp.altlinux.ru/pub/distributions/ALTLinux/platform/images/cloud/x86_64/alt-p10-cloud-x86_64.qcow2",
		"Alt_10_Server": "https://ftp.altlinux.ru/pub/distributions/ALTLinux/platform/images/cloud/x86_64/alt-server-p10-cloud-x86_64.qcow2",
		"Alt_11":        "https://ftp.altlinux.ru/pub/distributions/ALTLinux/images/p11/cloud/x86_64/alt-p11-cloud-x86_64.qcow2",
		"Alt_10_flant":  "https://static.storage-e2e.virtlab.flant.com/media/altp10.qcow2",
		//https://download.astralinux.ru/ui/native/mg-generic/
		"Astra_1_7_Max":   "https://download.astralinux.ru/artifactory/mg-generic/alse/cloudinit/alse-1.7-max-cloudinit-latest-amd64.qcow2",
		"Astra_1_8_Base":  "https://download.astralinux.ru/artifactory/mg-generic/alse/cloud/alse-1.8.1-base-cloud-mg13.3.0-amd64.qcow2",
		"Astra_1_7_flant": "https://static.storage-e2e.virtlab.flant.com/media/alse175.qcow2",
		"Astra_1_8_flant": "https://static.storage-e2e.virtlab.flant.com/media/alse181.qcow2",
		//https://cloud.centos.org/centos/
		"CentOS_9":  "https://cloud.centos.org/centos/9-stream/x86_64/images/CentOS-Stream-GenericCloud-x86_64-9-latest.x86_64.qcow2",
		"CentOS_10": "https://cloud.centos.org/centos/10-stream/x86_64/images/CentOS-Stream-GenericCloud-x86_64-10-latest.x86_64.qcow2",
		//https://almalinux.org/get-almalinux/#Cloud_Images
		"Alma_9_5": "https://repo.almalinux.org/almalinux/9/cloud/x86_64/images/AlmaLinux-9-GenericCloud-9.5-20241120.x86_64.qcow2",
		//https://alpinelinux.org/cloud/
		"Alpine_3_21": "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/cloud/generic_alpine-3.21.2-x86_64-bios-cloudinit-r0.qcow2",
		//https://gitlab.archlinux.org/archlinux/arch-boxes/
		"Arch": "https://geo.mirror.pkgbuild.com/images/latest/Arch-Linux-x86_64-cloudimg.qcow2",
		//https://fedoraproject.org/cloud/download
		"Fedora_41": "https://download.fedoraproject.org/pub/fedora/linux/releases/41/Cloud/x86_64/images/Fedora-Cloud-Base-Generic-41-1.4.x86_64.qcow2",
		//https://bsd-cloud-image.org/
		"FreeBsd_14_2":     "https://object-storage.public.mtl1.vexxhost.net/swift/v1/1dbafeefbd4f4c80864414a441e72dd2/bsd-cloud-image.org/images/freebsd/14.2/2024-12-08/ufs/freebsd-14.2-ufs-2024-12-08.qcow2",
		"NetBsd_10_1":      "https://object-storage.public.mtl1.vexxhost.net/swift/v1/1dbafeefbd4f4c80864414a441e72dd2/bsd-cloud-image.org/images/netbsd/10.1/2025-02-15/ufs/netbsd-10.1-2025-02-15.qcow2",
		"OpenBsd_7_6":      "https://github.com/hcartiaux/openbsd-cloud-image/releases/download/v7.6_2024-10-08-22-40/openbsd-min.qcow2",
		"DragonFlyBsd_6_4": "https://object-storage.public.mtl1.vexxhost.net/swift/v1/1dbafeefbd4f4c80864414a441e72dd2/bsd-cloud-image.org/images/dragonflybsd/6.4.0/2023-04-23/ufs/dragonflybsd-6.4.0-ufs-2023-04-23.qcow2",
		//https://download.freebsd.org/ftp/snapshots/
		"FreeBsd_15": "https://download.freebsd.org/ftp/snapshots/ISO-IMAGES/15.0/FreeBSD-15.0-CURRENT-amd64-20250213-6156da866e7d-275409-disc1.iso",
		//https://mirrors.slackware.com/slackware/
		"Slackware_15": "https://mirrors.slackware.com/slackware/slackware-iso/slackware64-15.0-iso/slackware64-15.0-install-dvd.iso",
	}

	VmCluster = []VmConfig{
		{"vm-ub22-1", "", 4, "8Gi", "Ubuntu_22", 20}, //master
		{"vm-ub22-2", "", 2, "6Gi", "Ubuntu_22", 20}, //setup DH
		{"vm-ub22-3", "", 2, "4Gi", "Ubuntu_22", 20},
		//{"vm-ub24-1", "", 2, "4Gi", "Ubuntu_24", 20},
		{"vm-de11-1", "", 2, "4Gi", "Debian_11", 20},
		//{"vm-re8-1", "", 2, "4Gi", "RedOS_8_flant", 20},
		//{"vm-as18-1", "", 2, "4Gi", "Astra_1_8_Base", 20},
		//{"vm-al10-1", "", 2, "4Gi", "Alt_10_flant", 30},
	}
)

func envInit() {
	if *nsFlag != "" {
		TestNS = *nsFlag
	}

	if licenseKey != "" {
		registryAuthToken := base64Encode("license-token:" + licenseKey)
		registryDockerCfg = base64Encode(fmt.Sprintf("{\"auths\":{\"dev-registry.deckhouse.io\":{\"auth\":\"%s\"}}}", registryAuthToken))
	}

	if *skipOptionalFlag && *standFlag != "stage" && *standFlag != "ci" {
		SkipOptional = true
	}

	if *treeFlag {
		TreeMode = true
	}
	if !*notParallelFlag {
		TreeMode, Parallel = true, true
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
		NestedSshKey = filepath.Join(KubePath, PrivKeyName)
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
