package multinode

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
	v1beta1 "github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
)

func TestClusterConfigGenerateMultipleServers(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server-1", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "server-2", Role: "server", Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "agent-1", Role: "agent", Address: "10.0.0.12", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "ha-cluster",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	servers := parsed.Spec.Hosts.Servers()
	if len(servers) != 2 {
		t.Errorf("server count = %d, want 2", len(servers))
	}
	agents := parsed.Spec.Hosts.Agents()
	if len(agents) != 1 {
		t.Errorf("agent count = %d, want 1", len(agents))
	}
}

func TestClusterConfigGeneratePreservesSSHPort(t *testing.T) {
	nodes := []environment.Node{
		{Name: "node", Role: "server", Address: "10.0.0.10", SSHUser: "root", SSHPort: 2222},
	}
	cfg := ClusterConfig{
		Name:       "custom-port",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if len(parsed.Spec.Hosts) != 1 {
		t.Fatalf("host count = %d, want 1", len(parsed.Spec.Hosts))
	}
	host := parsed.Spec.Hosts[0]
	if host.SSH == nil {
		t.Fatal("ssh config is nil")
	}
	if host.SSH.Port != 2222 {
		t.Errorf("ssh.port = %d, want 2222", host.SSH.Port)
	}
	if host.SSH.User != "root" {
		t.Errorf("ssh.user = %q, want root", host.SSH.User)
	}
}

func TestClusterConfigGeneratePreservesPrivateAddress(t *testing.T) {
	nodes := []environment.Node{
		{Name: "node", Role: "server", Address: "172.16.0.5", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "private",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if parsed.Spec.Hosts[0].PrivateAddress != "172.16.0.5" {
		t.Errorf("privateAddress = %q, want 172.16.0.5", parsed.Spec.Hosts[0].PrivateAddress)
	}
}

func TestClusterConfigGenerateNameInMetadata(t *testing.T) {
	cfg := ClusterConfig{
		Name:       "my-test-cluster",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes: []environment.Node{
			{Name: "node", Role: "server", Address: "10.0.0.1", SSHUser: "ubuntu", SSHPort: 22},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	text := string(data)
	if !strings.Contains(text, "name: my-test-cluster") {
		t.Errorf("manifest missing cluster name:\n%s", text)
	}
}

func TestClusterConfigGenerateRejectsAgentOnlyCluster(t *testing.T) {
	nodes := []environment.Node{
		{Name: "agent-1", Role: "agent", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "no-server",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
	}

	_, err := cfg.Generate()
	if err == nil {
		t.Fatal("expected an error for agent-only cluster")
	}
	if !strings.Contains(err.Error(), "at least one server") {
		t.Errorf("error = %v, want it to mention 'at least one server'", err)
	}
}

func TestClusterConfigGenerateRejectsEmptyK3sVersion(t *testing.T) {
	cfg := ClusterConfig{
		Name:    "test",
		KeyPath: "/tmp/id_ed25519",
		Nodes: []environment.Node{
			{Name: "node", Role: "server", Address: "10.0.0.1", SSHUser: "ubuntu", SSHPort: 22},
		},
	}

	_, err := cfg.Generate()
	if err == nil {
		t.Fatal("expected an error for missing K3sVersion")
	}
	if !strings.Contains(err.Error(), "k3s version is required") {
		t.Errorf("error = %v, want 'k3s version is required'", err)
	}
}

func TestClusterConfigGenerateRejectsEmptyKeyPath(t *testing.T) {
	cfg := ClusterConfig{
		Name:       "test",
		K3sVersion: "v1.30.0+k3s1",
		Nodes: []environment.Node{
			{Name: "node", Role: "server", Address: "10.0.0.1", SSHUser: "ubuntu", SSHPort: 22},
		},
	}

	_, err := cfg.Generate()
	if err == nil {
		t.Fatal("expected an error for missing KeyPath")
	}
	if !strings.Contains(err.Error(), "SSH key path is required") {
		t.Errorf("error = %v, want 'SSH key path is required'", err)
	}
}

func TestClusterConfigGenerateRejectsNoNodes(t *testing.T) {
	cfg := ClusterConfig{
		Name:       "test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
	}

	_, err := cfg.Generate()
	if err == nil {
		t.Fatal("expected an error for missing nodes")
	}
	if !strings.Contains(err.Error(), "at least one node is required") {
		t.Errorf("error = %v, want 'at least one node is required'", err)
	}
}
