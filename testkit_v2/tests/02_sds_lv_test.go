package integration

import (
	"fmt"
//	"strings"
	"testing"
//	"time"

//	"github.com/deckhouse/sds-e2e/funcs"
//	"github.com/melbahja/goph"
	coreapi "k8s.io/api/core/v1"
	util "github.com/deckhouse/sds-e2e/util"
//	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
)


var clr *util.KCluster

func TestPVC(t *testing.T) {
	defaultClr, err := util.InitKCluster("", "")
    if err != nil {
		t.Fatal("Kubeclient problem", err)
    }
	clr = defaultClr

	t.Run("PVC creating", testPVCCreate)
	t.Run("PVC resizing", testPVCResize)
	t.Run("PVC deleting", testPVCDelete)
}

func testPVCCreate(t *testing.T) {
	scName := "test-lvm-thick-immediate-retain"
	_, _ := clr.CreateSC(scName)

	//util.Infof("PVC creating for %s", bd.Name)
	_, err := clr.CreatePVC("test-pvc", scName, "1Gi")
	if err != nil {
		t.Error(err)
	}

//    t.Log("------------  pvc creating ------------- ")
//    pvc, err := funcs.CreatePVC(ctx, cl,
//        "test-pvc", "test-lvm-thick-immediate-retain", "1Gi", false)
//    pvcStatus, err := funcs.WaitPVCStatus(ctx, cl, pvc.Name)

//    t.Log(fmt.Sprintf("pvc status=%s", pvcStatus))
//    t.Log(fmt.Sprintf("pvc=%s created", pvc.Name))
//
//    t.Log("------------  pvc creating ------------- ")
//    err = funcs.EditSizePVC(ctx, cl, pvc.Name, "2Gi")
//    if err != nil {
//        t.Error(err)
//    }
//
//    t.Log("------------  pvc wait ------------- ")
//    pvcStatus, err = funcs.WaitPVCStatus(ctx, cl, pvc.Name)
//    if err != nil {
//        t.Error(err)
//    }
//    t.Log(fmt.Sprintf("pvc status=%s", pvcStatus))


}

func testPVCResize(t *testing.T) {
	//nodeList, _ := clr.GetNodes(map[string][]string{"OS": []string{"Ubuntu", "RedOS"}})

    pvcList, err := clr.GetTestPVC()
	if err != nil {
		t.Error("PVC getting:", err)
	}
	for _, pvc := range pvcList {
		//fmt.Printf("PVC %s size: %#v\n", pvc.Name, pvc.Size())
		origSize := pvc.Spec.Resources.Requests[coreapi.ResourceStorage]
		newSize := resource.MustParse("2Gi")  //.Value() | 1073741824 = 1Gi

    	// Update the PVC size
		util.Infof("PVC resize %s -> %s", origSize.String(), newSize.String())
    	pvc.Spec.Resources.Requests[coreapi.ResourceStorage] = newSize
		if err := clr.UpdatePVC(&pvc); err != nil {
            t.Error(fmt.Sprintf("PVC %s resizing (%s to %s) problem", pvc.Name, origSize.String(), newSize.String()), err)
		}
	}
}

func testPVCDelete(t *testing.T) {
}
