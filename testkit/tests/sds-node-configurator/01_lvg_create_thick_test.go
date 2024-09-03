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

func TestCreateLVG(t *testing.T) {
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
				Name: fmt.Sprintf("%s-lvg", device.Status.NodeName),
			},
			Spec: snc.LvmVolumeGroupSpec{
				ActualVGNameOnTheNode: "data",
				BlockDeviceNames:      []string{device.Name},
				Type:                  "Local",
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

	for _, ip := range []string{"10.10.10.180", "10.10.10.181", "10.10.10.182"} {
		client := funcs.GetSSHClient(ip, "user")
		out, _ := funcs.GetPVDisplay(client)
		fmt.Printf(out)
		out, _ = funcs.GetVGDisplay(client)
		fmt.Printf(out)
	}
}
