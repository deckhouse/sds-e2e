package integration

import (
	util "github.com/deckhouse/sds-e2e/util"
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
