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
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DataPath      = "../data"
	KubePath      = "../../../sds-e2e-cfg"
	RemoteAppPath = "/home/user"

	PrivKeyName   = "id_rsa_test"
	PubKeyName    = "id_rsa_test.pub"
	ConfigName    = "config.yml"
	ResourcesName = "resources.yml"

	pvcWaitInterval       = 1
	pvcWaitIterationCount = 20
	nsCleanUpSeconds      = 30 * 60
	retries               = 100
)

var (
	SkipOptional      = false
	startTime         = time.Now()
	TestNS            = fmt.Sprintf("e2e-tmp-%d%d", startTime.Minute(), startTime.Second())
	TestNSCleanUp     = ""
	licenseKey        = os.Getenv("licensekey")
	registryDockerCfg = "e30="
	Parallel          = false
	TreeMode          = false
	KeepState         = false
	fileLogger        *log.Logger

	ConfigTplName    = "config.yml.tpl"
	ResourcesTplName = "resources.yml.tpl"

	HypervisorKubeConfig = ""
	HvHost               = ""
	HvSshUser            = ""
	HvSshKey             = ""
	HvK8sPort            = "6445"
	HvSshClient          sshClient
	HvStorageClass       = "linstor-r1"

	NestedHost              = "127.0.0.1"
	NestedSshUser           = "user"
	NestedSshKey            = ""
	NestedK8sPort           = "6445"
	NestedClusterKubeConfig = "kube-nested.config"
	NestedSshClient         sshClient

	verboseFlag           = flag.Bool("verbose", false, "Output with Info messages")
	debugFlag             = flag.Bool("debug", false, "Output with Debug messages")
	treeFlag              = flag.Bool("tree", false, "Tests output in tree mode")
	kconfigFlag           = flag.String("kconfig", NestedClusterKubeConfig, "The k8s config path for test")
	hypervisorkconfigFlag = flag.String("hypervisorkconfig", "", "The k8s config path for vm creation")
	hvStorageClassFlag    = flag.String("hvstorageclass", HvStorageClass, "Hypervisor StorageClass name for nested cluster creation (virtual machines)")
	clusterNameFlag       = flag.String("kcluster", "", "The context of cluster to use for test")
	standFlag             = flag.String("stand", "", "Test stand name")
	nsFlag                = flag.String("namespace", "", "Test name space")
	nsReinitFlag          = flag.String("namespacereinit", "", "Test name space (reinitialize if exists)")
	nsCleanupFlag         = flag.String("namespacecleanup", "", "Test name space (delete after use)")
	sshhostFlag           = flag.String("sshhost", "127.0.0.1", "Test ssh host")
	sshkeyFlag            = flag.String("sshkey", os.Getenv("HOME")+"/.ssh/id_rsa", "Test ssh key")
	configTplFlag         = flag.String("nestedclusterconfigtemplate", ConfigTplName, "Test cluster config.yml template")
	resourcesTplFlag      = flag.String("nestedclusterresourcestemplate", ResourcesTplName, "Test cluster resources.yml template")
	skipOptionalFlag      = flag.Bool("skipoptional", false, "Skip optional tests (no required resources)")
	notParallelFlag       = flag.Bool("notparallel", false, "Run test groups in single mode")
	keepStateFlag         = flag.Bool("keepstate", false, "Don`t clean up after test finished")
	logFileFlag           = flag.String("logfile", "", "Write extended logs to file")

	NodeRequired    = map[string]NodeFilter{}
	VmCluster       = []VmConfig{}
	clusterTypeFlag = flag.String("clustertype", "Ubuntu 22 mini", "Set name of cluster nodes OS")
)

type clusterType struct {
	NodeRequired map[string]NodeFilter
	VmCluster    []VmConfig
}

var clusterTypeMap = map[string]clusterType{
	"Ubuntu 22 mini": clusterType{
		NodeRequired: map[string]NodeFilter{
			"Ubu22": {
				Name: "!%-master-%",
				Os:   "%Ubuntu 22.04%",
			},
		},
		VmCluster: []VmConfig{
			{"vm1-ub22", []string{"master"}, "", 4, 8, 20, "Ubuntu_22"},
			{"vm2-ub22", []string{"setup", "worker"}, "", 2, 6, 20, "Ubuntu_22"},
			{"vm3-ub22", []string{"worker"}, "", 2, 4, 20, "Ubuntu_22"},
		},
	},
	"Ubuntu 22 + Ubuntu 24 + Debian 11": clusterType{
		NodeRequired: map[string]NodeFilter{
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
		},
		VmCluster: []VmConfig{
			{"vm1-ub22", []string{"master"}, "", 4, 8, 20, "Ubuntu_22"},
			{"vm2-ub22", []string{"setup", "worker"}, "", 2, 6, 20, "Ubuntu_22"},
			{"vm3-ub22", []string{"worker"}, "", 2, 4, 20, "Ubuntu_22"},
			{"vm4-ub24", []string{"worker"}, "", 2, 4, 20, "Ubuntu_24"},
			{"vm5-de11", []string{"worker"}, "", 2, 4, 20, "Debian_11"},
		},
	},
	"Astra 1.7.3 flant": clusterType{
		NodeRequired: map[string]NodeFilter{
			"Astra": {
				Name: "!%-master-%",
				Os:   WhereLike{"Astra Linux"},
			},
		},
		VmCluster: []VmConfig{
			{"vm1-ub22", []string{"master"}, "", 4, 8, 20, "Ubuntu_22"},
			{"vm2-ub22", []string{"setup", "worker"}, "", 2, 6, 20, "Ubuntu_22"},
			{"vm3-as173", []string{"worker"}, "", 2, 4, 20, "Astra_173_base"},
			{"vm4-as173", []string{"worker"}, "", 2, 4, 20, "Astra_173_base"},
		},
	},

	"Astra 1.7.5 flant": clusterType{
		NodeRequired: map[string]NodeFilter{
			"Astra": {
				Name: "!%-master-%",
				Os:   WhereLike{"Astra Linux"},
			},
		},
		VmCluster: []VmConfig{
			{"vm1-ub22", []string{"master"}, "", 4, 8, 20, "Ubuntu_22"},
			{"vm2-ub22", []string{"setup", "worker"}, "", 2, 6, 20, "Ubuntu_22"},
			{"vm3-as175", []string{"worker"}, "", 2, 4, 20, "Astra_175_flant"},
			{"vm4-as175", []string{"worker"}, "", 2, 4, 20, "Astra_175_flant"},
		},
	},
	"Astra 1.8.1 flant": clusterType{
		NodeRequired: map[string]NodeFilter{
			"Astra": {
				Name: "!%-master-%",
				Os:   WhereLike{"Astra Linux"},
			},
		},
		VmCluster: []VmConfig{
			{"vm1-ub22", []string{"master"}, "", 4, 8, 20, "Ubuntu_22"},
			{"vm2-as181", []string{"setup", "worker"}, "", 2, 6, 20, "Astra_181_flant"},
			{"vm3-as181", []string{"worker"}, "", 2, 4, 20, "Astra_181_flant"},
			{"vm4-as181", []string{"worker"}, "", 2, 4, 20, "Astra_181_flant"},
		},
	},
	"RedOS 7.3 flant": clusterType{
		NodeRequired: map[string]NodeFilter{
			"Red7": {
				Name: "!%-master-%",
				Os:   WhereLike{"RedOS 7.3", "RED OS MUROM (7.3"},
				//Kernel: WhereLike{"6.1.52-1.el7.3.x86_64"},
			},
		},
		VmCluster: []VmConfig{
			{"vm1-red73", []string{"master"}, "", 4, 8, 20, "RedOS_7_3_flant"},
			{"vm2-ub22", []string{"setup", "worker"}, "", 2, 6, 20, "Ubuntu_22"},
			{"vm3-red73", []string{"worker"}, "", 2, 4, 20, "RedOS_7_3_flant"},
			{"vm4-red73", []string{"worker"}, "", 2, 4, 20, "RedOS_7_3_flant"},
		},
	},
	"RedOS 8 flant": clusterType{
		NodeRequired: map[string]NodeFilter{
			"Red8": {
				Name:   "!%-master-%",
				Os:     WhereLike{"RED OS 8"},
				Kernel: WhereLike{"6.6.6-1.red80.x86_64"},
			},
		},
		VmCluster: []VmConfig{
			{"vm1-red8", []string{"master"}, "", 4, 8, 20, "RedOS_8_flant"},
			{"vm2-ub22", []string{"setup", "worker"}, "", 2, 6, 20, "Ubuntu_22"},
			{"vm3-red8", []string{"worker"}, "", 2, 4, 20, "RedOS_8_flant"},
			{"vm4-red8", []string{"worker"}, "", 2, 4, 20, "RedOS_8_flant"},
		},
	},
	"Alt 10 flant": clusterType{
		NodeRequired: map[string]NodeFilter{
			"Alt10": {
				Name: "!%-master-%",
				Os:   WhereLike{"Alt 10"},
			},
		},
		VmCluster: []VmConfig{
			{"vm1-alt10", []string{"master"}, "", 4, 8, 30, "Alt_10_flant"},
			{"vm2-ubu22", []string{"setup", "worker"}, "", 2, 6, 20, "Ubuntu_22"},
			{"vm3-alt10", []string{"worker"}, "", 2, 4, 30, "Alt_10_flant"},
			{"vm4-alt10", []string{"worker"}, "", 2, 4, 30, "Alt_10_flant"},
		},
	},
}

// DH supported versions https://deckhouse.ru/products/kubernetes-platform/documentation/v1/supported_versions.html
var Images = map[string]string{ //qcow2, vmdk, vdi, iso, raw, raw.gz, raw.xz
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
	"Astra_173_base":  "https://download.astralinux.ru/artifactory/mg-generic/alse/cloudinit/alse-1.7.3-base-cloudinit-mg14.0.0-amd64.qcow2",
	"Astra_173_max":   "https://download.astralinux.ru/artifactory/mg-generic/alse/cloudinit/alse-1.7.3-max-cloudinit-mg14.0.0-amd64.qcow2",
	"Astra_1_7_Max":   "https://download.astralinux.ru/artifactory/mg-generic/alse/cloudinit/alse-1.7-max-cloudinit-latest-amd64.qcow2",
	"Astra_181_Base":  "https://download.astralinux.ru/artifactory/mg-generic/alse/cloud/alse-1.8.1-base-cloud-mg13.3.0-amd64.qcow2",
	"Astra_175_flant": "https://static.storage-e2e.virtlab.flant.com/media/alse175.qcow2",
	"Astra_181_flant": "https://static.storage-e2e.virtlab.flant.com/media/alse181.qcow2",
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

func envInit() {
	if *nsReinitFlag != "" {
		TestNS = *nsReinitFlag
		TestNSCleanUp = "reinit"
	} else if *nsCleanupFlag != "" {
		TestNS = *nsCleanupFlag
		TestNSCleanUp = "delete"
	} else if *nsFlag != "" {
		TestNS = *nsFlag
	} else {
		TestNSCleanUp = "free tmp"
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
		NestedK8sPort = "6443"
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

	HvStorageClass = *hvStorageClassFlag
	KeepState = *keepStateFlag

	if *logFileFlag != "" {
		f, err := os.OpenFile(*logFileFlag, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		fileLogger = log.New(f, "", log.LstdFlags)
	}

	ConfigTplName = *configTplFlag
	ResourcesTplName = *resourcesTplFlag

	ct, ok := clusterTypeMap[*clusterTypeFlag]
	if !ok {
		log.Fatalf("invalid cluster type: %s", *clusterTypeFlag)
	}
	NodeRequired = ct.NodeRequired
	VmCluster = ct.VmCluster
}
