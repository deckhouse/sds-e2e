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
	"fmt"
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

	fmt.Printf("err: %v\n", err)

	vmList, err := funcs.ListVM(ctx, cl, namespaceName)
	for _, item := range vmList {
		fmt.Printf("%s\n", item.Name)
	}

	err = funcs.CreateVM(ctx, cl, namespaceName, "vm1", "10.10.10.180", 4, "8Gi", "linstor-thin-r1", "http://static.storage-e2e.virtlab.flant.com/media/cloudubuntu22.img")
	fmt.Printf("err: %v\n", err)

}
