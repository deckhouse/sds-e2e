package integration

import (
	"testing"
	//"flag"
	util "github.com/deckhouse/sds-e2e/util"
)

var (
//testNamespace    = *flag.String("namespace", "sds-replicated-volume-e2e-test", "namespace in which test runs")
//stsCount         = *flag.Int("stsCount", 50, "number of sts instances")
//pvSize           = *flag.String("pvSize", "5Gi", "size of PV in Gi")
//pvResizedSize    = *flag.String("pvResizedSize", "5.1Gi", "size of resized PV in Gi")
//storageClassName = *flag.String("storageClassName", "linstor-r2", "storage class name")
//createRSP        = *flag.Bool("createRSP", true, "create RSP")
//createVM         = *flag.Bool("createVM", true, "create VM")
)

//func TestMain(m *testing.M) {
//	// call flag.Parse() here if TestMain uses flags
//
//	util.ClusterCreate()
//	m.Run()
//}

func TestNode(t *testing.T) {
	clr := util.GetCluster("", "")

	gns := clr.GetGroupNodes()
	//map[string][]string{
	//	"Astra":[]string(nil),
	//	"Deb11":[]string{"worker-deb-0"},
	//	"Ubu22":[]string{"worker-ubu-0", "worker-ubu-1"}
	//}

	astraNodes, ok := gns["Astra"]
	if !ok || len(astraNodes) == 0 {
		t.Error("No Astra node, really?")
	}

	ubuntuNodes, ok := gns["Ubu22"]
	if !ok || len(ubuntuNodes) < 2 {
		t.Fatal("No Ubuntu 22 nodes, impossible!")
	}
}
