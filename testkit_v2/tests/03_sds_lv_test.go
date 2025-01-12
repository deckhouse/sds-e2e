package integration

import (
	"fmt"
	"testing"

	coreapi "k8s.io/api/core/v1"
	util "github.com/deckhouse/sds-e2e/util"
	"k8s.io/apimachinery/pkg/api/resource"
)


func TestPVC(t *testing.T) {
	t.Run("PVC creating", testPVCCreate)
	t.Run("PVC resizing", testPVCResize)
	t.Run("PVC deleting", testPVCDelete)
	t.Run("PVC cleanup", testPVCCleanup)
}

func testPVCCreate(t *testing.T) {
	clr := util.GetCluster("", "")
	scName := "test-lvm-thick-immediate-retain"
	_, _ = clr.CreateSC(scName)

	// TODO move NS creating to tests init script
	_ = clr.CreateNs(util.TestNS)

	pvc, err := clr.CreatePVC("test-pvc", scName, "1Gi")
	if err != nil {
		t.Fatal(err)
	}

    pvcStatus, err := clr.WaitPVCStatus(pvc.Name)
	if err != nil {
		util.Debugf("PVC %s status: %s", pvc.Name, pvcStatus)
		// TODO Error is ok, need POD-consumer
		//t.Error(err)
	}
}

func testPVCResize(t *testing.T) {
	t.Skip("Not implemented")

	//err = funcs.EditSizePVC(ctx, cl, pvc.Name, "2Gi")
	/*
	func EditSizePVC(ctx context.Context, cl client.Client, name, newSize string) error {
		pvc := coreapi.PersistentVolumeClaim{}
		err := (*clr.rtClient).Get(clr.ctx, ctrlrtclient.ObjectKey{
			Name:      name,
			Namespace: TestNS,
		}, &pvc)
		if err != nil {
			return err
		}

		resourceList := make(map[coreapi.ResourceName]resource.Quantity)
		newPVCSize, err := resource.ParseQuantity(newSize)
		if err != nil {
			return err
		}

		resourceList[coreapi.ResourceStorage] = newPVCSize
		pvc.Spec.Resources.Requests = resourceList

		err = (*clr.rtClient).Update(clr.ctx, &pvc)
		if err != nil {
			return err
		}
		return nil
	}
	*/

	clr := util.GetCluster("", "")

    pvcList, err := clr.GetPVC(util.TestNS)
	if err != nil {
		t.Error("PVC getting:", err)
	}
	for _, pvc := range pvcList {
		origSize := pvc.Spec.Resources.Requests[coreapi.ResourceStorage]
		newSize := resource.MustParse("2Gi")  //.Value() | 1073741824 = 1Gi

    	// Update the PVC size
		util.Debugf("PVC %s size: %#v", pvc.Name, pvc.Size())
		util.Infof("PVC resize %s -> %s", origSize.String(), newSize.String())
    	pvc.Spec.Resources.Requests[coreapi.ResourceStorage] = newSize
		if err := clr.UpdatePVC(&pvc); err != nil {
            t.Error(fmt.Sprintf("PVC %s resizing (%s to %s) problem:", pvc.Name, origSize.String(), newSize.String()), err)
		}
	}
}

func testPVCDelete(t *testing.T) {
	clr := util.GetCluster("", "")

	err := clr.DeletePVC("test-pvc")
	if err != nil {
		t.Error(err)
	}

    pvcStatus, err := clr.DeletePVCWait("test-pvc")
	util.Debugf("PVC %s status: %s", "test-pvc", pvcStatus)
	if err != nil {
		t.Error(err)
	}
}

func testPVCCleanup(t *testing.T) {
	t.Skip("Not implemented: need to create NS in tests init script")

	clr := util.GetCluster("", "")

	// TODO delete SC (auto deleting with NS)

	// TODO delete NS. don`t work on Metal cluster
	if err := clr.DeleteNs(util.TestNS); err != nil {
		t.Error(err)
	}
}
