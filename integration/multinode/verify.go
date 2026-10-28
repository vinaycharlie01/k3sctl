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
	notReady := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node.Ready {
			ready[node.Name] = struct{}{}
			continue
		}
		notReady = append(notReady, node.Name)
	}
	sort.Strings(notReady)

	if len(ready) != len(expected) {
		if len(notReady) > 0 {
			return fmt.Errorf("expected %d Ready nodes, got %d: %v; not Ready: %v", len(expected), len(ready), sortedNames(ready), notReady)
		}
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

// VerifyEtcdMembers checks that the etcd member list output contains exactly
// the expected number of members. It works with both table and simple output
// formats from etcdctl.
func VerifyEtcdMembers(output string, expected int) error {
	if strings.TrimSpace(output) == "" {
		return fmt.Errorf("etcd member list output is empty")
	}

	count := 0
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip table borders and headers
		if strings.HasPrefix(trimmed, "+") || strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "ID") {
			continue
		}
		// Count lines that look like member entries: they contain a peer URL
		// (https://...:2380) or a comma-separated ID field.
		if strings.Contains(trimmed, ":2380") || strings.Contains(trimmed, "started") {
			count++
		}
	}

	if count != expected {
		return fmt.Errorf("etcd cluster has %d members, want %d; output:\n%s", count, expected, output)
	}
	return nil
}

// etcdMemberListResponse is the JSON response from the etcd v3 gRPC gateway
// POST /v3/cluster/member/list endpoint.
type etcdMemberListResponse struct {
	Members []EtcdMember `json:"members"`
}

// EtcdMember represents a single etcd cluster member from the REST API.
type EtcdMember struct {
	ID       json.Number `json:"ID"`
	Name     string      `json:"name"`
	PeerURLs []string    `json:"peerURLs"`
}

// VerifyEtcdMembersFromAPI parses the JSON response from the etcd v3 REST API
// and verifies that the cluster has exactly the expected number of members.
func VerifyEtcdMembersFromAPI(jsonOutput string, expected int) error {
	if strings.TrimSpace(jsonOutput) == "" {
		return fmt.Errorf("etcd member list API response is empty")
	}
	var resp etcdMemberListResponse
	if err := json.Unmarshal([]byte(jsonOutput), &resp); err != nil {
		return fmt.Errorf("parse etcd member list response: %w (output: %.200s)", err, jsonOutput)
	}
	count := len(resp.Members)
	if count != expected {
		return fmt.Errorf("etcd cluster has %d members, want %d", count, expected)
	}
	return nil
}

// ParseEtcdMembersFromAPI parses the JSON response from the etcd v3 REST API
// and returns the member list.
func ParseEtcdMembersFromAPI(jsonOutput string) ([]EtcdMember, error) {
	var resp etcdMemberListResponse
	if err := json.Unmarshal([]byte(jsonOutput), &resp); err != nil {
		return nil, fmt.Errorf("parse etcd member list response: %w", err)
	}
	return resp.Members, nil
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
