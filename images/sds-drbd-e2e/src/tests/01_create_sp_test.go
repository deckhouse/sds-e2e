package test

import (
	"flag"
	"fmt"
	"gocontainer/funcs"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"testing"
)

func TestCreatePool(t *testing.T) {
	kubeconfigPath := os.Getenv("kubeconfig")
	fmt.Printf(kubeconfigPath)
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
