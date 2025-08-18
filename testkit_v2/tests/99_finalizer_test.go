package integration

import (
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
)

func TestFinalizer(t *testing.T) {
	clr := util.GetCluster("", "")
	t.Cleanup(func() {
		if util.TestNSCleanUp == "delete" {
			util.Debugf("Dedeting namespace %s", util.TestNS)
			if err := clr.DeleteNs(util.NsFilter{Name: util.TestNS}); err != nil {
				util.Errorf("Can't delete namespace %s", util.TestNS)
			}
		}
	})
}
