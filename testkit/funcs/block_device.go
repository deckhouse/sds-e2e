package funcs

import (
	"context"
	"github.com/deckhouse/sds-e2e/sdsNodeConfiguratorApi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAPIBlockDevices(ctx context.Context, cl client.Client) (map[string]sdsNodeConfiguratorApi.BlockDevice, error) {
	listDevice := &sdsNodeConfiguratorApi.BlockDeviceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       sdsNodeConfiguratorApi.BlockDeviceKind,
			APIVersion: sdsNodeConfiguratorApi.TypeMediaAPIVersion,
		},
		ListMeta: metav1.ListMeta{},
		Items:    []sdsNodeConfiguratorApi.BlockDevice{},
	}

	err := cl.List(ctx, listDevice)
	if err != nil {
		return nil, err
	}

	devices := make(map[string]sdsNodeConfiguratorApi.BlockDevice, len(listDevice.Items))
	for _, blockDevice := range listDevice.Items {
		devices[blockDevice.Name] = blockDevice
	}
	return devices, nil
}
