package cluster_create

import (
	"flag"
)

var (
	licenseKey        = *flag.String("licensekey", "", "Registry license key")
	registryDockerCfg = *flag.String("registryDockerCfg", "", "Registry docker config")
)
