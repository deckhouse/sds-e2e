package test

import (
	"context"
	"sds-replicated-volume-e2e/funcs"
	"testing"
)

func TestDeleteStsPods(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.DeleteSts(ctx, cl, "d8-sds-replicated-volume-e2e-test")
	if err != nil {
		t.Error("Sts delete error", err)
	}
}
