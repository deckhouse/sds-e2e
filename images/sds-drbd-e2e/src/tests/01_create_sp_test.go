package test

import (
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
	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join("/app", "kube.config.internal")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		panic(err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	funcs.CreatePools(*dynamicClient)
}
