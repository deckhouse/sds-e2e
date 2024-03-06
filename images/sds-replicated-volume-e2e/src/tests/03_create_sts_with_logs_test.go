package test

import (
	"context"
	"sds-replicated-volume-e2e/funcs"
	"testing"
)

func TestCreateStsLogs(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.CreateLogSts(ctx, cl, "d8-sds-replicated-volume-e2e-test")
	if err != nil {
		t.Error("sts creation error", err)
	}

	funcs.WaitForLogStsPods(ctx, cl, "d8-sds-replicated-volume-e2e-test")
}
