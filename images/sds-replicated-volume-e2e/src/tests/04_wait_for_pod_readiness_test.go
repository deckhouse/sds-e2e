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

	for count := 0; count < 60; count++ {
		fmt.Printf("Wait for all pods to be ready")

		allPodsReady := true
		podList, err := funcs.ListPods(ctx, cl, "d8-sds-replicated-volume-e2e-test")
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

		if count == 60 {
			t.Errorf("Timeout waiting for all pods to be ready")
		}

		time.Sleep(time.Second * 5)
	}

	time.Sleep(time.Second * 10)
}
