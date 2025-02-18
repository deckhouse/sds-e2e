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

	//https://cloud-images.ubuntu.com/
	ImageUbuntu_22      = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	ImageUbuntu_24      = "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
	ImageUbuntu_24_vmdk = "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.vmdk"
	//https://cloud.debian.org/images/cloud/
	ImageDebian_11 = "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-genericcloud-amd64.qcow2"
	//RedOs
	ImageRedOS_7_3       = "https://files.red-soft.ru/redos/7.3/x86_64/iso/redos-MUROM-7.3.4-20231220.0-Everything-x86_64-DVD1.iso"
	ImageRedOS_7_3_qcow2 = "https://static.storage-e2e.virtlab.flant.com/media/redos733.qcow2"
	//https://ftp.altlinux.ru/pub/distributions/ALTLinux/
	ImageAlt_10        = "https://ftp.altlinux.ru/pub/distributions/ALTLinux/platform/images/cloud/x86_64/alt-p10-cloud-x86_64.qcow2"
	ImageAlt_10_Server = "https://ftp.altlinux.ru/pub/distributions/ALTLinux/platform/images/cloud/x86_64/alt-server-p10-cloud-x86_64.qcow2"
	ImageAlt_11        = "https://ftp.altlinux.ru/pub/distributions/ALTLinux/images/p11/cloud/x86_64/alt-p11-cloud-x86_64.qcow2"
	//https://download.astralinux.ru/ui/native/mg-generic/
	ImageAstra_1_7_Max  = "https://download.astralinux.ru/artifactory/mg-generic/alse/cloudinit/alse-1.7-max-cloudinit-latest-amd64.qcow2"
	ImageAstra_1_8_Base = "https://download.astralinux.ru/artifactory/mg-generic/alse/cloud/alse-1.8.1-base-cloud-mg13.3.0-amd64.qcow2"
	//https://cloud.centos.org/centos/
	ImageCentOS_9  = "https://cloud.centos.org/centos/9-stream/x86_64/images/CentOS-Stream-GenericCloud-x86_64-9-latest.x86_64.qcow2"
	ImageCentOS_10 = "https://cloud.centos.org/centos/10-stream/x86_64/images/CentOS-Stream-GenericCloud-x86_64-10-latest.x86_64.qcow2"
	//https://almalinux.org/get-almalinux/#Cloud_Images
	ImageAlma_9_5 = "https://repo.almalinux.org/almalinux/9/cloud/x86_64/images/AlmaLinux-9-GenericCloud-9.5-20241120.x86_64.qcow2"
	//https://alpinelinux.org/cloud/
	ImageAlpine_3_21 = "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/cloud/generic_alpine-3.21.2-x86_64-bios-cloudinit-r0.qcow2"
	//https://gitlab.archlinux.org/archlinux/arch-boxes/
	ImageArch = "https://geo.mirror.pkgbuild.com/images/latest/Arch-Linux-x86_64-cloudimg.qcow2"
	//https://fedoraproject.org/cloud/download
	ImageFedora_41 = "https://download.fedoraproject.org/pub/fedora/linux/releases/41/Cloud/x86_64/images/Fedora-Cloud-Base-Generic-41-1.4.x86_64.qcow2"
	//https://bsd-cloud-image.org/
	ImagrFreeBsd_14_2     = "https://object-storage.public.mtl1.vexxhost.net/swift/v1/1dbafeefbd4f4c80864414a441e72dd2/bsd-cloud-image.org/images/freebsd/14.2/2024-12-08/ufs/freebsd-14.2-ufs-2024-12-08.qcow2"
	ImageNetBsd_10_1      = "https://object-storage.public.mtl1.vexxhost.net/swift/v1/1dbafeefbd4f4c80864414a441e72dd2/bsd-cloud-image.org/images/netbsd/10.1/2025-02-15/ufs/netbsd-10.1-2025-02-15.qcow2"
	ImageOpenBsd_7_6      = "https://github.com/hcartiaux/openbsd-cloud-image/releases/download/v7.6_2024-10-08-22-40/openbsd-min.qcow2"
	ImageDragonFlyBsd_6_4 = "https://object-storage.public.mtl1.vexxhost.net/swift/v1/1dbafeefbd4f4c80864414a441e72dd2/bsd-cloud-image.org/images/dragonflybsd/6.4.0/2023-04-23/ufs/dragonflybsd-6.4.0-ufs-2023-04-23.qcow2"
	//https://download.freebsd.org/ftp/snapshots/
	ImagrFreeBsd_15 = "https://download.freebsd.org/ftp/snapshots/ISO-IMAGES/15.0/FreeBSD-15.0-CURRENT-amd64-20250213-6156da866e7d-275409-disc1.iso"
	//https://mirrors.slackware.com/slackware/
	ImageSlackware_15 = "https://mirrors.slackware.com/slackware/slackware-iso/slackware64-15.0-iso/slackware64-15.0-install-dvd.iso"
	//TODO RedHat
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
