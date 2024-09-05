package sds_node_configurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	"k8s.io/apimachinery/pkg/api/resource"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChangeLVGSize(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		t.Error("Kubeclient problem", err)
	}

	extCl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("Parent cluster kubeclient problem", err)
	}

	listDevice := &snc.LvmVolumeGroupList{}
	err = cl.List(ctx, listDevice)
	if err != nil {
		t.Error(err)
	}

	for _, LVMVG := range listDevice.Items {
		if len(LVMVG.Status.Nodes) == 0 {
			t.Error("LVMVG node is empty", LVMVG.Name)
		} else if len(LVMVG.Status.Nodes) == 0 || LVMVG.Status.Nodes[0].Devices[0].PVSize != resource.MustParse("20Gi") {
			t.Error(fmt.Sprintf("node name: %s, problem with size: %s, %s", LVMVG.Status.Nodes[0].Name, LVMVG.Status.Nodes[0].Devices[0].PVSize.String(), LVMVG.Status.Nodes[0].Devices[0].DevSize.String()))
		} else {
			fmt.Printf("node name: %s, size ok: %s, %s\n", LVMVG.Status.Nodes[0].Name, LVMVG.Status.Nodes[0].Devices[0].PVSize.String(), LVMVG.Status.Nodes[0].Devices[0].DevSize.String())
		}
	}

	vmdList, err := funcs.ListVMD(ctx, extCl, funcs.NamespaceName, "")
	for _, vmd := range vmdList {
		if strings.Contains(vmd.Name, "-data") {
			vmd.Spec.PersistentVolumeClaim.Size.Set(32212254720)
			err := extCl.Update(ctx, &vmd)
			if err != nil {
				t.Error("Disk update problem", err)
			}
		}
	}

	time.Sleep(5 * time.Second)

	t.Log(fmt.Sprintf("Waiting: VD to resize"))

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

	t.Log(fmt.Sprintf("VD resized"))

	listDataDisks := &v1alpha2.VirtualDiskList{}
	err = extCl.List(ctx, listDataDisks)
	if err != nil {
		t.Error("Disk retrieve failed", err)
	}
	for _, disk := range listDataDisks.Items {
		if strings.Contains(disk.Name, "-system") {
			continue
		}
		t.Log(fmt.Sprintf("Disk OK. Name: %s, status: %s, size: %s", disk.Name, disk.Status.Phase, disk.Status.Capacity))
	}

	time.Sleep(5 * time.Second)

	t.Log(fmt.Sprintf("Waiting: LVGs resize"))

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

	t.Log(fmt.Sprintf("LVGs resized"))

	for _, LVMVG := range listDevice.Items {
		if len(LVMVG.Status.Nodes) == 0 {
			t.Error("LVMVG node is empty", LVMVG.Name)
		} else if len(LVMVG.Status.Nodes) == 0 || LVMVG.Status.Nodes[0].Devices[0].PVSize != resource.MustParse("30Gi") {
			t.Error(fmt.Sprintf("node name: %s, problem with size (must be 30Gi): %s, %s", LVMVG.Status.Nodes[0].Name, LVMVG.Status.Nodes[0].Devices[0].PVSize.String(), LVMVG.Status.Nodes[0].Devices[0].DevSize.String()))
		} else {
			fmt.Printf("node name: %s, size ok: %s, %s\n", LVMVG.Status.Nodes[0].Name, LVMVG.Status.Nodes[0].Devices[0].PVSize.String(), LVMVG.Status.Nodes[0].Devices[0].DevSize.String())
		}
	}

	//for _, ip := range []string{"10.10.10.180", "10.10.10.181", "10.10.10.182"} {
	//	client := funcs.GetSSHClient(ip, "user")
	//	out, _ := funcs.GetLSBLK(client)
	//	fmt.Printf(out)
	//	out, _ = funcs.GetPVS(client)
	//	fmt.Printf(out)
	//	out, _ = funcs.GetVGS(client)
	//	fmt.Printf(out)
	//	out, _ = funcs.GetVGDisplay(client)
	//	fmt.Printf(out)
	//}
}
