package integration

import (
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
)

func TestNodeHealthCheck(t *testing.T) {
	clr := util.GetCluster("", "")

	nodeMap := clr.MapLabelNodes(nil)
	for label, nodes := range nodeMap {
		if len(nodes) == 0 {
			t.Errorf("No %s nodes", label)
		} else {
			util.Infof("%s nodes: %d", label, len(nodes))
		}
	}
}

func TestNode(t *testing.T) {
	clr := util.GetCluster("", "")

	nodeMap := clr.MapLabelNodes(nil)

	astraNodes, ok := nodeMap["Astra"]
	if !ok || len(astraNodes) == 0 {
		util.Warnf("No Astra node - not good")
	}

	ubuntuNodes, ok := nodeMap["Ubu22"]
	if !ok || len(ubuntuNodes) < 2 {
		t.Fatal("Few Ubuntu 22 nodes")
	}
}
