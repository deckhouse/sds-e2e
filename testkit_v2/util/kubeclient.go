package integration

import (
//	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
//	srv "github.com/deckhouse/sds-replicated-volume/api/v1alpha1"
//	virtualization "github.com/deckhouse/virtualization/api/core/v1alpha2"
//	"github.com/go-logr/logr"
//	v1 "k8s.io/api/core/v1"
//	sv1 "k8s.io/api/storage/v1"
//	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
//	apiruntime "k8s.io/apimachinery/pkg/runtime"
//	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
//	"k8s.io/client-go/rest"
//	"k8s.io/client-go/tools/clientcmd"
//	"os"
//	controllerruntime "sigs.k8s.io/controller-runtime"
//	"sigs.k8s.io/controller-runtime/pkg/client"
//	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
//
//	"fmt"
)

//func NewKubeClient(kubeconfigPath string) (client.Client, error) {
//	var config *rest.Config
//	var err error
//
//	if kubeconfigPath == "" {
//		kubeconfigPath = os.Getenv("kubeconfig")
//		fmt.Printf("\n>>>>>> %#v\n\n", kubeconfigPath)
//	}
//
//	controllerruntime.SetLogger(logr.New(ctrllog.NullLogSink{}))
//
//	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
//	if err != nil {
//		return nil, err
//	}
//
//	var (
//		resourcesSchemeFuncs = []func(*apiruntime.Scheme) error{
//			virtualization.AddToScheme,
//			srv.AddToScheme,
//			snc.AddToScheme,
//			clientgoscheme.AddToScheme,
//			extv1.AddToScheme,
//			v1.AddToScheme,
//			sv1.AddToScheme,
//		}
//	)
//
//	scheme := apiruntime.NewScheme()
//	for _, f := range resourcesSchemeFuncs {
//		err = f(scheme)
//		if err != nil {
//			return nil, err
//		}
//	}
//
//	clientOpts := client.Options{
//		Scheme: scheme,
//	}
//
//	return client.New(config, clientOpts)
//}
