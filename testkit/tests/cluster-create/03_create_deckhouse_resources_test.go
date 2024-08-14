package cluster_create

import (
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/melbahja/goph"
	"path/filepath"
	"testing"
)

func TestCreateDeckhouseResources(t *testing.T) {
	var out []byte

	auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
	funcs.LogFatalIfError(err, "")

	goph.DefaultTimeout = 0

	var masterClient *goph.Client
	masterClient = funcs.GetSSHClient(funcs.MasterNodeIP, "user", auth)
	defer masterClient.Close()

	out, err = masterClient.Run(fmt.Sprintf(funcs.DeckhouseResourcesInstallCommand, funcs.MasterNodeIP))
	funcs.LogFatalIfError(err, string(out))
	fmt.Printf("output: %s\n", out)
}
