package integration

import (
	"testing"
	"time"

	util "github.com/deckhouse/sds-e2e/util"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
)

func TestLVG(t *testing.T) {
	clr := util.GetCluster("", "")

	// Create all (split by group/node)
	for group, nodes := range clr.GetGroupNodes() {
		t.Run("create_"+group, func(t *testing.T) {
			if len(nodes) == 0 {
				t.Skip("no Nodes for case")
			}
			for _, nodeName := range nodes {
				t.Run(nodeName, func(t *testing.T) {
					t.Parallel()
					testLVGCreate(t, nodeName)
				})
			}
		})
	}

	/* [SAMPLE] Create all (split by node)
	t.Run("create", func(t *testing.T) {
		nodeMap, _ := clr.GetNodes()
		....
	})*/

	/* [SAMPLE] Create exact nodes (split by node)
	t.Run("create", func(t *testing.T) {
		nodeMap, _ := clr.GetNodes(util.Filter{NodeGroup: []string{"Deb11", "Ubu22", "Red7"}})
		....
	})*/

	time.Sleep(5 * time.Second)

	// Resize with exclusion (split by group/node)
	for group, nodes := range clr.GetGroupNodes(util.Filter{NotNodeGroup: []string{"Deb11"}}) {
		t.Run("resize_"+group, func(t *testing.T) {
			if len(nodes) == 0 {
				t.Skip("no Nodes for case")
			}
			for _, nodeName := range nodes {
				t.Run(nodeName, func(t *testing.T) {
					t.Parallel()
					testLVGResize(t, nodeName)
				})
			}
		})
	}

	// Delete
	t.Run("delete", testLVGDelete)

}

func testLVGCreate(t *testing.T, nodeName string) {
	clr := util.GetCluster("", "")
	lvgMap, _ := clr.GetLVGs(&util.Filter{Name: []string{"e2e-lvg-"}, Node: []string{nodeName}})
	if len(lvgMap) > 0 {
		util.Infof("test LVG already exists")
		return
	}

	bds, _ := clr.GetBDs(&util.Filter{Consumable: "true", Node: []string{nodeName}})
	// or check bd.Status.LVMVolumeGroupName for valid BDs
	if util.SkipFlag && len(bds) == 0 {
		util.Warnf("skip create LVG test for %s", nodeName)
		t.Skip("no Device to create LVG")
	}
	for bdName, bd := range bds {
		name := "e2e-lvg-" + nodeName[len(nodeName)-1:] + "-" + bdName[len(bdName)-3:]
		if _, err := clr.CreateLVG(name, nodeName, bdName); err != nil {
			t.Error("LVG creating:", err)
			continue
		}
		util.Infof("LVG created for BD %s", bd.Name)
		return
	}

	t.Fatal("no LVG created")
}

func testLVGResize(t *testing.T, nodeName string) {
	clr := util.GetCluster("", "")
	bds, _ := clr.GetBDs(&util.Filter{Consumable: "true", Node: []string{nodeName}})
	if util.SkipFlag && len(bds) == 0 {
		util.Warnf("skip resize LVG test for %s", nodeName)
		t.Skip("no Device to resize LVG")
	}
	bdMap := map[string]*snc.BlockDevice{}
	for _, bd := range bds {
		bdMap[bd.Status.NodeName] = &bd
	}
	lvgMap, _ := clr.GetLVGs(&util.Filter{Name: []string{"e2e-lvg-"}})
	lvgUpdated := false

	for _, lvg := range lvgMap {
		if len(lvg.Status.Nodes) == 0 {
			util.Errf("LVG: no nodes for %s", lvg.Name)
			continue
		//} else if lvg.Status.Nodes[0].Devices[0].PVSize.String() != "20Gi" || lvg.Status.Nodes[0].Devices[0].DevSize.String() != "20975192Ki" {
		//	t.Error(fmt.Sprintf("LVG %s: size problem %s, %s", lvg.Name, lvg.Status.Nodes[0].Devices[0].PVSize.String(), lvg.Status.Nodes[0].Devices[0].DevSize.String()))
		//} else {
		//	fmt.Printf("LVG %s: size ok %s, %s\n", lvg.Name, lvg.Status.Nodes[0].Devices[0].PVSize.String(), lvg.Status.Nodes[0].Devices[0].DevSize.String())
		}
		bd, ok := bdMap[lvg.Status.Nodes[0].Name]
		if !ok {
			util.Debugf("Have no extra BlockDevice for Node %s", lvg.Status.Nodes[0].Name)
			continue
		}
		origSize := lvg.Status.VGSize
		bdSelector := lvg.Spec.BlockDeviceSelector.MatchExpressions[0]
		bdSelector.Values = append(bdSelector.Values, bd.Name)
		if err := clr.UpdateLVG(&lvg); err != nil {
			util.Errf("LVG updating: %s", err)
			continue
		}

		if lvg.Status.VGSize.Value() <= origSize.Value() {
			util.Errf("LVG %s resize %s problem", lvg.Name, origSize.String())
			continue
		}
		lvgUpdated = true
	}

	if !lvgUpdated {
		t.Fatalf("No resized LVG for Node %s", nodeName)
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
	clr := util.GetCluster("", "")
	if err := clr.DeleteLVG(&util.Filter{Name: []string{"e2e-lvg-"}}); err != nil {
		t.Error("LVG deleting:", err)
	}
}
