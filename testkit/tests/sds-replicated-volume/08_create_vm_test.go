package sds_replicated_volume

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"strconv"
	"testing"
)

func TestCreateVm(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	sshPubKeyString := ""

	for count := 2; count <= 12; count++ {
		vmName := fmt.Sprintf("test-%d", count)
		vmIP := fmt.Sprintf("10.10.10.%d", count)
		vmCores := "1"
		vmMemory := "1Gi"
		vmStorageClass := "linstor-r2"
		fmt.Println(count)
		cpuCount, err := strconv.Atoi(vmCores)
		err = funcs.CreateVM(ctx, cl, testNamespace, vmName, vmIP, cpuCount, vmMemory, vmStorageClass, funcs.UbuntuCloudImage, sshPubKeyString, 6, 1)
		t.Error(fmt.Sprintf("virtualmachine \"%s\" already exists", vmName), err)
	}
}
