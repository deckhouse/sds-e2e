package funcs

import (
	"context"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAPIBlockDevices(ctx context.Context, cl client.Client) (map[string]snc.BlockDevice, error) {
	listDevice := &snc.BlockDeviceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       snc.BlockDeviceKind,
			APIVersion: snc.TypeMediaAPIVersion,
		},
		ListMeta: metav1.ListMeta{},
		Items:    []snc.BlockDevice{},
	}

	err := cl.List(ctx, listDevice)
	if err != nil {
		return nil, err
	}

	devices := make(map[string]snc.BlockDevice, len(listDevice.Items))
	for _, blockDevice := range listDevice.Items {
		devices[blockDevice.Name] = blockDevice
	}
	return devices, nil
}
