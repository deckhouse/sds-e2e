/*
Copyright 2024 Flant JSC

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

package main

import (
	"cluster-management/funcs"
	"cluster-management/tests"
	"context"
	"errors"
	"fmt"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	"github.com/melbahja/goph"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup

const (
	namespaceName       = "test1"
	masterNodeIP        = "10.10.10.180"
	installWorkerNodeIp = "10.10.10.181"
	workerNode2         = "10.10.10.182"
)

func logFatalIfError(err error, exclude ...string) {
	if err != nil {
		if len(exclude) > 0 {
			for _, excludeError := range exclude {
				if err.Error() == excludeError {
					return
				}
			}
		}
		log.Fatal(err.Error())
	}
}

func checkAndGetSSHKeys() (sshPubKeyString string) {
	if _, err := os.Stat("./id_rsa_test"); err == nil {
	} else if errors.Is(err, os.ErrNotExist) {
		funcs.GenerateRSAKeys("./id_rsa_test", "./id_rsa_test.pub")
	}

	sshPubKey, err := os.ReadFile("./id_rsa_test.pub")
	if err != nil {
		log.Fatal(err.Error())
	}

	return string(sshPubKey)
}

func nodeInstall(nodeIP string, installScript string, username string, auth goph.Auth) (out []byte) {
	defer wg.Done()
	nodeClient, err := goph.NewUnknown(username, nodeIP, auth)
	logFatalIfError(err)
	fmt.Printf("Install node %s\n", nodeIP)

	out, err = nodeClient.Run(fmt.Sprintf("base64 -d <<< %s | sudo -i bash", installScript))
	fmt.Printf(string(out))
	logFatalIfError(err)

	nodeClient.Close()

	return out
}

func main() {
	_, err := test.NewKubeClient()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	cl, err := test.NewKubeClient()

	err = funcs.CreateNamespace(ctx, cl, namespaceName)
	logFatalIfError(err, fmt.Sprintf("namespaces \"%s\" already exists", namespaceName))

	sshPubKeyString := checkAndGetSSHKeys()

	for _, vmItem := range [][]string{
		{"vm1", masterNodeIP, "4", "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img"},
		{"vm2", installWorkerNodeIp, "4", "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img"},
		{"vm3", workerNode2, "4", "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img"},
	} {
		cpuCount, err := strconv.Atoi(vmItem[2])
		err = funcs.CreateVM(ctx, cl, namespaceName, vmItem[0], vmItem[1], cpuCount, vmItem[3], vmItem[4], vmItem[5], sshPubKeyString)
		logFatalIfError(err, fmt.Sprintf("virtualmachines.virtualization.deckhouse.io \"%s\" already exists", vmItem[0]))
	}

	tries := 600
	allVMUp := true

	for count := 0; count < tries; count++ {
		allVMUp = true
		vmList, err := funcs.ListVM(ctx, cl, namespaceName)
		logFatalIfError(err)
		vmList, err = funcs.ListVM(ctx, cl, namespaceName)
		logFatalIfError(err)
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
	clusterConfig, err := os.ReadFile("config.yml.tpl")
	logFatalIfError(err)
	clusterConfigString := fmt.Sprintf(string(clusterConfig), registryDockerCfg, "%s")
	err = os.WriteFile("config.yml", []byte(clusterConfigString), 0644)
	logFatalIfError(err)

	auth, err := goph.Key("./id_rsa_test", "")
	logFatalIfError(err)

	goph.DefaultTimeout = 0

	var client *goph.Client

	tries = 600
	for count := 0; count < tries; count++ {
		client, err = goph.NewUnknown("user", installWorkerNodeIp, auth)
		if err == nil {
			break
		}

		time.Sleep(10 * time.Second)

		if count == tries-1 {
			log.Fatal("Timeout waiting for installer VM to be ready")
		}
	}
	logFatalIfError(err)

	defer client.Close()

	for _, item := range [][]string{
		{"config.yml", "/home/user/config.yml"},
		{"id_rsa_test", "/home/user/.ssh/id_rsa_test"},
		{"resources.yml", "/home/user/resources.yml"},
	} {
		err = client.Upload(item[0], item[1])
		logFatalIfError(err)
	}

	sshCommandList := []string{
		"sudo apt update && sudo apt -y install docker.io",
		fmt.Sprintf("sudo docker login -u license-token -p %s dev-registry.deckhouse.io", licenseKey),
	}

	var masterClient *goph.Client

	tries = 600
	for count := 0; count < tries; count++ {
		masterClient, err = goph.NewUnknown("user", masterNodeIP, auth)
		if err == nil {
			break
		}

		time.Sleep(10 * time.Second)

		if count == tries-1 {
			log.Fatal("Timeout waiting for master VM to be ready")
		}
	}
	defer masterClient.Close()

	log.Printf("Check Deckhouse existance")
	logFatalIfError(err)
	fmt.Printf(string(out))
	if strings.Contains(string(out), "cannot access '/opt/deckhouse'") {
		sshCommandList = append(sshCommandList, fmt.Sprintf("sudo docker run -t -v '/home/user/config.yml:/config.yml' -v '/home/user/.ssh/:/tmp/.ssh/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/.ssh/id_rsa_test --config=/config.yml", masterNodeIP))
	}
	sshCommandList = append(sshCommandList, fmt.Sprintf("sudo docker run -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/.ssh/:/tmp/.ssh/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=%s --ssh-agent-private-keys=/tmp/.ssh/id_rsa_test --resources=/resources.yml", masterNodeIP))

	for _, sshCommand := range sshCommandList {
		log.Printf("command: %s", sshCommand)
		out, err := client.Run(sshCommand)
		logFatalIfError(err)
		log.Printf("output: %s\n", out)
	}

	tries = 600
	for count := 0; count < tries; count++ {
		out, err = masterClient.Run("sudo /opt/deckhouse/bin/kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-worker -o json | jq '.data.\"bootstrap.sh\"' -r")
		if err == nil {
			break
		}

		time.Sleep(10 * time.Second)

		if count == tries-1 {
			log.Fatal("Timeout while retrieving node list")
		}
	}
	nodeList := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")

	nodeInstallScript, err := masterClient.Run("sudo /opt/deckhouse/bin/kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-worker -o json | jq '.data.\"bootstrap.sh\"' -r")
	logFatalIfError(err)

	for _, newNodeIP := range []string{installWorkerNodeIp, workerNode2} {
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
