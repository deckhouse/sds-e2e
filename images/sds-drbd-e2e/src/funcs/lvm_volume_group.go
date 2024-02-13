package funcs

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sds-drbd-e2e/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateLvmVolumeGroup(ctx context.Context, cl client.Client, lvmVolumeGroupName string, blockDeviceNames []string) error {
	lvmVolumeGroup := &v1alpha1.LvmVolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: lvmVolumeGroupName,
		},
		Spec: v1alpha1.LvmVolumeGroupSpec{
			ActualVGNameOnTheNode: "data",
			BlockDeviceNames:      blockDeviceNames,
			Type:                  "Local",
		},
		Status: v1alpha1.LvmVolumeGroupStatus{
			Health: "NonOperational",
		},
	}
	return cl.Create(ctx, lvmVolumeGroup)
}

func GetAPILvmVolumeGroup(ctx context.Context, cl client.Client) (map[string]v1alpha1.LvmVolumeGroup, error) {
	listDevice := &v1alpha1.LvmVolumeGroupList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LvmVolumeGroup",
			APIVersion: "v1alpha1",
		},
		ListMeta: metav1.ListMeta{},
		Items:    []v1alpha1.LvmVolumeGroup{},
	}

	cl.List(ctx, listDevice)

	lvmVolumeGroups := make(map[string]v1alpha1.LvmVolumeGroup, len(listDevice.Items))
	for _, lvmVolumeGroup := range listDevice.Items {
		lvmVolumeGroups[lvmVolumeGroup.Name] = lvmVolumeGroup
	}
	return lvmVolumeGroups, nil
}
