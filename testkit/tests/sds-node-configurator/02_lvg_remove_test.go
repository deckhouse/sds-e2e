package sds_node_configurator

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"path/filepath"
	"testing"
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
		t.Error("lvmVolumeGroup delete error", err)
	}
}
