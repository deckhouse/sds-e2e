package test

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sds-replicated-volume-e2e/funcs"
	"testing"
	"time"
)

func TestWaitStsPods(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	tries := 600

	for count := 0; count < tries; count++ {
		fmt.Printf("Wait for all pods to be ready\n")

		allPodsReady := true
		podList, err := funcs.ListPods(ctx, cl, testNamespace)
		if err != nil {
			t.Error("Pod list error", err)
		}
		for _, item := range podList {
			if item.Phase != corev1.PodRunning {
				allPodsReady = false
			}
		}

		if allPodsReady {
			break
		}

		time.Sleep(time.Second * 10)

		if count == tries-1 {
			t.Errorf("Timeout waiting for all pods to be ready")
		}

	}

	time.Sleep(time.Second * 10)
}
