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

package clusterManagement

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	"github.com/melbahja/goph"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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

func InitClusterCreate() {
	var out []byte

	if _, err := os.Stat(funcs.AppTmpPath); os.IsNotExist(err) {
		funcs.LogFatalIfError(os.Mkdir(funcs.AppTmpPath, 0644), "Cannot create temp dir")
	}

	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		funcs.LogFatalIfError(err, fmt.Sprintf("kubeclient connection problem %s", err))
	}

	err = funcs.CreateNamespace(ctx, cl, funcs.NamespaceName)
	funcs.LogFatalIfError(err, "", fmt.Sprintf("namespaces \"%s\" already exists", funcs.NamespaceName))

	sshPubKeyString := funcs.CheckAndGetSSHKeys(funcs.AppTmpPath, funcs.PrivKeyName, funcs.PubKeyName)

	for _, vmItem := range [][]string{
		{"vm1", funcs.MasterNodeIP, "4", "8Gi", "linstor-r1", funcs.UbuntuCloudImage},
		{"vm2", funcs.InstallWorkerNodeIp, "2", "4Gi", "linstor-r1", funcs.UbuntuCloudImage},
		{"vm3", funcs.WorkerNode2, "2", "4Gi", "linstor-r1", funcs.UbuntuCloudImage},
	} {
		cpuCount, err := strconv.Atoi(vmItem[2])
		err = funcs.CreateVM(ctx, cl, funcs.NamespaceName, vmItem[0], vmItem[1], cpuCount, vmItem[3], vmItem[4], vmItem[5], sshPubKeyString, 20, 20)
		funcs.LogFatalIfError(err, "", fmt.Sprintf("virtualmachines.virtualization.deckhouse.io \"%s\" already exists", vmItem[0]))
	}

	tries := 600
	allVMUp := true

	for count := 0; count < tries; count++ {
		allVMUp = true
		vmList, err := funcs.ListVM(ctx, cl, funcs.NamespaceName)
		funcs.LogFatalIfError(err, "")
		vmList, err = funcs.ListVM(ctx, cl, funcs.NamespaceName)
		funcs.LogFatalIfError(err, "")
		for _, item := range vmList {
			if item.Status != v1alpha2.MachineRunning {
				allVMUp = false
			}
		}

		if allVMUp {
			break
		}

		time.Sleep(10 * time.Second)

		if count == tries-1 {
			log.Fatal("Timeout waiting for all VMs to be ready")
		}

	}

	licenseKey := os.Getenv("licensekey")
	registryDockerCfg := os.Getenv("registryDockerCfg")

	for _, item := range [][]string{
		{funcs.ConfigTplName, filepath.Join(funcs.AppTmpPath, funcs.ConfigName)},
		{funcs.ResourcesTplName, filepath.Join(funcs.AppTmpPath, funcs.ResourcesName)},
	} {
		template, err := os.ReadFile(item[0])
		funcs.LogFatalIfError(err, "")
		renderedTemplateString := fmt.Sprintf(string(template), registryDockerCfg)
		err = os.WriteFile(item[1], []byte(renderedTemplateString), 0644)
		funcs.LogFatalIfError(err, "")
	}

	auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
	funcs.LogFatalIfError(err, "")

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

	log.Printf("Check Deckhouse existance")
	out, err = masterClient.Run("ls -1 /opt/deckhouse | wc -l")
	funcs.LogFatalIfError(err, string(out))
	if strings.Contains(string(out), "cannot access '/opt/deckhouse'") {
		sshCommandList = append(sshCommandList, fmt.Sprintf(funcs.DeckhouseInstallCommand, funcs.MasterNodeIP))
	}

	sshCommandList = append(sshCommandList, fmt.Sprintf(funcs.DeckhouseResourcesInstallCommand, funcs.MasterNodeIP))

	for _, sshCommand := range sshCommandList {
		log.Printf("command: %s", sshCommand)
		out, err := client.Run(sshCommand)
		funcs.LogFatalIfError(err, string(out))
		log.Printf("output: %s\n", out)
	}

	log.Printf("Nodes listing")

	out, err = masterClient.Run(funcs.NodesListCommand)
	funcs.LogFatalIfError(err, string(out))
	nodeList := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")

	for _, nodeItem := range nodeList {
		fmt.Println("node: ", nodeItem)
	}

	log.Printf("Getting master install script")

	nodeInstallScript := []byte("not found")
	for strings.Contains(string(nodeInstallScript), "not found") {
		nodeInstallScript, err = masterClient.Run(funcs.NodeInstallGenerationCommand)
		funcs.LogFatalIfError(err, "")
	}

	log.Printf("Setting up nodes")

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

	validTokenExists := false
	for !validTokenExists {
		out, err = masterClient.Run(fmt.Sprintf("sudo -i /bin/bash %s", filepath.Join(funcs.RemoteAppPath, funcs.UserCreateScriptName)))
		funcs.LogFatalIfError(err, string(out))
		out, err = masterClient.Run(fmt.Sprintf("cat %s", filepath.Join(funcs.RemoteAppPath, funcs.KubeConfigName)))
		funcs.LogFatalIfError(err, string(out))
		var validBase64 = regexp.MustCompile(`token: [A–Za-z0-9\+/=-_\.]{10,}`)
		validTokenExists = validBase64.MatchString(string(out))
		time.Sleep(10 * time.Second)
	}

	funcs.LogFatalIfError(masterClient.Download(filepath.Join(funcs.RemoteAppPath, funcs.KubeConfigName), filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName)), "")
}
