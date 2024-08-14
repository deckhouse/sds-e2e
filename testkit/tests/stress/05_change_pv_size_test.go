package stress

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	"testing"
	"time"
)

func TestChangeStsPvcSize(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	pvcList, err := funcs.ListPvcs(ctx, cl, testNamespace)
	for _, pvc := range pvcList {
		err = funcs.ChangePvcSize(ctx, cl, testNamespace, pvc.Name, pvResizedSize)
		if err != nil {
			t.Error("PVC size change error", err)
		}
	}

	allPvcChanged := true
	for count := 0; count < 600; count++ {
		pvcList, err = funcs.ListPvcs(ctx, cl, testNamespace)
		if err != nil {

			t.Error("PVC size change error", err)
		}
		allPvcChanged = true
		for _, pvc := range pvcList {
			kiPvResizedSize, err := funcs.ConvertUnit(pvResizedSize, "Ki")
			if err != nil {

				t.Error("PVC size change error (incorrect size)", err)
			}
			if pvc.Size != kiPvResizedSize {
				allPvcChanged = false
			}
		}

		if allPvcChanged {
			break
		}

		time.Sleep(time.Second * 5)
	}

	if allPvcChanged == false {
		t.Errorf("Timeout waiting for all pods to be ready")
	}

	time.Sleep(time.Second * 10)
}
