package sds_node_configurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddBDtoThinLVG(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		t.Error("Kubeclient problem", err)
	}

	extCl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("Parent cluster kubeclient problem", err)
	}

	for _, vmName := range []string{"vm1", "vm2", "vm3"} {
		vmdName := fmt.Sprintf("%s-thindata-add", vmName)
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

	time.Sleep(5 * time.Second)

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
				t.Log(fmt.Sprintf("Waiting: VD %s status: %s", disk.Name, disk.Status.Phase))
			}
		}
		if allVDRun {
			break
		}
	}

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
			if strings.Contains(disk.Name, "-thindata") && disk.Status.NodeName == lvg.Status.Nodes[0].Name {
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
				t.Log(fmt.Sprintf("Waiting: LVG %s status: %s", lvg.Name, lvg.Status.Phase))
			}
		}
		if allLVGRun {
			break
		}
	}
}
