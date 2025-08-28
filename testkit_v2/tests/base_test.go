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
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	util "github.com/deckhouse/sds-e2e/util"
	"github.com/deckhouse/sds-e2e/util/utiltype"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	persistentVolumeClaimType = "PersistentVolumeClaim"
	virtualDiskType           = "VirtualDisk"
	testPVCName               = "test-pvc"
	testVDName                = "test-vd"
	testDEName                = "test-de"
	testPodName               = "test-pod-for-data-export"
	testNameSpace             = "test-e2e"
	unsupportedExportType     = "UnsupportedKind"

	DataExportInProgressKey        = "storage.deckhouse.io/data-export-in-progress"
	DataExportRequestAnnotationKey = "storage.deckhouse.io/data-export-request"

	FinalizerName = "storage.deckhouse.io/data-exporter-controller"
)

var cluster *util.KCluster

func cleanUpBase() {
	cluster := util.EnsureCluster("", "")
	// _ = cluster.DeletePVC(dataExportPVCName)
	_ = cluster.DeleteDataExport(testDEName, util.TestNS)
}

func TestDataExport(t *testing.T) {
	t.Run("1", testDataExporterCreation)
	// t.Run("1", testDataExportWithUnexistingPVC)
}

func testDataExportHTTPRequests(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "10s")
	if err != nil {
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	dataExport, err := cluster.GetDataExport(testDEName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to create DataExport: %s", err.Error())
	}

	baseURL := dataExport.Status.Url
	basePath := "/api/v1/files"
	testTable := []struct {
		description string
		method      string
		path        string
		statusCode  int
	}{
		{
			description: "GET-запрос к несуществующему файлу",
			method:      http.MethodGet,
			path:        basePath + "/unexisting-file.txt",
			statusCode:  404,
		},
		{
			description: "GET-запрос к директории без завершающего слеша",
			method:      http.MethodGet,
			path:        basePath + "",
			statusCode:  400,
		},
		{
			description: "GET-запрос к некорректному маршруту",
			method:      http.MethodGet,
			path:        "/wrong/path",
			statusCode:  400,
		},
		{
			description: "HEAD-запрос к несуществующему файлу",
			method:      http.MethodHead,
			path:        basePath + "/unexisting-file.txt",
			statusCode:  404,
		},
		{
			description: "HEAD-запрос к несуществующему файлу",
			method:      http.MethodHead,
			path:        basePath + "/unexisting-file.txt",
			statusCode:  404,
		},
		{
			description: "HEAD-запрос к некорректному маршруту",
			method:      http.MethodHead,
			path:        "/wrong/path",
			statusCode:  400,
		},
	}

	client := &http.Client{}
	for _, test := range testTable {
		req, err := http.NewRequest(test.method, baseURL+test.path, nil)
		if err != nil {
			t.Fatalf("failed to create an HTTP-request: %s", err.Error())
		}

		// TODO parallel?
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to send HTTP-request: %s", err.Error())
		}

		if res.StatusCode != test.statusCode {
			t.Errorf("responce status code mismatch. Expected %d received %d", test.statusCode, res.StatusCode)
		}
	}
}

func testCreateVD(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	_, err := cluster.CreatePVCInTestNS(testPVCName, util.NestedDefaultStorageClass, "20Mi")
	if err != nil {
		t.Fatalf("failed to create PVC: %s", err.Error())
	}

	err = cluster.CreateVD("test-vd", util.TestNS, util.NestedDefaultStorageClass, 10240)
	if err != nil {
		t.Fatalf("failed to create VD: %s", err.Error())
	}

	_, err = cluster.CreateDataExport(testDEName, virtualDiskType, testVDName, util.TestNS, "1h", false)
	if err != nil {
		t.Fatalf("failed to create data export: %s", err.Error())
	}

}

func testDataExportTTLExpired(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "10s")
	if err != nil {
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	time.Sleep(10 * time.Second)

	dataExport, err := cluster.GetDataExport(testDEName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to get dublicate data export: %s", err.Error())
	}

	for _, cond := range dataExport.Status.Conditions {
		if cond.Type != "EXPIRED" {
			continue
		}
		if cond.Status != v1.ConditionTrue {
			t.Errorf("data export TTL has expited. Expected EXPIRED type to be Status true, but it should not")
		}
	}

	pvc, err := cluster.GetPVC(testPVCName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to get a PVC: %s", err.Error())
	}

	for annotation := range pvc.Annotations {
		if annotation == DataExportInProgressKey || annotation == DataExportRequestAnnotationKey {
			t.Errorf("pvc has annotation %s but it should not", annotation)
		}
	}

	for _, finalizer := range pvc.Finalizers {
		if finalizer == FinalizerName {
			t.Errorf("pvc has finalizer %s but it should not", finalizer)
		}
	}
}

func testDeleteDataExport(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "1h")
	if err != nil {
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	err = cluster.DeleteDataExport(testDEName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to delete a DataExport: %s", err.Error())
	}

	pvc, err := cluster.GetPVC(testPVCName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to get a PVC: %s", err.Error())
	}

	pvName := pvc.Spec.VolumeName
	pv, err := cluster.GetPV(pvName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to get a PV: %s", err.Error())
	}

	// TODO check all?
	if pv.Spec.ClaimRef.Name != pvc.Name {
		t.Fatalf("PV is not reatached to the original user PVC: ClaimRef.Name: %s, user PVC name %s", pv.Spec.ClaimRef.Name, pvc.Name)
	}

	if pv.Spec.ClaimRef.Namespace != pvc.Namespace {
		t.Fatalf("PV is not reatached to the original user PVC: ClaimRef.Namespace: %s, user PVC Namespace %s", pv.Spec.ClaimRef.Namespace, pvc.Namespace)
	}

	if pv.Spec.ClaimRef.UID != pvc.UID {
		t.Fatalf("PV is not reatached to the original user PVC: ClaimRef.UID: %s, user PVC UID %s", pv.Spec.ClaimRef.UID, pvc.UID)
	}

	if pv.Spec.ClaimRef.ResourceVersion != pvc.ResourceVersion {
		t.Fatalf("PV is not reatached to the original user PVC: ClaimRef.ResourceVersion: %s, user PVC ResourceVersion %s", pv.Spec.ClaimRef.ResourceVersion, pvc.ResourceVersion)
	}

	for annotation := range pvc.Annotations {
		if annotation == DataExportInProgressKey || annotation == DataExportRequestAnnotationKey {
			t.Errorf("pvc has annotation %s but it should not", annotation)
		}
	}

	for _, finalizer := range pvc.Finalizers {
		if finalizer == FinalizerName {
			t.Errorf("pvc has finalizer %s but it should not", finalizer)
		}
	}
}

func testExportTypeWhichIsAlreadyExported(t *testing.T) {
	// t.Cleanup(cleanUpBase)
	cluster := util.EnsureCluster("", "")

	err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "1h")
	if err != nil {
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	dublicateName := "duplicate-data-export"
	_, err = cluster.CreateDataExport(dublicateName, persistentVolumeClaimType, testPVCName, util.TestNS, "1h", false)
	if err != nil {
		t.Fatalf("you cant: %s", err.Error())
	}

	dataExport, err := cluster.GetDataExport(dublicateName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to get dublicate data export: %s", err.Error())
	}

	err = checkIfValidationFailed(dataExport)
	if err != nil {
		t.Fatalf("dataExport %s validation is not failed but it should: %s", testDEName, err.Error())
	}
}

func testDataExportWithUnsupportedExportType(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	_, err := cluster.CreateDataExport(testDEName, unsupportedExportType, "fake-kind-name", util.TestNS, "1h", false)
	if err == nil {
		t.Errorf("DataExport was created with an unsupported export type %s", err)
	}
}

func testDataExportWithNonExistentExportType(t *testing.T) {
	// t.Cleanup(cleanUpBase)

	cluster := util.EnsureCluster("", "")
	exportType := []string{"PersistentVolumeClaim", "VolumeSnapshot", "VirtualDisk", "VirtualDiskSnapshot"}

	for _, exportType := range exportType {
		_, err := cluster.CreateDataExport(testDEName, exportType, "fake-kind-name", util.TestNS, "1h", false)
		if err != nil {
			t.Fatalf("failed to create data export: %v", err)
		}
		time.Sleep(3 * time.Second)

		dataExport, err := cluster.GetDataExport(testDEName, util.TestNS)
		if err != nil {
			t.Fatalf("failed to get data export: %v", err)
		}

		err = checkIfValidationFailed(dataExport)
		if err != nil {
			t.Fatalf("dataExport %s validation is not failed but it should: %s", testDEName, err.Error())
		}
	}
}

func testDataExporterCreation(t *testing.T) {
	// t.Cleanup(cleanUpBase)

	cluster := util.EnsureCluster("", "")

	err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "1h")
	if err != nil {
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	dataExport, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("failed to await data export to become ready: %v", err)
	}

	err = testIfClusterUrlIsReady(dataExport)
	if err != nil {
		t.Error(err.Error())
	}

	// if dataExport.Spec.Publish {
	// 	err = testIfPublicUrlIsReady(dataExport)
	// 	if err != nil {
	// 		t.Error(err.Error())
	// 	}
	// }

	// err = testUnsupportedHTTPMethods(dataExport)
	// if err != nil {
	// 	t.Error(err.Error())
	// }
}

func testUnsupportedHTTPMethods(dataExport *utiltype.DataExport) error {
	unsupportedMethods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	client := http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s%s/%s/%s/api/v1/files/", dataExport.Status.Url, dataExport.Namespace, "pvc", dataExport.Spec.TargetRef.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, method := range unsupportedMethods {
		request, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create a new HTTP request: %w", err)
		}

		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("failed to send an HTTP %s request: %w", request.Method, err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusMethodNotAllowed {
			return fmt.Errorf("dataExport %s accepts %s HTTP method, but it shoudn't", dataExport.Name, request.Method)
		}
	}
	return nil
}

func testIfPublicUrlIsReady(dataExport *utiltype.DataExport) error {
	if dataExport.Status.PublicURL == "" {
		return fmt.Errorf("dataExport %s public URL is empty", dataExport.Name)
	}
	return nil
}

func testIfClusterUrlIsReady(dataExport *utiltype.DataExport) error {
	if dataExport.Status.Url == "" {
		return fmt.Errorf("dataExport %s URL is empty", dataExport.Name)
	}
	return nil
}

func testHTTP(dataExport *utiltype.DataExport) error {
	resp, err := http.Get(fmt.Sprintf("%s%s", dataExport.Status.Url, "health"))
	if err != nil {
		return fmt.Errorf("failed to chech server health: %w", err)
	}
	fmt.Printf("== Status: %s", resp.Status)
	return nil
}

func checkIfValidationFailed(dataExport *utiltype.DataExport) error {
	for _, cond := range dataExport.Status.Conditions {
		if cond.Type != "Ready" {
			continue
		}
		if cond.Reason == "ValidationFailed" {
			return nil
		}
	}
	return fmt.Errorf("validation is not failed")
}

func CreateDataExportWithPVC(cluster *util.KCluster, podName, pvcName, deName, duration string) error {
	pvc, err := cluster.CreatePVCInTestNS(pvcName, util.NestedDefaultStorageClass, "1Mi")
	if err != nil {
		return fmt.Errorf("failed to create PVC: %s", err.Error())
	}
	util.Infof("Created PVC: %s in namespace: %s", pvc.Name, pvc.Namespace)

	err = cluster.CreateDummyPod(podName, testNameSpace, pvcName)
	if err != nil {
		return fmt.Errorf("failed to create dummy pod: %s", err.Error())
	}

	_, err = cluster.WaitPVCStatus(pvcName)
	if err != nil {
		return fmt.Errorf("failed to wait pvc to become bound: %s", err.Error())
	}

	err = cluster.DeletePod(podName, util.TestNS)
	if err != nil {
		return fmt.Errorf("failed to delete test pod: %s", err.Error())
	}

	_, err = cluster.CreateDataExport(deName, persistentVolumeClaimType, pvcName, util.TestNS, duration, false)
	if err != nil {
		return fmt.Errorf("failed to create data export: %s", err.Error())
	}

	return nil
}
