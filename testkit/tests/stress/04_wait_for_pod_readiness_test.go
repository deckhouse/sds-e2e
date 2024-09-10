package stress

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

func TestWaitStsPods(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	tries := 600

	for count := 0; count < tries; count++ {
		//		fmt.Printf("Wait for all pods to be ready\n")

		allPodsReady := true
		podList := corev1.PodList{}
		opts := client.ListOption(&client.ListOptions{Namespace: testNamespace})
		err = cl.List(ctx, &podList, opts)
		if err != nil {
			t.Error("pvc list error", err)
		}
		for _, item := range podList.Items {
			if item.Status.Phase != corev1.PodRunning {
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
