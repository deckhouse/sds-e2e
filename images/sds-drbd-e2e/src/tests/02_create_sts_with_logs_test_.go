package test

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"sds-drbd-e2e/funcs"
	"testing"
)

func TestCreateStsLogs(t *testing.T) {
	kubeconfigPath := os.Getenv("kubeconfig")

	fmt.Printf(kubeconfigPath)
	if kubeconfigPath == "" {
		kubeconfigPath = filepath.Join("/app", "kube.config.internal")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		t.Error(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Error(err)
	}

	funcs.CreateLogSts(*clientset)
}
