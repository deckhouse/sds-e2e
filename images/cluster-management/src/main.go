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
	"cluster-management/tests"
	"cluster-management/v1alpha2"
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	namespaceName := "default"
	objs := v1alpha2.VirtualMachineList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err = cl.List(ctx, &objs, opts)
	print("1111")
	for _, item := range objs.Items {
		print(item)
	}
}
