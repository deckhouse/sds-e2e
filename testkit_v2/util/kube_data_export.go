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

package integration

import (
	"context"
	"time"

	"github.com/deckhouse/sds-e2e/util/utiltype"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cluster *KCluster) CreateDataExport(dataExportName, exportKindType, exportKindName, namespace, ttl string) (*utiltype.DataExport, error) {
	dataExport := &utiltype.DataExport{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DataExport",
			APIVersion: "deckhouse.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dataExportName,
			Namespace: namespace,
		},
		Spec: utiltype.DataexportSpec{
			Ttl:     ttl,
			Publish: true,
			TargetRef: utiltype.TargetRefSpec{
				Kind: exportKindType,
				Name: exportKindName,
			},
		},
	}

	cwt, cancel := context.WithTimeout(cluster.ctx, 5*time.Second)
	defer cancel()

	err := cluster.controllerRuntimeClient.Create(cwt, dataExport)
	if err != nil {
		return nil, err
	}

	return dataExport, nil
}

func (cluster *KCluster) DeleteDataExport(name, namespace string) error {
	dataExport := &utiltype.DataExport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	cwt, cancel := context.WithTimeout(cluster.ctx, 5*time.Second)
	defer cancel()

	err := cluster.controllerRuntimeClient.Delete(cwt, dataExport)
	if err != nil {
		return err
	}
	return nil
}

func (cluster *KCluster) CreateDummyPod(podName, namespace, pvcName string) error {
	if namespace == "" {
		namespace = TestNS
	}

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "main-container",
					Image: "nginx:latest",
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "storage",
							MountPath: "/data",
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "storage",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	cwt, cancel := context.WithTimeout(cluster.ctx, 5*time.Second)
	defer cancel()

	if err := cluster.controllerRuntimeClient.Create(cwt, &pod); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}
