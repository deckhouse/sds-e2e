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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
)

const ClusterConfigYaml = "00_cluster_deploy.yaml"

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

func TestClusterDeploy(t *testing.T) {
	// Get the test file directory
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	yamlPath := filepath.Join(testDir, ClusterConfigYaml)

	definition, hvCluster, err := util.UnmarshalYAMLConfig(yamlPath)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML config: %v", err)
	}

	// Ensure hosts are ready
	if err := util.EnsureHostsReady(hvCluster, definition); err != nil {
		t.Fatalf("Failed to ensure hosts are ready: %v", err)
	}

	// Prepare DKP cluster configuration
	licenseKey := os.Getenv("licensekey")
	dkpConfig := util.DKPClusterConfig{
		ClusterDefinition: definition,
		KubernetesVersion: "Automatic",
		PodSubnetCIDR:     "10.112.0.0/16",
		ServiceSubnetCIDR: "10.225.0.0/16",
		ClusterDomain:     "cluster.local",
		LicenseKey:        licenseKey,
		RegistryRepo:      "dev-registry.deckhouse.io/sys/deckhouse-oss",
	}

	// Ensure DKP cluster is ready
	cluster, err := util.EnsureDKPClusterReady(dkpConfig, hvCluster)
	if err != nil {
		t.Fatalf("Failed to ensure DKP cluster is ready: %v", err)
	}

	// Ensure modules are ready
	if err := util.EnsureModulesReady(cluster, testModules); err != nil {
		t.Fatalf("Failed to ensure modules are ready: %v", err)
	}

	util.Infof("Cluster deployment completed successfully")
}
