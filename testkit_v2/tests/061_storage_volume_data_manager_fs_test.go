package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
		t.Fatalf("failed to create a DataExport: %s", err.Error())
	}

	de, err := cluster.WaitDataExportURLReady(testDEName)
	if err != nil {
		t.Fatalf("failed to await DataExport URL: %v", err)
	}

	baseURL := de.Status.Url
	ns := de.Namespace
	name := de.Spec.TargetRef.Name

	baseFiles := fmt.Sprintf("%s/%s/%s/api/v1/files", ns, PVCShortType, name)
	baseBlock := fmt.Sprintf("%s/%s/%s/api/v1/block", ns, PVCShortType, name)

	testCases := []struct {
		description  string
		method       string
		path         string
		expectedCode int
	}{
		{
			description:  "GET /files should be 400",
			method:       http.MethodGet,
			path:         baseFiles,
			expectedCode: http.StatusBadRequest,
		},
		{
			description:  "GET /files/ should be 400",
			method:       http.MethodGet,
			path:         baseFiles + "/",
			expectedCode: http.StatusBadRequest,
		},
		{
			description:  "GET block/ should be 400",
			method:       http.MethodGet,
			path:         baseBlock + "/",
			expectedCode: http.StatusBadRequest,
		},
		{
			description:  "GET existing block device should be 200",
			method:       http.MethodGet,
			path:         baseBlock,
			expectedCode: http.StatusOK,
		},
		{
			description:  "GET non-existing block device should be 404",
			method:       http.MethodGet,
			path:         baseBlock + "/nonexistent",
			expectedCode: http.StatusNotFound,
		},
		{
			description:  "GET incorrect path should be 400",
			method:       http.MethodGet,
			path:         "wrong/path",
			expectedCode: http.StatusBadRequest,
		},
		{
			description:  "HEAD existing block device should be 200",
			method:       http.MethodHead,
			path:         baseBlock,
			expectedCode: http.StatusOK,
		},
		{
			description:  "HEAD non-existing block device should be 404",
			method:       http.MethodHead,
			path:         baseBlock + "/nonexistent",
			expectedCode: http.StatusNotFound,
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
	token, err := cluster.CreateDataExporterBearer("e2e-user", 24*60*60)
	if err != nil {
		t.Fatalf("failed to create bearer token: %s", err.Error())
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			fullURL := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(tc.path, "/")
			fmt.Printf("curl URL: %s (method %s)\n", fullURL, tc.method)

			var cmd []string
			if tc.method == http.MethodHead {
				cmd = []string{"curl", "-skI", "-H", "Authorization: Bearer " + token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", fullURL}
			} else {
				cmd = []string{"curl", "-sk", "-H", "Authorization: Bearer " + token, "-w", "\n__HTTP_CODE__:%{http_code}__END__", "-X", tc.method, fullURL}
			}
			stdout, stderr, err := cluster.ExecNode(nodeName, cmd)
			if err != nil {
				t.Fatalf("curl failed on node %s: %v\nstderr: %s\nstdout: %s", nodeName, err, stderr, stdout)
			}

			marker := "__HTTP_CODE__:"
			endMarker := "__END__"
			idx := strings.LastIndex(stdout, marker)
			if idx == -1 {
				t.Fatalf("failed to parse curl output: no status marker found. Output: %q", stdout)
			}
			rest := stdout[idx+len(marker):]
			endIdx := strings.Index(rest, endMarker)
			if endIdx == -1 {
				t.Fatalf("failed to parse curl output: no end marker found. Output: %q", stdout)
			}
			codeStr := strings.TrimSpace(rest[:endIdx])

			code, convErr := strconv.Atoi(codeStr)
			if convErr != nil {
				t.Fatalf("failed to parse HTTP status code from curl output %q: %v", codeStr, convErr)
			}

			if code != tc.expectedCode {
				t.Errorf("response status code mismatch. Expected %d, received %d", tc.expectedCode, code)
			}
		})
	}
}
