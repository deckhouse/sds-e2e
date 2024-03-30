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

const (
	namespaceName = "test1"
)

func logFatalIfError(err error, exclude ...string) {
	if len(exclude) > 0 {
		for _, excludeError := range exclude {
			if err.Error() == excludeError {
				return
			}
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}

func checkAndGetSSHKeys() (sshPubKeyString string) {
	if _, err := os.Stat("./id_rsa_test"); err == nil {
	} else if errors.Is(err, os.ErrNotExist) {
		funcs.GenerateRSAKeys("./id_rsa_test", "./id_rsa_test.pub")
	}

	sshPubKey, err := os.ReadFile("./id_rsa_test.pub")
	if err != nil {
		log.Fatal(err)
	}

	return string(sshPubKey)
}

func nodeInstall(nodeIP string, installScript string, username string, auth goph.Auth) (out []byte) {
	nodeClient, err := goph.NewUnknown(username, nodeIP, auth)
	fmt.Printf(err.Error())
	logFatalIfError(err)
	defer nodeClient.Close()

	fmt.Sprintf("base64 -d <<< %s | sudo bash", installScript)

	out, err = nodeClient.Run(fmt.Sprintf("base64 -d <<< %s | sudo bash", installScript))
	logFatalIfError(err)

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
		{"vm1", "10.10.10.180", "4", "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img"},
		{"vm2", "10.10.10.181", "4", "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img"},
		{"vm3", "10.10.10.182", "4", "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img"},
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
			log.Printf("%s, #%v", item.Status, item.Status == v1alpha2.MachineRunning)
		}

		if allVMUp {
			break
		}

		time.Sleep(time.Second * 10)

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
	client, err := goph.NewUnknown("user", "10.10.10.181", auth)
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

	masterClient, err := goph.NewUnknown("user", "10.10.10.180", auth)
	logFatalIfError(err)
	defer masterClient.Close()

	log.Printf("Check Deckhouse existance")
	out, err := masterClient.Run("ls -1 /opt/deckhouse | wc -l")
	logFatalIfError(err)
	if string(out) == "0" {
		sshCommandList = append(sshCommandList, "sudo docker run -t -v '/home/user/config.yml:/config.yml' -v '/home/user/.ssh/:/tmp/.ssh/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap --ssh-user=user --ssh-host=10.10.10.180 --ssh-agent-private-keys=/tmp/.ssh/id_rsa_test --config=/config.yml")
	}
	sshCommandList = append(sshCommandList, "sudo docker run -t -v '/home/user/resources.yml:/resources.yml' -v '/home/user/.ssh/:/tmp/.ssh/' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap-phase create-resources --ssh-user=user --ssh-host=10.10.10.180 --ssh-agent-private-keys=/tmp/.ssh/id_rsa_test --resources=/resources.yml")

	for _, sshCommand := range sshCommandList {
		log.Printf("command: %s", sshCommand)
		out, err := client.Run(sshCommand)
		logFatalIfError(err)
		log.Printf("output: %s\n", out)
	}

	out, err = masterClient.Run("sudo /opt/deckhouse/bin/kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-worker -o json | jq '.data.\"bootstrap.sh\"' -r")
	logFatalIfError(err)
	nodeList := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")

	nodeInstallScript, err := masterClient.Run("sudo /opt/deckhouse/bin/kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-worker -o json | jq '.data.\"bootstrap.sh\"' -r")
	logFatalIfError(err)

	var wg sync.WaitGroup

	for _, newNodeIP := range []string{"10.10.10.181", "10.10.10.182"} {
		needInstall := true
		for _, nodeIP := range nodeList {
			if nodeIP == newNodeIP {
				needInstall = false
			}
		}

		if needInstall == true {
			nodeInstall(newNodeIP, string(nodeInstallScript), "user", auth)
		}
	}
	wg.Wait()
}
