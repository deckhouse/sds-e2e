package funcs

import (
	"cluster-management/v1alpha2"
	"context"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VM struct {
	Name  string
	Phase corev1.PodPhase
}

func ListVM(ctx context.Context, cl client.Client, namespaceName string) ([]VM, error) {
	objs := v1alpha2.VirtualMachineList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	podList := []VM{}
	for _, item := range objs.Items {
		podList = append(podList, VM{Name: item.Name})
	}

	return podList, nil
}
