package cluster_create

import (
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/melbahja/goph"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallDeckhouse(t *testing.T) {
	var out []byte

	fmt.Println(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName))

	auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
	funcs.LogFatalIfError(err, fmt.Sprintf("access error"))

	goph.DefaultTimeout = 0

	var client *goph.Client
	var masterClient *goph.Client

	client = funcs.GetSSHClient(funcs.InstallWorkerNodeIp, "user", auth)
	defer client.Close()
	masterClient = funcs.GetSSHClient(funcs.MasterNodeIP, "user", auth)
	defer masterClient.Close()

	for _, item := range [][]string{
		{filepath.Join(funcs.AppTmpPath, funcs.ConfigName), filepath.Join(funcs.RemoteAppPath, funcs.ConfigName), "installWorker"},
		{filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), filepath.Join(funcs.RemoteAppPath, funcs.PrivKeyName), "installWorker"},
		{filepath.Join(funcs.AppTmpPath, funcs.ResourcesName), filepath.Join(funcs.RemoteAppPath, funcs.ResourcesName), "installWorker"},
		{funcs.UserCreateScriptName, filepath.Join(funcs.RemoteAppPath, funcs.UserCreateScriptName), "masterNode"},
	} {
		if item[2] == "installWorker" {
			err = client.Upload(item[0], item[1])
		} else {
			err = masterClient.Upload(item[0], item[1])
		}
		funcs.LogFatalIfError(err, "")
	}

	out = []byte("Unable to lock directory")
	for strings.Contains(string(out), "Unable to lock directory") {
		out, err = client.Run("sudo apt update && sudo apt -y install docker.io")
		fmt.Println(string(out))
	}

	sshCommandList := []string{
		fmt.Sprintf("sudo docker login -u license-token -p %s dev-registry.deckhouse.io", licenseKey),
	}

	fmt.Printf("Check Deckhouse existance")
	out, err = masterClient.Run("ls -1 /opt/deckhouse | wc -l")
	funcs.LogFatalIfError(err, string(out))
	if strings.Contains(string(out), "cannot access '/opt/deckhouse'") {
		sshCommandList = append(sshCommandList, fmt.Sprintf(funcs.DeckhouseInstallCommand, funcs.MasterNodeIP))
	}

	for _, sshCommand := range sshCommandList {
		fmt.Printf("command: %s", sshCommand)
		out, err := client.Run(sshCommand)
		funcs.LogFatalIfError(err, string(out))
		fmt.Printf("output: %s\n", out)
	}
}
