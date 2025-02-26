package integration

import (
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
)

func TestNode(t *testing.T) {
	clr := util.GetCluster("", "")

	nodeMap := clr.MapLabelNodes(nil)

	astraNodes, ok := nodeMap["Astra"]
	if !ok || len(astraNodes) == 0 {
		t.Error("No Astra node - not good")
	}

	ubuntuNodes, ok := nodeMap["Ubu22"]
	if !ok || len(ubuntuNodes) < 2 {
		t.Fatal("Few Ubuntu 22 nodes")
	}
}
