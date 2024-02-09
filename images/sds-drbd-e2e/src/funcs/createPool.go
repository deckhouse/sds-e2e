package funcs

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

var ctx = context.Background()

var lvmVolumeGroupRes = schema.GroupVersionResource{Group: "storage.deckhouse.io", Version: "v1alpha1", Resource: "lvmvolumegroups"}
var blockDeviceRes = schema.GroupVersionResource{Group: "storage.deckhouse.io", Version: "v1alpha1", Resource: "blockdevices"}
var drbdStoragePoolRes = schema.GroupVersionResource{Group: "storage.deckhouse.io", Version: "v1alpha1", Resource: "drbdstoragepools"}
var drbdStorageClassRes = schema.GroupVersionResource{Group: "storage.deckhouse.io", Version: "v1alpha1", Resource: "drbdstorageclasses"}

func CreateLvmVolumeGroup(client dynamic.DynamicClient, blockDeviceName string, lvmVolumeGroupName string, vgName string) (*unstructured.Unstructured, error) {
	lvmVolumeGroupObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "storage.deckhouse.io/v1alpha1",
			"kind":       "LvmVolumeGroup",
			"metadata": map[string]interface{}{
				"name": lvmVolumeGroupName,
			},
			"spec": map[string]interface{}{
				"actualVGNameOnTheNode": vgName,
				"blockDeviceNames": []string{
					blockDeviceName,
				},
				"type": "Local",
			},
		},
	}

	output, err := client.Resource(lvmVolumeGroupRes).Create(ctx, lvmVolumeGroupObj, metav1.CreateOptions{})
	return output, err
}

func CreateDrbdStoragePool(client dynamic.DynamicClient, lvmVolumeGroupList []map[string]interface{}, storagePoolName string) (*unstructured.Unstructured, error) {
	drbdStoragePoolObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "storage.deckhouse.io/v1alpha1",
			"kind":       "DRBDStoragePool",
			"metadata": map[string]interface{}{
				"name": storagePoolName,
			},
			"spec": map[string]interface{}{
				"lvmvolumegroups": lvmVolumeGroupList,
				"type":            "LVM",
			},
		},
	}

	output, err := client.Resource(drbdStoragePoolRes).Create(ctx, drbdStoragePoolObj, metav1.CreateOptions{})
	return output, err
}

func CreateDrbdStorageClass(client dynamic.DynamicClient, storageClassName string, replication string, isDefault bool) (*unstructured.Unstructured, error) {
	drbdStorageClassObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "storage.deckhouse.io/v1alpha1",
			"kind":       "DRBDStorageClass",
			"metadata": map[string]interface{}{
				"name": storageClassName,
			},
			"spec": map[string]interface{}{
				"isDefault":     isDefault,
				"reclaimPolicy": "Delete",
				"replication":   replication,
				"storagePool":   "data",
				"topology":      "Ignored",
				"volumeAccess":  "PreferablyLocal",
			},
		},
	}

	output, err := client.Resource(drbdStorageClassRes).Create(ctx, drbdStorageClassObj, metav1.CreateOptions{})
	return output, err
}

func CreatePools(client dynamic.DynamicClient) {
	listedResources, err := client.Resource(blockDeviceRes).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Fatal(err)
	}

	for _, item := range listedResources.Items {
		blockDeviceName, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
		nodeName, _, _ := unstructured.NestedString(item.Object, "status", "nodeName")
		fmt.Printf("%s\n\n", blockDeviceName)
		fmt.Printf("%s\n\n", nodeName)

		output, err := CreateLvmVolumeGroup(client, blockDeviceName, nodeName, "lvm")

		fmt.Printf("%s\n%s\n", output, err)
	}

	lvmVolumeGroupList := []map[string]interface{}{}
	listedResources, err = client.Resource(lvmVolumeGroupRes).List(ctx, metav1.ListOptions{})
	for _, item := range listedResources.Items {
		lvmvgName, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
		fmt.Printf("%s\n", lvmvgName)
		lvmVolumeGroupList = append(lvmVolumeGroupList, map[string]interface{}{"name": lvmvgName, "thinpoolname": ""})
	}
	fmt.Printf("%s\n", lvmVolumeGroupList)

	output, err := CreateDrbdStoragePool(client, lvmVolumeGroupList, "data")
	fmt.Printf("%s\n%s\n", output, err)

	output, err = CreateDrbdStorageClass(client, "linstor-r1", "None", false)
	fmt.Printf("%s\n%s\n", output, err)

	output, err = CreateDrbdStorageClass(client, "linstor-r2", "Availability", true)
	fmt.Printf("%s\n%s\n", output, err)
}
