package test

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sds-node-configurator-e2e/funcs"
	"sds-node-configurator-e2e/v1alpha1"
	"testing"
)

func Test11(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient(t)
	if err != nil {
		t.Error("kubeclient error", err)
	}

	devices, err := funcs.GetAPIBlockDevices(ctx, cl, t)
	if err != nil {
		t.Error("get error", err)
	}

	for key, item := range devices {
		t.Log(key)
		t.Log(item.ObjectMeta.Name)
		t.Log(item.Status.NodeName)
		lvmVolumeGroup := &v1alpha1.LvmVolumeGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name: item.Status.NodeName,
			},
			Spec: v1alpha1.LvmVolumeGroupSpec{
				ActualVGNameOnTheNode: "data",
				BlockDeviceNames:      []string{item.ObjectMeta.Name},
				Type:                  "Local",
			},
			Status: v1alpha1.LvmVolumeGroupStatus{
				Health: "NonOperational",
			},
		}
		t.Log(cl.Create(ctx, lvmVolumeGroup))
	}
}
