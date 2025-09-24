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

func TestNodeHealthCheck(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	nodeMap := cluster.MapLabelNodes(nil)
	for label, nodes := range nodeMap {
		if len(nodes) == 0 {
			t.Errorf("No %s nodes", label)
		} else {
			util.Infof("%s nodes: %d", label, len(nodes))
		}
	}
}
