package funcs

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sds-drbd-e2e/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateDrbdStoragePool(ctx context.Context, cl client.Client, drbdStoragePoolName string, lvmVolumeGroups []v1alpha1.DRBDStoragePoolLVMVolumeGroups) error {
	lvmVolumeGroup := &v1alpha1.DRBDStoragePool{
		ObjectMeta: metav1.ObjectMeta{
			Name: drbdStoragePoolName,
		},
		Spec: v1alpha1.DRBDStoragePoolSpec{
			LvmVolumeGroups: lvmVolumeGroups,
			Type:            "LVM",
		},
		Status: v1alpha1.DRBDStoragePoolStatus{
			Phase: "Updating",
		},
	}
	return cl.Create(ctx, lvmVolumeGroup)
}

func CreateDrbdStorageClass(ctx context.Context, cl client.Client, drbdStorageClassName string, replication string, isDefault bool) error {
	lvmVolumeGroup := &v1alpha1.DRBDStorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: drbdStorageClassName,
		},
		Spec: v1alpha1.DRBDStorageClassSpec{
			IsDefault:     isDefault,
			StoragePool:   "data",
			ReclaimPolicy: "Delete",
			Replication:   replication,
			VolumeAccess:  "PreferablyLocal",
			Topology:      "Ignored",
			Zones:         []string{},
		},
	}
	return cl.Create(ctx, lvmVolumeGroup)
}

func CreatePools(ctx context.Context, cl client.Client) error {
	var lvmVolumeGroupList []v1alpha1.DRBDStoragePoolLVMVolumeGroups
	listedResources, _ := GetAPILvmVolumeGroup(ctx, cl)
	for _, item := range listedResources {
		lvmVolumeGroupName := item.ObjectMeta.Name
		lvmVolumeGroupList = append(lvmVolumeGroupList, v1alpha1.DRBDStoragePoolLVMVolumeGroups{Name: lvmVolumeGroupName, ThinPoolName: ""})
	}

	err := CreateDrbdStoragePool(ctx, cl, "data", lvmVolumeGroupList)
	if err != nil {
		return err
	}

	err = CreateDrbdStorageClass(ctx, cl, "linstor-r1", "None", false)
	if err != nil {
		return err
	}

	err = CreateDrbdStorageClass(ctx, cl, "linstor-r2", "Availability", true)
	if err != nil {
		return err
	}

	return nil
}
