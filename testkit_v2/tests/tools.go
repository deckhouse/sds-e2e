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
	removeTestDisks()
}

// Remove LVGs, VMBDs, VDs, BDs
func removeTestDisks() {
	cluster := util.EnsureCluster("", "")

	lvgs, _ := cluster.ListLVG(util.LvgFilter{Name: "%e2e-lvg-%"})
	for _, lvg := range lvgs {
		nName := lvg.Spec.Local.NodeName
		_, _, _ = cluster.ExecNode(nName, []string{"sudo", lvmD8, "lvremove", "-y", lvg.Name})
	}
	_ = cluster.DeleteLvgAndWait(util.LvgFilter{Name: "%e2e-lvg-%"})

	if util.HypervisorKubeConfig != "" {
		hvCluster := util.EnsureCluster(util.HypervisorKubeConfig, "")
		_ = hvCluster.DeleteVmbdAndWait(util.VmBdFilter{NameSpace: util.TestNS})
		_ = hvCluster.DeleteVdAndWait(util.VdFilter{NameSpace: util.TestNS, Name: "!%-system%"})
	}
	_ = cluster.DeleteBdAndWait()
}

// Provides N devices with size M on node
func getOrCreateConsumableBlockDevices(nName string, size int64, count int) ([]snc.BlockDevice, error) {
	cluster := util.EnsureCluster("", "")
	bds, _ := cluster.ListBD(util.BdFilter{Node: nName, Consumable: true, Size: float32(size)})
	if len(bds) >= int(count) {
		return bds, nil
	}

	if util.HypervisorKubeConfig == "" {
		return nil, fmt.Errorf("Not enough bds on %s: %d of %d", nName, len(bds), count)
	}
	hvCluster := util.EnsureCluster(util.HypervisorKubeConfig, "")
	for i := len(bds); i < count; i++ {
		err := hvCluster.CreateVMBD(nName, nName+"-data-"+util.RandString(4), util.HvStorageClass, size)
		if err != nil {
			return nil, err
		}
	}

	if err := util.RetrySec(30, func() error {
		bds, _ := cluster.ListBD(util.BdFilter{Node: nName, Consumable: true, Size: float32(size)})
		if len(bds) < int(count) {
			return fmt.Errorf("Not enough bds on %s: %d of %d", nName, len(bds), count)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return cluster.ListBD(util.BdFilter{Node: nName, Consumable: true, Size: float32(size)})
}
