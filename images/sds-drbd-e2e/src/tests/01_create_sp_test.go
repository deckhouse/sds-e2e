package test

import (
	"flag"
	"gocontainer/funcs"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"
	"testing"
)

func CreatePoolTest(t *testing.T) {
	kubeconfig := flag.String("kubeconfig", filepath.Join("/app", "kube.config.internal"), "(optional) absolute path to the kubeconfig file")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	funcs.CreatePools(*dynamicClient)
}
