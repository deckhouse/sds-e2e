package test

import (
	"context"
	"sds-replicated-volume-e2e/funcs"
	"testing"
)

func TestDeleteNamespace(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.DeleteNamespace(ctx, cl, "d8-sds-replicated-volume-e2e-test")
	if err != nil {
		t.Error("namespace delete error", err)
	}
}
