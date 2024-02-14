package funcs

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sds-node-configurator-e2e/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAPIBlockDevices(ctx context.Context, cl client.Client) (map[string]v1alpha1.BlockDevice, error) {
	listDevice := &v1alpha1.BlockDeviceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.BlockDeviceKind,
			APIVersion: v1alpha1.TypeMediaAPIVersion,
		},
		ListMeta: metav1.ListMeta{},
		Items:    []v1alpha1.BlockDevice{},
	}

	err := cl.List(ctx, listDevice)
	if err != nil {
		return nil, err
	}

	devices := make(map[string]v1alpha1.BlockDevice, len(listDevice.Items))
	for _, blockDevice := range listDevice.Items {
		devices[blockDevice.Name] = blockDevice
	}
	return devices, nil
}
