package test

import (
	"context"
	"sds-replicated-volume-e2e/funcs"
	"testing"
	"time"
)

func TestCreateStsLogs(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.CreateSts(ctx, cl, testNamespace, 15)
	if err != nil {
		t.Error("sts creation error", err)
	}

	time.Sleep(time.Second * 10)
}
