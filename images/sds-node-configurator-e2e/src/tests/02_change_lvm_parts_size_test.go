package test

import (
	"github.com/melbahja/goph"
	"path/filepath"
	"sds-node-configurator-e2e/funcs"
	"strings"
	"testing"
)

func TestLvmPartsSizeChange(t *testing.T) {
	for _, ip := range []string{"10.10.10.180", "10.10.10.181", "10.10.10.182"} {
		auth, err := goph.Key(filepath.Join(AppTmpPath, PrivKeyName), "")
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
