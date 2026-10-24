package multinode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
)

func TestParseNodesEmptyItems(t *testing.T) {
	nodes, err := ParseNodes(`{"items":[]}`)
	if err != nil {
		t.Fatalf("ParseNodes() error = %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("len(nodes) = %d, want 0", len(nodes))
	}
}

func TestParseNodesMultipleConditions(t *testing.T) {
	input := `{"items":[{"metadata":{"name":"node1"},"status":{"conditions":[
		{"type":"DiskPressure","status":"False"},
		{"type":"MemoryPressure","status":"False"},
		{"type":"Ready","status":"True"},
		{"type":"PIDPressure","status":"False"}
	]}}]}`
	nodes, err := ParseNodes(input)
	if err != nil {
		t.Fatalf("ParseNodes() error = %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d, want 1", len(nodes))
	}
	if !nodes[0].Ready {
		t.Error("node should be Ready despite multiple conditions")
	}
}

func TestVerifyNodesEmptyExpectedEmptyOutput(t *testing.T) {
	err := VerifyNodes(`{"items":[]}`, []string{})
	if err != nil {
		t.Fatalf("VerifyNodes() error = %v, want nil", err)
	}
}

func TestVerifyNodesExpectedButNoOutput(t *testing.T) {
	err := VerifyNodes(`{"items":[]}`, []string{"server"})
	if err == nil {
		t.Fatal("VerifyNodes() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "expected 1 Ready nodes, got 0") {
		t.Errorf("error = %v, want count mismatch message", err)
	}
}

func TestVerifyVersionWithTrailingNewline(t *testing.T) {
	output := "k3s version v1.30.0+k3s1 (abc123)\n\n"
	if err := VerifyVersion(output, "v1.30.0+k3s1"); err != nil {
		t.Fatalf("VerifyVersion() error = %v", err)
	}
}

func TestVerifyVersionSubstringMatch(t *testing.T) {
	output := "k3s version v1.30.0+k3s1 (abc123)"
	if err := VerifyVersion(output, "v1.30.0"); err != nil {
		t.Fatalf("VerifyVersion() should match substring, error = %v", err)
	}
}

func TestVerifyStatusSkipsHeaderLine(t *testing.T) {
	nodes := []environment.Node{{Name: "server"}}
	output := "HOST     ROLE    STATUS\nserver   server  running\n"
	if err := VerifyStatus(output, nodes); err != nil {
		t.Fatalf("VerifyStatus() error = %v", err)
	}
}

func TestVerifyStatusMultipleNodes(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server"},
		{Name: "agent1"},
		{Name: "agent2"},
	}
	output := "server   server  running\nagent1   agent   running\nagent2   agent   running\n"
	if err := VerifyStatus(output, nodes); err != nil {
		t.Fatalf("VerifyStatus() error = %v", err)
	}
}

func TestVerifyStatusPartiallyRunning(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server"},
		{Name: "agent1"},
	}
	output := "server   server  running\nagent1   agent   stopped\n"
	err := VerifyStatus(output, nodes)
	if err == nil {
		t.Fatal("VerifyStatus() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), `"agent1"`) {
		t.Errorf("error = %v, want it to mention agent1", err)
	}
}

func TestVerifyKubeconfigValidAddress(t *testing.T) {
	output := "apiVersion: v1\nclusters:\n- cluster:\n    server: https://192.168.1.10:6443\n"
	if err := VerifyKubeconfig(output, "https://192.168.1.10:6443"); err != nil {
		t.Fatalf("VerifyKubeconfig() error = %v", err)
	}
}

func TestVerifyKubeconfigMismatch(t *testing.T) {
	output := "server: https://10.0.0.1:6443"
	err := VerifyKubeconfig(output, "https://10.0.0.99:6443")
	if err == nil {
		t.Fatal("VerifyKubeconfig() error = nil, want a mismatch error")
	}
}

func TestCheckAPIURLAcceptsValidStatuses(t *testing.T) {
	for _, status := range []int{
		http.StatusOK,
		http.StatusUnauthorized,
		http.StatusForbidden,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()

			if err := checkAPIURL(context.Background(), server.URL); err != nil {
				t.Fatalf("checkAPIURL() error = %v", err)
			}
		})
	}
}

func TestCheckAPIURLRejectsServerError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	err := checkAPIURL(context.Background(), server.URL)
	if err == nil {
		t.Fatal("checkAPIURL() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error = %v, want it to mention status 502", err)
	}
}

func TestCheckAPIURLContextCancellation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := checkAPIURL(ctx, server.URL); err == nil {
		t.Fatal("checkAPIURL() error = nil, want a context error")
	}
}

func TestNodeStatusStruct(t *testing.T) {
	status := NodeStatus{Name: "test-node", Ready: true}
	if status.Name != "test-node" {
		t.Errorf("Name = %q", status.Name)
	}
	if !status.Ready {
		t.Error("Ready should be true")
	}
}

func TestSortedNames(t *testing.T) {
	names := map[string]struct{}{"charlie": {}, "alpha": {}, "bravo": {}}
	sorted := sortedNames(names)
	if len(sorted) != 3 {
		t.Fatalf("len(sorted) = %d, want 3", len(sorted))
	}
	if sorted[0] != "alpha" || sorted[1] != "bravo" || sorted[2] != "charlie" {
		t.Errorf("sorted = %v, want [alpha bravo charlie]", sorted)
	}
}

func TestSortedNamesEmpty(t *testing.T) {
	sorted := sortedNames(map[string]struct{}{})
	if len(sorted) != 0 {
		t.Errorf("len(sorted) = %d, want 0", len(sorted))
	}
}
