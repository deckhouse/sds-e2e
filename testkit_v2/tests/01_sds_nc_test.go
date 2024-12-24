package integration

import (
	"fmt"
//	"strings"
	"testing"
	"time"

//	"github.com/deckhouse/sds-e2e/funcs"
//	"github.com/melbahja/goph"
//	coreapi "k8s.io/api/core/v1"
	util "github.com/deckhouse/sds-e2e/util"
    snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
//	"k8s.io/apimachinery/pkg/api/resource"
)


var clr *util.KCluster

func TestNode(t *testing.T) {
	defaultClr, err := util.InitKCluster("", "")
    if err != nil {
		t.Fatal("Kubeclient problem", err)
    }
	clr = defaultClr

	t.Run("Node check", testPerNode)
}

func TestLVG(t *testing.T) {
	defaultClr, err := util.InitKCluster("", "")
    if err != nil {
		t.Fatal("Kubeclient problem", err)
    }
	clr = defaultClr

	t.Run("LVG creating", testLVGCreate)
	time.Sleep(5 * time.Second)
	t.Run("LVG resizing", testLVGResize)
	t.Run("LVG deleting", testLVGDelete)
}

func testPerNode(t *testing.T) {
	//nodeList, _ := clr.GetNodes(nil)
	nodeMap, err := clr.GetNodes(&util.Filter{Os: []string{"Ubuntu", "RedOS"}})
	if err != nil {
		t.Fatal("node error", err)
	}

    for _, node := range nodeMap {
		util.Infof(node.ObjectMeta.Name)
    }
}

func testLVGCreate(t *testing.T) {
	bdMap := map[string]*snc.BlockDevice{}
    lvgList, _ := clr.GetTestLVGs()
    for _, lvg := range lvgList {
        if len(lvg.Status.Nodes) == 0 {
			continue
        }
		bdMap[lvg.Status.Nodes[0].Name] = nil
	}

    bds, _ := clr.GetBDs(&util.Filter{Consumable: "true"})
	// or check bd.Status.LVMVolumeGroupName for valid BDs
    for _, bd := range bds {
		if _, ok := bdMap[bd.Status.NodeName]; ok {
			// only one test LVG for node
			continue
		}

		if _, err := clr.CreateTestLVG(bd.Status.NodeName, bd.Name); err != nil {
			t.Error("LVG creating:", err)
		}
		util.Infof("LVG created for BD %s", bd.Name)
		bdMap[bd.Status.NodeName] = &bd
    }
}

func testLVGResize(t *testing.T) {
    bds, _ := clr.GetBDs(&util.Filter{Consumable: "true", Os: []string{"Ubuntu", "RedOS"}})
	bdMap := map[string]*snc.BlockDevice{}
    for _, bd := range bds {
		bdMap[bd.Status.NodeName] = &bd
    }
    lvgList, err := clr.GetTestLVGs()
	if err != nil {
		t.Error("LVG getting:", err)
	}
	lvgUpdated := false

    for _, lvg := range lvgList {
		origSize := lvg.Status.VGSize
        if len(lvg.Status.Nodes) == 0 {
            t.Error(fmt.Sprintf("LVG: no nodes for %s", lvg.Name))
			continue
//        } else if lvg.Status.Nodes[0].Devices[0].PVSize.String() != "20Gi" || lvg.Status.Nodes[0].Devices[0].DevSize.String() != "20975192Ki" {
//            t.Error(fmt.Sprintf("LVG %s: size problem %s, %s", lvg.Name, lvg.Status.Nodes[0].Devices[0].PVSize.String(), lvg.Status.Nodes[0].Devices[0].DevSize.String()))
//        } else {
//            fmt.Printf("LVG %s: size ok %s, %s\n", lvg.Name, lvg.Status.Nodes[0].Devices[0].PVSize.String(), lvg.Status.Nodes[0].Devices[0].DevSize.String())
        }
		bd, ok := bdMap[lvg.Status.Nodes[0].Name]
		if !ok {
			util.Warnf("Have no extra BlockDevice for Node %s", lvg.Status.Nodes[0].Name)
			continue
		}
		bdSelector := lvg.Spec.BlockDeviceSelector.MatchExpressions[0]
		bdSelector.Values = append(bdSelector.Values, bd.Name)
		_ = clr.UpdateLVG(&lvg)

		if lvg.Status.VGSize.Value() <= origSize.Value() {
			t.Error(fmt.Sprintf("LVG %s resize %s problem", lvg.Name, origSize.String()))
		}
		lvgUpdated = true
    }

	if !lvgUpdated {
		util.Errf("No accessible resources for LVG resize test")
	}

/*
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
*/
}

func testLVGDelete(t *testing.T) {
	if err := clr.DelTestLVG();  err != nil {
		t.Error("LVG deleting:", err)
	}
}
