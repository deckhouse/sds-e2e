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
	"log"
	"os"
)

func main() {
	_, err := test.NewKubeClient()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	cl, err := test.NewKubeClient()
	//	if err != nil {
	//		t.Error("kubeclient error", err)
	//	}

	namespaceName := "test1"

	err = funcs.CreateNamespace(ctx, cl, namespaceName)
	if err != nil {
		if err.Error() != fmt.Sprintf("namespaces \"%s\" already exists", namespaceName) {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	vmList, err := funcs.ListVM(ctx, cl, namespaceName)
	for _, item := range vmList {
		fmt.Printf("%s\n", item.Name)
	}

	if _, err := os.Stat("./id_rsa_test"); err == nil {
		fmt.Printf("RSA keys exists")
	} else if errors.Is(err, os.ErrNotExist) {
		funcs.GenerateRSAKeys("./id_rsa_test", "./id_rsa_test.pub")
	}

	sshPubKey, err := os.ReadFile("./id_rsa_test.pub")
	if err != nil {
		log.Fatal(err)
	}
	sshPubKeyString := string(sshPubKey)

	fmt.Printf(sshPubKeyString)

	//output, err := funcs.RemoteRun("user", "10.10.10.181", "user", "ls -l /")
	//fmt.Printf("output: %v\n", output)
	//fmt.Printf("err: %v\n", err)
	//
	//output, err = funcs.RemoteRun("user", "10.10.10.181", "user", "sudo apt update && sudo apt -y install docker.io")
	//fmt.Printf("output: %v\n", output)
	//fmt.Printf("err: %v\n", err)
	//
	//licenseKey := os.Getenv("licensekey")
	//fmt.Printf(licenseKey)
	//sshCommand := fmt.Sprintf("sudo docker login -u license-token -p %s dev-registry.deckhouse.io", licenseKey)
	////" && mkdir -p .ssh && sudo docker run --pull=always -t -v '/home/user/config.yml:/config.yml' dev-registry.deckhouse.io/sys/deckhouse-oss/install:main dhctl bootstrap --ssh-user=user --ssh-host=10.10.10.180 --ssh-password=user --config=/config.yml", licenseKey)
	//
	//output, err = funcs.RemoteRun("user", "10.10.10.181", "user", sshCommand)
	//fmt.Printf("output: %v\n", output)
	//fmt.Printf("err: %v\n", err)

	err = funcs.CreateVM(ctx, cl, namespaceName, "vm1", "10.10.10.180", 4, "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img", sshPubKeyString)
	fmt.Printf("err: %v\n", err)
	err = funcs.CreateVM(ctx, cl, namespaceName, "vm2", "10.10.10.181", 4, "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img", sshPubKeyString)
	fmt.Printf("err: %v\n", err)
	err = funcs.CreateVM(ctx, cl, namespaceName, "vm3", "10.10.10.182", 4, "8Gi", "linstor-r1", "https://cloud-images.ubuntu.com/jammy/20240306/jammy-server-cloudimg-amd64.img", sshPubKeyString)
	fmt.Printf("err: %v\n", err)

	vmList, err = funcs.ListVM(ctx, cl, namespaceName)
	for _, item := range vmList {
		fmt.Printf(item.Name, item.Status)
	}

}
