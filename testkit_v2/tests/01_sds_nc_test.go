package integration

import (
	"fmt"
//	"strings"
	"testing"
//	"time"
//	"log"

//	"github.com/deckhouse/sds-e2e/funcs"
//	"github.com/melbahja/goph"
	coreapi "k8s.io/api/core/v1"
	util "github.com/deckhouse/sds-e2e/util"
	"k8s.io/apimachinery/pkg/api/resource"
)


var clr *util.KCluster

func TestNode(t *testing.T) {
	defaultClr, err := util.InitKCluster("", "")
    if err != nil {
		t.Fatal("Kubeclient problem", err)
    }
	clr = defaultClr

	t.Run("Node check", testPerNode)
	t.Run("LVG creating", testLVGCreate)
	t.Run("LVG resizing", testLVGResize)
}

func testPerNode(t *testing.T) {
	nodeList, err := clr.GetNodes(map[string][]string{"OS": []string{"Ubuntu", "RedOS"}})
	if err != nil {
		t.Fatal("node error", err)
	}

    for _, node := range nodeList {
		fmt.Println(node.ObjectMeta.Name)
    }
}

func testLVGCreate(t *testing.T) {
	//nodeList, _ := clr.GetNodes(nil)
    bdList, _ := clr.GetBDs(map[string][]string{"Name": []string{"dev-06d870aec3f7a7794c02803e1d08b941f74e4fde"}})
    for _, bd := range bdList {
		fmt.Printf("BD: %#v\n", bd.Name)
//        if err := clr.AddLVG(bd.Status.NodeName, bd.Name); err != nil {
//            t.Error("LVG creating", err)
//        }
    }
}

/*
- Создание LVG на всех 3х нодах
- Расширение LVG путем добавления нового BD только на 2х нодах:
 - Ubuntu 22.04
 - РедОС  7.3
*/

func testLVGResize(t *testing.T) {
	//nodeList, _ := clr.GetNodes(map[string][]string{"OS": []string{"Ubuntu", "RedOS"}})

    pvcList, _ := clr.GetTestPVC()
	for _, pvc := range pvcList {
		fmt.Printf("PVC %s size: %#v\n", pvc.Name, pvc.Size())
		fmt.Printf("PVC %s size2: %#v\n", pvc.Name, pvc.Spec.Resources.Requests[coreapi.ResourceStorage])
		r := pvc.Spec.Resources.Requests[coreapi.ResourceStorage]
		fmt.Printf("PVC %s size3: %#v\n", pvc.Name, r.Value())
		fmt.Printf("PVC %s size4: %#v\n", pvc.Name, r.String())

    	// Update the PVC size
    	pvc.Spec.Resources.Requests[coreapi.ResourceStorage] = resource.MustParse("2Gi")  // 1073741824 = 1Gi

		updPVC, err := clr.UpdatePVC(&pvc)
    	fmt.Printf("PVC %s updated to new size: %v\n", updPVC.Name, updPVC.Spec.Resources.Requests[coreapi.ResourceStorage])
		fmt.Printf("ERR: %#v\n", err)
		//fmt.Printf("ERR: %#v\n", err.StatusError.ErrStatus.Details)
	}

/*
    lvgList, _ := clr.GetTestLVG()
    for _, lvg := range lvgList {
        if len(lvg.Status.Nodes) == 0 {
            t.Error(fmt.Sprintf("LVG %s: node is empty", lvg.Name))
        } else if lvg.Status.Nodes[0].Devices[0].PVSize.String() != "20Gi" || lvg.Status.Nodes[0].Devices[0].DevSize.String() != "20975192Ki" {
            t.Error(fmt.Sprintf("LVG %s: size problem %s, %s", lvg.Name, lvg.Status.Nodes[0].Devices[0].PVSize.String(), lvg.Status.Nodes[0].Devices[0].DevSize.String()))
        } else {
            fmt.Printf("LVG %s: size ok %s, %s\n", lvg.Name, lvg.Status.Nodes[0].Devices[0].PVSize.String(), lvg.Status.Nodes[0].Devices[0].DevSize.String())
        }
    }

    vmdList, _ := clr.GetTestVMD() // kube_vm.config
    if len(vmdList) == 0 {
        t.Error("Disk update problem, no VMDs")
    }
    for _, vmd := range vmdList {
        if strings.Contains(vmd.Name, "-data") {
            vmd.Spec.PersistentVolumeClaim.Size.Set(32212254720)
            err := clr.UpdVMD(&vmd)
            if err != nil {
                t.Error("Disk update problem", err)
            }
        }
    }
*/
}
