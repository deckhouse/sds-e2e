/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"fmt"
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
)

const (
	testPrefix = "e2e-01-"
)

func cleanup01() {
	if !util.KeepState {
		removeTestDisks()
	}
}

func TestLvg(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup01)

	t.Run("create", func(t *testing.T) {
		clr.RunTestGroupNodes(t, nil, directLVGCreate)
		if err := clr.WaitLVGsReady(util.LvgFilter{Name: util.WhereLike{testPrefix}}); err != nil {
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

	lvgs, _ := clr.ListLVG(util.LvgFilter{Name: util.WhereLike{testPrefix}, Node: util.WhereIn{t.Node.Name}})
	if len(lvgs) > 0 {
		t.Skipf("LVG already exists for %s", t.Node.Name)
	}

	if util.HypervisorKubeConfig != "" {
		// create bd on VM
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		for i := 1; i <= bdCount; i++ {
			vmdName := fmt.Sprintf("%s-data-%d", t.Node.Name, i)
			err := hypervisorClr.CreateVMBD(t.Node.Name, vmdName, HvStorageClass, 6)
			if err != nil {
				t.Fatalf("Hypervisor CreateVMBD error: %s", err.Error())
			}
			util.Debugf("Attach VMBD %s", vmdName)
		}

		_ = hypervisorClr.WaitVmbdAttached(util.VmBdFilter{NameSpace: util.TestNS, VmName: t.Node.Name})
	}

	var bds []snc.BlockDevice
	if err := util.RetrySec(5, func() error {
		bds, _ = clr.ListBD(util.BdFilter{Node: t.Node.Name, Consumable: true})
		if len(bds) < bdCount {
			return fmt.Errorf("%s: not enough Device to create LVG (%d < %d)", t.Node.Name, len(bds), bdCount)
		}
		return nil
	}); err != nil {
		t.Fatal(err.Error())
	}

	for _, bd := range bds[:bdCount] {
		name := testPrefix + t.Node.Name[len(t.Node.Name)-1:] + "-" + bd.Name[len(bd.Name)-3:]
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
		_ = hypervisorClr.CreateVMBD(t.Node.Name, vmdName, HvStorageClass, 8)

		_ = hypervisorClr.WaitVmbdAttached(util.VmBdFilter{NameSpace: util.TestNS, VmName: t.Node.Name})
	}

	lvgs, _ := clr.ListLVG(util.LvgFilter{Name: util.WhereLike{testPrefix}, Node: util.WhereIn{t.Node.Name}})
	if len(lvgs) == 0 || len(lvgs[0].Status.Nodes) == 0 {
		t.Fatalf("No LVG for Node %s", t.Node.Name)
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
	if err := clr.DeleteLVG(util.LvgFilter{Name: util.WhereLike{testPrefix}}); err != nil {
		t.Fatalf("LVG deleting error: %s", err.Error())
	}

	if err := util.RetrySec(10, func() error {
		lvgs, err := clr.ListLVG(util.LvgFilter{Name: util.WhereLike{testPrefix}})
		if err != nil {
			return err
		}
		if len(lvgs) > 0 {
			return fmt.Errorf("LVGs not deleted: %d", len(lvgs))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if util.HypervisorKubeConfig != "" {
		hypervisorClr := util.GetCluster(util.HypervisorKubeConfig, "")
		err := hypervisorClr.DeleteVmbdAndWait(util.VmBdFilter{NameSpace: util.TestNS})
		if err != nil {
			t.Errorf("VMBD deleting error: %s", err)
		}
		err = hypervisorClr.DeleteVdAndWait(util.VdFilter{NameSpace: util.TestNS, Name: "!%-system%"})
		if err != nil {
			t.Errorf("VD deleting error: %s", err)
		}
	}
}
