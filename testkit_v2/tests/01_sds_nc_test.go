package integration

import (
	"fmt"
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
)

func TestLvgCreate(t *testing.T) {
	clr := util.GetCluster("", "")

	clr.RunTestGroupNodes(t, directLVGCreate)

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

	err := clr.CheckStatusLVGs(util.LvgFilter{Name: util.Cond{Contains: []string{"e2e-lvg-"}}})
	if err != nil {
		t.Error(err.Error())
	}
}

func TestLvgResize(t *testing.T) {
	clr := util.GetCluster("", "")
	// Resize (exclusion "Deb11" for example)
	for group, nodes := range clr.GetGroupNodes(util.NodeFilter{NodeGroup: util.Cond{NotIn: []string{"Deb11"}}}) {
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
}

func TestLvgDelete(t *testing.T) {
	clr := util.GetCluster("", "")
	if err := clr.DeleteLVG(util.LvgFilter{Name: util.Cond{Contains: []string{"e2e-lvg-"}}}); err != nil {
		t.Error("LVG deleting:", err)
	}
}

func directLVGCreate(t *testing.T, tNode util.TestNode) {
	util.Infof("Start LVG create for %s.%s", tNode.GroupName, tNode.Name)
	nodeName, bdCount := tNode.Name, (tNode.Id%3)+1
	clr := util.GetCluster("", "")

	lvgMap, _ := clr.GetLVGs(util.LvgFilter{Name: util.Cond{Contains: []string{"e2e-lvg-"}}, Node: util.Cond{In: []string{nodeName}}})
	if len(lvgMap) > 0 {
		util.Infof("test LVG already exists")
		return
	}

	if util.HypervisorKubeConfig != "" {
		// create bd
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		for i := 1; i <= bdCount; i++ {
			vmdName := fmt.Sprintf("%s-data-%d", nodeName, i)
			util.Debugf("AttachVMBD %s", vmdName)
			hypervisorClr.AttachVMBD(nodeName, vmdName, "linstor-r1", 10)
		}

		_ = hypervisorClr.WaitVMBDs(util.TestNS, nodeName, "")
	}

	bds, _ := clr.GetBDs(util.BdFilter{Node: util.Cond{In: []string{nodeName}}, Consumable: util.Cond{In: []string{"true"}}})
	if len(bds) < bdCount {
		t.Errorf("not enough Device to create LVG (%d < %d)", len(bds), bdCount)
		return
	}

	for bdName, bd := range bds {
		if bdCount <= 0 {
			break
		}

		name := "e2e-lvg-" + nodeName[len(nodeName)-1:] + "-" + bdName[len(bdName)-3:]
		if _, err := clr.CreateLVG(name, nodeName, bdName); err != nil {
			util.Errf("LVG creating:", err.Error())
			continue
		}
		util.Debugf("LVG %s created for BD %s", name, bd.Name)
		bdCount--
	}

	if bdCount > 0 {
		t.Errorf("Not all LVGs created")
	}
}

func testLVGResize(t *testing.T, nodeName string) {
	clr := util.GetCluster("", "")
	bds, _ := clr.GetBDs(util.BdFilter{Node: util.Cond{In: []string{nodeName}}, Consumable: util.Cond{In: []string{"true"}}})
	if len(bds) == 0 {
		if util.SkipOptional {
			util.Warnf("skip resize LVG test for %s", nodeName)
			t.Skip("no Device to resize LVG")
		}
		t.Fatal("no Device to resize LVG")
	}
	bdMap := map[string]*snc.BlockDevice{}
	for _, bd := range bds {
		bdMap[bd.Status.NodeName] = &bd
	}
	lvgMap, _ := clr.GetLVGs(util.LvgFilter{Name: util.Cond{Contains: []string{"e2e-lvg-"}}})
	lvgUpdated := false

	for _, lvg := range lvgMap {
		if len(lvg.Status.Nodes) == 0 {
			util.Errf("LVG: no nodes for %s", lvg.Name)
			continue
		}
		bd, ok := bdMap[lvg.Status.Nodes[0].Name]
		if !ok {
			util.Debugf("Have no extra BlockDevice for Node %s", lvg.Status.Nodes[0].Name)
			util.Debugf("%#v", bdMap)
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
}
