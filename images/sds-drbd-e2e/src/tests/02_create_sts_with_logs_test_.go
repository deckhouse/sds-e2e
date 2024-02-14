package test

import (
	"context"
	"sds-drbd-e2e/funcs"
	"testing"
)

func TestCreateStsLogs(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	funcs.CreateLogSts(ctx, cl)
}
