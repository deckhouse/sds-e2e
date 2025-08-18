/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"net/http"
	"testing"
	"time"

	util "github.com/deckhouse/sds-e2e/util"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDataExportName = "test-data-export"
	testPVCName        = "test-pvc-for-data-export"
	testPVName         = "test-pv-for-data-export"
	testPodName        = "test-pod-for-data-export"
	pvcType            = "PersistentVolumeClaim"
	scType             = "StorageClass"
)

func TestDataExport(t *testing.T) {
	t.Run("DataExport URL publish false", testCheckUrlReadinessWithPublichFalse)
	// t.Run("Exporting Pod accepts only GET and HEAD methods", testWrongHTTPMethods)
	// t.Run("DataExport with unsupported device", testWrongExportDevice)
	// t.Run("DataExport with unexisting supported device", testWrongExportDeviceName)
}

func cleanUp06() {
	clr := util.GetCluster("", "")
	_ = clr.DeletePVC(testPVCName)
	_ = clr.DeleteSC(testDataExportName)
	_ = clr.DeleteDataExport(testDataExportName)
	_ = clr.DeletePV(testPVName)
}

//TODO сделать нумерацию и переносить названия тестов в документ
//TODO сделать метод ensureWorkingStorageClass

func testCheckUrlReadinessWithPublichFalse(t *testing.T) {
	// t.Cleanup(cleanUp06)

	clr := util.GetCluster("", "")

	_, err := clr.CreateSC(testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test sc: %s", err.Error())
	}

	// _, err = clr.CreatePV(testPVName, "1Gi", testDataExportName)
	// if err != nil {
	// 	t.Fatalf("failed to create a test pv: %s", err.Error())
	// }

	_, err = clr.CreatePVC(testPVCName, testDataExportName, "1Gi", testPVName)
	if err != nil {
		t.Fatalf("failed to create a test pvс: %s", err.Error())
	}

	_, err = clr.WaitPVCStatus(testPVCName)
	if err != nil {
		t.Fatalf("failed await PVC ready status: %s", err.Error())
	}

	dataExport, err := clr.CreateDataExport(pvcType, testPVCName, testDataExportName, "1h", false)
	if err != nil {
		t.Fatalf("failed to create a test DataExport: %s", err.Error())
	}

	// dataExport, err := clr.WaitDataExportURLReady(testDataExportName)
	// if err != nil {
	// 	t.Fatalf("failed await for DataExport URL to become ready: %s", err.Error())
	// }

	if dataExport.Status.Url != "" {
		t.Errorf("DataExport URLs not ready. PublicURL: %s, Url: %s", dataExport.Status.PublicURL, dataExport.Status.Url)
	}
}

func testCheckUrlReadinessWithPublichTrue(t *testing.T) {
	t.Cleanup(cleanUp06)

	clr := util.GetCluster("", "")

	_, err := clr.CreateSC(testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test sc: %s", err.Error())
	}

	_, err = clr.CreatePV(testPVName, "1Gi", testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test pv: %s", err.Error())
	}

	_, err = clr.CreatePVC(testPVCName, testDataExportName, "1Gi", testPVName)
	if err != nil {
		t.Fatalf("failed to create a test pvс: %s", err.Error())
	}

	_, err = clr.WaitPVCStatus(testPVCName)
	if err != nil {
		t.Fatalf("failed await PVC ready status: %s", err.Error())
	}

	_, err = clr.CreateDataExport(pvcType, testPVCName, testDataExportName, "1h", true)
	if err != nil {
		t.Fatalf("failed to create a test DataExport: %s", err.Error())
	}

	dataExport, err := clr.WaitDataExportURLReady(testDataExportName)
	if err != nil {
		t.Fatalf("failed await for DataExport URL to become ready: %s", err.Error())
	}

	if dataExport.Status.PublicURL != "" && dataExport.Status.Url != "" {
		t.Errorf("DataExport URLs not ready. PublicURL: %s, Url: %s", dataExport.Status.PublicURL, dataExport.Status.Url)
	}
}

func testWrongHTTPMethods(t *testing.T) {
	t.Cleanup(cleanUp06)

	clr := util.GetCluster("", "")

	_, err := clr.CreateSC(testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test sc: %s", err.Error())
	}

	_, err = clr.CreatePV(testPVName, "1Gi", testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test pv: %s", err.Error())
	}

	pvc, err := clr.CreatePVC(testPVCName, testDataExportName, "1Gi", testPVName)
	if err != nil {
		t.Fatalf("failed to create a test pvс: %s", err.Error())
	}

	_, err = clr.WaitPVCStatus(testPVCName)

	dataExport, err := clr.CreateDataExport(pvcType, pvc.Name, testDataExportName, "1h", true)
	if err != nil {
		t.Fatalf("failed to create a test DataExport: %s", err.Error())
	}

	requests := make([]*http.Request, 4)
	methods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, m := range methods {
		req, err := http.NewRequest(m, dataExport.Status.Url, nil)
		if err != nil {
			t.Fatalf("failed to create an HTTP request: %s", err.Error())
		}
		requests = append(requests, req)
	}

	client := &http.Client{}
	for _, r := range requests {
		_, err = client.Do(r)
		if err == nil {
			t.Errorf("exporting pod accepds http method %s but it should not", r.Method)
		}
	}
}

func testWrongExportDevice(t *testing.T) {
	t.Cleanup(cleanUp06)

	clr := util.GetCluster("", "")

	sc, err := clr.CreateSC(testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test sc: %s", err.Error())
	}

	dataExport, err := clr.CreateDataExport(scType, sc.Name, testDataExportName, "1h", true)
	if err != nil {
		t.Fatalf("failed to create a test DataExport: %s", err.Error())
	}

	for _, cond := range dataExport.Status.Conditions {
		if cond.Type != "Ready" {
			continue
		}
		if cond.Status != "False" && cond.Message != "data export validation failed" {
			t.Errorf("DataExport Ready condition status is not cool")
		}
	}
}

func testWrongExportDeviceName(t *testing.T) {
	t.Cleanup(cleanUp06)

	clr := util.GetCluster("", "")

	_, err := clr.CreateSC(testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test sc: %s", err.Error())
	}

	_, err = clr.CreatePV(testPVName, "1Gi", testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test pv: %s", err.Error())
	}

	_, err = clr.CreatePVC(testPVCName, testDataExportName, "1Gi", testPVName)
	if err != nil {
		t.Fatalf("failed to create a test pvс: %s", err.Error())
	}

	_, err = clr.WaitPVCStatus(testPVCName)
	if err != nil {
		t.Fatalf("failed await PVC ready status: %s", err.Error())
	}

	dataExport, err := clr.CreateDataExport(scType, "wrongDeviceName", testDataExportName, "1h", true)
	if err != nil {
		t.Fatalf("failed to create a test DataExport: %s", err.Error())
	}

	for _, cond := range dataExport.Status.Conditions {
		if cond.Type != "Ready" {
			continue
		}
		if cond.Status != "False" && cond.Message != "data export validation failed" {
			t.Errorf("DataExport Ready condition status is not cool")
		}
	}
}

func testDataExportWithBoundPVC(t *testing.T) {
	t.Cleanup(cleanUp06)

	clr := util.GetCluster("", "")

	_, err := clr.CreateSC(testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test sc: %s", err.Error())
	}

	_, err = clr.CreatePV(testPVName, "1Gi", testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test pv: %s", err.Error())
	}

	_, err = clr.CreatePVC(testPVCName, testDataExportName, "1Gi", testPVName)
	if err != nil {
		t.Fatalf("failed to create a test pvс: %s", err.Error())
	}

	err = clr.CreatePodWithPVC(util.TestNS, testPodName, testPVCName)
	if err != nil {
		t.Fatalf("failed to create a test pod: %s", err.Error())
	}

	_, err = clr.WaitPVCStatus(testPVCName)
	if err != nil {
		t.Fatalf("failed await PVC ready status: %s", err.Error())
	}

	dataExport, err := clr.CreateDataExport(scType, "wrongDeviceName", testDataExportName, "1h", true)
	if err != nil {
		t.Fatalf("failed to create a test DataExport: %s", err.Error())
	}

	for _, cond := range dataExport.Status.Conditions {
		if cond.Type != "Ready" {
			continue
		}
		if cond.Status != "False" && cond.Message != "data export validation failed" {
			t.Errorf("DataExport Ready condition status is not cool")
		}
	}
}

func testexportPodTTLExpired(t *testing.T) {
	t.Cleanup(cleanUp06)

	clr := util.GetCluster("", "")

	_, err := clr.CreateSC(testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test sc: %s", err.Error())
	}

	_, err = clr.CreatePV(testPVName, "1Gi", testDataExportName)
	if err != nil {
		t.Fatalf("failed to create a test pv: %s", err.Error())
	}

	_, err = clr.CreatePVC(testPVCName, testDataExportName, "1Gi", testPVName)
	if err != nil {
		t.Fatalf("failed to create a test pvс: %s", err.Error())
	}

	err = clr.CreatePodWithPVC(util.TestNS, testPodName, testPVCName)
	if err != nil {
		t.Fatalf("failed to create a test pod: %s", err.Error())
	}

	_, err = clr.WaitPVCStatus(testPVCName)
	if err != nil {
		t.Fatalf("failed await PVC ready status: %s", err.Error())
	}

	dataExport, err := clr.CreateDataExport(scType, "wrongDeviceName", testDataExportName, "1s", true)
	if err != nil {
		t.Fatalf("failed to create a test DataExport: %s", err.Error())
	}

	time.Sleep(time.Second)

	for _, cond := range dataExport.Status.Conditions {
		if cond.Type != "Expired" {
			continue
		}
		if cond.Status != "True" {
			t.Error("Exporting Pod TTL has expired, but DataExport's 'Expired' condition isn't 'True'")
		}
	}

	pv, err := clr.GetPV(testPVName)
	if err != nil {
		t.Fatalf("failed to get test pv: %s", err.Error())
	}

	pvc, err := clr.GetPVC(testPVCName)
	if err != nil {
		t.Fatalf("failed to get test pvс: %s", err.Error())
	}

	if pv.Spec.ClaimRef.Name != pvc.Name {
		t.Errorf("PV isn't reattached to the original PVC. Current clainref name %s, must be %s", pv.Spec.ClaimRef.Name, pvc.Name)
	}

	if pv.Spec.ClaimRef.Namespace != pvc.Namespace {
		t.Errorf("PV isn't reattached to the original PVC. Current name %s, must be %s", pv.Spec.ClaimRef.Namespace, pvc.Namespace)
	}

	if pv.Spec.ClaimRef.UID != pvc.UID {
		t.Errorf("PV isn't reattached to the original PVC. Current UID %s, must be %s", pv.Spec.ClaimRef.UID, pvc.UID)
	}

	if pv.Spec.ClaimRef.ResourceVersion != pvc.ResourceVersion {
		t.Errorf("PV isn't reattached to the original PVC. Current ResourceVersion %s, must be %s", pv.Spec.ClaimRef.ResourceVersion, pvc.ResourceVersion)
	}

	for annotation := range pvc.Annotations {
		if annotation == "storage.deckhouse.io/data-export-in-progress" || annotation == "storage.deckhouse.io/data-export-request" {
			t.Errorf("original PVC contains annotation than shouldn't exist by this time")
		}
	}

	for _, finalizer := range pvc.Finalizers {
		if finalizer == "storage.deckhouse.io/data-exporter-controller" {
			t.Errorf("original PVC contains a finalizer than shouldn't exist by this time")
		}
	}
}

// func testDataExportWithUsedPVC(t *testing.T) {
// 	clr := util.GetCluster("", "")

// 	// Clean up any existing resources
// 	_ = clr.DeletePod("", "test-pod-1")
// 	_ = clr.DeletePod("", "test-pod-2")
// 	_ = clr.DeletePVC("test-pvc-used")
// 	_ = clr.DeleteSC(dataExportTestSCName)

// 	// Create storage class
// 	_, err := clr.CreateSC(dataExportTestSCName)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create PVC
// 	_, err = clr.CreatePVC("test-pvc-used", dataExportTestSCName, "1Gi", "")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Wait for PVC to be bound
// 	pvcStatus, err := clr.WaitPVCStatus("test-pvc-used")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Logf("PVC status: %s", pvcStatus)

// 	// Create first pod that uses the PVC
// 	err = clr.CreatePodWithPVC("", "test-pod-1", "test-pvc-used")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Wait for first pod to be running
// 	podStatus, err := clr.WaitPodStatus("", "test-pod-1")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Logf("First pod status: %s", podStatus)

// 	// Try to create DataExport for the PVC that is already in use
// 	dataExport, err := clr.CreateDataExport("test-pvc-used", "test-data-export-used")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	t.Logf("DataExport created: %s", dataExport.Name)

// 	// Wait for DataExport status to be updated
// 	dataExportWithStatus, err := clr.WaitDataExportStatus("test-data-export-used")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Check conditions
// 	if len(dataExportWithStatus.Status.Conditions) == 0 {
// 		t.Fatal("DataExport should have conditions set")
// 	}

// 	// Find the "Ready" condition
// 	var readyCondition *metav1.Condition
// 	for _, condition := range dataExportWithStatus.Status.Conditions {
// 		if condition.Type == "Ready" {
// 			readyCondition = &condition
// 			break
// 		}
// 	}

// 	if readyCondition == nil {
// 		t.Fatal("DataExport should have a 'Ready' condition")
// 	}

// 	// Check that the condition status is "False"
// 	if readyCondition.Status != metav1.ConditionFalse {
// 		t.Errorf("Expected Ready condition status to be 'False', got '%s'", readyCondition.Status)
// 	}

// 	// Check that the message is "PVC validation Failed"
// 	expectedMessage := "PVC validation Failed"
// 	if readyCondition.Message != expectedMessage {
// 		t.Errorf("Expected Ready condition message to be '%s', got '%s'", expectedMessage, readyCondition.Message)
// 	}

// 	t.Logf("DataExport Ready condition: Status=%s, Message=%s", readyCondition.Status, readyCondition.Message)

// 	// Clean up
// 	_ = clr.DeletePod("", "test-pod-1")
// 	_ = clr.DeletePVC("test-pvc-used")
// 	_ = clr.DeleteSC(dataExportTestSCName)
// }

// func testDataExportWithVirtualDiskInUse(t *testing.T) {
// 	clr := util.GetCluster("", "")

// 	// Clean up any existing resources
// 	_ = clr.DeleteVDByName(util.TestNS, "test-vd-inuse")
// 	_ = clr.DeleteSC(dataExportTestSCName)

// 	// Create storage class
// 	_, err := clr.CreateSC(dataExportTestSCName)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	err = clr.CreateVDWithInUseCondition(util.TestNS, "test-vd-inuse", dataExportTestSCName, 1, "UsedByVM")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Wait for VirtualDisk to be ready
// 	vd, err := clr.WaitVDStatus(util.TestNS, "test-vd-inuse")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Verify that the InUse condition is set
// 	var inUseCondition *metav1.Condition
// 	for _, condition := range vd.Status.Conditions {
// 		if condition.Type == "InUse" {
// 			inUseCondition = &condition
// 			break
// 		}
// 	}

// 	if inUseCondition == nil {
// 		t.Fatal("VirtualDisk should have an 'InUse' condition")
// 	}

// 	if inUseCondition.Status != metav1.ConditionTrue {
// 		t.Errorf("Expected InUse condition status to be 'True', got '%s'", inUseCondition.Status)
// 	}

// 	if inUseCondition.Reason != "UsedByVM" {
// 		t.Errorf("Expected InUse condition reason to be 'UsedByVM', got '%s'", inUseCondition.Reason)
// 	}

// 	t.Logf("VirtualDisk InUse condition: Status=%s, Reason=%s", inUseCondition.Status, inUseCondition.Reason)

// 	// Try to create DataExport for the VirtualDisk that is in use
// 	dataExport, err := clr.CreateDataExportForVirtualDisk("test-vd-inuse", "test-data-export-vd")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	t.Logf("DataExport created: %s", dataExport.Name)

// 	// Wait for DataExport status to be updated
// 	dataExportWithStatus, err := clr.WaitDataExportStatus("test-data-export-vd")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Check conditions
// 	if len(dataExportWithStatus.Status.Conditions) == 0 {
// 		t.Fatal("DataExport should have conditions set")
// 	}

// 	// Find the "Ready" condition
// 	var readyCondition *metav1.Condition
// 	for _, condition := range dataExportWithStatus.Status.Conditions {
// 		if condition.Type == "Ready" {
// 			readyCondition = &condition
// 			break
// 		}
// 	}

// 	if readyCondition == nil {
// 		t.Fatal("DataExport should have a 'Ready' condition")
// 	}

// 	// Check that the condition status is "False"
// 	if readyCondition.Status != metav1.ConditionFalse {
// 		t.Errorf("Expected Ready condition status to be 'False', got '%s'", readyCondition.Status)
// 	}

// 	// Check that the message is "Validation Failed"
// 	expectedMessage := "Validation Failed"
// 	if readyCondition.Message != expectedMessage {
// 		t.Errorf("Expected Ready condition message to be '%s', got '%s'", expectedMessage, readyCondition.Message)
// 	}

// 	t.Logf("DataExport Ready condition: Status=%s, Message=%s", readyCondition.Status, readyCondition.Message)

// 	// Clean up
// 	_ = clr.DeleteVDByName(util.TestNS, "test-vd-inuse")
// 	_ = clr.DeleteSC(dataExportTestSCName)
// }
