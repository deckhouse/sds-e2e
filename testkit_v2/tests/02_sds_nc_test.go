package integration

import (
//	"fmt"
	"testing"
	"time"

	util "github.com/deckhouse/sds-e2e/util"
//	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
)

func TestLVGByGroup(t *testing.T) {
	clr := util.GetCluster("", "")

	for g, nodes := range clr.GetGroupNodes(nil) {
		t.Run(g, func(t *testing.T) {
			// Create
			nodeMap, _ := clr.GetNodes(&util.Filter{Node: nodes})
			for nodeName, _ := range nodeMap {
				t.Run("create " + nodeName, func(t *testing.T) {
					testLVGCreate(t, nodeName)
				})
			}

			// Resize
			time.Sleep(5 * time.Second)
			nodeMap, _ = clr.GetNodes(&util.Filter{Node: nodes, NotOs: []string{"Debian"}})
			for nodeName, node := range nodeMap {
				t.Run("resize " + nodeName, func(t *testing.T) {
					util.Infof("Node image: %s", node.Status.NodeInfo.OSImage)
					if err := testLVGResize(t, nodeName); err != nil {
						t.Error(err)
					}
				})
			}

			// Delete
			t.Run("delete", testLVGDelete)
		})
	}
}

/*
func testLVGCreate(t *testing.T, nodeName string) {
	clr := util.GetCluster("", "")
	lvgMap, _ := clr.GetLVGs(&util.Filter{Name: []string{"e2e-lvg-"}, Node: []string{nodeName}})
	if len(lvgMap) > 0 {
		util.Infof("test LVG already exists")
		return
	}

	bds, _ := clr.GetBDs(&util.Filter{Consumable: "true", Node: []string{nodeName}})
	if util.SkipFlag && len(bds) == 0 {
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

func testLVGResize(t *testing.T, nodeName string) error {
	clr := util.GetCluster("", "")
	bds, _ := clr.GetBDs(&util.Filter{Consumable: "true", Node: []string{nodeName}})
	if util.SkipFlag && len(bds) == 0 {
		t.Skip("no Device to resize LVG")
	}
	bdMap := map[string]*snc.BlockDevice{}
	for _, bd := range bds {
		bdMap[bd.Status.NodeName] = &bd
	}
	lvgMap, err := clr.GetLVGs(&util.Filter{Name: []string{"e2e-lvg-"}})
	if err != nil {
		return err
	}
	lvgUpdated := false

	for _, lvg := range lvgMap {
		if len(lvg.Status.Nodes) == 0 {
			util.Errf("LVG: no nodes for %s", lvg.Name)
			continue
		}
		bd, ok := bdMap[lvg.Status.Nodes[0].Name]
		if !ok {
			continue
		}
		origSize := lvg.Status.VGSize
		bdSelector := lvg.Spec.BlockDeviceSelector.MatchExpressions[0]
		bdSelector.Values = append(bdSelector.Values, bd.Name)
		if err = clr.UpdateLVG(&lvg); err != nil {
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
		return fmt.Errorf("No resized LVG for Node %s", nodeName)
	}

	return nil
}

func testLVGDelete(t *testing.T) {
	clr := util.GetCluster("", "")
	if err := clr.DeleteLVG(&util.Filter{Name: []string{"e2e-lvg-"}});  err != nil {
		t.Error("LVG deleting:", err)
	}
}
*/
