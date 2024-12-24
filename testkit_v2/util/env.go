package integration

import (
	"os"
	"flag"
)

const (
	defaultNamespace = "default"
	testpodNamespace = "test-pods"
	testNS = "test1"

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

	UbuntuCloudImage = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	ImageCloudUbuntu2204 = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
	ImageCloudDebian11 = "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-genericcloud-amd64.raw"
	//ImageCloudRedOS73 = "https://files.red-soft.ru/redos/7.3/x86_64/iso/redos-MUROM-7.3.4-20231220.0-Everything-x86_64-DVD1.iso"
	ImageCloudRedOS73 = "https://static.storage-e2e.virtlab.flant.com/media/redos733.qcow2"

	ImageYaCloudUbuntu2204 = "b1gbp6lurl0smp6ci3js"  // folderID: b1g1oe1s72nr8b95qkgn
	ImageYaCloudDebian11 = "fd81j47dsud5nvq3498i"  // family_id: debian-11-oslogin
	ImageYaCloudRedOS73 = "fd8s4p1p4od29db5u8mi"  // family_id: redsoft-red-os-standart-server-7-3


	DeckhouseInstallCommand          = "sudo -i docker run --network=host -t -v '/home/user/config.yml:/config.yml' -v '/home/user/:/tmp/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --config=/config.yml"
	DeckhouseResourcesInstallCommand = "sudo -i docker run --network=host -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/:/tmp/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --resources=/resources.yml"

	NodeInstallGenerationCommand = "sudo -i kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-worker -o json | /opt/deckhouse/bin/jq '.data.\"bootstrap.sh\"' -r"
	NodesListCommand             = "sudo -i kubectl get nodes -owide | grep -v NAME | awk '{ print $6 }'"
)

var (
	//fakepubsubNodePort = flag.Int("fakepubsub-node-port", 30303, "The port to use for connecting sub tests with the fakepubsub service (for configuring PUBSUB_EMULATOR_HOST)")
	clusterPathFlag = flag.String("kubeconf", "", "The k8s config path for test")
	clusterNameFlag = flag.String("kubecontext", "", "The context of cluster to use for test")
	vmOS = flag.String("virtos", "", "Deploy virtual machine with specified OS")
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
