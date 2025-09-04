package integration

import (
	"fmt"
	"net/http"
	"testing"

	util "github.com/deckhouse/sds-e2e/util"
)

// TestStorageVolumeDataManagerBlock covers block-device HTTP API basic flows
func TestStorageVolumeDataManagerBlock(t *testing.T) {
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
			expected: http.StatusNotFound,
		},
		{
			name:     "GET incorrect path should be 400",
			method:   http.MethodGet,
			path:     "wrong/path",
			expected: http.StatusBadRequest,
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
