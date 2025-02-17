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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Svc struct {
	Name string
}

func ListServices(ctx context.Context, cl client.Client, namespaceName string, svcSearch string) ([]Svc, error) {
	objs := corev1.ServiceList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	svcList := []Svc{}
	for _, item := range objs.Items {
		if svcSearch == "" || svcSearch == item.Name {
			svcList = append(svcList, Svc{Name: item.Name})
		}
	}

	return svcList, nil
}

func CreateService(ctx context.Context, cl client.Client, namespaceName string, name string) (*corev1.Service, error) {
	svcObj := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespaceName, Labels: map[string]string{"service": "v1"}},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"service": "v1"},
			Ports: []corev1.ServicePort{
				{Protocol: "TCP", Port: 31022, TargetPort: intstr.FromInt32(22)},
			},
		},
	}

	err := cl.Create(ctx, svcObj)
	if err != nil {
		return nil, err
	}

	return svcObj, nil
}
