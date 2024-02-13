package test

import (
	"context"
	"sds-drbd-e2e/funcs"
	"testing"
)

func TestCreatePool(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	funcs.CreatePools(ctx, cl, t)
}
