package funcs

import (
	"context"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	PodClassKind          = "Pod"
	PodAPIVersion         = "v1"
	WaitIntervalPOD       = 3
	WaitIterationCountPOD = 20
)

func CreatePod(ctx context.Context, cl client.Client, name, pvcName string, blockMode bool, command, args []string) (string, error) {

	var vds []v1.VolumeDevice
	var vms []v1.VolumeMount

	if blockMode {
		vd := v1.VolumeDevice{
			Name:       "nginx-persistent-storage",
			DevicePath: "/dev/sdx",
		}
		vds = append(vds, vd)
		vms = nil
		name = name + NamePrefixBlock
	} else {
		vm := v1.VolumeMount{
			Name:      "nginx-persistent-storage",
			MountPath: "/usr/share/test-data",
		}
		vms = append(vms, vm)
		vds = nil
		name = name + NamePrefixFS
	}

	var cs []v1.Container
	c := v1.Container{
		Name:          "nginx-container",
		Image:         "nginx",
		VolumeDevices: vds,
		VolumeMounts:  vms,
		Command:       command,
		Args:          args,
	}
	cs = append(cs, c)

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       PodClassKind,
			APIVersion: PodAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NameSpace,
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
			RestartPolicy: "Never",
		},
	}

	err := cl.Create(ctx, &pod)
	if err != nil {
		return "", err
	}
	return name, nil
}

func DeletePod(ctx context.Context, cl client.Client, name string) error {
	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       PodClassKind,
			APIVersion: PodAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NameSpace,
		},
	}
	err := cl.Delete(ctx, &pod)
	if err != nil {
		return err
	}
	return nil
}

func WaitPodStatus(ctx context.Context, cl client.Client, name string) (string, error) {
	pod := v1.Pod{}

	for i := 0; i < WaitIterationCountPOD; i++ {
		time.Sleep(WaitIntervalPOD * time.Second)
		cl.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: NameSpace,
		}, &pod)

		if len(pod.Status.ContainerStatuses) != 0 {
			if pod.Status.ContainerStatuses[0].State.Waiting != nil {
				fmt.Println("container Waiting...")
				//return pod.Status.ContainerStatuses[0].Image, nil
			}

			if pod.Status.ContainerStatuses[0].State.Running != nil {
				fmt.Println("container Running....")
			}

			if pod.Status.ContainerStatuses[0].State.Terminated != nil {
				fmt.Println("container Terminated...")
				return pod.Status.ContainerStatuses[0].State.Terminated.Reason, nil
			}
		}
	}
	return "", errors.New("the waiting time for the pod to be ready has expired")
}

func WaitDeletePod(ctx context.Context, cl client.Client, name string) (string, error) {
	pod := v1.Pod{}
	for i := 0; i < WaitIterationCountPOD; i++ {
		time.Sleep(WaitIntervalPOD * time.Second)
		err := cl.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: NameSpace,
		}, &pod)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return "Deleted", nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf("the waiting time %d for the pod to be ready has expired",
		WaitIterationCountPOD*WaitIntervalPOD))
}
