package multinode

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
)

// NodeStatus is the subset of a Kubernetes Node used by the assertions.
type NodeStatus struct {
	Name  string
	Ready bool
}

// ParseNodes extracts node readiness from `kubectl get nodes -o json`.
func ParseNodes(output string) ([]NodeStatus, error) {
	var list struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Status struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &list); err != nil {
		return nil, fmt.Errorf("parse node list: %w", err)
	}

	nodes := make([]NodeStatus, 0, len(list.Items))
	for _, item := range list.Items {
		status := NodeStatus{Name: item.Metadata.Name}
		for _, condition := range item.Status.Conditions {
			if condition.Type == "Ready" {
				status.Ready = condition.Status == "True"
			}
		}
		nodes = append(nodes, status)
	}
	return nodes, nil
}

// VerifyNodes asserts that exactly the expected nodes are Ready.
func VerifyNodes(output string, expected []string) error {
	nodes, err := ParseNodes(output)
	if err != nil {
		return err
	}

	ready := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if node.Ready {
			ready[node.Name] = struct{}{}
		}
	}

	if len(ready) != len(expected) {
		return fmt.Errorf("expected %d Ready nodes, got %d: %v", len(expected), len(ready), sortedNames(ready))
	}
	for _, name := range expected {
		if _, ok := ready[name]; !ok {
			return fmt.Errorf("node %q is not Ready; Ready nodes: %v", name, sortedNames(ready))
		}
	}
	return nil
}

// VerifyVersion checks the version reported by `k3s --version`.
func VerifyVersion(output, want string) error {
	if want == "" {
		return fmt.Errorf("expected version is empty")
	}
	if !strings.Contains(output, want) {
		return fmt.Errorf("installed K3s version does not contain %q: %s", want, strings.TrimSpace(output))
	}
	return nil
}

// VerifyStatus checks that every host in `k3sctl config status` reports
// "running". The table has a HOST, ROLE and STATUS column.
func VerifyStatus(output string, nodes []environment.Node) error {
	statuses := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		statuses[fields[0]] = fields[2]
	}

	for _, node := range nodes {
		status, ok := statuses[node.Name]
		if !ok {
			return fmt.Errorf("host %q missing from status output", node.Name)
		}
		if status != "running" {
			return fmt.Errorf("host %q status is %q, want %q", node.Name, status, "running")
		}
	}
	return nil
}

// VerifyKubeconfig checks that the kubeconfig points at the requested API
// address.
func VerifyKubeconfig(output, apiAddress string) error {
	want := "server: " + apiAddress
	if !strings.Contains(output, want) {
		return fmt.Errorf("kubeconfig does not contain %q", want)
	}
	return nil
}

func sortedNames(names map[string]struct{}) []string {
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// CheckAPIServer verifies that the Kubernetes API endpoint responds over TLS.
// The API requires authentication, so 401 and 403 responses also prove that the
// endpoint is serving.
func CheckAPIServer(ctx context.Context, address string) error {
	return checkAPIURL(ctx, fmt.Sprintf("https://%s:6443/version", address))
}

// checkAPIURL verifies that the given endpoint responds over TLS. It accepts
// 200, 401 and 403 because the API requires authentication.
func checkAPIURL(ctx context.Context, endpoint string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			// The cluster uses a self-signed serving certificate.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build API request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer func() { _ = response.Body.Close() }()

	switch response.StatusCode {
	case http.StatusOK, http.StatusUnauthorized, http.StatusForbidden:
		return nil
	default:
		return fmt.Errorf("unexpected status %d from %s", response.StatusCode, endpoint)
	}
}
