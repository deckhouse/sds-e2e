/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package funcs

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Namespace struct {
	Name string
}

func CreateNamespace(ctx context.Context, cl client.Client, namespaceName string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}

	return cl.Create(ctx, namespace)
}

func DeleteNamespace(ctx context.Context, cl client.Client, namespaceName string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}

	return cl.Delete(ctx, namespace)
}

func ListNamespace(ctx context.Context, cl client.Client, namespaceSearch string) ([]Namespace, error) {
	objs := corev1.NamespaceList{}
	opts := client.ListOption(&client.ListOptions{})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	namespaceList := []Namespace{}
	for _, item := range objs.Items {
		if namespaceSearch == "" || namespaceSearch == item.Name {
			namespaceList = append(namespaceList, Namespace{Name: item.Name})
		}
	}

	return namespaceList, nil
}
