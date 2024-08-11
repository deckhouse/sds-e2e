package sds_node_configurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"github.com/melbahja/goph"
	"path/filepath"
	"strings"
	"testing"
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
		} else if len(LVMVG.Status.Nodes) == 0 || LVMVG.Status.Nodes[0].Devices[0].PVSize.String() != "20Gi" || LVMVG.Status.Nodes[0].Devices[0].DevSize.String() != "20975192Ki" {
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

	for _, ip := range []string{funcs.MasterNodeIP, funcs.InstallWorkerNodeIp, funcs.WorkerNode2} {
		auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
		if err != nil {
			t.Error("SSH connection problem", err)
		}
		client := funcs.GetSSHClient(ip, "user", auth)
		defer client.Close()

		funcs.ExecuteSSHCommandWithCheck(client, ip, "sudo vgs", []string{"data", "20.00g"})
		funcs.ExecuteSSHCommandWithCheck(client, ip, "sudo vgdisplay -C", []string{"data", "20.00g"})
		funcs.ExecuteSSHCommandWithCheck(client, ip, "sudo lsblk", []string{"sdc", "20G"})
		funcs.ExecuteSSHCommandWithCheck(client, ip, "sudo pvs", []string{"/dev/sdc", "20G"})
	}
}
