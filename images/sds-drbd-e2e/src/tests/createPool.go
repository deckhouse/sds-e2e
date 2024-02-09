package tests

import (
	"context"
	"flag"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
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

	CreatePools(*dynamicClient)
}

func CreatePools(client dynamic.DynamicClient) {
	bdRes := schema.GroupVersionResource{Group: "storage.deckhouse.io", Version: "v1alpha1", Resource: "blockdevices"}
	lvmvgRes := schema.GroupVersionResource{Group: "storage.deckhouse.io", Version: "v1alpha1", Resource: "lvmvolumegroups"}
	drbdspRes := schema.GroupVersionResource{Group: "storage.deckhouse.io", Version: "v1alpha1", Resource: "drbdstoragepools"}
	drbdscRes := schema.GroupVersionResource{Group: "storage.deckhouse.io", Version: "v1alpha1", Resource: "drbdstorageclasses"}

	ctx := context.Background()

	listedResources, err := client.Resource(bdRes).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Fatal(err)
	}

	for _, item := range listedResources.Items {
		bdName, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
		nodeName, _, _ := unstructured.NestedString(item.Object, "status", "nodeName")
		fmt.Printf("%s\n\n", bdName)
		fmt.Printf("%s\n\n", nodeName)
		lvmvgobj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "storage.deckhouse.io/v1alpha1",
				"kind":       "LvmVolumeGroup",
				"metadata": map[string]interface{}{
					"name": nodeName,
				},
				"spec": map[string]interface{}{
					"actualVGNameOnTheNode": "lvmthin",
					"blockDeviceNames": []string{
						bdName,
					},
					"type": "Local",
				},
			},
		}
		output, err := client.Resource(lvmvgRes).Create(ctx, lvmvgobj, metav1.CreateOptions{})
		fmt.Printf("%s\n%s\n", output, err)
	}

	lvmvgList := []map[string]interface{}{}
	listedResources, err = client.Resource(lvmvgRes).List(ctx, metav1.ListOptions{})
	for _, item := range listedResources.Items {
		lvmvgName, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
		fmt.Printf("%s\n", lvmvgName)
		lvmvgList = append(lvmvgList, map[string]interface{}{"name": lvmvgName, "thinpoolname": ""})
	}

	fmt.Printf("%s\n", lvmvgList)

	drbdspobj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "storage.deckhouse.io/v1alpha1",
			"kind":       "DRBDStoragePool",
			"metadata": map[string]interface{}{
				"name": "data",
			},
			"spec": map[string]interface{}{
				"lvmvolumegroups": lvmvgList,
				"type":            "LVM",
			},
		},
	}

	output, err := client.Resource(drbdspRes).Create(ctx, drbdspobj, metav1.CreateOptions{})
	fmt.Printf("%s\n%s\n", output, err)

	drbdSCObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "storage.deckhouse.io/v1alpha1",
			"kind":       "DRBDStorageClass",
			"metadata": map[string]interface{}{
				"name": "linstor-r1",
			},
			"spec": map[string]interface{}{
				"isDefault":     false,
				"reclaimPolicy": "Delete",
				"replication":   "None",
				"storagePool":   "data",
				"topology":      "Ignored",
				"volumeAccess":  "PreferablyLocal",
			},
		},
	}

	output, err = client.Resource(drbdscRes).Create(ctx, drbdSCObj, metav1.CreateOptions{})
	fmt.Printf("%s\n%s\n", output, err)

	drbdSCObj = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "storage.deckhouse.io/v1alpha1",
			"kind":       "DRBDStorageClass",
			"metadata": map[string]interface{}{
				"name": "linstor-r2",
			},
			"spec": map[string]interface{}{
				"isDefault":     true,
				"reclaimPolicy": "Delete",
				"replication":   "Availability",
				"storagePool":   "data",
				"topology":      "Ignored",
				"volumeAccess":  "PreferablyLocal",
			},
		},
	}

	output, err = client.Resource(drbdscRes).Create(ctx, drbdSCObj, metav1.CreateOptions{})
	fmt.Printf("%s\n%s\n", output, err)

}
