package integration

import (
	"fmt"
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
)

func TestLvg(t *testing.T) {
	clr := util.GetCluster("", "")

	t.Run("create", func(t *testing.T) {
		clr.RunTestGroupNodes(t, nil, directLVGCreate)
		if err := clr.CheckLVGsReady(util.LvgFilter{Name: "%e2e-lvg-%"}); err != nil {
			t.Fatal(err.Error())
		}
	})

	t.Run("resize", func(t *testing.T) {
		clr.RunTestGroupNodes(t, util.WhereNotLike{"Deb"}, directLVGResize)
	})

	t.Run("delete", directLVGDelete)
}

func directLVGCreate(t *util.T) {
	bdCount := (t.Node.Id % 3) + 1
	clr := util.GetCluster("", "")

	lvgs, _ := clr.ListLVG(util.LvgFilter{Name: "%e2e-lvg-%", Node: util.WhereIn{t.Node.Name}})
	if len(lvgs) > 0 {
		t.Skipf("LVG already exists for %s", t.Node.Name)
	}

	if util.HypervisorKubeConfig != "" {
		// create bd on VM
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		for i := 1; i <= bdCount; i++ {
			vmdName := fmt.Sprintf("%s-data-%d", t.Node.Name, i)
			err := hypervisorClr.CreateVMBD(t.Node.Name, vmdName, "linstor-r1", 6)
			if err != nil {
				t.Fatalf("Hypervisor CreateVMBD error: %s", err.Error())
			}
			util.Debugf("Attach VMBD %s", vmdName)
		}

		_ = hypervisorClr.WaitVmbdAttached(util.VmBdFilter{NameSpace: util.TestNS, VmName: t.Node.Name})
	}

	bds, _ := clr.ListBD(util.BdFilter{Node: t.Node.Name, Consumable: true})
	if len(bds) < bdCount {
		t.Errorf("%s: not enough Device to create LVG (%d < %d)", t.Node.Name, len(bds), bdCount)
		return
	}

	for _, bd := range bds[:bdCount] {
		name := "e2e-lvg-" + t.Node.Name[len(t.Node.Name)-1:] + "-" + bd.Name[len(bd.Name)-3:]
		if err := clr.CreateLVG(name, t.Node.Name, []string{bd.Name}); err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}
		util.Debugf("LVG %s created for BD %s", name, bd.Name)
	}
}

func directLVGResize(t *util.T) {
	clr := util.GetCluster("", "")

	if util.HypervisorKubeConfig != "" {
		// create bd on VM
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		vmdName := fmt.Sprintf("%s-data-%d", t.Node.Name, 21)
		util.Debugf("Add VMBD %s", vmdName)
		_ = hypervisorClr.CreateVMBD(t.Node.Name, vmdName, "linstor-r1", 8)

		_ = hypervisorClr.WaitVmbdAttached(util.VmBdFilter{NameSpace: util.TestNS, VmName: t.Node.Name})
	}

	lvgs, _ := clr.ListLVG(util.LvgFilter{Name: "%e2e-lvg-%", Node: util.WhereIn{t.Node.Name}})
	if len(lvgs) == 0 || len(lvgs[0].Status.Nodes) == 0 {
		t.Skipf("No LVG for Node %s", t.Node.Name)
	}
	lvg := &lvgs[len(lvgs)-1]

	var bdExtra *snc.BlockDevice
	bds, _ := clr.ListBD(util.BdFilter{Node: t.Node.Name, Consumable: true})
	for _, bd := range bds {
		if lvg.Status.Nodes[0].Name == bd.Status.NodeName {
			bdExtra = &bd
			break
		}
	}
	if bdExtra == nil {
		t.Skipf("No consumable BD for Node %s", t.Node.Name)
	}

	origSize := lvg.Status.VGSize.Value()
	bdSelector := lvg.Spec.BlockDeviceSelector.MatchExpressions[0]
	bdSelector.Values = append(bdSelector.Values, bdExtra.Name)
	lvg.Spec.BlockDeviceSelector.MatchExpressions[0] = bdSelector
	if err := clr.UpdateLVG(lvg); err != nil {
		t.Fatalf("LVG updating: %s", err.Error())
	}

	if err := util.RetrySec(30, func() error {
		lvg, _ := clr.GetLvg(lvg.Name)
		if lvg.Status.VGSize.Value() > origSize {
			return nil
		}
		return fmt.Errorf("LVG %s no resize", lvg.Name)
	}); err != nil {
		t.Fatal(err.Error())
	}
}

func directLVGDelete(t *testing.T) {
	clr := util.GetCluster("", "")
	if err := clr.DeleteLVG(util.LvgFilter{Name: "%e2e-lvg-%"}); err != nil {
		t.Fatalf("LVG deleting error: %s", err.Error())
	}

	if err := util.RetrySec(10, func() error {
		lvgs, err := clr.ListLVG(util.LvgFilter{Name: "%e2e-lvg-%"})
		if err != nil {
			return err
		}
		if len(lvgs) > 0 {
			return fmt.Errorf("LVGs not deleted: %d", len(lvgs))
		}
		return nil
	}); err != nil {
		t.Fatal(err.Error())
	}

	if util.HypervisorKubeConfig != "" {
		// delete virtual disks
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		err := hypervisorClr.DeleteVMBD(util.VmBdFilter{NameSpace: util.TestNS})
		if err != nil {
			util.Errorf("VMBD deleting error:", err)
		}

		if err := util.RetrySec(20, func() error {
			vds, err := hypervisorClr.ListVD(util.VdFilter{NameSpace: util.TestNS, Name: "!%-system%"})
			if err != nil {
				return err
			}
			if len(vds) > 0 {
				return fmt.Errorf("VDs not deleted: %d", len(vds))
			}
			return nil
		}); err != nil {
			t.Fatal(err.Error())
		}
	}
}
