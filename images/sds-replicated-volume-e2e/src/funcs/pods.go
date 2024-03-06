package funcs

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Pod struct {
	Name  string
	Phase corev1.PodPhase
}

func ListPods(ctx context.Context, cl client.Client, namespaceName string) ([]Pod, error) {
	objs := corev1.PodList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	podList := []Pod{}
	for _, item := range objs.Items {
		podList = append(podList, Pod{Name: item.Name, Phase: item.Status.Phase})
	}

	return podList, nil
}
