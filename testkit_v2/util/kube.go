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
	"fmt"

	logr "github.com/go-logr/logr"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlrtlog "sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1nfs "github.com/deckhouse/csi-nfs/api/v1alpha1"
	"github.com/deckhouse/sds-e2e/util/utiltype"
	v1app "k8s.io/api/apps/v1"
	coreapi "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storapi "k8s.io/api/storage/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"

	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	srv "github.com/deckhouse/sds-replicated-volume/api/v1alpha1"
	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
)

type KCluster struct {
	name                    string
	ctx                     context.Context
	restCfg                 *rest.Config
	controllerRuntimeClient ctrlrtclient.Client
	goClient                *kubernetes.Clientset
	dyClient                *dynamic.DynamicClient
}

/*  Config  */

func NewRestConfig(configPath, clusterName string) (*rest.Config, error) {
	var cfg *rest.Config
	var loader clientcmd.ClientConfigLoader
	if configPath != "" {
		loader = &clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath}
	} else {
		loader = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	overrides := clientcmd.ConfigOverrides{}
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
		overrides.CurrentContext = clusterName
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader, &overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed create rest config: %w", err)
	}

	return cfg, nil
}

/*  Kuber Client  */

func NewKubeRTClient(cfg *rest.Config) (ctrlrtclient.Client, error) {
	// Add options
	var resourcesSchemeFuncs = []func(*apiruntime.Scheme) error{
		virt.AddToScheme,
		srv.AddToScheme,
		snc.AddToScheme,
		kubescheme.AddToScheme,
		extapi.AddToScheme,
		coreapi.AddToScheme,
		storapi.AddToScheme,
		D8SchemeBuilder.AddToScheme,
		metav1.AddMetaToScheme,
		v1alpha1nfs.AddToScheme,
		utiltype.AddToScheme,
	}

	scheme := apiruntime.NewScheme()
	for _, f := range resourcesSchemeFuncs {
		err := f(scheme)
		if err != nil {
			return nil, err
		}
	}
	clientOpts := ctrlrtclient.Options{
		Scheme: scheme,
	}

	// Init client
	ctrlrtlog.SetLogger(logr.FromContextOrDiscard(context.Background()))
	cl, err := ctrlrtclient.New(cfg, clientOpts)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

func NewKubeGoClient(cfg *rest.Config) (*kubernetes.Clientset, error) {
	cl, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

func NewKubeDyClient(cfg *rest.Config) (*dynamic.DynamicClient, error) {
	cl, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

// CreateDataExporterBearer creates a ServiceAccount and ClusterRoleBinding, then issues a token
// for use as Bearer in HTTP requests. Token TTL is in seconds.
func (cluster *KCluster) CreateDataExporterBearer(userName string, ttlSeconds int64) (string, error) {
	ns := "test-e2e"

	sa := &coreapi.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: userName, Namespace: ns}}
	if err := cluster.controllerRuntimeClient.Create(cluster.ctx, sa); err != nil && !apierrors.IsAlreadyExists(err) {
		return "", fmt.Errorf("create SA: %w", err)
	}

	crbName := "data-exporter-binding-" + userName
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: crbName},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "cluster-admin"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: userName, Namespace: ns}},
	}
	if err := cluster.controllerRuntimeClient.Create(cluster.ctx, crb); err != nil && !apierrors.IsAlreadyExists(err) {
		return "", fmt.Errorf("create CRB: %w", err)
	}

	exp := ttlSeconds
	tr := &authv1.TokenRequest{Spec: authv1.TokenRequestSpec{ExpirationSeconds: &exp}}
	tok, err := cluster.goClient.CoreV1().ServiceAccounts(ns).CreateToken(cluster.ctx, userName, tr, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}
	return tok.Status.Token, nil
}

/*  Kuber Cluster object  */

func InitKCluster(configPath, clusterName string) (*KCluster, error) {
	if clusterName == "" {
		clusterName = *clusterNameFlag
	}
	if configPath == "" {
		configPath = NestedClusterKubeConfig
	}

	restCfg, err := NewRestConfig(configPath, clusterName)
	if err != nil {
		Critf("Can't connect cluster %s", clusterName)
		return nil, err
	}

	rcl, err := NewKubeRTClient(restCfg)
	if err != nil {
		Critf("Can't connect cluster %s", clusterName)
		return nil, err
	}

	gcl, err := NewKubeGoClient(restCfg)
	if err != nil {
		Critf("Can't connect cluster %s", clusterName)
		return nil, err
	}

	dcl, err := NewKubeDyClient(restCfg)
	if err != nil {
		Critf("Can't connect cluster %s", clusterName)
		return nil, err
	}

	cluster := KCluster{
		name:                    clusterName,
		ctx:                     context.Background(),
		restCfg:                 restCfg,
		controllerRuntimeClient: rcl,
		goClient:                gcl,
		dyClient:                dcl,
	}

	return &cluster, nil
}

/*  Name Space  */

type NsFilter struct {
	Name     any
	ExistSec any
}

type nsType = coreapi.Namespace

func (f *NsFilter) Apply(nss []nsType) (resp []nsType) {
	for _, ns := range nss {
		if f.Name != nil && !CheckCondition(f.Name, ns.Name) {
			continue
		}

		resp = append(resp, ns)
	}
	return
}

func (cluster *KCluster) ListNs(filters ...NsFilter) ([]nsType, error) {
	nsList := coreapi.NamespaceList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	err := cluster.controllerRuntimeClient.List(cluster.ctx, &nsList, opts)
	if err != nil {
		Warnf("Can't get namespaces: %s", err.Error())
		return nil, err
	}

	if len(filters) == 0 {
		return nsList.Items, nil
	}

	resp := nsList.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (cluster *KCluster) CreateNs(nsName string) error {
	namespace := &coreapi.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}

	err := cluster.controllerRuntimeClient.Create(cluster.ctx, namespace)
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}

	Errorf("Can't create namespace %s", nsName)
	return err
}

func (cluster *KCluster) DeleteNs(filters ...NsFilter) error {
	nsList, err := cluster.ListNs(filters...)
	if err != nil {
		return err
	}
	allNsList, err := cluster.ListNs()
	if err != nil {
		return err
	} else if len(nsList) == len(allNsList) {
		return fmt.Errorf("Fatal mistake protection: can`t delete all namespaces")
	}

	for _, ns := range nsList {
		if err := cluster.controllerRuntimeClient.Delete(cluster.ctx, &ns); err != nil {
			return err
		}
	}

	return nil
}

// TODO add context
func (cluster *KCluster) DeleteNsAndWait(filters ...NsFilter) error {
	if err := cluster.DeleteNs(filters...); err != nil {
		return err
	}
	return RetrySec(20, func() error {
		nsList, err := cluster.ListNs(filters...)
		if err != nil {
			return err
		}

		if len(nsList) > 0 {
			return fmt.Errorf("Can't delete %d namespaces: %s, ...", len(nsList), nsList[0].Name)
		}
		Debugf("Namespaces deleted")
		return nil
	})
}

func (cluster *KCluster) CheckDeploymentReady(nsName, deploymentName string) (bool, error) {
	deployment := &v1app.Deployment{}
	err := cluster.controllerRuntimeClient.Get(cluster.ctx, ctrlrtclient.ObjectKey{
		Name:      deploymentName,
		Namespace: nsName,
	}, deployment)

	if err != nil {
		return false, fmt.Errorf("failed to get deployment %s in namespace %s: %w", deploymentName, nsName, err)
	}

	if deployment.Status.ReadyReplicas == deployment.Status.Replicas {
		return true, nil
	}

	return false, nil
}

func (cluster *KCluster) WaitUntilDeploymentReady(nsName, deploymentName string, timeoutSec int) error {
	Debugf("Waiting for deployment %s in namespace %s to be ready for %d seconds...", deploymentName, nsName, timeoutSec)
	return RetrySec(timeoutSec, func() error {
		ready, err := cluster.CheckDeploymentReady(nsName, deploymentName)
		if err != nil {
			return err
		}
		if !ready {
			return fmt.Errorf("deployment %s in namespace %s is not ready", deploymentName, nsName)
		}
		Debugf("Deployment %s in namespace %s is ready", deploymentName, nsName)
		return nil
	})
}
