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
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func CreateSts(ctx context.Context, cl client.Client, namespaceName string, pvSize string, stsCount int, storageClassName string) error {
	for count := 0; count <= stsCount; count++ {
		fs := corev1.PersistentVolumeFilesystem
		var replicas int32 = 3
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("flog-generator-%d", count),
				Namespace: namespaceName,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app.kubernetes.io/name": fmt.Sprintf("flog-generator-%d", count)},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.kubernetes.io/name": fmt.Sprintf("flog-generator-%d", count)},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Image:        "ex42zav/flog:0.4.3",
								Name:         "flog-generator",
								Command:      []string{"/bin/sh"},
								Args:         []string{"-c", "/srv/flog/run.sh 2>&1 | tee -a /var/log/flog/fake.log"},
								Env:          []corev1.EnvVar{{Name: "FLOG_BATCH_SIZE", Value: "1024000"}, {Name: "FLOG_TIME_INTERVAL", Value: "1"}},
								VolumeMounts: []corev1.VolumeMount{{Name: "flog-pv", MountPath: "/var/log/flog"}},
							},
							{
								Image: "blacklabelops/logrotate",
								Name:  "logrotate",
								Env: []corev1.EnvVar{
									{Name: "LOGS_DIRECTORIES", Value: "/var/log/flog"},
									{Name: "LOGROTATE_INTERVAL", Value: "hourly"},
									{Name: "LOGROTATE_COPIES", Value: "2"},
									{Name: "LOGROTATE_SIZE", Value: "500M"},
									{Name: "LOGROTATE_CRONSCHEDULE", Value: "0 2 * * * *"},
								},
								VolumeMounts: []corev1.VolumeMount{{Name: "flog-pv", MountPath: "/var/log/flog"}},
							},
						},
						RestartPolicy:      "Always",
						DNSPolicy:          "ClusterFirst",
						ServiceAccountName: "default",
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "flog-pv",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse(pvSize),
								},
							},
							StorageClassName: &storageClassName,
							VolumeMode:       &fs,
						},
					},
				},
			},
		}

		fmt.Printf("Creating sts number %d\n", count)
		err := cl.Create(ctx, sts)
		if err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
	}

	return nil
}

func DeleteSts(ctx context.Context, cl client.Client, namespaceName string) error {
	objs := appsv1.StatefulSetList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return err
	}

	for _, item := range objs.Items {
		err := cl.Delete(ctx,
			&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: item.Name, Namespace: namespaceName}},
			client.DeleteOption(&client.DeleteOptions{}))
		if err != nil {
			return err
		}
	}

	return nil
}
