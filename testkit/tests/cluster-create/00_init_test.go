package cluster_create

import (
	"flag"
	"os"
	"testing"
)

var licenseKey = ""
var registryDockerCfg = ""

func TestMain(m *testing.M) {
	licenseKey = *flag.String("licenseKey", "", "Registry license key")
	registryDockerCfg = *flag.String("registryDockerCfg", "", "Registry docker config")

	flag.Parse()

	exitVal := m.Run()

	os.Exit(exitVal)
}
