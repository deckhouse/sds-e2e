package stress

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	"testing"
	"time"
)

func TestCreateStsLogs(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.CreateSts(ctx, cl, testNamespace, pvSize, stsCount, storageClassName)
	if err != nil {
		t.Error("sts creation error", err)
	}

	time.Sleep(time.Second * 10)
}
