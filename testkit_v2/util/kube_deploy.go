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

func (clr *KCluster) GetDaemonSet(nsName, dsName string) (*appsapi.DaemonSet, error) {
	ds, err := (*clr.goClient).AppsV1().DaemonSets(nsName).Get(clr.ctx, dsName, metav1.GetOptions{})
	if err != nil {
		Debugf("Can't get '%s.%s' DS: %s", nsName, dsName, err.Error())
		return nil, err
	}

	return ds, nil
}

func (clr *KCluster) ListDaemonSet(nsName string) ([]appsapi.DaemonSet, error) {
	dsList, err := (*clr.goClient).AppsV1().DaemonSets(nsName).List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		Debugf("Can't get '%s' DS list: %s", nsName, err.Error())
		return nil, err
	}

	return dsList.Items, nil
}

/*  Service  */

func (clr *KCluster) ListSvc(nsName string) ([]coreapi.Service, error) {
	svcList := coreapi.ServiceList{}
	optsList := ctrlrtclient.ListOptions{}
	if nsName != "" {
		fmt.Println("NS: ", nsName)
		optsList.Namespace = nsName
	}

	opts := ctrlrtclient.ListOption(&optsList)
	if err := clr.rtClient.List(clr.ctx, &svcList, opts); err != nil {
		return nil, err
	}

	return svcList.Items, nil
}

func (clr *KCluster) CreateSvcNodePort(nsName, sName string, selector map[string]string, port, nodePort int) error {
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

	err := clr.rtClient.Create(clr.ctx, &svc)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
