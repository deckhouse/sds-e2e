/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
		pvcs := corev1.PersistentVolumeClaimList{}
		opts := client.ListOption(&client.ListOptions{Namespace: testNamespace})
		err := cl.List(ctx, &pvcs, opts)
		if err != nil {
			t.Error("PVC size change error", err)
		}

		allPvcChanged = true
		for _, pvc := range pvcs.Items {
			parse := resource.MustParse(pvResizedSize)
			if parse.Size() != pvc.Status.Capacity.Storage().Size() {
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
