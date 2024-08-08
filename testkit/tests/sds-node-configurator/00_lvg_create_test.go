package sds_node_configurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"github.com/melbahja/goph"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"
	"testing"
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
		t.Error("error listing block devices")
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

	for _, ip := range []string{funcs.MasterNodeIP, funcs.InstallWorkerNodeIp, funcs.WorkerNode2} {
		auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
		if err != nil {
			t.Error("SSH connection problem", err)
		}
		client := funcs.GetSSHClient(ip, "user", auth)
		defer client.Close()

		out, err := client.Run("sudo vgdisplay -C")
		if !strings.Contains(string(out), "data") || !strings.Contains(string(out), "20.00g") || err != nil {
			t.Error("error running vgdisplay -C", err)
		}
	}
}
