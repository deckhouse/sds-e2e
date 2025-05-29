package integration

import (
	"fmt"
	"strings"
	"testing"
	"time"

	util "github.com/deckhouse/sds-e2e/util"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ================ LVM THICK TESTS ================

// 1 - Create LVMVolumeGroup. Check VG, PV auto creating
func TestLvgThickCreateCascade(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name
		cmdPrefix := "sudo " + lvmD8 + " "
		if t.Node.Id%2 == 1 {
			cmdPrefix = "sudo lvm "
			if strings.Contains(t.Node.Raw.Status.NodeInfo.OSImage, "Debian") {
				_, _, _ = clr.ExecNode(nName, []string{"sudo", "apt", "-y", "install", "lvm2"})
			}
		}

		lvg, err := directLvgCreate(nName, 2)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := clr.ExecNodeRespContains(nName, cmdPrefix+"vgdisplay --units B", []string{
			"VG Name\\s+" + lvg.Name,
			"VG Size\\s+21[45]\\d{7} B", //2.00 GiB(2147483648 B) +- 20 MiB
			"Alloc PE / Size[\\s\\d]+ 0 / 0 B",
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespContains(nName, cmdPrefix+"pvdisplay --units B", []string{
			"VG Name\\s+" + lvg.Name,
			"PV Size\\s+21[45]\\d{7} B  /", //2.00 GiB(2147483648 B) +- 20 MiB
		}); err != nil {
			t.Error(err.Error())
		}
	})
}

// 2 - Delete LVMVolumeGroup. Check VG, PV auto deleting
func TestLvgThickDeleteCascadeManually(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name
		lvg, err := directLvgCreate(nName, 3)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		_ = clr.DeleteLvgAndWait(util.LvgFilter{Name: lvg.Name})

		if err := clr.ExecNodeRespNotContains(nName, vgdisplayCmd, []string{
			"VG Name ",
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespNotContains(nName, pvdisplayCmd, []string{
			"VG Name ",
		}); err != nil {
			t.Error(err.Error())
		}
	})
}

// 3 - Increase BlockDevice size. Check LVG, PV, VG resizing
func TestLvgThickDiskResize(t *testing.T) {
	clr := util.GetCluster("", "")
	if util.HypervisorKubeConfig == "" {
		t.Fatal("No HypervisorKubeConfig to resize VD")
	}
	prepareClr()
	t.Cleanup(cleanup05)

	hvClr := util.GetCluster(util.HypervisorKubeConfig, "")
	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name
		lvg, err := directLvgCreate(nName, 1)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{1}, []string{"1.00g"}, "1.00g"); err != nil {
			t.Error(err.Error())
		}

		vdList, _ := hvClr.ListVD(util.VdFilter{NameSpace: util.TestNS, Name: "%" + t.Node.Name + "-data-%"})
		if len(vdList) == 0 {
			t.Fatalf("Non VD for node %s", t.Node.Name)
		}
		for _, vd := range vdList {
			vd.Spec.PersistentVolumeClaim.Size = resource.NewQuantity(2*1024*1024*1024, resource.BinarySI)
			if err := hvClr.UpdateVd(&vd); err != nil {
				t.Fatal(err.Error())
			}
		}

		if err := util.RetrySec(10, func() error {
			return checkNodeLvgSize(lvg.Name, []float32{2}, []string{"2.00g"}, "2.00g")
		}); err != nil {
			t.Error(err.Error())
		}
	})
}

// 4 - Add second BlockDevice to LVG. Check LVG, PV, VG resizing
func TestLvgThickAddBd(t *testing.T) {
	clr := util.GetCluster("", "")
	if util.HypervisorKubeConfig == "" {
		t.Fatal("No HypervisorKubeConfig to add VD")
	}
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name
		lvg, err := directLvgCreate(nName, 1)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{1}, []string{"1.00g"}, "1.00g"); err != nil {
			t.Error(err.Error())
		}

		bds, err := getOrCreateConsumableBlockDevices(nName, 2, 1)
		if err != nil {
			t.Fatal(err.Error())
		}
		bdSelector := lvg.Spec.BlockDeviceSelector.MatchExpressions[0]
		bdSelector.Values = append(bdSelector.Values, bds[0].Name)
		lvg.Spec.BlockDeviceSelector.MatchExpressions[0] = bdSelector
		if err := clr.UpdateLVG(lvg); err != nil {
			t.Fatalf("LVG updating: %s", err.Error())
		}

		if err := util.RetrySec(20, func() error {
			lvg, err := clr.GetLvg(lvg.Name)
			if err != nil {
				return err
			}
			if lvg.Status.VGSize.Value() != int64(3)*1024*1024*1024 {
				return fmt.Errorf("VG %s size: %d != 3Gi", lvg.Name, lvg.Status.VGSize.Value())
			}
			return nil
		}); err != nil {
			t.Fatal(err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{1, 2}, []string{"1.00g", "2.00g"}, "3.00g"); err != nil {
			t.Error(err.Error())
		}
	})
}

// 5 - Reconnect BlockDevice to another path. Check LVG no changes
func TestLvgThickReconnectBd(t *testing.T) {
	clr := util.GetCluster("", "")
	if util.HypervisorKubeConfig == "" {
		t.Fatal("No HypervisorKubeConfig to add VD")
	}
	prepareClr()
	t.Cleanup(cleanup05)

	hvClr := util.GetCluster(util.HypervisorKubeConfig, "")
	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name
		lvg, err := directLvgCreate(nName, 1)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{1}, []string{"1.00g"}, "1.00g"); err != nil {
			t.Error(err.Error())
		}

		vmbds, _ := hvClr.ListVMBD(util.VmBdFilter{VmName: nName, Name: "%-data-%"})
		oldBdPath := lvg.Status.Nodes[0].Devices[0].Path

		err = hvClr.DetachVmbd(util.VmBdFilter{VmName: nName, Name: "%-data-%"})
		if err != nil {
			t.Fatal(err.Error())
		}
		bdName := lvg.Spec.BlockDeviceSelector.MatchExpressions[0].Values[0]
		_ = clr.DeleteBd(util.BdFilter{Name: bdName})

		_, _ = getOrCreateConsumableBlockDevices(nName, 2, 1)

		for _, vmbd := range vmbds {
			_ = hvClr.AttachVmbd(nName, vmbd.Name)
		}
		_ = hvClr.WaitVmbdAttached(util.VmBdFilter{VmName: nName})

		lvg, err = clr.GetLvg(lvg.Name)
		if err != nil {
			t.Fatal(err.Error())
		}
		if len(lvg.Status.Nodes) == 0 {
			t.Fatalf("No nodes in LVG %s status", lvg.Name)
		}
		newBdPath := lvg.Status.Nodes[0].Devices[0].Path
		if newBdPath == oldBdPath {
			t.Fatalf("Can`t change BD path on node %s", nName)
		}
		if lvg.Status.Phase != "Ready" {
			t.Fatalf("LVG %s not Ready: %s", lvg.Name, lvg.Status.Phase)
		}
		if err := checkNodeLvgSize(lvg.Name, []float32{1}, []string{"1.00g"}, "1.00g"); err != nil {
			t.Error(err.Error())
		}
	})
}

// 6 - Add new LV to empty VG. Check VG allocated size increase
func TestVgThickAddLv(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name

		vgName := "e2e-vg-" + util.RandString(4)
		bds, err := getOrCreateConsumableBlockDevices(nName, 1, 1)
		if err != nil {
			t.Fatal(err.Error())
		}
		stOut, stErr, err := clr.ExecNode(nName, []string{"sudo", lvmD8, "vgcreate", vgName, bds[0].Status.Path})
		if err != nil {
			util.Debugf("vgcreate stOut: %s", stOut)
			util.Debugf("vgcreate stErr: %s", stErr)
			t.Fatal(err.Error())
		}

		if err := clr.ExecNodeRespContains(nName, vgdisplayCmd, []string{
			"VG Name\\s+" + vgName,
			"Alloc PE / Size[\\s\\d]+ 0 / 0",
		}); err != nil {
			t.Error(err.Error())
		}

		stOut, stErr, err = clr.ExecNode(nName, []string{"sudo", lvmD8, "lvcreate", "-L", "500m", vgName})
		if err != nil {
			util.Debugf("vgcreate stOut: %s", stOut)
			util.Debugf("vgcreate stErr: %s", stErr)
			t.Fatal(err.Error())
		}

		if err := clr.ExecNodeRespContains(nName, vgdisplayCmd, []string{
			"VG Name\\s+" + vgName,
			"Alloc PE / Size[\\s\\d]+ / 500.00 MiB",
		}); err != nil {
			t.Error(err.Error())
		}
	})
}

// ================ LVM THIN TESTS ================

// 1 - Create LVMVolumeGroup on ThinPools. Check VG, PV, LV auto creating
func TestLvgThinCreateCascade(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name
		cmdPrefix := "sudo " + lvmD8 + " "
		if t.Node.Id%2 == 1 {
			cmdPrefix = "sudo lvm "
			if strings.Contains(t.Node.Raw.Status.NodeInfo.OSImage, "Debian") {
				_, _, _ = clr.ExecNode(nName, []string{"sudo", "apt", "-y", "install", "lvm2"})
			}
		}

		lvg, err := directLvgTpCreate(nName, 3.33)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := clr.ExecNodeRespContains(nName, cmdPrefix+"vgdisplay --units B", []string{
			"VG Name\\s+" + lvg.Name,
			"VG Size\\s+4(27|28|29|30|31)\\d{7} B",         //4.00 GiB(4294967296 B) +- 20 MiB
			"Alloc PE / Size\\s+\\d+ / 3(49|5\\d)\\d{7} B", //3.3 GiB(3543348019 B) +- 50 MiB
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespContains(nName, cmdPrefix+"pvdisplay --units B", []string{
			"VG Name\\s+" + lvg.Name,
			"PV Size\\s+4(27|28|29|30|31)\\d{7} B  /", //4.00 GiB(4294967296 B) +- 20 MiB
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespContains(nName, cmdPrefix+"lvdisplay", []string{
			"LV Name\\s+thin-e2e-01",
			"LV Name\\s+thin-e2e-02",
			"LV Size\\s+1.00 GiB",
			"LV Size\\s+2.33 GiB",
		}); err != nil {
			t.Error(err.Error())
		}
	})
}

// 2 - Delete LV before LVMVolumeGroup. Check VG, PV auto deleting
func TestLvgThinDeleteCascadeManually(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		var out string
		nName := t.Node.Name

		lvg, err := directLvgTpCreate(t.Node.Name, 1.8)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		vgName := lvg.Spec.ActualVGNameOnTheNode
		out, _, err = clr.ExecNode(nName, []string{"sudo", lvmD8, "lvremove", "-y", "/dev/" + vgName + "/thin-e2e-01"})
		if err != nil {
			util.Debugf("lvremove output: %s", out)
			t.Fatal(err.Error())
		}

		_ = clr.DeleteLvgAndWait(util.LvgFilter{Name: lvg.Name})

		if err := clr.ExecNodeRespNotContains(nName, vgdisplayCmd, []string{
			"VG Name ",
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespNotContains(nName, pvdisplayCmd, []string{
			"VG Name ",
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespNotContains(nName, lvdisplayCmd, []string{
			"LV Name ",
		}); err != nil {
			t.Error(err.Error())
		}
	})
}

// 3 - Delete LVMVolumeGroup. Check VG, PV still exist. Delete LV. Check VG, PV auto deleting
func TestLvgThinDeleteCascadeK8s(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		var out string
		nName := t.Node.Name

		lvg, err := directLvgTpCreate(nName, 1.6)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}
		vgName := lvg.Spec.ActualVGNameOnTheNode

		_ = clr.DeleteLVG(util.LvgFilter{Name: lvg.Name})
		time.Sleep(4 * time.Second)

		if err := clr.ExecNodeRespContains(nName, vgdisplayCmd, []string{
			"VG Name\\s+" + lvg.Name,
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespContains(nName, pvdisplayCmd, []string{
			"VG Name\\s+" + lvg.Name,
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespContains(nName, lvdisplayCmd, []string{
			"LV Name\\s+thin-e2e-01",
		}); err != nil {
			t.Error(err.Error())
		}

		out, _, err = clr.ExecNode(nName, []string{"sudo", lvmD8, "lvremove", "-y", "/dev/" + vgName + "/thin-e2e-01"})
		if err != nil {
			util.Debugf("lvremove output: %s", out)
			t.Fatal(err.Error())
		}
		time.Sleep(3 * time.Second)

		if err := clr.ExecNodeRespNotContains(nName, vgdisplayCmd, []string{
			"VG Name\\s+",
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespNotContains(nName, pvdisplayCmd, []string{
			"VG Name\\s+",
		}); err != nil {
			t.Error(err.Error())
		}
		if err := clr.ExecNodeRespNotContains(nName, lvdisplayCmd, []string{
			"LV Name\\s+",
		}); err != nil {
			t.Error(err.Error())
		}
	})
}

// 4.1 - Increase BlockDevice size. Check LVG, PV, VG resizing. Check ThinPools no changes
func TestLvgThinDiskResize(t *testing.T) {
	clr := util.GetCluster("", "")
	if util.HypervisorKubeConfig == "" {
		t.Fatal("No HypervisorKubeConfig to resize VD")
	}
	prepareClr()
	t.Cleanup(cleanup05)

	hvClr := util.GetCluster(util.HypervisorKubeConfig, "")
	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		lvg, err := directLvgTpCreate(t.Node.Name, 2.34)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{3}, []string{"660.00m"}, "660.00m"); err != nil {
			t.Error(err.Error())
		}
		if err = thinPoolsCheck(lvg.Name, 1.0, 1.34); err != nil {
			t.Error(err.Error())
		}

		vdList, _ := hvClr.ListVD(util.VdFilter{NameSpace: util.TestNS, Name: "%" + t.Node.Name + "-data-%"})
		if len(vdList) == 0 {
			t.Fatalf("Non VD for node %s", t.Node.Name)
		}
		for _, vd := range vdList {
			vd.Spec.PersistentVolumeClaim.Size = resource.NewQuantity(4*1024*1024*1024, resource.BinarySI)
			if err := hvClr.UpdateVd(&vd); err != nil {
				t.Fatal(err.Error())
			}
		}

		if err := util.RetrySec(10, func() error {
			return checkNodeLvgSize(lvg.Name, []float32{4}, []string{"1.64g"}, "1.64g")
		}); err != nil {
			t.Error(err.Error())
		}
		if err = thinPoolsCheck(lvg.Name, 1.0, 1.34); err != nil {
			t.Error(err.Error())
		}
	})
}

// 4.2 - Increase ThinPool size. Check LVG, PV, VG, ThinPool resizing
func TestLvgThinPoolResize(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		lvg, err := directLvgTpCreate(t.Node.Name, 2.34)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{3}, []string{"660.00m"}, "660.00m"); err != nil {
			t.Error(err.Error())
		}
		if err = thinPoolsCheck(lvg.Name, 1.0, 1.34); err != nil {
			t.Error(err.Error())
		}

		if lvg.Spec.ThinPools[0].Size != "1.0Gi" {
			t.Fatalf("Invalid ThinPool size: %s != 1.0Gi", lvg.Spec.ThinPools[0].Size)
		}
		lvg.Spec.ThinPools[0].Size = "1.21Gi"
		if err := clr.UpdateLVG(lvg); err != nil {
			t.Fatalf("LVG updating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{3}, []string{"444.00m"}, "444.00m"); err != nil {
			t.Error(err.Error())
		}
		if err = thinPoolsCheck(lvg.Name, 1.21, 1.34); err != nil {
			t.Error(err.Error())
		}
	})
}

// 4.3 - Increase ThinPool size over VG. Check LVG, PV, VG, ThinPool no changes
func TestLvgThinPoolOversize(t *testing.T) {
	clr := util.GetCluster("", "")
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		lvg, err := directLvgTpCreate(t.Node.Name, 2.34)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{3}, []string{"660.00m"}, "660.00m"); err != nil {
			t.Error(err.Error())
		}
		if err = thinPoolsCheck(lvg.Name, 1.0, 1.34); err != nil {
			t.Error(err.Error())
		}

		if lvg.Spec.ThinPools[0].Size != "1.0Gi" {
			t.Fatalf("Invalid ThinPool size: %s != 1.0Gi", lvg.Spec.ThinPools[0].Size)
		}
		lvg.Spec.ThinPools[0].Size = "1.8Gi"
		if err := clr.UpdateLVG(lvg); err != nil {
			t.Fatalf("LVG updating: %s", err.Error())
		}

		time.Sleep(3 * time.Second)
		lvg, _ = clr.GetLvg(lvg.Name)
		if lvg.Spec.ThinPools[0].Size != "1.8Gi" {
			t.Errorf("ThinPool size: %s != 1.8Gi", lvg.Spec.ThinPools[0].Size)
		}
		if lvg.Status.ConfigurationApplied != "False" {
			t.Errorf("LVG ConfigurationApplied: %s", lvg.Status.ConfigurationApplied)
		}
		if err := checkNodeLvgSize(lvg.Name, []float32{3}, []string{"660.00m"}, "660.00m"); err != nil {
			t.Error(err.Error())
		}
		if err = thinPoolsCheck(lvg.Name, 1.0, 1.34); err != nil {
			t.Error(err.Error())
		}
	})
}

// 5 - Add second BlockDevice to LVG. Check LVG, PV, VG resizing. Check ThinPools no changes
func TestLvgThinAddBd(t *testing.T) {
	clr := util.GetCluster("", "")

	if util.HypervisorKubeConfig == "" {
		t.Fatal("No HypervisorKubeConfig to add VD")
	}
	prepareClr()
	t.Cleanup(cleanup05)

	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name
		lvg, err := directLvgTpCreate(nName, 1.7)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{2}, []string{"296.00m"}, "296.00m"); err != nil {
			t.Error(err.Error())
		}
		if err = thinPoolsCheck(lvg.Name, 1.7); err != nil {
			t.Error(err.Error())
		}

		bds, err := getOrCreateConsumableBlockDevices(nName, 1, 1)
		if err != nil {
			t.Fatal(err.Error())
		}
		bdSelector := lvg.Spec.BlockDeviceSelector.MatchExpressions[0]
		bdSelector.Values = append(bdSelector.Values, bds[0].Name)
		lvg.Spec.BlockDeviceSelector.MatchExpressions[0] = bdSelector
		if err := clr.UpdateLVG(lvg); err != nil {
			t.Fatalf("LVG updating: %s", err.Error())
		}

		if err := util.RetrySec(20, func() error {
			lvg, _ = clr.GetLvg(lvg.Name)
			if lvg.Status.VGSize.Value() != int64(3)*1024*1024*1024 {
				return fmt.Errorf("VG %s size: %d != 3Gi", lvg.Name, lvg.Status.VGSize.Value())
			}
			return nil
		}); err != nil {
			t.Fatal(err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{2, 1}, []string{"296.00m", "1.00g"}, "<1.29g"); err != nil {
			t.Error(err.Error())
		}
		if err = thinPoolsCheck(lvg.Name, 1.7); err != nil {
			t.Error(err.Error())
		}
	})
}

// 6 - Reconnect BlockDevice to another path. Check LVG no changes
func TestLvgThinReconnectBd(t *testing.T) {
	clr := util.GetCluster("", "")
	if util.HypervisorKubeConfig == "" {
		t.Fatal("No HypervisorKubeConfig to add VD")
	}
	prepareClr()
	t.Cleanup(cleanup05)

	hvClr := util.GetCluster(util.HypervisorKubeConfig, "")
	clr.RunTestGroupNodes(t, nil, func(t *util.T) {
		nName := t.Node.Name
		lvg, err := directLvgTpCreate(nName, 1.1)
		if err != nil {
			t.Fatalf("LVG creating: %s", err.Error())
		}

		if err := checkNodeLvgSize(lvg.Name, []float32{2}, []string{"912.00m"}, "912.00m"); err != nil {
			t.Error(err.Error())
		}

		vmbds, _ := hvClr.ListVMBD(util.VmBdFilter{VmName: nName, Name: "%-data-%"})
		oldBdPath := lvg.Status.Nodes[0].Devices[0].Path

		err = hvClr.DetachVmbd(util.VmBdFilter{VmName: nName, Name: "%-data-%"})
		if err != nil {
			t.Fatal(err.Error())
		}
		bdName := lvg.Spec.BlockDeviceSelector.MatchExpressions[0].Values[0]
		_ = clr.DeleteBd(util.BdFilter{Name: bdName})

		_, _ = getOrCreateConsumableBlockDevices(nName, 2, 1)

		for _, vmbd := range vmbds {
			_ = hvClr.AttachVmbd(nName, vmbd.Name)
		}
		_ = hvClr.WaitVmbdAttached(util.VmBdFilter{VmName: nName})
		time.Sleep(time.Second)

		lvg, err = clr.GetLvg(lvg.Name)
		if err != nil {
			t.Fatal(err.Error())
		}
		newBdPath := lvg.Status.Nodes[0].Devices[0].Path
		if newBdPath == oldBdPath {
			t.Fatalf("Can`t change BD path on node %s", nName)
		}
		if lvg.Status.Phase != "Ready" {
			t.Fatalf("LVG %s not Ready: %s", lvg.Name, lvg.Status.Phase)
		}
		if err := checkNodeLvgSize(lvg.Name, []float32{2}, []string{"912.00m"}, "912.00m"); err != nil {
			t.Error(err.Error())
		}
	})
}

// ================ HELP TOOLS ================

func cleanup05() {
	if !util.KeepState {
		removeTestDisks()
	}
}

func checkNodeLvgSize(lvgName string, vSize []float32, vFree []string, vgFree string) error {
	clr := util.GetCluster("", "")

	lvg, _ := clr.GetLvg(lvgName)
	if len(lvg.Status.Nodes[0].Devices) != len(vSize) {
		return fmt.Errorf("LVG %s devices: %d != %d", lvgName, len(lvg.Status.Nodes[0].Devices), len(vSize))
	}

	nName := lvg.Spec.Local.NodeName
	vgName := lvg.Spec.ActualVGNameOnTheNode
	nChecks := map[string][]string{
		lsblkCmd: []string{},
		pvsCmd:   []string{},
	}
	vgSize := float32(0)
	validDiff := int64(5) * 1024 * 1024

	for devId, sizeG := range vSize {
		dSize := lvg.Status.Nodes[0].Devices[devId].DevSize.Value()
		pSize := lvg.Status.Nodes[0].Devices[devId].PVSize.Value()
		path := lvg.Status.Nodes[0].Devices[devId].Path
		sizeB := int64(sizeG * 1024 * 1024 * 1024)
		if dSize < sizeB-validDiff || dSize > sizeB+validDiff || pSize != sizeB {
			return fmt.Errorf("%s LVG size != %.2fG: dev %d pv %d", nName, sizeG, dSize, pSize)
		}
		vgSize += sizeG

		nChecks[lsblkCmd] = append(nChecks[lsblkCmd],
			fmt.Sprintf("%s [\\d\\s:]* %.0fG\\s+0\\s+disk\\s*\\n", path[len(path)-3:], sizeG))
		nChecks[pvsCmd] = append(nChecks[pvsCmd],
			fmt.Sprintf("%s\\s+%s [a-z\\d\\s-]+ %.2fg\\s+%s", path, lvg.Name, sizeG, vFree[devId]))
	}
	nChecks[vgsCmd] = []string{fmt.Sprintf("%s [a-z\\d\\s-]* %.2fg\\s+%s", vgName, vgSize, vgFree)}
	nChecks[vgdisplayCmd+" "+vgName] = []string{fmt.Sprintf("VG Size\\s+%.2f GiB\\n", vgSize)}

	vgSizeB := int64(vgSize) * 1024 * 1024 * 1024
	if lvg.Status.VGSize.Value() < vgSizeB-validDiff || lvg.Status.VGSize.Value() > vgSizeB+validDiff {
		return fmt.Errorf("%s VG size: %d != %.2fG", nName, lvg.Status.VGSize.Value(), vgSize)
	}

	for cmd, resp := range nChecks {
		if err := clr.ExecNodeRespContains(nName, cmd, resp); err != nil {
			return err
		}
	}

	return nil
}

func thinPoolsCheck(lvgName string, sizes ...float32) error {
	clr := util.GetCluster("", "")

	lvg, _ := clr.GetLvg(lvgName)
	tps := lvg.Status.ThinPools

	if len(tps) != len(sizes) {
		return fmt.Errorf("ThinPools size: %d != %d", len(tps), len(sizes))
	}

	for _, tp := range tps {
		tpOk := false
		for _, s := range sizes {
			s64 := int64(s * 1024 * 1024 * 1024)
			validDiff := int64(10) * 1024 * 1024
			if tp.ActualSize.Value() > s64-validDiff && tp.ActualSize.Value() < s64+validDiff {
				tpOk = true
				break
			}
		}
		if tp.AllocatedSize.Value() != 0 {
			return fmt.Errorf("ThinPool %s AllocatedSize != 0", tp.Name)
		}
		if !tpOk {
			return fmt.Errorf("ThinPool %s invalid ActualSize: %d", tp.Name, tp.ActualSize.Value())
		}
	}

	return nil
}

func directLvgCreate(nName string, size int64) (*snc.LVMVolumeGroup, error) {
	clr := util.GetCluster("", "")
	bds, err := getOrCreateConsumableBlockDevices(nName, size, 1)
	if err != nil {
		return nil, err
	}

	bd := bds[0]
	lvgName := "e2e-lvg-" + bd.Name[len(bd.Name)-4:]
	err = clr.CreateLvgWithCheck(lvgName, nName, []string{bd.Name})
	if err != nil {
		return nil, err
	}

	return clr.GetLvg(lvgName)
}

func directLvgTpCreate(nName string, size float32) (*snc.LVMVolumeGroup, error) {
	clr := util.GetCluster("", "")
	bds, err := getOrCreateConsumableBlockDevices(nName, int64(size+0.9999), 1)
	if err != nil {
		return nil, err
	}

	bd := bds[0]
	lvgName := "e2e-lvg-" + bd.Name[len(bd.Name)-4:]
	if size >= 2 {
		err = clr.CreateLvgExt(lvgName, nName, map[string]any{
			"bds": []string{bd.Name},
			"thinpools": []snc.LVMVolumeGroupThinPoolSpec{{
				Name:            "thin-e2e-01",
				Size:            "1.0Gi",
				AllocationLimit: "130%",
			}, {
				Name:            "thin-e2e-02",
				Size:            fmt.Sprintf("%.2fGi", size-1),
				AllocationLimit: "150%",
			}},
		})
	} else {
		err = clr.CreateLvgExt(lvgName, nName, map[string]any{
			"bds": []string{bd.Name},
			"thinpools": []snc.LVMVolumeGroupThinPoolSpec{{
				Name:            "thin-e2e-01",
				Size:            fmt.Sprintf("%.1fGi", size),
				AllocationLimit: "125%",
			}},
		})
	}
	if err != nil {
		return nil, err
	}

	if err := clr.WaitLVGsReady(util.LvgFilter{Name: lvgName}); err != nil {
		return nil, err
	}

	return clr.GetLvg(lvgName)
}
