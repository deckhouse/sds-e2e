package integration

import (
	"fmt"

	util "github.com/deckhouse/sds-e2e/util"
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
)

const (
	lvmD8        = "/opt/deckhouse/sds/bin/lvm.static"
	lsblkCmd     = "sudo /opt/deckhouse/bin/lsblk"
	vgsCmd       = "sudo " + lvmD8 + " vgs"
	pvsCmd       = "sudo " + lvmD8 + " pvs"
	vgdisplayCmd = "sudo " + lvmD8 + " vgdisplay"
	pvdisplayCmd = "sudo " + lvmD8 + " pvdisplay"
	lvdisplayCmd = "sudo " + lvmD8 + " lvdisplay"
)

// Remove all deprecated resources from cluster
func prepareClr() {
	rmLvgBd()
}

// Remove LVGs, VMBDs, VDs, BDs
func rmLvgBd() {
	clr := util.GetCluster("", "")

	lvgs, _ := clr.ListLVG(util.LvgFilter{Name: "%e2e-lvg-%"})
	for _, lvg := range lvgs {
		nName := lvg.Spec.Local.NodeName
		_, _, _ = clr.ExecNode(nName, []string{"sudo", lvmD8, "lvremove", "-y", lvg.Name})
	}
	_ = clr.DeleteLvgWithCheck(util.LvgFilter{Name: "%e2e-lvg-%"})

	if util.HypervisorKubeConfig != "" {
		hvClr := util.GetCluster(util.HypervisorKubeConfig, "")
		_ = hvClr.DeleteVmbdWithCheck(util.VmBdFilter{NameSpace: util.TestNS})
		_ = hvClr.DeleteVdWithCheck(util.VdFilter{NameSpace: util.TestNS, Name: "!%-system%"})
	}
	_ = clr.DeleteBdWithCheck()
}

// Provides N devices with size M on node
func ensureBdConsumable(nName string, size int64, count int) ([]snc.BlockDevice, error) {
	clr := util.GetCluster("", "")
	bds, _ := clr.ListBD(util.BdFilter{Node: nName, Consumable: true, Size: float32(size)})
	if len(bds) >= int(count) {
		return bds, nil
	}

	if util.HypervisorKubeConfig == "" {
		return nil, fmt.Errorf("Not enough bds on %s: %d of %d", nName, len(bds), count)
	}
	hvClr := util.GetCluster(util.HypervisorKubeConfig, "")
	for i := len(bds); i < count; i++ {
		err := hvClr.CreateVMBD(nName, nName+"-data-"+util.RandString(4), "linstor-r1", size)
		if err != nil {
			return nil, err
		}
	}

	if err := util.RetrySec(30, func() error {
		bds, _ := clr.ListBD(util.BdFilter{Node: nName, Consumable: true, Size: float32(size)})
		if len(bds) < int(count) {
			return fmt.Errorf("Not enough bds on %s: %d of %d", nName, len(bds), count)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return clr.ListBD(util.BdFilter{Node: nName, Consumable: true, Size: float32(size)})
}
