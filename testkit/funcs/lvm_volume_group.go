package funcs

import (
	"context"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateLvmVolumeGroup(ctx context.Context, cl client.Client, lvmVolumeGroupName string, blockDeviceNames []string) error {
	lvmVolumeGroup := &snc.LvmVolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: lvmVolumeGroupName,
		},
		Spec: snc.LvmVolumeGroupSpec{
			ActualVGNameOnTheNode: "data",
			BlockDeviceNames:      blockDeviceNames,
			Type:                  "Local",
		},
	}
	return cl.Create(ctx, lvmVolumeGroup)
}

func GetLvmVolumeGroups(ctx context.Context, cl client.Client) (map[string]snc.LvmVolumeGroup, error) {
	listDevice := &snc.LvmVolumeGroupList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LvmVolumeGroup",
			APIVersion: "v1alpha1",
		},
		ListMeta: metav1.ListMeta{},
		Items:    []snc.LvmVolumeGroup{},
	}

	err := cl.List(ctx, listDevice)
	if err != nil {
		return nil, err
	}

	lvmVolumeGroups := make(map[string]snc.LvmVolumeGroup, len(listDevice.Items))
	for _, lvmVolumeGroup := range listDevice.Items {
		lvmVolumeGroups[lvmVolumeGroup.Name] = lvmVolumeGroup
	}
	return lvmVolumeGroups, nil
}
