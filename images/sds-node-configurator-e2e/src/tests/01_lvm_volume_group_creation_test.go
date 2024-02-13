package test

import (
	"context"
	"sds-node-configurator-e2e/funcs"
	"testing"
)

func TestLvmVolumeGroupCreation(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	devices, err := funcs.GetAPIBlockDevices(ctx, cl, t)
	if err != nil {
		t.Error("get error", err)
	}

	for _, item := range devices {
		t.Log(funcs.CreateLvmVolumeGroup(ctx, cl, t, item.Status.NodeName, []string{item.ObjectMeta.Name}))
	}
}
