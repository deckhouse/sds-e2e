package funcs

import (
	"context"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	srv "github.com/deckhouse/sds-replicated-volume/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateDrbdStoragePool(ctx context.Context, cl client.Client, drbdStoragePoolName string, lvmVolumeGroups []srv.ReplicatedStoragePoolLVMVolumeGroups) error {
	lvmVolumeGroup := &srv.ReplicatedStoragePool{
		ObjectMeta: metav1.ObjectMeta{
			Name: drbdStoragePoolName,
		},
		Spec: srv.ReplicatedStoragePoolSpec{
			LvmVolumeGroups: lvmVolumeGroups,
			Type:            "LVM",
		},
		Status: srv.ReplicatedStoragePoolStatus{
			Phase: "Updating",
		},
	}
	return cl.Create(ctx, lvmVolumeGroup)
}

func CreateReplicatedStorageClass(ctx context.Context, cl client.Client, drbdStorageClassName string, replication string, isDefault bool) error {
	lvmVolumeGroup := &srv.ReplicatedStorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: drbdStorageClassName,
		},
		Spec: srv.ReplicatedStorageClassSpec{
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

func CreateReplicatedStoragePool(ctx context.Context, cl client.Client) error {
	var lvmVolumeGroupList []srv.ReplicatedStoragePoolLVMVolumeGroups
	listDevice := &snc.LvmVolumeGroupList{}

	err := cl.List(ctx, listDevice)
	if err != nil {
		return err
	}

	for _, item := range listDevice.Items {
		lvmVolumeGroupName := item.ObjectMeta.Name
		lvmVolumeGroupList = append(lvmVolumeGroupList, srv.ReplicatedStoragePoolLVMVolumeGroups{Name: lvmVolumeGroupName, ThinPoolName: ""})
	}

	err = CreateDrbdStoragePool(ctx, cl, "data", lvmVolumeGroupList)
	if err != nil {
		if err.Error() != "replicatedstoragepools.storage.deckhouse.io \"data\" already exists" {
			return err
		}
	}

	err = CreateReplicatedStorageClass(ctx, cl, "linstor-r1", "None", false)
	if err != nil {
		if err.Error() != "replicatedstorageclasses.storage.deckhouse.io \"linstor-r1\" already exists" {
			return err
		}
	}

	err = CreateReplicatedStorageClass(ctx, cl, "linstor-r2", "Availability", true)
	if err != nil {
		if err.Error() != "replicatedstorageclasses.storage.deckhouse.io \"linstor-r2\" already exists" {
			return err
		}
	}

	return nil
}
