package sds_node_configurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"path/filepath"
	"testing"
	"time"
)

func TestRemoveLVG(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		t.Error("Kubeclient problem", err)
	}

	listDevice := &snc.LvmVolumeGroupList{}
	err = cl.List(ctx, listDevice)
	if err != nil {
		t.Error("lvmVolumeGroup list error", err)
	}

	for _, item := range listDevice.Items {
		err = cl.Delete(ctx, &item)
		if err != nil {
			t.Error("lvmVolumeGroup delete error", err)
		}
	}

	time.Sleep(5 * time.Second)

	for {
		allLVGRun := true
		listLVG := snc.LvmVolumeGroupList{}
		err = cl.List(ctx, &listLVG)
		if err != nil {
			t.Error("LVG retrieve failed", err)
		}

		for _, lvg := range listLVG.Items {
			if lvg.Status.Phase != "Ready" {
				allLVGRun = false
			}
		}
		if allLVGRun {
			break
		}
	}
	for _, ip := range []string{"10.10.10.180", "10.10.10.181", "10.10.10.182"} {
		client := funcs.GetSSHClient(ip, "user")
		out, _ := funcs.GetPVDisplay(client)
		fmt.Printf(out)
		out, _ = funcs.GetVGDisplay(client)
		fmt.Printf(out)
	}
}
