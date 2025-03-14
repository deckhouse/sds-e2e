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

package stress

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	"strconv"
	"testing"
	"time"
)

func TestCreateVm(t *testing.T) {
	if createVM {
		ctx := context.Background()
		cl, err := funcs.NewKubeClient("")
		if err != nil {
			t.Error("kubeclient error", err)
		}

		sshPubKeyString := ""

		for count := 0; count <= 20; count++ {
			vmName := fmt.Sprintf("test-%d", count)
			vmIP := fmt.Sprintf("10.10.10.%d", count+2)
			vmCores := "1"
			vmMemory := "1Gi"
			cpuCount, err := strconv.Atoi(vmCores)
			err = funcs.CreateVM(ctx, cl, testNamespace, vmName, vmIP, cpuCount, vmMemory, storageClassName, funcs.UbuntuCloudImage, sshPubKeyString, 6, 1)
			if err != nil {
				t.Error(fmt.Sprintf("virtualmachine \"%s\" creation problem", vmName), err)
			}
		}

		tries := 600
		allVMUp := true

		for count := 0; count < tries; count++ {
			allVMUp = true
			vmList, err := funcs.ListVM(ctx, cl, testNamespace)
			funcs.LogFatalIfError(err, "")
			vmList, err = funcs.ListVM(ctx, cl, testNamespace)
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
				t.Error("Timeout waiting for all VMs to be ready")
			}
		}
	}
}
