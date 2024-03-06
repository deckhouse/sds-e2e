package funcs

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListPods(ctx context.Context, cl client.Client, namespaceName string) error {
	objs := corev1.PodList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return err
	}

	for count, item := range objs.Items {
		fmt.Printf("%d: #%v", count, item)
	}

	return nil
}

func WaitForLogStsPods(ctx context.Context, cl client.Client, namespaceName string) error {
	objs := corev1.PodList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return err
	}

	for _, item := range objs.Items {
		//		fmt.Printf("%d: #%v", count, item)
		fmt.Printf("#%v", item.Status.Phase)
	}

	return nil
}
