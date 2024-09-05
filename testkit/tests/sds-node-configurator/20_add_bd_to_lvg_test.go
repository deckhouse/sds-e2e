package sds_node_configurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddBDtoLVG(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		t.Error("Kubeclient problem", err)
	}

	extCl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("Parent cluster kubeclient problem", err)
	}

	t.Log(fmt.Sprintf("Waiting: creating VD"))

	for _, vmName := range []string{"vm1", "vm2", "vm3"} {
		vmdName := fmt.Sprintf("%s-data-add", vmName)
		_, err = funcs.CreateVMD(ctx, extCl, funcs.NamespaceName, vmdName, funcs.StorageClass, 5)
		if err != nil {
			t.Error("Disk creation failed", err)
		}

		err = extCl.Create(ctx, &v1alpha2.VirtualMachineBlockDeviceAttachment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmdName,
				Namespace: funcs.NamespaceName,
			},
			Spec: v1alpha2.VirtualMachineBlockDeviceAttachmentSpec{
				VirtualMachineName: vmName,
				BlockDeviceRef: v1alpha2.VMBDAObjectRef{
					Kind: "VirtualDisk",
					Name: vmdName,
				},
			},
		})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Error("Disk attach failed", err)
		}
	}

	time.Sleep(30 * time.Second)

	for {
		allVDRun := true
		listDataDisks := &v1alpha2.VirtualDiskList{}
		err = extCl.List(ctx, listDataDisks)
		if err != nil {
			t.Error("Disk retrieve failed", err)
		}
		for _, disk := range listDataDisks.Items {
			if disk.Status.Phase != "Ready" {
				allVDRun = false
			}
		}
		if allVDRun {
			break
		}
	}

	t.Log(fmt.Sprintf("VD created"))

	listDataDisks := &v1alpha2.VirtualDiskList{}
	err = extCl.List(ctx, listDataDisks)
	if err != nil {
		t.Error("Disk retrieve failed", err)
	}
	for _, disk := range listDataDisks.Items {
		t.Log(fmt.Sprintf("Disk name: %s, status: %s, size: %s", disk.Name, disk.Status.Phase, disk.Status.Capacity))
	}

	t.Log(fmt.Sprintf("Waiting: adding BD to LVG"))

	listLVG := &snc.LvmVolumeGroupList{}
	err = cl.List(ctx, listLVG)
	if err != nil {
		t.Error("Lvm volume group list failed", err)
	}

	listBlockDevice := &snc.BlockDeviceList{}
	err = cl.List(ctx, listBlockDevice)
	if err != nil {
		t.Error("Block device list failed", err)
	}

	for _, lvg := range listLVG.Items {
		devices := []string{}
		for _, disk := range listBlockDevice.Items {
			if strings.Contains(disk.Name, "-data") && disk.Status.NodeName == lvg.Status.Nodes[0].Name {
				devices = append(devices, disk.Name)
			}
		}

		lvg.Spec.BlockDeviceNames = devices
		err = cl.Update(ctx, &lvg)
		if err != nil {
			t.Error("Lvm volume group update failed", err)
		}
	}

	time.Sleep(5 * time.Second)

	for {
		allLVGRun := true
		listLVG := snc.LvmVolumeGroupList{}
		err = cl.List(ctx, &listLVG)
		if err != nil {
			t.Error("LVG retrieve failed", err)
		}

		for _, lvg := range listLVG.Items {
			if lvg.Status.Phase != "Ready" {
				allLVGRun = false
			}
		}
		if allLVGRun {
			break
		}
	}

	t.Log(fmt.Sprintf("BD added to LVG"))

	time.Sleep(5 * time.Second)

	listLVG = &snc.LvmVolumeGroupList{}
	err = cl.List(ctx, listLVG)
	if err != nil {
		t.Error("Lvm volume group list failed", err)
	}
	for _, LVMVG := range listLVG.Items {
		if len(LVMVG.Status.Nodes) == 0 {
			t.Error("LVMVG node is empty", LVMVG.Name)
		} else if len(LVMVG.Status.Nodes) == 0 || LVMVG.Status.VGSize != resource.MustParse("30Gi") {
			t.Error(fmt.Sprintf("LVG name: %s, problem with size (must be 30Gi): %s", LVMVG.Name, LVMVG.Status.VGSize.String()))
		} else {
			fmt.Printf("LVG name: %s, size ok: %s\n", LVMVG.Name, LVMVG.Status.VGSize.String())
		}
	}

	//for _, ip := range []string{"10.10.10.180", "10.10.10.181", "10.10.10.182"} {
	//	client := funcs.GetSSHClient(ip, "user")
	//	out, _ := funcs.GetVGS(client)
	//	fmt.Printf(out)
	//	out, _ = funcs.GetVGDisplay(client)
	//	fmt.Printf(out)
	//}
}
