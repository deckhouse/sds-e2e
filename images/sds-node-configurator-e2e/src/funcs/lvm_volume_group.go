package funcs

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sds-node-configurator-e2e/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func CreateLvmVolumeGroup(ctx context.Context, cl client.Client, t *testing.T, lvmVolumeGroupName string, blockDeviceNames []string) error {
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
