package funcs

import (
	"context"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StorageClassKind                                  = "StorageClass"
	StorageClassAPIVersion                            = "storage.k8s.io/v1"
	StorageClassVolumeBindingModeImmediate            = "Immediate"
	StorageClassVolumeBindingModeWaitForFirstConsumer = "WaitForFirstConsumer"
	StorageClassReclaimPolicyDelete                   = "Delete"
	StorageClassReclaimPolicyRetain                   = "Retain"
)

func CreateStorageClass(ctx context.Context, cl client.Client, name, lvmType, lvmVolumeGroups, volumeBindingMode, reclaimPolicy string) error {

	vbm := storagev1.VolumeBindingImmediate
	switch volumeBindingMode {
	case StorageClassVolumeBindingModeImmediate:
		vbm = storagev1.VolumeBindingImmediate
	case StorageClassVolumeBindingModeWaitForFirstConsumer:
		vbm = storagev1.VolumeBindingWaitForFirstConsumer
	}

	rp := v1.PersistentVolumeReclaimDelete
	switch reclaimPolicy {
	case StorageClassReclaimPolicyDelete:
		rp = v1.PersistentVolumeReclaimDelete
	case StorageClassReclaimPolicyRetain:
		rp = v1.PersistentVolumeReclaimRetain

	}

	cs := storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       StorageClassKind,
			APIVersion: StorageClassAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Provisioner: "lvm.csi.storage.deckhouse.io",
		Parameters: map[string]string{
			"lvm.csi.storage.deckhouse.io/lvm-type":            lvmType,
			"lvm.csi.storage.deckhouse.io/volume-binding-mode": volumeBindingMode,
			"lvm.csi.storage.deckhouse.io/lvm-volume-groups":   lvmVolumeGroups,
		},
		ReclaimPolicy:        &rp,
		MountOptions:         nil,
		AllowVolumeExpansion: nil,
		VolumeBindingMode:    &vbm,
	}

	err := cl.Create(ctx, &cs)
	if err != nil {
		return err
	}
	return nil
}

func DeleteStorageClass(ctx context.Context, cl client.Client, name string) error {
	sc := storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       StorageClassKind,
			APIVersion: StorageClassAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}

	err := cl.Delete(ctx, &sc)
	if err != nil {
		return err
	}
	return nil
}
