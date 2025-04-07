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

func TestNode(t *testing.T) {
	clr := util.GetCluster("", "")

	nodeMap := clr.MapLabelNodes(nil)

	astraNodes, ok := nodeMap["Astra"]
	if !ok || len(astraNodes) == 0 {
		t.Error("No Astra node - not good")
	}

	ubuntuNodes, ok := nodeMap["Ubu22"]
	if !ok || len(ubuntuNodes) < 2 {
		t.Fatal("Few Ubuntu 22 nodes")
	}
}
