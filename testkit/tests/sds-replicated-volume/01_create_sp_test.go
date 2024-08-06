package sds_replicated_volume

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	"testing"
	"time"
)

func TestCreatePool(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.CreateReplicatedStoragePool(ctx, cl)
	if err != nil {
		t.Error("Pool creation error", err)
	}

	time.Sleep(time.Second * 10)
}
