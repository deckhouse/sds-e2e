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
