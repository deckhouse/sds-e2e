package cluster_create

import "os"

var (
	licenseKey        = os.Getenv("licensekey")
	registryDockerCfg = os.Getenv("registryDockerCfg")
)
