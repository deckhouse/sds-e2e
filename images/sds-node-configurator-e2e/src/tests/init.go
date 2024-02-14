package test

import (
	v1 "k8s.io/api/core/v1"
	sv1 "k8s.io/api/storage/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"sds-node-configurator-e2e/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewKubeClient() (client.Client, error) {
	var config *rest.Config
	var err error

	kubeconfigPath := os.Getenv("kubeconfig")
	//	if kubeconfigPath == "" {
	//		kubeconfigPath = filepath.Join("/app", "kube.config.internal")
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	//	} else {
	//		config = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
	//			clientcmd.NewDefaultClientConfigLoadingRules(),
	//			&clientcmd.ConfigOverrides{},
	//		)
	//	}

	if err != nil {
		return nil, err
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

	clientOpts := client.Options{
		Scheme: scheme,
	}

	return client.New(config, clientOpts)
}
