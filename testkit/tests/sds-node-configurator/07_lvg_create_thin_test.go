package sds_node_configurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateThinLVG(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		funcs.LogFatalIfError(err, "kubeclient problem")
	}

	listDevice := &snc.BlockDeviceList{}
	err = cl.List(ctx, listDevice)
	if err != nil {
		t.Error("error listing block devices", err)
	}

	for _, device := range listDevice.Items {
		lvmVolumeGroup := &snc.LvmVolumeGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-thin-lvg", device.Status.NodeName),
			},
			Spec: snc.LvmVolumeGroupSpec{
				ActualVGNameOnTheNode: "datathin",
				BlockDeviceNames:      []string{device.Name},
				Type:                  "Local",
				ThinPools:             []snc.LvmVolumeGroupThinPoolSpec{{Name: "thinpool", Size: "50%"}},
			},
		}
		err = cl.Create(ctx, lvmVolumeGroup)
		if err != nil {
			t.Error("error creating lvm volume group")
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

	//
	//for _, ip := range []string{funcs.MasterNodeIP, funcs.InstallWorkerNodeIp, funcs.WorkerNode2} {
	//	auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
	//	if err != nil {
	//		t.Error("SSH connection problem", err)
	//	}
	//	client := funcs.GetSSHClient(ip, "user", auth)
	//	defer client.Close()
	//
	//	out, err := client.Run("sudo vgdisplay -C")
	//	fmt.Println(string(out))
	//	if !strings.Contains(string(out), "data") || !strings.Contains(string(out), "20.00g") || err != nil {
	//		t.Error("error running vgdisplay -C", err)
	//	}
	//}
}
