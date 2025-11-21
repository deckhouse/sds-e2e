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

// Modules definition for the cluster deployment
var testModules = []util.ModuleConfig{
	{
		Name:         "snapshot-controller",
		Version:      1,
		Enabled:      true,
		Settings:     map[string]any{},
		Dependencies: []string{},
	},
	{
		Name:    "sds-node-configurator",
		Version: 1,
		Enabled: true,
		Settings: map[string]any{
			"enableThinProvisioning": true,
		},
		Dependencies: []string{"sds-local-volume"},
	},
	{
		Name:         "sds-local-volume",
		Version:      1,
		Enabled:      true,
		Settings:     map[string]any{},
		Dependencies: []string{"snapshot-controller"},
	},
	{
		Name:         "sds-replicated-volume",
		Version:      1,
		Enabled:      true,
		Settings:     map[string]any{},
		Dependencies: []string{"sds-node-configurator"},
	},
}

// Cluster definition for healthcheck test
var testClusterDefinition = util.ClusterDefinition{
	Masters: []util.ClusterNode{
		{
			Hostname: "master-1",
			HostType: util.HostTypeVM,
			Role:     util.ClusterRoleMaster,
			OSType:   util.OSTypeMap["Ubuntu 22.04 6.2.0-39-generic"],
			Auth: util.NodeAuth{
				Method: util.AuthMethodSSHKey,
				User:   "user",
				SSHKey: "", // Public key that will be deployed to the node - value or filepath
			},
			CPU:      4,
			RAM:      8,
			DiskSize: 30,
		},
	},
	Workers: []util.ClusterNode{
		{
			Hostname: "worker-1",
			HostType: util.HostTypeVM,
			Role:     util.ClusterRoleWorker,
			OSType:   util.OSTypeMap["Ubuntu 22.04 6.2.0-39-generic"],
			Auth: util.NodeAuth{
				Method: util.AuthMethodSSHKey,
				User:   "user",
				SSHKey: "", // Public key that will be deployed to the node - value or filepath
			},
			CPU:      2,
			RAM:      6,
			DiskSize: 30,
		},
		{
			Hostname: "worker-2",
			HostType: util.HostTypeVM,
			Role:     util.ClusterRoleWorker,
			OSType:   util.OSTypeMap["Ubuntu 22.04 6.2.0-39-generic"],
			Auth: util.NodeAuth{
				Method: util.AuthMethodSSHKey,
				User:   "user",
				SSHKey: "", // Public key that will be deployed to the node - value or filepath
			},
			CPU:      2,
			RAM:      6,
			DiskSize: 30,
		},
	},
}

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
