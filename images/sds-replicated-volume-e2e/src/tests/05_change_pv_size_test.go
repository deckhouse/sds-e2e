package test

import (
	"context"
	"sds-replicated-volume-e2e/funcs"
	"testing"
)

func TestChangeStsPvcSize(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	pvcsNames, err := funcs.ListPvcNames(ctx, cl, "d8-sds-replicated-volume-e2e-test")
	for _, pvcName := range pvcsNames {
		err = funcs.ChangePvcSize(ctx, cl, "d8-sds-replicated-volume-e2e-test", pvcName, "1.1Gi")
		if err != nil {
			t.Error("Pods waiting error", err)
		}
	}
}
