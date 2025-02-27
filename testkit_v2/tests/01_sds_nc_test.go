package integration

import (
	"fmt"
	"testing"
	"time"

	util "github.com/deckhouse/sds-e2e/util"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
)

func directLVGCreate(t *testing.T, tNode util.TestNode) {
	nodeName, bdCount := tNode.Name, (tNode.Id%3)+1
	clr := util.GetCluster("", "")

	lvgs, _ := clr.ListLVG(util.LvgFilter{Name: "%e2e-lvg-%", Node: util.WhereIn{nodeName}})
	if len(lvgs) > 0 {
		util.Infof("test LVG already exists")
		return
	}

	if util.HypervisorKubeConfig != "" {
		// create bd on VM
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		for i := 1; i <= bdCount; i++ {
			vmdName := fmt.Sprintf("%s-data-%d", nodeName, i)
			util.Debugf("Attach VMBD %s", vmdName)
			_ = hypervisorClr.CreateVMBD(nodeName, vmdName, "linstor-r1", 6)
		}

		_ = hypervisorClr.WaitVMBD(util.VmBdFilter{NameSpace: util.TestNS, VmName: nodeName})
	}

	bds, _ := clr.ListBD(util.BdFilter{Node: nodeName, Consumable: true})
	if len(bds) < bdCount {
		t.Errorf("%s: not enough Device to create LVG (%d < %d)", tNode.Name, len(bds), bdCount)
		return
	}

	for _, bd := range bds {
		if bdCount <= 0 {
			break
		}

		name := "e2e-lvg-" + nodeName[len(nodeName)-1:] + "-" + bd.Name[len(bd.Name)-3:]
		if _, err := clr.CreateLVG(name, nodeName, bd.Name); err != nil {
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

func TestLvgCreate(t *testing.T) {
	clr := util.GetCluster("", "")

	clr.RunTestGroupNodes(t, directLVGCreate)

	err := clr.CheckStatusLVGs(util.LvgFilter{Name: "%e2e-lvg-%"})
	if err != nil {
		t.Error(err.Error())
	}
}

func directLVGResize(t *testing.T, nodeName string) {
	clr := util.GetCluster("", "")

	if util.HypervisorKubeConfig != "" {
		// create bd on VM
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		vmdName := fmt.Sprintf("%s-data-%d", nodeName, 21)
		util.Debugf("Add VMBD %s", vmdName)
		_ = hypervisorClr.CreateVMBD(nodeName, vmdName, "linstor-r1", 8)

		_ = hypervisorClr.WaitVMBD(util.VmBdFilter{NameSpace: util.TestNS, VmName: nodeName})
	}

	bds, _ := clr.ListBD(util.BdFilter{Node: nodeName, Consumable: true})
	if len(bds) == 0 {
		if util.SkipOptional {
			util.Warnf("skip resize LVG test for %s", nodeName)
		}
		t.Errorf("no Device to resize '%s' LVGs", nodeName)
		return
	}
	bdMap := map[string]*snc.BlockDevice{}
	for _, bd := range bds {
		bdMap[bd.Status.NodeName] = &bd
	}
	lvgs, _ := clr.ListLVG(util.LvgFilter{Name: "%e2e-lvg-%"})
	lvgUpdated := false

	for _, lvg := range lvgs {
		if len(lvg.Status.Nodes) == 0 {
			util.Errf("LVG: no nodes for %s", lvg.Name)
			continue
		}
		bd, ok := bdMap[lvg.Status.Nodes[0].Name]
		if !ok {
			util.Debugf("Have no extra BlockDevice for Node %s", lvg.Status.Nodes[0].Name)
			continue
		}
		lvgName, origSize := lvg.Name, lvg.Status.VGSize
		bdSelector := lvg.Spec.BlockDeviceSelector.MatchExpressions[0]
		bdSelector.Values = append(bdSelector.Values, bd.Name)
		lvg.Spec.BlockDeviceSelector.MatchExpressions[0] = bdSelector
		if err := clr.UpdateLVG(&lvg); err != nil {
			util.Errf("LVG updating: %s", err)
			continue
		}

		for i := 0; ; i++ {
			newLvgs, _ := clr.ListLVG(util.LvgFilter{Name: lvgName})
			lvg = newLvgs[0]
			if lvg.Status.VGSize.Value() > origSize.Value() {
				lvgUpdated = true
				break
			}
			if i >= 5 {
				util.Errf("LVG %s resize problem: %d <= %d", lvgName, lvg.Status.VGSize.Value(), origSize.Value())
				break
			}
			time.Sleep(5 * time.Second)
		}
	}

	if !lvgUpdated {
		t.Errorf("No resized LVG for Node %s", nodeName)
	}
}

func TestLvgResize(t *testing.T) {
	clr := util.GetCluster("", "")
	// Resize (exclusion "Deb11" for example)
	for group, nodes := range clr.MapLabelNodes(util.WhereNotIn{"Deb11"}) {
		t.Run("resize_"+group, func(t *testing.T) {
			if len(nodes) == 0 {
				t.Skip("no Nodes for case")
			}
			for _, nodeName := range nodes {
				t.Run(nodeName, func(t *testing.T) {
					t.Parallel()
					directLVGResize(t, nodeName)
				})
			}
		})
	}
}

func TestLvgDelete(t *testing.T) {
	clr := util.GetCluster("", "")
	if err := clr.DeleteLVG(util.LvgFilter{Name: "%e2e-lvg-%"}); err != nil {
		t.Error("LVG deleting:", err)
	}

	if util.HypervisorKubeConfig != "" {
		// delete virtual disks
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		_ = hypervisorClr.DeleteVMBD(util.VmBdFilter{NameSpace: util.TestNS})
	}
}
