package sds_replicated_volume

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	"testing"
	"time"
)

func TestDeleteStsPods(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.DeleteSts(ctx, cl, testNamespace)
	if err != nil {
		t.Error("Sts delete error", err)
	}

	time.Sleep(time.Second * 10)
}
