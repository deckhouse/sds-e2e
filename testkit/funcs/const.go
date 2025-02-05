package funcs

const (
	AppTmpPath     = "/app/tmp"
	KubeConfigName = "kube.config"

	PrivKeyName = "id_rsa_test"
	PubKeyName  = "id_rsa_test.pub"

	NamespaceName       = "test1"
	MasterNodeIP        = "10.10.10.80"
	InstallWorkerNodeIp = "10.10.10.81"
	WorkerNode2         = "10.10.10.82"

	RemoteAppPath = "/home/user"

	ConfigTplName        = "config.yml.tpl"
	ConfigName           = "config.yml"
	ResourcesTplName     = "resources.yml.tpl"
	ResourcesName        = "resources.yml"
	UserCreateScriptName = "createuser.sh"

	UbuntuCloudImage = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"

	DeckhouseInstallCommand          = "sudo -i docker run --network=host -t -v '/home/user/config.yml:/config.yml' -v '/home/user/:/tmp/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --config=/config.yml"
	DeckhouseResourcesInstallCommand = "sudo -i docker run --network=host -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/:/tmp/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/id_rsa_test --resources=/resources.yml"

	NodeInstallGenerationCommand = "sudo -i kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-worker -o json | /opt/deckhouse/bin/jq '.data.\"bootstrap.sh\"' -r"
	NodesListCommand             = "sudo -i kubectl get nodes -owide | grep -v NAME | awk '{ print $6 }'"
)
