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
			vmStorageClass := "linstor-r2"
			cpuCount, err := strconv.Atoi(vmCores)
			err = funcs.CreateVM(ctx, cl, testNamespace, vmName, vmIP, cpuCount, vmMemory, vmStorageClass, funcs.UbuntuCloudImage, sshPubKeyString, 6, 1)
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
