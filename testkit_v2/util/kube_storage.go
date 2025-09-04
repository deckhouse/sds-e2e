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

	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	srv "github.com/deckhouse/sds-replicated-volume/api/v1alpha1"
	coreapi "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultLVMVolumeGroupNamePrefix = "default-lvg-"
	DefaultLVMVolumeGroupSize       = "10Gi"
	DefaultVGNameOnTheNode          = "vg-default"
)

/*  Block Device  */

type BdFilter struct {
	Name       any
	Node       any
	Consumable any
	Size       float32
}

func (f *BdFilter) Apply(bds []snc.BlockDevice) (resp []snc.BlockDevice) {
	for _, bd := range bds {
		if f.Name != nil && !CheckCondition(f.Name, bd.Name) {
			continue
		}
		if f.Node != nil && !CheckCondition(f.Node, bd.Status.NodeName) {
			continue
		}
		if f.Consumable != nil && !CheckCondition(f.Consumable, bd.Status.Consumable) {
			continue
		}
		if f.Size != 0 {
			sizeB := int64(f.Size) * 1024 * 1024 * 1024
			validDiff := int64(10 * 1024 * 1024)
			if bd.Status.Size.Value() < sizeB-validDiff || bd.Status.Size.Value() > sizeB+validDiff {
				continue
			}
		}

		resp = append(resp, bd)
	}
	return
}

func (cluster *KCluster) ListBD(filters ...BdFilter) ([]snc.BlockDevice, error) {
	bdList := &snc.BlockDeviceList{}
	err := cluster.controllerRuntimeClient.List(cluster.ctx, bdList)
	if err != nil {
		Warnf("Can't get BDs: %s", err.Error())
		return nil, err
	}

	resp := bdList.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (cluster *KCluster) DeleteBd(filters ...BdFilter) error {
	bdList, err := cluster.ListBD(filters...)
	if err != nil {
		return err
	}
	for _, bd := range bdList {
		if err := cluster.controllerRuntimeClient.Delete(cluster.ctx, &bd); err != nil {
			return err
		}
	}

	return nil
}

func (cluster *KCluster) DeleteBdAndWait(filters ...BdFilter) error {
	if err := cluster.DeleteBd(filters...); err != nil {
		return err
	}
	return RetrySec(20, func() error {
		bds, err := cluster.ListBD(filters...)
		if err != nil {
			return err
		}

		if len(bds) > 0 {
			return fmt.Errorf("not deleted BDs: %d (%s, ...)", len(bds), bds[0].Name)
		}
		Debugf("BDs deleted")
		return nil
	})
}

/*  LVM Volume Group  */

type LvgFilter struct {
	Name  any
	Node  any
	Phase any
}

func (f *LvgFilter) Apply(lvgs []snc.LVMVolumeGroup) (resp []snc.LVMVolumeGroup) {
	for _, lvg := range lvgs {
		if f.Name != nil && !CheckCondition(f.Name, lvg.Name) {
			continue
		}
		if f.Node != nil && (len(lvg.Status.Nodes) == 0 || !CheckCondition(f.Node, lvg.Status.Nodes[0].Name)) {
			continue
		}
		if f.Phase != nil && !CheckCondition(f.Phase, lvg.Status.Phase) {
			continue
		}

		resp = append(resp, lvg)
	}
	return
}

func (cluster *KCluster) GetLvg(lvgName string) (*snc.LVMVolumeGroup, error) {
	lvg := snc.LVMVolumeGroup{}
	err := cluster.controllerRuntimeClient.Get(cluster.ctx, ctrlrtclient.ObjectKey{
		Name: lvgName,
	}, &lvg)
	if err != nil {
		return nil, err
	}
	return &lvg, nil
}

// ListLVG returns a list of LVM Volume Groups filtered by the provided filters.
// If no filters are provided, it returns all LVM Volume Groups in the cluster.
// Example usage:
//
//	lvgList, err := cluster.ListLVG(LvgFilter{Name: "my-lvg"})
//	lvgList, err := cluster.ListLVG() // Get all LVGs
//	lvgList, err := cluster.ListLVG(LvgFilter{Node: "node1"}, LvgFilter{Phase: "Ready"})
func (cluster *KCluster) ListLVG(filters ...LvgFilter) ([]snc.LVMVolumeGroup, error) {
	lvgList := &snc.LVMVolumeGroupList{}
	if err := cluster.controllerRuntimeClient.List(cluster.ctx, lvgList); err != nil {
		Warnf("Can't get LVGs: %s", err.Error())
		return nil, err
	}

	resp := lvgList.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (cluster *KCluster) WaitLVGsReady(filters ...LvgFilter) error {
	filtersNotReady := append(filters, LvgFilter{Phase: "!Ready"})
	if err := RetrySec(35, func() error {
		lvgs, err := cluster.ListLVG(filtersNotReady...)
		if err != nil {
			return err
		}
		if len(lvgs) > 0 {
			return fmt.Errorf("LVGs not Ready: %d (%s, ...)", len(lvgs), lvgs[0].Name)
		}
		return nil
	}); err != nil {
		return err
	}

	lvgs, err := cluster.ListLVG(filters...)
	if err != nil {
		return err
	}
	Debugf("LVGs is ready: %d", len(lvgs))
	for _, lvg := range lvgs {
		if len(lvg.Status.Nodes) == 0 {
			return fmt.Errorf("no nodes in LVG %s status", lvg.Name)
		}
	}

	return nil
}

func (cluster *KCluster) CreateLVG(name, nodeName string, bds []string) error {
	Debugf("Creating LVG %s (node %s, bds %v)", name, nodeName, bds)
	lvmVolumeGroup := &snc.LVMVolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: snc.LVMVolumeGroupSpec{
			ActualVGNameOnTheNode: name,
			BlockDeviceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "kubernetes.io/metadata.name", Operator: metav1.LabelSelectorOpIn, Values: bds},
				},
			},
			Type:  "Local",
			Local: snc.LVMVolumeGroupLocalSpec{NodeName: nodeName},
		},
	}
	err := cluster.controllerRuntimeClient.Create(cluster.ctx, lvmVolumeGroup)
	if err != nil {
		Errorf("Can't create LVG %s (node %s, bds %v)", name, nodeName, bds)
		return err
	}
	return nil
}

func (cluster *KCluster) CreateLvgWithCheck(name, nodeName string, bds []string) error {
	if err := cluster.CreateLVG(name, nodeName, bds); err != nil {
		return err
	}

	return cluster.WaitLVGsReady(LvgFilter{Name: name})
}

func (cluster *KCluster) CreateLvgExt(name, nodeName string, ext map[string]any) error {
	lvg := &snc.LVMVolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: snc.LVMVolumeGroupSpec{
			ActualVGNameOnTheNode: name,
			BlockDeviceSelector:   &metav1.LabelSelector{},
			Type:                  "Local",
			Local:                 snc.LVMVolumeGroupLocalSpec{NodeName: nodeName},
		},
	}
	if bds, ok := ext["bds"]; ok {
		lvg.Spec.BlockDeviceSelector.MatchExpressions = []metav1.LabelSelectorRequirement{{
			Key:      "kubernetes.io/metadata.name",
			Operator: metav1.LabelSelectorOpIn,
			Values:   bds.([]string),
		}}
	}
	if tp, ok := ext["thinpools"]; ok {
		lvg.Spec.ThinPools = tp.([]snc.LVMVolumeGroupThinPoolSpec)
	}
	err := cluster.controllerRuntimeClient.Create(cluster.ctx, lvg)
	if err != nil {
		Errorf("Can't create LVG %s/%s", nodeName, name)
		return err
	}
	return nil
}

func (cluster *KCluster) UpdateLVG(lvg *snc.LVMVolumeGroup) error {
	err := cluster.controllerRuntimeClient.Update(cluster.ctx, lvg)
	if err != nil {
		Errorf("Can't update LVG %s", lvg.Name)
		return err
	}

	return nil
}

func (cluster *KCluster) DeleteLVG(filters ...LvgFilter) error {
	lvgs, _ := cluster.ListLVG(filters...)

	for _, lvg := range lvgs {
		if err := cluster.controllerRuntimeClient.Delete(cluster.ctx, &lvg); err != nil {
			return err
		}
	}

	return nil
}

func (cluster *KCluster) DeleteLvgAndWait(filters ...LvgFilter) error {
	if err := cluster.DeleteLVG(filters...); err != nil {
		return err
	}

	return RetrySec(15, func() error {
		lvgs, err := cluster.ListLVG(filters...)
		if err != nil {
			return err
		}
		if len(lvgs) > 0 {
			return fmt.Errorf("LVGs not deleted: %d", len(lvgs))
		}
		Debugf("LVGs deleted")
		return nil
	})
}

/*  Storage Class  */

func (cluster *KCluster) CreateLocalThickStorageClass(name string) (*storagev1.StorageClass, error) {
	lvmType := "Thick"
	lvmVolGroups := "- name: vg-w1\n- name: vg-w2"

	volBindingMode := storagev1.VolumeBindingImmediate

	reclaimPolicy := coreapi.PersistentVolumeReclaimDelete

	volExpansion := true

	sc := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Provisioner: "local.csi.storage.deckhouse.io",
		Parameters: map[string]string{
			"local.csi.storage.deckhouse.io/lvm-type":            lvmType,
			"local.csi.storage.deckhouse.io/volume-binding-mode": string(volBindingMode),
			"local.csi.storage.deckhouse.io/lvm-volume-groups":   lvmVolGroups,
		},
		ReclaimPolicy:        &reclaimPolicy,
		MountOptions:         nil,
		AllowVolumeExpansion: &volExpansion,
		VolumeBindingMode:    &volBindingMode,
	}

	if err := cluster.controllerRuntimeClient.Create(cluster.ctx, sc); err != nil {
		Errorf("Can't create SC %s", sc.Name)
		return nil, err
	}
	return sc, nil
}

func (cluster *KCluster) CreateDefaultStorageClass(name string) (*storagev1.StorageClass, error) {
	enableThinProvisioning := true
	err := cluster.EnsureSDSReplicatedVolumeModuleEnabled(enableThinProvisioning)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure SDS Replicated Volume module enabled: %w", err)
	}

	err = cluster.WaitUntilSDSReplicatedVolumeModuleReady()
	if err != nil {
		return nil, fmt.Errorf("failed to wait until SDS Replicated Volume module is ready: %w", err)
	}

	lvgs, err := cluster.EnsureEveryNodeHasLVMVolumeGroup(DefaultLVMVolumeGroupSize)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure every node has LVM Volume Group: %w", err)
	}

	storagePool, err := cluster.CreateDefaultStoragePool(NestedDefaultReplicatedStoragePoolName, lvgs)
	if err != nil {
		return nil, fmt.Errorf("failed create default StoragePool: %w", err)
	}

	replicatedStorageClass, err := cluster.CreateReplicatedStorageClass(NestedDefaultStorageClass, "None", storagePool.Name)
	if err != nil {
		return nil, fmt.Errorf("failed create default StoragePool: %w", err)
	}

	cwt, cancel := context.WithTimeout(cluster.ctx, 5*time.Second)
	defer cancel()

	storageClass := &storagev1.StorageClass{}
	err = cluster.controllerRuntimeClient.Get(cwt, ctrlrtclient.ObjectKeyFromObject(replicatedStorageClass), storageClass)
	if err != nil {
		return nil, fmt.Errorf("failed create default StorageClass: %w", err)
	}

	return storageClass, nil

}

func (cluster *KCluster) CreateReplicatedStorageClass(drbdStorageClassName, replication, storagePoolName string) (*srv.ReplicatedStorageClass, error) {
	rsc := &srv.ReplicatedStorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: drbdStorageClassName,
		},
		Spec: srv.ReplicatedStorageClassSpec{
			StoragePool:   storagePoolName,
			ReclaimPolicy: "Delete",
			Replication:   replication,
			VolumeAccess:  "Any",
			Topology:      "Ignored",
			Zones:         []string{},
		},
	}

	cwt, cancel := context.WithTimeout(cluster.ctx, 5*time.Second)
	defer cancel()

	err := cluster.controllerRuntimeClient.Create(cwt, rsc)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return rsc, nil
		}
		return nil, fmt.Errorf("failed to create a replicated storage class: %w", err)
	}

	return rsc, nil
}

func (cluster *KCluster) CreateDefaultStoragePool(name string, lvmVolumeGroups []string) (*srv.ReplicatedStoragePool, error) {
	rspLVGs := make([]srv.ReplicatedStoragePoolLVMVolumeGroups, 0, len(lvmVolumeGroups))
	for _, name := range lvmVolumeGroups {
		rspLVGs = append(rspLVGs, srv.ReplicatedStoragePoolLVMVolumeGroups{
			Name:         name,
			ThinPoolName: "",
		})
	}

	storagePool := &srv.ReplicatedStoragePool{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: srv.ReplicatedStoragePoolSpec{
			LVMVolumeGroups: rspLVGs,
			Type:            "LVM",
		},
		Status: srv.ReplicatedStoragePoolStatus{
			Phase: "Updating",
		},
	}

	cwt, cancel := context.WithTimeout(cluster.ctx, 5*time.Second)
	defer cancel()

	err := cluster.controllerRuntimeClient.Create(cwt, storagePool)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return storagePool, nil
		}
		return nil, fmt.Errorf("failed to create a default ReplicatedStoragePool: %s", err.Error())
	}
	return storagePool, nil
}

func (cluster *KCluster) EnsureStorageClass(storageClassName string) (*storagev1.StorageClass, error) {
	storageClass := &storagev1.StorageClass{}

	err := cluster.controllerRuntimeClient.Get(cluster.ctx, ctrlrtclient.ObjectKey{Name: storageClassName}, storageClass)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}

		// StorageClass does not exist, create it
		storageClass, err = cluster.CreateDefaultStorageClass(storageClassName)
		if err != nil {
			return nil, fmt.Errorf("failed to create StorageClass %s: %w", storageClassName, err)
		}
	}

	return storageClass, nil
}

/*  Persistent Volume Claims  */

func (cluster *KCluster) ListPVC(nsName string) ([]coreapi.PersistentVolumeClaim, error) {
	pvcList, err := (*cluster.goClient).CoreV1().PersistentVolumeClaims(nsName).List(cluster.ctx, metav1.ListOptions{})
	if err != nil {
		Debugf("Can't get '%s' PVCs: %s", nsName, err.Error())
		return nil, err
	}

	return pvcList.Items, nil
}

func (cluster *KCluster) CreatePVCInTestNS(name, scName, size string) (*coreapi.PersistentVolumeClaim, error) {
	_, err := cluster.EnsureStorageClass(scName)
	if err != nil {
		return nil, err
	}

	resourceList := make(map[coreapi.ResourceName]resource.Quantity)
	sizeStorage, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
	resourceList[coreapi.ResourceStorage] = sizeStorage
	volMode := coreapi.PersistentVolumeFilesystem

	pvc := coreapi.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: TestNS,
		},
		Spec: coreapi.PersistentVolumeClaimSpec{
			StorageClassName: &scName,
			AccessModes: []coreapi.PersistentVolumeAccessMode{
				coreapi.ReadWriteOnce,
			},
			Resources: coreapi.VolumeResourceRequirements{
				Requests: resourceList,
			},
			VolumeMode: &volMode,
		},
	}

	err = cluster.controllerRuntimeClient.Create(cluster.ctx, &pvc)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return &pvc, nil
		}
		return nil, err
	}
	return &pvc, nil
}

func (cluster *KCluster) WaitPVCStatus(name string) (string, error) {
	pvc := coreapi.PersistentVolumeClaim{}
	for i := 0; i < pvcWaitIterationCount; i++ {
		Infof("Waiting PVC %s to become BOUND. Attempt %d of %d", name, i+1, pvcWaitIterationCount)
		err := cluster.controllerRuntimeClient.Get(cluster.ctx, ctrlrtclient.ObjectKey{
			Name:      name,
			Namespace: TestNS,
		}, &pvc)
		if err != nil {
			Debugf("Get PVC error: %v", err)
		}
		if pvc.Status.Phase == coreapi.ClaimBound {
			return string(pvc.Status.Phase), nil
		}

		if len(pvc.Status.Phase) == 0 {
			return "Deleted", nil
		}

		time.Sleep(pvcWaitInterval * time.Second)
	}
	return string(pvc.Status.Phase), fmt.Errorf("the waiting time %d or the pvc to be ready has expired",
		pvcWaitInterval*pvcWaitIterationCount)
}

func (cluster *KCluster) DeletePVC(name string) error {
	pvc := coreapi.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: TestNS,
		},
	}

	if err := cluster.controllerRuntimeClient.Delete(cluster.ctx, &pvc); err != nil {
		return err
	}
	return nil
}

func (cluster *KCluster) UpdatePVC(pvc *coreapi.PersistentVolumeClaim) error {
	err := cluster.controllerRuntimeClient.Update(cluster.ctx, pvc)
	if err != nil {
		Warnf("Can't update PVC %s", pvc.Name)
		return err
	}

	return nil
}

func (cluster *KCluster) EnsureEveryNodeHasLVMVolumeGroup(lvmVolumeGroupSize string) ([]string, error) {
	readyLVMVolumeGroups, err := cluster.ListLVG(LvgFilter{Phase: "Ready"})
	if err != nil {
		return nil, fmt.Errorf("failed to list LVM Volume Groups: %w", err)
	}

	nodes, err := cluster.ListNode()
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	createdLVGNames := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeName := node.Name
		lvgName := DefaultLVMVolumeGroupNamePrefix + nodeName

		// Check if the LVM Volume Group already exists for the node
		lvgExists := false
		for _, lvg := range readyLVMVolumeGroups {
			if lvg.Name == lvgName {
				lvgExists = true
				break
			}
		}

		if !lvgExists {
			Debugf("Creating LVM Volume Group %s for node %s", lvgName, nodeName)
			bds, err := GetOrCreateConsumableBlockDevices(nodeName, 60, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to create block devices: %s", err.Error())
			}

			bdNames := make([]string, 0, len(bds))
			for _, bd := range bds {
				bdNames = append(bdNames, bd.Name)
			}

			err = cluster.CreateLvgWithCheck(lvgName, nodeName, bdNames)
			if err != nil {
				return nil, fmt.Errorf("failed to create LVM Volume Group %s for node %s: %w", lvgName, nodeName, err)
			}
			createdLVGNames = append(createdLVGNames, lvgName)
		} else {
			Debugf("LVM Volume Group %s already exists for node %s", lvgName, nodeName)
		}
	}

	Debugf("All nodes have LVM Volume Groups")

	return createdLVGNames, nil
}

func (cluster *KCluster) GetPVC(name, nameSpace string) (*v1.PersistentVolumeClaim, error) {
	pvc := &v1.PersistentVolumeClaim{}

	cwt, cancel := context.WithTimeout(cluster.ctx, 5*time.Second)
	defer cancel()

	pvc, err := cluster.goClient.CoreV1().PersistentVolumeClaims(nameSpace).Get(cwt, name, metav1.GetOptions{})
	if err != nil {
		return pvc, err
	}

	return pvc, nil
}

func (cluster *KCluster) GetPV(name, nameSpace string) (*v1.PersistentVolume, error) {
	pv := &v1.PersistentVolume{}

	cwt, cancel := context.WithTimeout(cluster.ctx, 5*time.Second)
	defer cancel()

	err := cluster.controllerRuntimeClient.Get(cwt, ctrlrtclient.ObjectKey{Name: name, Namespace: nameSpace}, pv)
	if err != nil {
		return pv, err
	}

	return pv, nil
}
