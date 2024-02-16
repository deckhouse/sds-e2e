package funcs

import (
	"context"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	PersistentVolumeClaimKind       = "PersistentVolumeClaim"
	PersistentVolumeClaimAPIVersion = "v1"
	WaitIntervalPVC                 = 1
	WaitIterationCountPVC           = 10
	DeletedStatusPVC                = "Deleted"
)

func CreatePVC(ctx context.Context, cl client.Client, name, storageClassName, size string, blockMode bool) (v1.PersistentVolumeClaim, error) {
	resourceList := make(map[v1.ResourceName]resource.Quantity)
	sizeStorage, err := resource.ParseQuantity(size)
	if err != nil {
		return v1.PersistentVolumeClaim{}, err
	}
	resourceList[v1.ResourceStorage] = sizeStorage

	var pvm v1.PersistentVolumeMode
	if blockMode {
		pvm = v1.PersistentVolumeBlock
		name = name + NamePrefixBlock
	} else {
		pvm = v1.PersistentVolumeFilesystem
		name = name + NamePrefixFS
	}

	pvc := v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       PersistentVolumeClaimKind,
			APIVersion: PersistentVolumeClaimAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NameSpace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.VolumeResourceRequirements{
				Requests: resourceList,
			},
			VolumeMode: &pvm,
		},
	}

	err = cl.Create(ctx, &pvc)
	if err != nil {
		return v1.PersistentVolumeClaim{}, err
	}
	return pvc, nil
}

func DeletePVC(ctx context.Context, cl client.Client, name string) error {
	pvc := v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       PersistentVolumeClaimKind,
			APIVersion: PersistentVolumeClaimAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NameSpace,
		},
	}

	err := cl.Delete(ctx, &pvc)
	if err != nil {
		return err
	}
	return nil
}

func WaitPVCStatus(ctx context.Context, cl client.Client, name string) (string, error) {
	pvc := v1.PersistentVolumeClaim{}
	for i := 0; i < WaitIterationCountPVC; i++ {
		err := cl.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: NameSpace,
		}, &pvc)
		if err != nil {
			//if kerrors.IsNotFound(err) {
			//	return "", err
			//}
		}
		fmt.Printf("pvc %s...\n", pvc.Status.Phase)
		if pvc.Status.Phase == v1.ClaimBound {
			return string(pvc.Status.Phase), nil
		}

		if len(pvc.Status.Phase) == 0 {
			return DeletedStatusPVC, nil
		}

		time.Sleep(WaitIntervalPVC * time.Second)
	}
	return "", errors.New(fmt.Sprintf("the waiting time %d or the pvc to be ready has expired",
		WaitIntervalPVC*WaitIterationCountPVC))
}

func WaitDeletePVC(ctx context.Context, cl client.Client, name string) (string, error) {
	pod := v1.Pod{}
	for i := 0; i < WaitIterationCountPVC; i++ {
		time.Sleep(WaitIntervalPVC * time.Second)
		err := cl.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: NameSpace,
		}, &pod)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return DeletedStatusPVC, nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf("the waiting time %d for the pod to be ready has expired",
		WaitIterationCountPVC*WaitIntervalPVC))
}

func EditSizePVC(ctx context.Context, cl client.Client, name, newSize string) error {
	pvc := v1.PersistentVolumeClaim{}
	err := cl.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: NameSpace,
	}, &pvc)
	if err != nil {
		return err
	}

	resourceList := make(map[v1.ResourceName]resource.Quantity)
	newPVCSize, err := resource.ParseQuantity(newSize)
	if err != nil {
		return err
	}

	resourceList[v1.ResourceStorage] = newPVCSize
	pvc.Spec.Resources.Requests = resourceList

	err = cl.Update(ctx, &pvc)
	if err != nil {
		return err
	}
	return nil
}
