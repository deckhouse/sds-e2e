package funcs

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PersistentVolumeClaimKind       = "PersistentVolumeClaim"
	PersistentVolumeClaimAPIVersion = "v1"
	NamePrefixBlock                 = "-block"
	NamePrefixFS                    = "-fs"
)

func CreatePVC(ctx context.Context, cl client.Client, name, storageClassName, size string, blockMode bool) error {
	resourceList := make(map[v1.ResourceName]resource.Quantity)
	sizeStorage, err := resource.ParseQuantity(size)
	if err != nil {
		return err
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
		return err
	}
	return nil
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
