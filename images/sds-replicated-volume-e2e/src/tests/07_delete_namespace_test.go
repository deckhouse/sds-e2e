package test

import (
	"context"
	"fmt"
	"sds-replicated-volume-e2e/funcs"
	"testing"
	"time"
)

func TestDeleteNamespace(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.DeleteNamespace(ctx, cl, testNamespace)
	if err != nil {
		t.Error("namespace delete error", err)
	}

	tries := 600

	for count := 0; count < tries; count++ {
		fmt.Printf("Wait for namespace %s to be deleted\n", testNamespace)

		namespaceList, err := funcs.ListNamespace(ctx, cl, testNamespace)
		if err != nil {
			t.Error("Namespace list error", err)
		}
		if len(namespaceList) == 0 {
			break
		}

		time.Sleep(time.Second * 10)

		if count == tries-1 {
			t.Errorf("Timeout waiting for namespace %s to be deleted", testNamespace)
		}

	}

}
