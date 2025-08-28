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

package integration

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	podWaitIterationCount = 20
	podWaitInterval       = 10
)

func (cluster *KCluster) WaitPodDeletion(name, namespace string) error {
	pod := &v1.Pod{}

	for range podWaitIterationCount {
		сtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		err := cluster.controllerRuntimeClient.Get(сtx, client.ObjectKey{Name: name, Namespace: namespace}, pod)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get Pod %s: %w", name, err)
		}
		time.Sleep(podWaitInterval * time.Second)
	}

	return fmt.Errorf("failed to await pod %s deletion: pod still exists", name)
}
