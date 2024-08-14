package funcs

import (
	"context"
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type pvcList struct {
	Name string
	Size string
}

func ListPvcs(ctx context.Context, cl client.Client, namespaceName string) ([]pvcList, error) {
	objs := corev1.PersistentVolumeClaimList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	pvcs := []pvcList{}
	for _, item := range objs.Items {
		pvcs = append(pvcs, pvcList{Name: item.ObjectMeta.Name, Size: item.Status.Capacity.Storage().String()})
	}

	return pvcs, nil
}

func ChangePvcSize(ctx context.Context, cl client.Client, namespaceName string, pvcName string, pvResizedSize string) error {
	payload := []patchUInt32Value{{
		Op:    "replace",
		Path:  "/spec/resources/requests/storage",
		Value: pvResizedSize,
	}}
	payloadBytes, _ := json.Marshal(payload)
	err := cl.Patch(ctx,
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Namespace: namespaceName, Name: pvcName}},
		client.RawPatch(types.JSONPatchType, payloadBytes),
		&client.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}
