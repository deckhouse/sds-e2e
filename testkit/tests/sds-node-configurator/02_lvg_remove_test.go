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

package sds_node_configurator

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"path/filepath"
	"testing"
)

func TestRemoveLVG(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		t.Error("Kubeclient problem", err)
	}

	listDevice := &snc.LvmVolumeGroupList{}
	err = cl.List(ctx, listDevice)
	if err != nil {
		t.Error("lvmVolumeGroup list error", err)
	}

	for _, item := range listDevice.Items {
		err = cl.Delete(ctx, &item)
		t.Error("lvmVolumeGroup delete error", err)
	}
}
