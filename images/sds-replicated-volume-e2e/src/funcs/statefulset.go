package funcs

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type patchUInt32Value struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func CreateSts(ctx context.Context, cl client.Client, namespaceName string, stsCount int) error {
	for count := 0; count <= stsCount; count++ {
		fs := corev1.PersistentVolumeFilesystem
		storageClassName := "linstor-r2"
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
								Env:          []corev1.EnvVar{{Name: "FLOG_BATCH_SIZE", Value: "2048000"}, {Name: "FLOG_TIME_INTERVAL", Value: "1"}},
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
									corev1.ResourceStorage: resource.MustParse("10Gi"),
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
			return err
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
