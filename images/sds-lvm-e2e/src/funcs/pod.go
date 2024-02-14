package funcs

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PodClassKind  = "Pod"
	PodAPIVersion = "v1"
)

func CreatePod(ctx context.Context, cl client.Client, name, pvcName string, blockMode bool) error {

	var vds []v1.VolumeDevice
	var vms []v1.VolumeMount

	if blockMode {
		vd := v1.VolumeDevice{
			Name:       "nginx-persistent-storage",
			DevicePath: "/dev/sdx",
		}
		vds = append(vds, vd)
		vms = nil
	} else {
		vm := v1.VolumeMount{
			Name:      "nginx-persistent-storage",
			MountPath: "/usr/share/test-data",
		}
		vms = append(vms, vm)
		vds = nil
	}

	var cs []v1.Container
	c := v1.Container{
		Name:          "nginx-container",
		Image:         "nginx",
		VolumeDevices: vds,
		VolumeMounts:  vms,
	}
	cs = append(cs, c)

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       PodClassKind,
			APIVersion: PodAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: cs,
			Volumes: []v1.Volume{
				{
					Name: "nginx-persistent-storage",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	err := cl.Create(ctx, &pod)
	if err != nil {
		return err
	}
	return nil
}
