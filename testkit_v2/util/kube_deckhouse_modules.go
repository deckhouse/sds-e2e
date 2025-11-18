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
	v1alpha1nfs "github.com/deckhouse/csi-nfs/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SnapshotControllerModuleName      = "snapshot-controller"
	SnapshotControllerModuleNamespace = "d8-snapshot-controller"
	SnapshotControllerDeploymentName  = "snapshot-controller"

	SDSLocalVolumeModuleName                  = "sds-local-volume"
	SDSLocalVolumeModuleNamespace             = "d8-sds-local-volume"
	SDSLocalVolumeCSIControllerDeploymentName = "csi-controller"
	SDSLocalVolumeCSINodeDaemonSetName        = "csi-node"

	SDSNodeConfiguratorModuleName      = "sds-node-configurator"
	SDSNodeConfiguratorModuleNamespace = "d8-sds-node-configurator"
	SDSNodeConfiguratorDaemonSetName   = "sds-node-configurator"

	SDSReplicatedVolumeModuleName               = "sds-replicated-volume"
	SDSReplicatedVolumeModuleNamespace          = "d8-sds-replicated-volume"
	SDSReplicatedVolumeControllerDeploymentName = "sds-replicated-volume-controller"

	ModuleReadyTimeout = 720 // Timeout for module to be ready (in seconds) - 12*60
)

// EnsureModule creates and configure deckhouse module if it does not exist. Check and modify existing module if needed.
func (cluster *KCluster) EnsureSDSNodeConfiguratorModuleEnabled(enableThinProvisioning bool) error {
	sdsNodeConfiguratorModuleConfig := generateModuleConfig(
		SDSNodeConfiguratorModuleName,
		1,    // Version
		true, // Enabled
		map[string]any{
			"enableThinProvisioning": enableThinProvisioning,
		},
	)

	err := cluster.EnsureModuleEnabled(sdsNodeConfiguratorModuleConfig)
	if err != nil {
		return err
	}

	return nil
}

// EnsureModule creates and configure deckhouse module if it does not exist. Check and modify existing module if needed.
func (cluster *KCluster) EnsureSDSReplicatedVolumeModuleEnabled(enableThinProvisioning bool) error {
	err := cluster.EnsureSDSNodeConfiguratorModuleEnabled(enableThinProvisioning)
	if err != nil {
		return err
	}

	// Create or update SDS Replicated Volume module configuration
	// This module is dependent on the SDS Node Configurator module, so it should be created after it.
	sdsReplicatedVolumeModuleConfig := generateModuleConfig(
		SDSReplicatedVolumeModuleName,
		1,    // Version
		true, // Enabled
		map[string]any{},
	)

	err = cluster.EnsureModuleEnabled(sdsReplicatedVolumeModuleConfig)
	if err != nil {
		return err
	}

	return nil
}

func (cluster *KCluster) EnsureModuleEnabled(moduleConfig *v1alpha1nfs.ModuleConfig) error {
	err := cluster.controllerRuntimeClient.Create(cluster.ctx, moduleConfig)

	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
		// Module already exists, update it
		existingModuleConfig := &v1alpha1nfs.ModuleConfig{}
		err = cluster.controllerRuntimeClient.Get(cluster.ctx, ctrlrtclient.ObjectKey{Name: SDSNodeConfiguratorModuleName}, existingModuleConfig)
		if err != nil {
			return err
		}
		moduleConfig.ResourceVersion = existingModuleConfig.ResourceVersion
		err = cluster.controllerRuntimeClient.Update(cluster.ctx, moduleConfig)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cluster *KCluster) WaitUntilSDSReplicatedVolumeModuleReady() error {
	Debugf("Waiting for SDS Replicated Volume module to be ready...")

	return cluster.WaitUntilDeploymentReady(SDSReplicatedVolumeModuleNamespace, SDSReplicatedVolumeControllerDeploymentName, ModuleReadyTimeout)
}

func generateModuleConfig(name string, version int, enabled bool, settings map[string]any) *v1alpha1nfs.ModuleConfig {
	return &v1alpha1nfs.ModuleConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1nfs.ModuleConfigSpec{
			Version:  version,
			Enabled:  ptr.To(enabled),
			Settings: settings,
		},
	}
}
