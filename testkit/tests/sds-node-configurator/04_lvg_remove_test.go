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
				t.Log(fmt.Sprintf("Waiting: LVG %s status: %s", lvg.Name, lvg.Status.Phase))
			}
		}
		if allLVGRun {
			break
		}
	}
}
