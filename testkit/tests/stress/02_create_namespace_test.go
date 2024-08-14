package stress

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	"testing"
	"time"
)

func TestCreateNamespace(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.CreateNamespace(ctx, cl, testNamespace)
	if err != nil {
		t.Error("namespace creation error", err)
	}

	time.Sleep(time.Second * 10)
}
