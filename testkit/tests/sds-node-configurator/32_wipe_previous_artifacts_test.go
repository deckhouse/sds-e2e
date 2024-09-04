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

func TestRemovePreviousArtifactsBeforeThin(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		funcs.LogFatalIfError(err, "kubeclient problem")
	}

	t.Log(fmt.Sprintf("Waiting: deleting LVG"))

	listDevice := &snc.LvmVolumeGroupList{}
	err = cl.List(ctx, listDevice)
	if err != nil {
		t.Error("lvmVolumeGroup list error", err)
	}

	for _, item := range listDevice.Items {
		item.Finalizers = []string{}
		err = cl.Update(ctx, &item)
		if err != nil {
			t.Error("lvmVolumeGroup update error", err)
		}
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

	t.Log(fmt.Sprintf("LVG deleted"))

	t.Log(fmt.Sprintf("Waiting: deleting BD"))

	listBlockDevice := &snc.BlockDeviceList{}
	err = cl.List(ctx, listBlockDevice)
	if err != nil {
		t.Error("BlockDevice list error", err)
	}

	for _, item := range listBlockDevice.Items {
		item.Finalizers = []string{}
		err = cl.Update(ctx, &item)
		if err != nil {
			t.Error("BlockDevice update error", err)
		}
		err = cl.Delete(ctx, &item)
		if err != nil {
			t.Error("BlockDevice delete error", err)
		}
	}

	time.Sleep(5 * time.Second)

	t.Log(fmt.Sprintf("BlockDevice deleted"))

}
