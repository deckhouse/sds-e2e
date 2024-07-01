package funcs

import (
	"context"
	"github.com/deckhouse/sds-e2e/sdsNodeConfiguratorApi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateLvmVolumeGroup(ctx context.Context, cl client.Client, lvmVolumeGroupName string, blockDeviceNames []string) error {
	lvmVolumeGroup := &sdsNodeConfiguratorApi.LvmVolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: lvmVolumeGroupName,
		},
		Spec: sdsNodeConfiguratorApi.LvmVolumeGroupSpec{
			ActualVGNameOnTheNode: "data",
			BlockDeviceNames:      blockDeviceNames,
			Type:                  "Local",
		},
		Status: sdsNodeConfiguratorApi.LvmVolumeGroupStatus{
			Health: "NonOperational",
		},
	}
	return cl.Create(ctx, lvmVolumeGroup)
}

func GetLvmVolumeGroups(ctx context.Context, cl client.Client) (map[string]sdsNodeConfiguratorApi.LvmVolumeGroup, error) {
	listDevice := &sdsNodeConfiguratorApi.LvmVolumeGroupList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LvmVolumeGroup",
			APIVersion: "v1alpha1",
		},
		ListMeta: metav1.ListMeta{},
		Items:    []sdsNodeConfiguratorApi.LvmVolumeGroup{},
	}

	err := cl.List(ctx, listDevice)
	if err != nil {
		return nil, err
	}

	lvmVolumeGroups := make(map[string]sdsNodeConfiguratorApi.LvmVolumeGroup, len(listDevice.Items))
	for _, lvmVolumeGroup := range listDevice.Items {
		lvmVolumeGroups[lvmVolumeGroup.Name] = lvmVolumeGroup
	}
	return lvmVolumeGroups, nil
}
