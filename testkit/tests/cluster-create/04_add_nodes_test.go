package cluster_create

import (
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/melbahja/goph"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var wg sync.WaitGroup

func nodeInstall(nodeIP string, installScript string, username string, auth goph.Auth) (out []byte) {
	defer wg.Done()
	nodeClient, err := goph.NewUnknown(username, nodeIP, auth)
	funcs.LogFatalIfError(err, "")
	fmt.Printf("Install node %s\n", nodeIP)

	out, err = nodeClient.Run(fmt.Sprintf("base64 -d <<< %s | sudo -i bash", installScript))
	funcs.LogFatalIfError(err, string(out))

	nodeClient.Close()

	return out
}

func TestAddNodes(t *testing.T) {
	var out []byte

	auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
	funcs.LogFatalIfError(err, "")

	goph.DefaultTimeout = 0

	var masterClient *goph.Client
	masterClient = funcs.GetSSHClient(funcs.MasterNodeIP, "user", auth)
	defer masterClient.Close()

	out, err = masterClient.Run(funcs.NodesListCommand)
	funcs.LogFatalIfError(err, string(out))
	nodeList := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")

	for _, nodeItem := range nodeList {
		fmt.Println("node: ", nodeItem)
	}

	fmt.Printf("Getting node install script")

	nodeInstallScript := []byte("not found")
	for strings.Contains(string(nodeInstallScript), "not found") {
		nodeInstallScript, err = masterClient.Run(funcs.NodeInstallGenerationCommand)
		funcs.LogFatalIfError(err, "")
	}

	fmt.Printf("Setting up nodes")

	for _, newNodeIP := range []string{funcs.InstallWorkerNodeIp, funcs.WorkerNode2} {
		needInstall := true
		for _, nodeIP := range nodeList {
			if nodeIP == newNodeIP {
				needInstall = false
			}
		}

		if needInstall == true {
			wg.Add(1)
			go nodeInstall(newNodeIP, strings.ReplaceAll(string(nodeInstallScript), "\n", ""), "user", auth)
		}
	}
	wg.Wait()
}
