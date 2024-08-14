package cluster_create

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestCreateLVG(t *testing.T) {
	if _, err := os.Stat(funcs.AppTmpPath); os.IsNotExist(err) {
		funcs.LogFatalIfError(os.Mkdir(funcs.AppTmpPath, 0644), "Cannot create temp dir")
	}

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
			t.Error("Timeout waiting for all VMs to be ready")
		}

	}

}
