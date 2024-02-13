package test

import (
	"context"
	"sds-node-configurator-e2e/funcs"
	"testing"
)

func Test11(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient(t)
	if err != nil {
		t.Error("kubeclient error", err)
	}

	t.Logf("test")

	devices, err := funcs.GetAPIBlockDevices(ctx, cl, t)
	if err != nil {
		t.Error("get error", err)
	}

	t.Logf("%#v", devices)
}
