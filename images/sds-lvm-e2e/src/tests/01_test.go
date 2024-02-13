package test

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StorageClassKind       = "StorageClass"
	StorageClassAPIVersion = "storage.k8s.io/v1"
)

func Test11(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = CreateStorageClass(ctx, cl)
	if err != nil {
		t.Error(err)
	}
}

func CreateStorageClass(ctx context.Context, cl client.Client) error {
	vbm := storagev1.VolumeBindingImmediate
	rp := v1.PersistentVolumeReclaimDelete

	cs := storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       StorageClassKind,
			APIVersion: StorageClassAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lvm-thick-immediate-delete",
			Namespace: "default",
		},
		Provisioner: "lvm.csi.storage.deckhouse.io",
		Parameters: map[string]string{
			"lvm.csi.storage.deckhouse.io/lvm-type":            "Thick",
			"lvm.csi.storage.deckhouse.io/volume-binding-mode": "Immediate",
			"lvm.csi.storage.deckhouse.io/lvm-volume-groups":   "- name: vg-w1\n- name: vg-w2",
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
