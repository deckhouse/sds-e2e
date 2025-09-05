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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	util "github.com/deckhouse/sds-e2e/util"
	"github.com/deckhouse/sds-e2e/util/utiltype"
	coreapi "k8s.io/api/core/v1"
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

	PVCShortType = "pvc"

	DataExportInProgressKey        = "storage.deckhouse.io/data-export-in-progress"
	DataExportRequestAnnotationKey = "storage.deckhouse.io/data-export-request"

	FinalizerName = "storage.deckhouse.io/data-exporter-controller"

	HelloFileMD5 = "40375e146b382bc870c143b4b5aa2cbd"
	tokenTTL     = 24 * 60 * 60
)

// TestDataExport runs all data export related tests
func TestDataExport(t *testing.T) {
	// t.Run("DataExporterCreation", testCreateVD)
	// t.Run("DataExportWithUnsupportedType", testDataExportWithUnsupportedExportType)
	// t.Run("DataExportWithNonExistentType", testDataExportWithNonExistentExportType)
	// t.Run("DataExportTTLExpired", testDataExportTTLExpired)
	// t.Run("DeleteDataExport", testDeleteDataExport)
	// t.Run("ExportTypeAlreadyExported", testExportTypeWhichIsAlreadyExported)
	// t.Run("HTTPRequests", testDataExportHTTPRequests)
	t.Run("dec", TestStorageVolumeDataManagerBlock)
}

func TestStorageVolumeDataManagerBlock(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	// t.Cleanup(func() {
	// 	cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	// })

	if err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "2h"); err != nil {
		t.Fatalf("Failed to create DataExport: %v", err)
	}

	de, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("Failed waiting for DataExport URL: %v", err)
	}

	baseURL := de.Status.Url
	baseFiles := fmt.Sprintf("%s/%s/%s/api/v1/files", de.Namespace, PVCShortType, de.Spec.TargetRef.Name)
	baseBlock := fmt.Sprintf("%s/%s/%s/api/v1/block", de.Namespace, PVCShortType, de.Spec.TargetRef.Name)

	nodes, err := cluster.ListNode()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	nodeName := nodes[0].Name

	token, err := cluster.CreateAuthToken("e2e-user", util.TestNS, 24*60*60)
	if err != nil {
		t.Fatalf("Failed to create auth token: %v", err)
	}

	tests := []struct {
		name     string
		method   string
		path     string
		expected int
	}{
		{
			name:     "GET /files should be 400",
			method:   http.MethodGet,
			path:     baseFiles,
			expected: http.StatusBadRequest,
		},
		{
			name:     "GET /files/ should be 400",
			method:   http.MethodGet,
			path:     baseFiles + "/",
			expected: http.StatusBadRequest,
		},
		{
			name:     "GET block/ should be 400",
			method:   http.MethodGet,
			path:     baseBlock + "/",
			expected: http.StatusBadRequest,
		},
		{
			name:     "GET existing block device should be 200",
			method:   http.MethodGet,
			path:     baseBlock,
			expected: http.StatusOK,
		},
		{
			name:     "GET non-existing block device should be 404",
			method:   http.MethodGet,
			path:     baseBlock + "/nonexistent",
			expected: http.StatusBadRequest,
		},
		{
			name:     "GET incorrect path should be 400",
			method:   http.MethodGet,
			path:     "wrong/path",
			expected: http.StatusNotFound,
		},
		{
			name:     "HEAD existing block device should be 200",
			method:   http.MethodHead,
			path:     baseBlock,
			expected: http.StatusOK,
		},
		{
			name:     "HEAD non-existing block device should be 404",
			method:   http.MethodHead,
			path:     baseBlock + "/nonexistent",
			expected: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := []string{"curl", "-sk", "-H", "Authorization: Bearer " + token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", "-X", tt.method, baseURL + tt.path}
			stdout, stderr, err := cluster.ExecNode(nodeName, args)
			if err != nil {
				t.Fatalf("Curl failed: %v, stderr: %s, stdout: %s", err, stderr, stdout)
			}

			res, err := parseCurlExecOutput(stdout)
			if err != nil {
				t.Fatalf("Failed to parse curl output: %v", err)
			}

			if res.StatusCode != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, res.StatusCode)
			}
		})
	}
}

// testDataExportRoutingValidation: базовая маршрутизация и валидация путей
func testDataExportRoutingValidation(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	})

	if err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "2h"); err != nil {
		t.Fatalf("Failed to create DataExport: %v", err)
	}

	dataExport, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("Failed waiting for DataExport URL: %v", err)
	}

	baseURL := dataExport.Status.Url
	basePath := fmt.Sprintf("%s/%s/%s/api/v1/files", dataExport.Namespace, PVCShortType, dataExport.Spec.TargetRef.Name)
	baseBlockPath := fmt.Sprintf("%s/%s/%s/api/v1/block", dataExport.Namespace, PVCShortType, dataExport.Spec.TargetRef.Name)

	nodes, err := cluster.ListNode()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	nodeName := nodes[0].Name

	token, err := cluster.CreateAuthToken("e2e-user", util.TestNS, tokenTTL)
	if err != nil {
		t.Fatalf("Failed to create auth token: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected int
	}{
		{
			name:     "GET incorrect route should be 404",
			path:     baseURL + "wrong/path",
			expected: http.StatusNotFound,
		},
		{
			name:     "GET /files should be 400",
			path:     baseURL + basePath,
			expected: http.StatusBadRequest,
		},
		{
			name:     "GET /block should be 400",
			path:     baseURL + baseBlockPath,
			expected: http.StatusBadRequest,
		},
		{
			name:     "GET /block/ should be 400",
			path:     baseURL + baseBlockPath + "/",
			expected: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"curl", "-sk", "-H", "Authorization: Bearer " + token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", "-X", http.MethodGet, tt.path}
			stdout, stderr, err := cluster.ExecNode(nodeName, args)
			if err != nil {
				t.Fatalf("Curl failed: %v, stderr: %s, stdout: %s", err, stderr, stdout)
			}

			res, err := parseCurlExecOutput(stdout)
			if err != nil {
				t.Fatalf("Failed to parse curl output: %v", err)
			}

			if res.StatusCode != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, res.StatusCode)
			}
		})
	}
}

// testDataExportAuth: аутентификация
func testDataExportAuth(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	})

	if err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "2h"); err != nil {
		t.Fatalf("Failed to create DataExport: %v", err)
	}

	de, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("Failed waiting for DataExport URL: %v", err)
	}

	basePath := fmt.Sprintf("%s/%s/%s/api/v1/files/hello.txt", de.Namespace, PVCShortType, de.Spec.TargetRef.Name)
	url := de.Status.Url + basePath

	nodes, err := cluster.ListNode()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	nodeName := nodes[0].Name

	tests := []struct {
		name     string
		token    string
		expected int
	}{
		{
			name:     "GET request with missing bearer should return 401",
			token:    "",
			expected: http.StatusUnauthorized,
		},
		{
			name:     "GET request with invalid bearer should return 401",
			token:    "invalid-token",
			expected: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"curl", "-sk", "-H", "Authorization: Bearer " + tt.token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", "-X", http.MethodGet, url}
			stdout, stderr, err := cluster.ExecNode(nodeName, args)
			if err != nil {
				t.Fatalf("Curl failed: %v, stderr: %s, stdout: %s", err, stderr, stdout)
			}

			res, err := parseCurlExecOutput(stdout)
			if err != nil {
				t.Fatalf("Failed to parse curl output: %v", err)
			}

			if res.StatusCode != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, res.StatusCode)
			}
		})
	}
}

// testDataExportFilesContent: файлы (контент)
func testDataExportFilesContent(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	})

	// Создание DataExport
	if err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "2h"); err != nil {
		t.Fatalf("Failed to create DataExport: %v", err)
	}

	// Ожидание готовности URL
	de, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("Failed waiting for DataExport URL: %v", err)
	}

	// Подготовка базового пути
	basePath := fmt.Sprintf("%s/%s/%s/api/v1/files/hello.txt", de.Namespace, PVCShortType, de.Spec.TargetRef.Name)
	url := de.Status.Url + basePath

	// Получение первого узла
	nodes, err := cluster.ListNode()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	nodeName := nodes[0].Name

	// Создание токена
	token, err := cluster.CreateAuthToken("e2e-user", util.TestNS, 24*60*60)
	if err != nil {
		t.Fatalf("Failed to create auth token: %v", err)
	}

	// Тестовые случаи
	tests := []struct {
		name               string
		extectedStatusCode int
		expectMD5          string
	}{
		{
			name:               "GET existing file with md5",
			extectedStatusCode: http.StatusOK,
			expectMD5:          HelloFileMD5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"curl", "-sk", "-H", "Authorization: Bearer " + token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", "-X", http.MethodGet, url}
			stdout, stderr, err := cluster.ExecNode(nodeName, args)
			if err != nil {
				t.Fatalf("Curl failed: %v, stderr: %s, stdout: %s", err, stderr, stdout)
			}

			res, err := parseCurlExecOutput(stdout)
			if err != nil {
				t.Fatalf("error parsing curl exec output: %v", err)
			}

			if res.StatusCode != tt.extectedStatusCode {
				t.Errorf("Expected status %d, got %d", tt.extectedStatusCode, res.StatusCode)
			}

			hash := md5.Sum([]byte(res.Body))
			if gotMD5 := hex.EncodeToString(hash[:]); gotMD5 != tt.expectMD5 {
				t.Errorf("MD5 mismatch: expected %s, got %s", tt.expectMD5, gotMD5)
			}
		})
	}
}

// testDataExportFilesHeaders: файлы (метаданные/заголовки)
func testDataExportFilesHeaders(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	})

	if err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "2h"); err != nil {
		t.Fatalf("Failed to create DataExport: %v", err)
	}

	de, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("Failed waiting for DataExport URL: %v", err)
	}

	basePath := fmt.Sprintf("%s/%s/%s/api/v1/files/hello.txt", de.Namespace, PVCShortType, de.Spec.TargetRef.Name)
	url := de.Status.Url + basePath

	nodes, err := cluster.ListNode()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	nodeName := nodes[0].Name

	token, err := cluster.CreateAuthToken("e2e-user", util.TestNS, 24*60*60)
	if err != nil {
		t.Fatalf("Failed to create auth token: %v", err)
	}

	tests := []struct {
		name              string
		expectedStatus    int
		expectedHeader    string
		expectedHeaderVal string
	}{
		{
			name:              "HEAD existing file returns Content-Length",
			expectedStatus:    http.StatusOK,
			expectedHeader:    "Content-Length",
			expectedHeaderVal: "9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"curl", "-skI", "-H", "Authorization: Bearer " + token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", url}
			stdout, stderr, err := cluster.ExecNode(nodeName, args)
			if err != nil {
				t.Fatalf("Curl failed: %v, stderr: %s, stdout: %s", err, stderr, stdout)
			}

			res, err := parseCurlExecOutput(stdout)
			if err != nil {
				t.Fatalf("Failed to parse curl output: %v", err)
			}

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			val, found := findHeader(res.Headers, tt.expectedHeader)
			if !found {
				t.Errorf("Header %s not found", tt.expectedHeader)
			} else if val != tt.expectedHeaderVal {
				t.Errorf("%s mismatch: expected %s, got %s", tt.expectedHeader, tt.expectedHeaderVal, val)
			}
		})
	}
}

// testDataExportDirectoriesGroup: директории
func testDataExportDirectoriesGroup(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	})

	if err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "2h"); err != nil {
		t.Fatalf("Failed to create DataExport: %v", err)
	}

	de, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("Failed waiting for DataExport URL: %v", err)
	}

	basePath := fmt.Sprintf("%s/%s/%s/api/v1/files/", de.Namespace, PVCShortType, de.Spec.TargetRef.Name)
	url := de.Status.Url + basePath

	nodes, err := cluster.ListNode()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	nodeName := nodes[0].Name

	token, err := cluster.CreateAuthToken("e2e-user", util.TestNS, 24*60*60)
	if err != nil {
		t.Fatalf("Failed to create auth token: %v", err)
	}

	tests := []struct {
		name     string
		method   string
		expected int
		checkDir bool
	}{
		{
			name:     "GET directory lists hello.txt",
			method:   http.MethodGet,
			expected: http.StatusOK,
			checkDir: true,
		},
		{
			name:     "HEAD directory returns 400",
			method:   http.MethodHead,
			expected: http.StatusBadRequest,
			checkDir: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"curl", "-sk"}
			if tt.method == http.MethodHead {
				args = append(args, "-I")
			} else {
				args = append(args, "-X", tt.method)
			}
			args = append(args, "-H", "Authorization: Bearer "+token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", "-X", tt.method, url)

			stdout, stderr, err := cluster.ExecNode(nodeName, args)
			if err != nil {
				t.Fatalf("Curl failed: %v, stderr: %s, stdout: %s", err, stderr, stdout)
			}

			res, err := parseCurlExecOutput(stdout)
			if err != nil {
				t.Fatalf("Failed to parse curl output: %v", err)
			}

			if res.StatusCode != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, res.StatusCode)
			}

			if tt.checkDir {
				var obj struct {
					Items []struct {
						Name string `json:"name"`
					} `json:"items"`
				}
				if err := json.Unmarshal([]byte(res.Body), &obj); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				if len(obj.Items) != 1 || obj.Items[0].Name != "hello.txt" {
					t.Errorf("Expected directory listing with one file 'hello.txt', got %v", obj.Items)
				}
			}
		})
	}
}

// testDataExportMethodNotAllowedGroup: недопустимые методы
func testDataExportMethodNotAllowedGroup(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	})

	if err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "2h"); err != nil {
		t.Fatalf("Failed to create DataExport: %v", err)
	}

	de, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("Failed waiting for DataExport URL: %v", err)
	}

	basePath := fmt.Sprintf("%s/%s/%s/api/v1/files/hello.txt", de.Namespace, PVCShortType, de.Spec.TargetRef.Name)
	url := de.Status.Url + basePath

	nodes, err := cluster.ListNode()
	if err != nil || len(nodes) == 0 {
		t.Fatalf("Failed to list nodes: %v", err)
	}
	nodeName := nodes[0].Name

	token, err := cluster.CreateAuthToken("e2e-user", util.TestNS, 24*60*60)
	if err != nil {
		t.Fatalf("Failed to create auth token: %v", err)
	}

	tests := []struct {
		name     string
		method   string
		expected int
	}{
		{
			name:     "POST should be 405",
			method:   http.MethodPost,
			expected: http.StatusMethodNotAllowed,
		},
		{
			name:     "PUT should be 405",
			method:   http.MethodPut,
			expected: http.StatusMethodNotAllowed,
		},
		{
			name:     "PATCH should be 405",
			method:   http.MethodPatch,
			expected: http.StatusMethodNotAllowed,
		},
		{
			name:     "DELETE should be 405",
			method:   http.MethodDelete,
			expected: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"curl", "-sk", "-H", "Authorization: Bearer " + token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", "-X", tt.method, url}
			stdout, stderr, err := cluster.ExecNode(nodeName, args)
			if err != nil {
				t.Fatalf("Curl failed: %v, stderr: %s, stdout: %s", err, stderr, stdout)
			}

			res, err := parseCurlExecOutput(stdout)
			if err != nil {
				t.Fatalf("Failed to parse curl output: %v", err)
			}

			if res.StatusCode != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, res.StatusCode)
			}
		})
	}
}

// testDataExportHTTPRequests tests various HTTP requests to data export endpoints
func testDataExportHTTPRequests(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	// t.Cleanup(func() {
	// 	cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	// })

	err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "2h")
	if err != nil {
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	_, err = cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("failed to wait DataExport URL to be ready: %s", err.Error())
	}

	dataExport, err := cluster.GetDataExport(testDEName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to create DataExport: %s", err.Error())
	}

	baseURL := dataExport.Status.Url
	fmt.Printf("DataExport URL: %s\n", baseURL)
	basePath := fmt.Sprintf("%s/%s/%s/api/v1/files", dataExport.Namespace, PVCShortType, dataExport.Spec.TargetRef.Name)
	baseBlockPath := fmt.Sprintf("%s/%s/%s/api/v1/block", dataExport.Namespace, PVCShortType, dataExport.Spec.TargetRef.Name)
	secretDirPath := basePath + "/secret/"
	symlinkPath := basePath + "/hosts_symlink"
	lockedFilePath := basePath + "/locked.txt"

	testCases := []struct {
		description             string
		method                  string
		path                    string
		statusCode              int
		expectHeader            string
		expectHeaderValue       string
		bodyContains            []string
		expectJSONItemsContains []string
		expectMD5               string
		noAuth                  bool
		overrideToken           string
	}{
		{
			description: "Test GET request to unexisting file", // GET-запрос к несуществующему файлу
			method:      http.MethodGet,
			path:        basePath + "/unexisting-file.txt",
			statusCode:  http.StatusNotFound,
		},
		{
			description: "Test GET request to directory without trailing slash", // GET-запрос к директории без завершающего слеша
			method:      http.MethodGet,
			path:        basePath + "",
			statusCode:  http.StatusBadRequest,
		},
		{
			description: "Test GET request to incorrect route", // GET-запрос к некорректному маршруту
			method:      http.MethodGet,
			path:        "wrong/path",
			statusCode:  http.StatusNotFound,
		},
		{
			description: "Test HEAD request to unexisting file", // HEAD-запрос к несуществующему файлу
			method:      http.MethodHead,
			path:        basePath + "/unexisting-file.txt",
			statusCode:  http.StatusNotFound,
		},
		{
			description: "Test HEAD request to incorrect route", // HEAD-запрос к некорректному маршруту
			method:      http.MethodHead,
			path:        "wrong/path",
			statusCode:  http.StatusNotFound,
		},
		{
			description: "Test unsupported POST method",
			method:      http.MethodPost,
			path:        basePath + "/hello.txt",
			statusCode:  http.StatusMethodNotAllowed,
		},
		{
			description: "Test unsupported PUT method",
			method:      http.MethodPut,
			path:        basePath + "/hello.txt",
			statusCode:  http.StatusMethodNotAllowed,
		},
		{
			description: "Test unsupported PATCH method",
			method:      http.MethodPatch,
			path:        basePath + "/hello.txt",
			statusCode:  http.StatusMethodNotAllowed,
		},
		{
			description: "Test unsupported DELETE method",
			method:      http.MethodDelete,
			path:        basePath + "/hello.txt",
			statusCode:  http.StatusMethodNotAllowed,
		},
		{
			description: "Test GET request to file URL with trailing slash", // GET-запрос к файлу с URL, заканчивающимся на слеш
			method:      http.MethodGet,
			path:        basePath + "/hello.txt/",
			statusCode:  400,
		},
		{
			description: "Test HEAD request to file URL with trailing slash", // HEAD-запрос к файлу с URL, заканчивающимся на слеш
			method:      http.MethodHead,
			path:        basePath + "/hello.txt/",
			statusCode:  400,
		},
		{
			description:       "Test HEAD request to existing file", // HEAD-запрос к существующему файлу
			method:            http.MethodHead,
			path:              basePath + "/hello.txt",
			statusCode:        http.StatusOK,
			expectHeader:      "Content-Length",
			expectHeaderValue: "9",
		},
		{
			description: "Test GET request to existing file with md5", // GET-запрос к существующему файлу c md5
			method:      http.MethodGet,
			path:        basePath + "/hello.txt",
			statusCode:  http.StatusOK,
			expectMD5:   HelloFileMD5,
		},
		{
			description: "Test GET request to restricted directory should be 403", // GET-запрос к директории с ограниченным доступом
			method:      http.MethodGet,
			path:        secretDirPath,
			statusCode:  http.StatusForbidden,
		},
		{
			description: "Test HEAD request to restricted directory should be 400", // HEAD-запрос к директории с ограниченным доступом 23
			method:      http.MethodHead,
			path:        secretDirPath,
			statusCode:  http.StatusBadRequest,
		},
		{
			description: "Test GET request to symlink outside root should be 403", // GET-запрос к символической ссылке за пределами корневой директории
			method:      http.MethodGet,
			path:        symlinkPath,
			statusCode:  http.StatusForbidden,
		},
		{
			description: "Test HEAD request to symlink outside root should be 403", // HEAD-запрос к символической ссылке за пределами корневой директории
			method:      http.MethodHead,
			path:        symlinkPath,
			statusCode:  http.StatusForbidden,
		},
		{
			description: "Test GET request to files root should be 400", // GET-запрос на /block, /block/ или files
			method:      http.MethodGet,
			path:        basePath,
			statusCode:  http.StatusBadRequest,
		},
		{
			description: "Test GET request to block without trailing slash should be 400", // GET-запрос на /block, /block/ или files
			method:      http.MethodGet,
			path:        baseBlockPath,
			statusCode:  http.StatusBadRequest,
		},
		{
			description: "Test GET request to block with trailing slash should be 400", // GET-запрос на /block, /block/ или files
			method:      http.MethodGet,
			path:        baseBlockPath + "/",
			statusCode:  http.StatusBadRequest,
		},
		{
			description: "Test GET request to restricted file should be 403", // GET-запрос к файлу с ограниченным доступом
			method:      http.MethodGet,
			path:        lockedFilePath,
			statusCode:  http.StatusForbidden,
		},
		{
			description: "Test HEAD request to restricted file should be 403", // HEAD-запрос к файлу с ограниченным доступом
			method:      http.MethodHead,
			path:        lockedFilePath,
			statusCode:  http.StatusForbidden,
		},
		{
			description:             "Test GET request to existing directory", // GET-запрос к существующей директории
			method:                  http.MethodGet,
			path:                    basePath + "/",
			statusCode:              http.StatusOK,
			expectJSONItemsContains: []string{"hello.txt"},
		},
		{
			description: "Test GET with missing bearer token should be 401", // Токен отсутствует
			method:      http.MethodGet,
			path:        basePath + "/hello.txt",
			statusCode:  http.StatusUnauthorized,
			noAuth:      true,
		},
		{
			description:   "Test GET with invalid bearer token should be 401", // Токен невалидный
			method:        http.MethodGet,
			path:          basePath + "/hello.txt",
			statusCode:    http.StatusUnauthorized,
			overrideToken: "invalid-token",
		},
		{
			description: "Test HEAD request to directory should be 400", // HEAD-запрос к директории
			method:      http.MethodHead,
			path:        basePath + "/",
			statusCode:  http.StatusBadRequest,
		},
	}

	// Pick a node to run curl from (command executes in host namespaces via nsenter)
	nodes, err := cluster.ListNode()
	if err != nil {
		t.Fatalf("failed to list nodes: %s", err.Error())
	}
	if len(nodes) == 0 {
		t.Fatalf("no nodes available to run HTTP checks")
	}
	nodeName := nodes[0].Name

	// Obtain bearer token to access data-exporter
	token, err := cluster.CreateAuthToken("e2e-user", util.TestNS, 24*60*60)
	if err != nil {
		t.Fatalf("failed to create bearer token: %s", err.Error())
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			fmt.Printf("baseURL+tc.path: %s%s\n", baseURL, tc.path)
			fullURL := baseURL + tc.path
			fmt.Printf("curl URL: %s (method %s)\n", fullURL, tc.method)

			authHeader := []string{}
			if !tc.noAuth {
				useToken := token
				if tc.overrideToken != "" {
					useToken = tc.overrideToken
				}
				authHeader = []string{"-H", "Authorization: Bearer " + useToken}
			}

			var cmd []string
			if tc.method == http.MethodHead {
				cmd = append([]string{"curl", "-skI"}, authHeader...)
				cmd = append(cmd, []string{"-w", "\n__HTTP_CODE__:%{http_code}__END__", fullURL}...)
			} else {
				cmd = append([]string{"curl", "-sk"}, authHeader...)
				cmd = append(cmd, []string{"-w", "\n__HTTP_CODE__:%{http_code}__END__", "-X", tc.method, fullURL}...)
			}
			stdout, stderr, err := cluster.ExecNode(nodeName, cmd)
			if err != nil {
				t.Fatalf("curl failed on node %s: %v\nstderr: %s\nstdout: %s", nodeName, err, stderr, stdout)
			}

			parsed, perr := parseCurlExecOutput(stdout)
			if perr != nil {
				t.Fatalf("failed to parse curl output: %v. Output: %q", perr, stdout)
			}

			bodyRaw := parsed.BodyRaw
			body := parsed.Body
			code := parsed.StatusCode

			t.Logf("Response body (%s %s):\n%s", tc.method, fullURL, body)

			if code != tc.statusCode {
				t.Errorf("response status code mismatch. Expected %d, received %d", tc.statusCode, code)
			}

			// Optional MD5 check for GET file
			if tc.expectMD5 != "" && tc.method == http.MethodGet && code == http.StatusOK {
				h := md5.Sum([]byte(bodyRaw))
				got := hex.EncodeToString(h[:])
				if got != tc.expectMD5 {
					t.Errorf("MD5 mismatch: expected %s, got %s", tc.expectMD5, got)
				}
			}

			// Optional JSON body assertions
			if len(tc.bodyContains) > 0 || len(tc.expectJSONItemsContains) > 0 {
				// Try to parse JSON
				var obj struct {
					APIVersion string `json:"apiVersion"`
					Items      []struct {
						Name string `json:"name"`
					} `json:"items"`
				}
				if err := json.Unmarshal([]byte(body), &obj); err == nil {
					if len(tc.expectJSONItemsContains) > 0 {
						nameSet := map[string]struct{}{}
						for _, it := range obj.Items {
							nameSet[it.Name] = struct{}{}
						}
						for _, want := range tc.expectJSONItemsContains {
							if _, ok := nameSet[want]; !ok {
								t.Errorf("directory listing does not contain %q", want)
							}
						}
					}
				} else {
					// Fallback to substring checks if not a JSON body
					for _, sub := range tc.bodyContains {
						if !strings.Contains(body, sub) {
							t.Errorf("response body does not contain expected substring: %s", sub)
						}
					}
				}
			}

			// Optional header assertion (works for HEAD where body contains headers)
			if tc.expectHeader != "" {
				lowHeader := strings.ToLower(tc.expectHeader) + ":"
				found := false
				for _, line := range strings.Split(body, "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(strings.ToLower(line), lowHeader) {
						val := strings.TrimSpace(strings.TrimPrefix(line, line[:len(lowHeader)]))
						if val != tc.expectHeaderValue {
							t.Errorf("header %s mismatch. Expected %q, got %q", tc.expectHeader, tc.expectHeaderValue, val)
						}
						found = true
						break
					}
				}
				if !found {
					t.Errorf("header %s not found in response", tc.expectHeader)
				}
			}
		})
	}
}

func testCreateVD(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	// t.Cleanup(func() {
	// 	cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	// 	cluster.DeleteVD(util.VdFilter{Name: testVDName, NameSpace: util.TestNS})
	// })

	_, err := cluster.CreatePVCInTestNS(testPVCName, util.NestedDefaultStorageClass, "20Mi")
	if err != nil {
		t.Fatalf("failed to create PVC: %s", err.Error())
	}

	err = cluster.CreateVD(testVDName, util.TestNS, util.NestedDefaultStorageClass, 10240)
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

	// t.Cleanup(func() {
	// 	cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	// })

	err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "30s")
	if err != nil {
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	time.Sleep(31 * time.Second)

	dataExport, err := cluster.GetDataExport(testDEName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to get data export: %s", err.Error())
	}

	// Expect EXPIRED condition to be True (always present by design)
	for _, cond := range dataExport.Status.Conditions {
		if cond.Type == "EXPIRED" {
			if cond.Status != v1.ConditionTrue {
				t.Errorf("data export TTL has expired but EXPIRED condition is not true: got %s (%s)", cond.Status, cond.Reason)
			}
			break
		}
	}

	// Wait until annotations and finalizer are cleared from PVC (fresh GET each time)
	if err := waitPVCAnnotationsAndFinalizersCleared(cluster, testPVCName, 60*time.Second); err != nil {
		t.Error(err)
	}
}

func waitPVCAnnotationsAndFinalizersCleared(cluster *util.KCluster, pvcName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pvc, err := cluster.GetPVC(pvcName, util.TestNS)
		if err != nil {
			return fmt.Errorf("failed to get PVC: %w", err)
		}

		stale := false
		for annotation := range pvc.Annotations {
			if annotation == DataExportInProgressKey || annotation == DataExportRequestAnnotationKey {
				stale = true
				break
			}
		}
		if !stale {
			finalizerPresent := false
			for _, f := range pvc.Finalizers {
				if f == FinalizerName {
					finalizerPresent = true
					break
				}
			}
			if !finalizerPresent {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("PVC %s still has stale metadata: annotations=%v, finalizers=%v", pvcName, pvc.Annotations, pvc.Finalizers)
		}
		time.Sleep(2 * time.Second)
	}
}

func testDeleteDataExport(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	})

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
		t.Fatalf("failed to get PVC: %s", err.Error())
	}

	pvName := pvc.Spec.VolumeName
	pv, err := cluster.GetPV(pvName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to get PV: %s", err.Error())
	}

	// TODO check all?
	if pv.Spec.ClaimRef.Name != pvc.Name {
		t.Errorf("PV is not reattached to original PVC: ClaimRef.Name: %s, PVC name %s", pv.Spec.ClaimRef.Name, pvc.Name)
	}

	if pv.Spec.ClaimRef.Namespace != pvc.Namespace {
		t.Errorf("PV is not reattached to original PVC: ClaimRef.Namespace: %s, PVC Namespace %s", pv.Spec.ClaimRef.Namespace, pvc.Namespace)
	}

	if pv.Spec.ClaimRef.UID != pvc.UID {
		t.Errorf("PV is not reattached to original PVC: ClaimRef.UID: %s, PVC UID %s", pv.Spec.ClaimRef.UID, pvc.UID)
	}

	waitPVCAnnotationsAndFinalizersCleared(cluster, testPVCName, 60*time.Second)
}

func testExportTypeWhichIsAlreadyExported(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
		cleanupDataExport(cluster, "duplicate-data-export", testPVCName, testPodName)
	})

	err := CreateDataExportWithPVC(cluster, testPodName, testPVCName, testDEName, "1h")
	if err != nil {
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	duplicateName := "duplicate-data-export"
	_, err = cluster.CreateDataExport(duplicateName, persistentVolumeClaimType, testPVCName, util.TestNS, "1h", false)
	if err != nil {
		t.Fatalf("failed to create duplicate data export: %s", err.Error())
	}

	dataExport, err := cluster.GetDataExport(duplicateName, util.TestNS)
	if err != nil {
		t.Fatalf("failed to get duplicate data export: %s", err.Error())
	}

	if err = checkIfValidationFailed(dataExport); err != nil {
		t.Fatalf("dataExport %s validation should have failed: %s", duplicateName, err.Error())
	}
}

func testDataExportWithUnsupportedExportType(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	_, err := cluster.CreateDataExport(testDEName, unsupportedExportType, "fake-kind-name", util.TestNS, "1h", false)
	if err == nil {
		t.Error("DataExport was created with an unsupported export type, but should have failed")
	}
}

func testDataExportWithNonExistentExportType(t *testing.T) {
	cluster := util.EnsureCluster("", "")

	exportTypes := []string{"PersistentVolumeClaim", "VolumeSnapshot", "VirtualDisk", "VirtualDiskSnapshot"}

	for _, exportType := range exportTypes {
		t.Run(exportType, func(t *testing.T) {
			t.Cleanup(func() {
				cleanupDataExport(cluster, testDEName, "fake-kind-name", testPodName)
			})

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
				t.Fatalf("dataExport %s validation should have failed: %s", testDEName, err.Error())
			}
		})
	}
}

func testDataExporterCreation(t *testing.T) {
	cluster := util.EnsureCluster("", "")
	t.Cleanup(func() {
		cleanupDataExport(cluster, testDEName, testPVCName, testPodName)
	})

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
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("failed to send HTTP %s request: %w", request.Method, err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusMethodNotAllowed {
			return fmt.Errorf("dataExport %s accepts %s HTTP method, but it shouldn't", dataExport.Name, request.Method)
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
	url := fmt.Sprintf("%s%s", dataExport.Status.Url, "health")
	fmt.Printf("== URL: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to check server health: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("== Status: %s", resp.Status)
	return nil
}

// CurlExecResult represents a parsed result of a curl command executed via ExecNode.
// It assumes that curl was invoked with: -w "\n__HTTP_CODE__:%{http_code}__END__"
// so that the HTTP status code is appended to the end of stdout after a marker.
type CurlExecResult struct {
	StatusCode int
	BodyRaw    string // raw body (as-is), without the trailing newline added by -w
	Body       string // trimmed body (for logging/convenience)
	Headers    string // raw headers (as-is), without the trailing newline added by -w
}

// parseCurlExecOutput parses stdout produced by curl with the marker
// "__HTTP_CODE__:" and closing "__END__" and returns a structured result.
// Expected stdout tail: "\n__HTTP_CODE__:XYZ__END__"
func parseCurlExecOutput(stdout string) (*CurlExecResult, error) {
	const (
		marker    = "__HTTP_CODE__:"
		endMarker = "__END__"
	)

	idx := strings.LastIndex(stdout, marker)
	if idx == -1 {
		return nil, fmt.Errorf("curl output parse error: status marker not found")
	}

	rest := stdout[idx+len(marker):]
	endIdx := strings.Index(rest, endMarker)
	if endIdx == -1 {
		return nil, fmt.Errorf("curl output parse error: end marker not found")
	}
	codeStr := strings.TrimSpace(rest[:endIdx])

	bodyRaw := stdout[:idx]
	bodyRaw = strings.TrimSuffix(bodyRaw, "\n")
	body := strings.TrimSpace(bodyRaw)

	code, convErr := strconv.Atoi(codeStr)
	if convErr != nil {
		return nil, fmt.Errorf("curl output parse error: invalid status code %q: %w", codeStr, convErr)
	}

	return &CurlExecResult{
		StatusCode: code,
		BodyRaw:    bodyRaw,
		Body:       body,
		Headers:    bodyRaw,
	}, nil
}

func findHeader(headers, headerName string) (string, bool) {
	for _, line := range strings.Split(headers, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), strings.ToLower(headerName)+":") {
			return strings.TrimSpace(line[len(headerName)+1:]), true
		}
	}
	return "", false
}

func checkIfValidationFailed(dataExport *utiltype.DataExport) error {
	for _, cond := range dataExport.Status.Conditions {
		if cond.Type == "Ready" && cond.Reason == "ValidationFailed" {
			return nil
		}
	}
	return fmt.Errorf("validation did not fail as expected")
}

func checkPVCAnnotationsAndFinalizers(t *testing.T, pvc *coreapi.PersistentVolumeClaim) {
	for annotation := range pvc.Annotations {
		if annotation == DataExportInProgressKey || annotation == DataExportRequestAnnotationKey {
			t.Errorf("PVC has annotation %s but it should not", annotation)
		}
	}

	for _, finalizer := range pvc.Finalizers {
		if finalizer == FinalizerName {
			t.Errorf("PVC has finalizer %s but it should not", finalizer)
		}
	}
}

func cleanupDataExport(cluster *util.KCluster, deName, pvcName, podName string) {
	_ = cluster.DeleteDataExport(deName, util.TestNS)
	_ = cluster.DeletePod(podName, util.TestNS)
	_ = cluster.DeletePVC(pvcName)
}

func CreateDataExportWithPVC(cluster *util.KCluster, podName, pvcName, deName, duration string) error {
	pvc, err := cluster.CreatePVCInTestNS(pvcName, util.NestedDefaultStorageClass, "1Mi")
	if err != nil {
		return fmt.Errorf("failed to create PVC: %w", err)
	}
	util.Infof("Created PVC: %s in namespace: %s", pvc.Name, pvc.Namespace)

	err = cluster.CreateDummyPod(podName, testNameSpace, pvcName)
	if err != nil {
		return fmt.Errorf("failed to create dummy pod: %w", err)
	}

	// Wait until the writer pod is running and ready to ensure the file write completes
	if err := cluster.WaitPodReady(podName, util.TestNS); err != nil {
		return fmt.Errorf("failed to wait writer pod ready: %w", err)
	}

	_, err = cluster.WaitPVCStatus(pvcName)
	if err != nil {
		return fmt.Errorf("failed to wait PVC to become bound: %w", err)
	}

	err = cluster.DeletePod(podName, util.TestNS)
	if err != nil {
		return fmt.Errorf("failed to delete test pod: %w", err)
	}

	if err = cluster.WaitPodDeletion(podName, util.TestNS); err != nil {
		return fmt.Errorf("failed to wait for pod deletion: %w", err)
	}

	_, err = cluster.CreateDataExport(deName, persistentVolumeClaimType, pvcName, util.TestNS, duration, false)
	if err != nil {
		return fmt.Errorf("failed to create data export: %w", err)
	}

	return nil
}
