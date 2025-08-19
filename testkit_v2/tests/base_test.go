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
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
)

func TestDataExporterBase(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	// nodeConfigurator.TestPrepare
	// sdsReplicatedVolume.TestPrepare

	// context := context.Background()
	// enableThinProvisioning := true
	// err := cluster.EnableSDSNodeConfiguratorModule(enableThinProvisioning)
	// if err != nil {
	// 	t.Fatalf("Failed to enable SDS Node Configurator module: %v", err)
	// }
	// randomNamespaceName := util.RandString(10)

	pvc, err := cluster.CreatePVCInTestNS("test-pvc", util.NestedDefaultStorageClass, "10Gi")
	if err != nil {
		t.Fatalf("Failed to create PVC: %v", err)
	}
	// defer func() {
	// 	if err := cluster.DeletePVCInTestNS(pvc.Name); err != nil {
	// 		t.Fatalf("Failed to delete PVC: %v", err)
	// 	}
	// }()

	util.Infof("Created PVC: %s in namespace: %s", pvc.Name, pvc.Namespace)

	// err = clister.DeletePVCInTestNS(pvc.Name)
}
