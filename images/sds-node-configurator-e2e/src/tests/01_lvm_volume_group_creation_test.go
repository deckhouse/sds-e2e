package test

import (
	"context"
	"github.com/melbahja/goph"
	"path/filepath"
	"sds-node-configurator-e2e/funcs"
	"strings"
	"testing"
)

const (
	appTmpPath = "/app/tmp"

	privKeyName = "id_rsa_test"
)

func TestLvmVolumeGroupCreation(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error("kubeclient error", err)
	}

	devices, err := funcs.GetAPIBlockDevices(ctx, cl)
	if err != nil {
		t.Error("get error", err)
	}

	for _, item := range devices {
		t.Log(funcs.CreateLvmVolumeGroup(ctx, cl, item.Status.NodeName, []string{item.ObjectMeta.Name}))
	}

	for _, ip := range []string{"10.10.10.180", "10.10.10.181", "10.10.10.182"} {
		auth, err := goph.Key(filepath.Join(appTmpPath, privKeyName), "")
		if err != nil {
			t.Fatal(err)
		}
		client := funcs.GetSSHClient(ip, "user", auth)
		defer client.Close()
		out, err := client.Run("sudo vgdisplay -C")
		if !strings.Contains(string(out), "data") || !strings.Contains(string(out), "20.00g") || err != nil {
			t.Fatal("vgdisplay -C error", err)
		}
		t.Log("vgdisplay -C", string(out))
	}
}
