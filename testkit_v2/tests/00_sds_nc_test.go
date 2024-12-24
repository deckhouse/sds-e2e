package integration

import (
	"fmt"
	"strings"
	"testing"
	"time"
//	"log"

//	"github.com/deckhouse/sds-e2e/funcs"
//	"github.com/melbahja/goph"
	util "github.com/deckhouse/sds-e2e/util"
)


var clr *util.KCluster

func TestLVG(t *testing.T) {
	defaultClr, err := util.InitKCluster("", "")
    if err != nil {
		t.Fatal("Kubeclient problem", err)
    }
	clr = defaultClr

	t.Run("LVG creating for each BlockDevice", testCreateLVG)
	time.Sleep(5 * time.Second)
	t.Run("LVG size changing", testChangeLVGSize)
	t.Run("LVG removing", testRemoveLVG)
}

func testCreateLVG(t *testing.T) {
// test BlockDevice
	bds, _ := clr.GetBDs()
    for _, bd := range bds {
		if _, err := clr.CreateLVG("", bd.Status.NodeName, bd.Name); err != nil {
            t.Error("LVG creating", err)
        }
    }
// ----

// TODO test vgdisplay (запкстить когда начнет работать выполнение команд на node)
/*
    for _, ip := range []string{funcs.MasterNodeIP, funcs.InstallWorkerNodeIp, funcs.WorkerNode2} {
		fmt.Println(">>> 5 >>>")
        auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
		fmt.Println(">>> 5.1 >>>")
        if err != nil {
            t.Error("SSH connection problem", err)
        }
        client := funcs.GetSSHClient(ip, "user", auth)
		fmt.Println(">>> 5.2 >>>")
        defer client.Close()

        out, err := client.Run("sudo vgdisplay -C")
		fmt.Println(">>> 5.3 >>>")
        fmt.Printf("OUT: %#v, ERR: %#v\n", string(out), err)
        if !strings.Contains(string(out), "data") || !strings.Contains(string(out), "20.00g") || err != nil {
            t.Error("error running vgdisplay -C", err)
        }
		fmt.Println(">>> 5.4 >>>")
    }
*/
}

func testChangeLVGSize(t *testing.T) {
	lvgList, _ := clr.GetTestLVGs()
	for _, lvg := range lvgList {
		if len(lvg.Status.Nodes) == 0 {
			t.Error(fmt.Sprintf("LVG %s: node is empty", lvg.Name))
		} else if lvg.Status.Nodes[0].Devices[0].PVSize.String() != "20Gi" || lvg.Status.Nodes[0].Devices[0].DevSize.String() != "20975192Ki" {
			t.Error(fmt.Sprintf("LVG %s: size problem %s, %s", lvg.Name, lvg.Status.Nodes[0].Devices[0].PVSize.String(), lvg.Status.Nodes[0].Devices[0].DevSize.String()))
//    node name: d-shipkov-worker-0, problem with size: 22524Mi, 22Gi
		} else {
			fmt.Printf("LVG %s: size ok %s, %s\n", lvg.Name, lvg.Status.Nodes[0].Devices[0].PVSize.String(), lvg.Status.Nodes[0].Devices[0].DevSize.String())
		}
	}

	vmdList, _ := clr.GetTestVMD() // kube_vm.config
	if len(vmdList) == 0 {
		t.Error("Disk update problem, no VMDs")
	}
	for _, vmd := range vmdList {
		if strings.Contains(vmd.Name, "-data") {
			vmd.Spec.PersistentVolumeClaim.Size.Set(32212254720)
			err := clr.UpdateVMD(&vmd)
			if err != nil {
				t.Error("Disk update problem", err)
			}
		}
	}

	// TODO test vgdisplay (запкстить когда начнет работать выполнение команд на node)
/*
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
*/
}

func testRemoveLVG(t *testing.T) {
	if err := clr.DelTestLVG(); err != nil {
		t.Error("lvmVolumeGroup delete error", err)
	}
}

//func TestTest1(t *testing.T) {
//	t.Error("The problem", fmt.Errorf("Fail 1"))
//	log.Println("Private Key generated")
////	log.Fatal("ooops")
//	t.Error("The problem", fmt.Errorf("Fail 2"))
//	t.Fatal("The problem", fmt.Errorf("Fail 3"))
//	t.Error("The problem", fmt.Errorf("Fail 4"))
//	return
//}
