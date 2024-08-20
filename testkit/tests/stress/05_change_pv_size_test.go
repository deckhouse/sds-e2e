package stress

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

func TestChangeStsPvcSize(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	pvcList := corev1.PersistentVolumeClaimList{}
	opts := client.ListOption(&client.ListOptions{Namespace: testNamespace})
	err = cl.List(ctx, &pvcList, opts)
	if err != nil {
		t.Error("pvc list error", err)
	}

	for _, pvc := range pvcList.Items {
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(pvResizedSize)
		err = cl.Update(ctx, &pvc)
		if err != nil {
			t.Error("PVC size change error", err)
		}
	}

	allPvcChanged := true
	for count := 0; count < 600; count++ {
		pvcList := corev1.PersistentVolumeClaimList{}
		opts := client.ListOption(&client.ListOptions{Namespace: testNamespace})
		err = cl.List(ctx, &pvcList, opts)
		if err != nil {
			t.Error("pvc list error", err)
		}

		allPvcChanged = true
		for _, pvc := range pvcList.Items {
			if pvc.Spec.Resources.Requests[corev1.ResourceStorage] != resource.MustParse(pvResizedSize) {
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
