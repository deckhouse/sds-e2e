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
	"fmt"
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
	coreapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	scName = "test-lvm-thick-immediate-retain"
)

func TestPVC(t *testing.T) {
	t.Run("PVC creating", testPVCCreate)
	t.Run("PVC resizing", testPVCResize)
	t.Run("PVC deleting", testPVCDelete)
}

func testPVCCreate(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	_, _ = cluster.CreateLocalThickStorageClass(scName)

	pvc, err := cluster.CreatePVCInTestNS("test-pvc", scName, "1Gi", false)
	if err != nil {
		t.Fatal(err)
	}

	pvcStatus, err := cluster.WaitPVCStatus(pvc.Name)
	if err != nil {
		util.Debugf("PVC %s status: %s", pvc.Name, pvcStatus)
	}
}

func testPVCResize(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	pvcList, err := cluster.ListPVC(util.TestNS)
	if err != nil {
		t.Error("PVC getting:", err)
	}
	for _, pvc := range pvcList {
		origSize := pvc.Spec.Resources.Requests[coreapi.ResourceStorage]
		newSize := resource.MustParse("2Gi")

		// Update the PVC size
		util.Debugf("PVC %s size: %#v", pvc.Name, pvc.Size())
		util.Infof("PVC resize %s -> %s", origSize.String(), newSize.String())
		pvc.Spec.Resources.Requests[coreapi.ResourceStorage] = newSize
		if err := cluster.UpdatePVC(&pvc); err != nil {
			t.Error(fmt.Sprintf("PVC %s resizing (%s to %s) problem:", pvc.Name, origSize.String(), newSize.String()), err)
		}
	}
}

func testPVCDelete(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	err := cluster.DeletePVC("test-pvc")
	if err != nil {
		t.Error(err)
	}
}
