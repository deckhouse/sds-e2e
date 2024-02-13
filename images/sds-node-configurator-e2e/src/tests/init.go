package test

import (
	v1 "k8s.io/api/core/v1"
	sv1 "k8s.io/api/storage/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
	"sds-node-configurator-e2e/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func NewKubeClient(t *testing.T) (client.Client, error) {
	//clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
	//	clientcmd.NewDefaultClientConfigLoadingRules(),
	//	&clientcmd.ConfigOverrides{},
	//)
	//
	//config, err := clientConfig.ClientConfig()
	//if err != nil {
	//	return nil, err
	//}

	kubeconfigPath := filepath.Join("/app", "kube.config.internal")
	//kubeconfigPath := os.Getenv("kubeconfig")
	//
	//
	//t.Logf(kubeconfigPath)
	//if kubeconfigPath == "" {
	//	kubeconfigPath = filepath.Join("/app", "kube.config.internal")
	//}

	t.Log("1")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		panic(err.Error())
	}

	var (
		resourcesSchemeFuncs = []func(*apiruntime.Scheme) error{
			v1alpha1.AddToScheme,
			clientgoscheme.AddToScheme,
			extv1.AddToScheme,
			v1.AddToScheme,
			sv1.AddToScheme,
		}
	)

	scheme := apiruntime.NewScheme()
	for _, f := range resourcesSchemeFuncs {
		err = f(scheme)
		if err != nil {
			return nil, err
		}
	}

	managerOpts := client.Options{
		Scheme: scheme,
	}

	return client.New(config, managerOpts)
	//
	//mgr, err := manager.New(config, managerOpts)
	//if err != nil {
	//	return nil, err
	//}
	//
	//return mgr.GetClient(), nil
}
