package integration

import (
	"flag"
	"os"
)

const (
	defaultNamespace = "default"
	testpodNamespace = "test-pods"
	TestNS           = "test1"
	//NameSpace = "sds-local-volume"

	PVCKind               = "PersistentVolumeClaim"
	PVCAPIVersion         = "v1"
	PVCWaitInterval       = 1
	PVCWaitIterationCount = 20
	PVCDeletedStatus      = "Deleted"

	// VVV to remove VVV
	AppTmpPath     = "/app/tmp"
	KubeConfigName = "kube.config"

	PrivKeyName = "id_rsa_test"
	PubKeyName  = "id_rsa_test.pub"

	NamespaceName       = "test1"
	MasterNodeIP        = "10.10.10.180"
	InstallWorkerNodeIp = "10.10.10.181"
	WorkerNode2         = "10.10.10.182"

	RemoteAppPath = "/home/user"

	ConfigTplName        = "config.yml.tpl"
	ConfigName           = "config.yml"
	ResourcesTplName     = "resources.yml.tpl"
	ResourcesName        = "resources.yml"
	UserCreateScriptName = "createuser.sh"

	UbuntuCloudImage     = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	ImageCloudUbuntu2204 = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	ImageCloudDebian11   = "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-genericcloud-amd64.raw"
	//ImageCloudRedOS73 = "https://files.red-soft.ru/redos/7.3/x86_64/iso/redos-MUROM-7.3.4-20231220.0-Everything-x86_64-DVD1.iso"
	ImageCloudRedOS73 = "https://static.storage-e2e.virtlab.flant.com/media/redos733.qcow2"

	ImageYaCloudUbuntu2204 = "b1gbp6lurl0smp6ci3js" // folderID: b1g1oe1s72nr8b95qkgn
	ImageYaCloudDebian11   = "fd81j47dsud5nvq3498i" // family_id: debian-11-oslogin
	ImageYaCloudRedOS73    = "fd8s4p1p4od29db5u8mi" // family_id: redsoft-red-os-standart-server-7-3

	DeckhouseInstallCommand          = "sudo -i docker run --network=host -t -v '/home/user/config.yml:/config.yml' -v '/home/user/:/tmp/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --config=/config.yml"
	DeckhouseResourcesInstallCommand = "sudo -i docker run --network=host -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/:/tmp/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --resources=/resources.yml"

	NodeInstallGenerationCommand = "sudo -i kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-worker -o json | /opt/deckhouse/bin/jq '.data.\"bootstrap.sh\"' -r"
	NodesListCommand             = "sudo -i kubectl get nodes -owide | grep -v NAME | awk '{ print $6 }'"
)

var (
	//fakepubsubNodePort = flag.Int("fakepubsub-node-port", 30303, "The port to use for connecting sub tests with the fakepubsub service (for configuring PUBSUB_EMULATOR_HOST)")
	clusterPathFlag = flag.String("kconfig", "", "The k8s config path for test")
	clusterNameFlag = flag.String("kcontext", "", "The context of cluster to use for test")
	vmOS            = flag.String("virtos", "", "Deploy virtual machine with specified OS")
	NodeRequired    = map[string]Filter{
		"Ubu22": Filter{
			Os: []string{"Ubuntu 22.04"},
		},
		"Ubu24_ultra": Filter{
			Os:      []string{"Ubuntu 24"},
			Kernel:  []string{"5.15.0-122", "5.15.0-128", "5.15.0-127"},
			Kubelet: []string{"v1.28.15"},
		},
		"Deb11": Filter{
			Os:     []string{"Debian 11", "Debian GNU/Linux 11"},
			Kernel: []string{"5.10.0-33-cloud-amd64"},
		},
		"Red7": Filter{
			Os:     []string{"RedOS 7.3", "RED OS MUROM (7.3)"},
			Kernel: []string{"6.1.52-1.el7.3.x86_64"},
		},
		"Alt10": Filter{
			Os: []string{"Alt 10"},
		},
		"Astra": Filter{
			Os: []string{"Astra Linux"},
		},
	}
	SkipFlag = false // TODO not on Prod/Ci
)

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
		return configPath
	}

	if *clusterPathFlag != "" {
		return *clusterPathFlag
	}

	return os.Getenv("kubeconfig")
}
