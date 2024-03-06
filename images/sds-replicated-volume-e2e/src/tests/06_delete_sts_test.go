package test

import (
	"context"
	"sds-replicated-volume-e2e/funcs"
	"testing"
	"time"
)

func TestDeleteStsPods(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.DeleteSts(ctx, cl, testNamespace)
	if err != nil {
		t.Error("Sts delete error", err)
	}

	time.Sleep(time.Second * 10)
}
