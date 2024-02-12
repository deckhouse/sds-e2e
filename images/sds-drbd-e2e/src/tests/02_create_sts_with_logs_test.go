package test

import (
	"flag"
	"fmt"
	"gocontainer/funcs"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateStsLogs(t *testing.T) {
	kubeconfigPath := os.Getenv("kubeconfig")

	fmt.Printf(kubeconfigPath)
	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join("/app", "kube.config.internal")
	}

	if flag.Lookup("kubeconfig") == nil {
		kubeconfig := flag.String("kubeconfig", kubeconfigPath, "(optional) absolute path to the kubeconfig file")
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	funcs.CreateLogSts(*clientset)
}
