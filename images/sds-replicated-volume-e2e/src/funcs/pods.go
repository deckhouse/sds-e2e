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

func WaitForPodsReadiness(ctx context.Context, cl client.Client, namespaceName string) error {
	objs := corev1.PodList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return err
	}

	for count := 0; count < 60; count++ {
		fmt.Printf("Wait for all pods to be ready")

		allPodsReady := true
		for _, item := range objs.Items {
			if item.Status.Phase != corev1.PodRunning {
				allPodsReady = false
			}
		}

		if allPodsReady {
			break
		}

		if count == 60 {
			return fmt.Errorf("Timeout waiting for all pods to be ready")
		}
	}

	return nil
}
