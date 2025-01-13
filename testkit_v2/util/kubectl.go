package integration

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func RunPod(ctx context.Context, cl ctrlruntimeclient.Client) {
	// TODO implement
	nodeName := "d-shipkov-worker-0"

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "test1", Name: "bar"}}
	binding := &corev1.Binding{Target: corev1.ObjectReference{Name: nodeName}}
	_ = cl.SubResource("binding").Create(ctx, pod, binding)
}
