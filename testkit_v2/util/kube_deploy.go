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
	"fmt"

	appsapi "k8s.io/api/apps/v1"
	coreapi "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

/*  Daemon Set  */

func (cluster *KCluster) GetDaemonSet(nsName, dsName string) (*appsapi.DaemonSet, error) {
	ds, err := (*cluster.goClient).AppsV1().DaemonSets(nsName).Get(cluster.ctx, dsName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func (cluster *KCluster) ListDaemonSet(nsName string) ([]appsapi.DaemonSet, error) {
	dsList, err := (*cluster.goClient).AppsV1().DaemonSets(nsName).List(cluster.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return dsList.Items, nil
}

/*  Service  */

func (cluster *KCluster) ListSvc(nsName string) ([]coreapi.Service, error) {
	svcList := coreapi.ServiceList{}
	optsList := ctrlrtclient.ListOptions{}
	if nsName != "" {
		fmt.Println("NS: ", nsName)
		optsList.Namespace = nsName
	}

	opts := ctrlrtclient.ListOption(&optsList)
	if err := cluster.controllerRuntimeClient.List(cluster.ctx, &svcList, opts); err != nil {
		return nil, err
	}

	return svcList.Items, nil
}

func (cluster *KCluster) CreateSvcNodePort(nsName, sName string, selector map[string]string, port, nodePort int) error {
	svc := coreapi.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sName,
			Namespace: nsName,
		},
		Spec: coreapi.ServiceSpec{
			Type:     coreapi.ServiceTypeNodePort,
			Selector: selector,
			Ports: []coreapi.ServicePort{{
				Port:     int32(port),
				NodePort: int32(nodePort),
			}},
		},
	}

	err := cluster.controllerRuntimeClient.Create(cluster.ctx, &svc)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
