package multinode

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
	v1beta1 "github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func testNodes() []environment.Node {
	return []environment.Node{
		{Name: "k3sctl-run-1-server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "k3sctl-run-1-agent1", Role: "agent", Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "k3sctl-run-1-agent2", Role: "agent", Address: "10.0.0.12", SSHUser: "ubuntu", SSHPort: 22},
	}
}

func validClusterConfig() ClusterConfig {
	return ClusterConfig{
		Name:       "k3sctl-run-1",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/k3sctl-run-1/id_ed25519",
		Nodes:      testNodes(),
	}
}

func TestClusterConfigGenerateProducesValidManifest(t *testing.T) {
	cfg := validClusterConfig()

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	text := string(data)
	for _, want := range []string{
		"apiVersion: k3sctl.k3sproject.io/v1beta1",
		"kind: Cluster",
		"name: k3sctl-run-1",
		"hostname: k3sctl-run-1-server",
		"version: v1.30.0+k3s1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("generated manifest missing %q:\n%s", want, text)
		}
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if parsed.APIVersion != v1beta1.APIVersion {
		t.Errorf("apiVersion = %q", parsed.APIVersion)
	}
	if parsed.Kind != "Cluster" {
		t.Errorf("kind = %q", parsed.Kind)
	}
	if parsed.Metadata.Name != cfg.Name {
		t.Errorf("metadata.name = %q", parsed.Metadata.Name)
	}
	if parsed.Spec == nil || parsed.Spec.K3s == nil {
		t.Fatal("spec.k3s is missing")
	}
	if parsed.Spec.K3s.Version != cfg.K3sVersion {
		t.Errorf("spec.k3s.version = %q", parsed.Spec.K3s.Version)
	}
	if len(parsed.Spec.Hosts) != len(cfg.Nodes) {
		t.Fatalf("spec.hosts length = %d, want %d", len(parsed.Spec.Hosts), len(cfg.Nodes))
	}

	for i, node := range cfg.Nodes {
		host := parsed.Spec.Hosts[i]
		if string(host.Role) != string(node.Role) {
			t.Errorf("host[%d].role = %q, want %q", i, host.Role, node.Role)
		}
		if host.PrivateAddress != node.Address {
			t.Errorf("host[%d].privateAddress = %q, want %q", i, host.PrivateAddress, node.Address)
		}
		if host.Hostname != node.Name {
			t.Errorf("host[%d].hostname = %q, want %q", i, host.Hostname, node.Name)
		}
		if host.SSH == nil {
			t.Fatalf("host[%d].ssh is missing", i)
		}
		if host.SSH.Address != node.Address {
			t.Errorf("host[%d].ssh.address = %q, want %q", i, host.SSH.Address, node.Address)
		}
		if host.SSH.User != node.SSHUser {
			t.Errorf("host[%d].ssh.user = %q, want %q", i, host.SSH.User, node.SSHUser)
		}
		if host.SSH.Port != node.SSHPort {
			t.Errorf("host[%d].ssh.port = %d, want %d", i, host.SSH.Port, node.SSHPort)
		}
		if host.SSH.KeyPath == nil || *host.SSH.KeyPath != cfg.KeyPath {
			t.Errorf("host[%d].ssh.keyPath does not point at the generated key", i)
		}
	}

	parsed.SetDefaults()
	if err := parsed.Validate(); err != nil {
		t.Fatalf("generated manifest does not validate: %v", err)
	}

	if roles := parsed.Spec.Hosts.Servers(); len(roles) != 1 {
		t.Errorf("server count = %d, want 1", len(roles))
	}
	if agents := parsed.Spec.Hosts.Agents(); len(agents) != 2 {
		t.Errorf("agent count = %d, want 2", len(agents))
	}
}

func TestClusterConfigGenerateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ClusterConfig)
		want   string
	}{
		{
			name:   "missing k3s version",
			mutate: func(c *ClusterConfig) { c.K3sVersion = "" },
			want:   "k3s version is required",
		},
		{
			name:   "missing key path",
			mutate: func(c *ClusterConfig) { c.KeyPath = "" },
			want:   "SSH key path is required",
		},
		{
			name:   "no nodes",
			mutate: func(c *ClusterConfig) { c.Nodes = nil },
			want:   "at least one node is required",
		},
		{
			name:   "unprefixed k3s version",
			mutate: func(c *ClusterConfig) { c.K3sVersion = "1.30.0+k3s1" },
			want:   "validate generated cluster config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validClusterConfig()
			tt.mutate(&cfg)

			_, err := cfg.Generate()
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestClusterConfigGenerateDefaultsRole(t *testing.T) {
	nodes := []environment.Node{
		{Name: "only", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "run",
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
		t.Fatalf("spec.hosts length = %d, want 1", len(parsed.Spec.Hosts))
	}
	if parsed.Spec.Hosts[0].Role != cluster.RoleServer {
		t.Errorf("role = %q, want %q", parsed.Spec.Hosts[0].Role, cluster.RoleServer)
	}
}
