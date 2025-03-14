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
	"testing"
	"time"
)

func TestDeleteNamespace(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("kubeclient error", err)
	}

	err = funcs.DeleteNamespace(ctx, cl, testNamespace)
	if err != nil {
		t.Error("namespace delete error", err)
	}

	tries := 600

	for count := 0; count < tries; count++ {
		//		fmt.Printf("Wait for namespace %s to be deleted\n", testNamespace)

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
