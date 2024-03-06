package test

import (
	"context"
	"fmt"
	"sds-replicated-volume-e2e/funcs"
	"testing"
	"time"
)

func TestChangeStsPvcSize(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	pvcList, err := funcs.ListPvcs(ctx, cl, testNamespace)
	for _, pvc := range pvcList {
		err = funcs.ChangePvcSize(ctx, cl, testNamespace, pvc.Name, "1.1Gi")
		if err != nil {
			t.Error("PVC size change error", err)
		}
	}

	for count := 0; count < 60; count++ {
		fmt.Printf("Wait for all pvc to change size")

		allPvcChanged := true
		pvcList, err = funcs.ListPvcs(ctx, cl, testNamespace)
		if err != nil {

			t.Error("PVC size change error", err)
		}
		for _, pvc := range pvcList {
			if pvc.Size != "1153434Ki" {
				allPvcChanged = false
			}
		}

		if allPvcChanged {
			break
		}

		if count == 60 {
			t.Errorf("Timeout waiting for all pods to be ready")
		}

		time.Sleep(time.Second * 5)
	}

	time.Sleep(time.Second * 10)
}
