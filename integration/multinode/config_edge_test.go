package multinode

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/nirantarasi/k3sctl/integration/environment"
	v1beta1 "github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
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

func TestClusterConfigGenerateWithNodeID(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "agent1", Role: "agent", Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "nodeid-test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s:        &K3sOverrides{WithNodeID: true},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	for i, host := range parsed.Spec.Hosts {
		if !host.WithNodeID {
			t.Errorf("host[%d].withNodeID = false, want true", i)
		}
	}
}

func TestClusterConfigGenerateSecurityOverrides(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "security-test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s: &K3sOverrides{
			Security: &cluster.SecurityConfig{
				PreferBundledBin: true,
			},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if parsed.Spec.K3s.Security == nil {
		t.Fatal("security is nil")
	}
	if !parsed.Spec.K3s.Security.PreferBundledBin {
		t.Error("preferBundledBin = false, want true")
	}
}

func TestClusterConfigGenerateServerOverrides(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "server-test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s: &K3sOverrides{
			Server: &cluster.ServerConfig{
				SupervisorMetrics: true,
				EnablePprof:       true,
			},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if parsed.Spec.K3s.Server == nil {
		t.Fatal("server is nil")
	}
	if !parsed.Spec.K3s.Server.SupervisorMetrics {
		t.Error("supervisorMetrics = false, want true")
	}
	if !parsed.Spec.K3s.Server.EnablePprof {
		t.Error("enablePprof = false, want true")
	}
}

func TestClusterConfigGenerateKubeconfigOverrides(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "kc-test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s: &K3sOverrides{
			Kubeconfig: &cluster.KubeconfigConfig{
				Mode: "0600",
				Path: "/tmp/kubeconfig.yaml",
			},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if parsed.Spec.K3s.Kubeconfig == nil {
		t.Fatal("kubeconfig is nil")
	}
	if parsed.Spec.K3s.Kubeconfig.Mode != "0600" {
		t.Errorf("kubeconfig.mode = %q, want 0600", parsed.Spec.K3s.Kubeconfig.Mode)
	}
	if parsed.Spec.K3s.Kubeconfig.Path != "/tmp/kubeconfig.yaml" {
		t.Errorf("kubeconfig.path = %q, want /tmp/kubeconfig.yaml", parsed.Spec.K3s.Kubeconfig.Path)
	}
}

func TestClusterConfigGenerateRuntimeOverrides(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "runtime-test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s: &K3sOverrides{
			Runtime: &cluster.RuntimeConfig{
				ContainerRuntimeEndpoint: "unix:///run/containerd/containerd.sock",
				ImageServiceEndpoint:     "unix:///run/containerd/containerd.sock",
			},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if parsed.Spec.K3s.Runtime == nil {
		t.Fatal("runtime is nil")
	}
	if parsed.Spec.K3s.Runtime.ContainerRuntimeEndpoint != "unix:///run/containerd/containerd.sock" {
		t.Errorf("containerRuntimeEndpoint = %q", parsed.Spec.K3s.Runtime.ContainerRuntimeEndpoint)
	}
	if parsed.Spec.K3s.Runtime.ImageServiceEndpoint != "unix:///run/containerd/containerd.sock" {
		t.Errorf("imageServiceEndpoint = %q", parsed.Spec.K3s.Runtime.ImageServiceEndpoint)
	}
}

func TestClusterConfigGenerateAgentOverrides(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "agent1", Role: "agent", Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "agent-test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s: &K3sOverrides{
			Agent: &cluster.AgentConfig{
				LBServerPort: 6444,
			},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if parsed.Spec.K3s.Agent == nil {
		t.Fatal("agent is nil")
	}
	if parsed.Spec.K3s.Agent.LBServerPort != 6444 {
		t.Errorf("lbServerPort = %d, want 6444", parsed.Spec.K3s.Agent.LBServerPort)
	}
}

func TestClusterConfigGenerateEmbeddedRegistryOverrides(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "registry-test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s: &K3sOverrides{
			EmbeddedRegistry: &cluster.EmbeddedRegistryConfig{
				Enabled: true,
			},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	if parsed.Spec.K3s.EmbeddedRegistry == nil {
		t.Fatal("embeddedRegistry is nil")
	}
	if !parsed.Spec.K3s.EmbeddedRegistry.Enabled {
		t.Error("enabled = false, want true")
	}
}

func TestClusterConfigGenerateLabelsOverrides(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "agent1", Role: "agent", Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "labels-test",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s: &K3sOverrides{
			Labels: map[string]string{
				"env": "integration",
			},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	for i, host := range parsed.Spec.Hosts {
		if host.Labels == nil || host.Labels["env"] != "integration" {
			t.Errorf("host[%d].labels[env] = %q, want integration", i, host.Labels["env"])
		}
	}
}

func TestClusterConfigGenerateAllOverridesCombined(t *testing.T) {
	nodes := []environment.Node{
		{Name: "server", Role: "server", Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "agent1", Role: "agent", Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
	}
	cfg := ClusterConfig{
		Name:       "all-flags",
		K3sVersion: "v1.30.0+k3s1",
		KeyPath:    "/tmp/id_ed25519",
		Nodes:      nodes,
		K3s: &K3sOverrides{
			WithNodeID:       true,
			Security:         &cluster.SecurityConfig{PreferBundledBin: true},
			Server:           &cluster.ServerConfig{SupervisorMetrics: true, EnablePprof: true},
			Kubeconfig:       &cluster.KubeconfigConfig{Mode: "0600"},
			Agent:            &cluster.AgentConfig{LBServerPort: 6444},
			EmbeddedRegistry: &cluster.EmbeddedRegistryConfig{Enabled: true},
			Labels:           map[string]string{"k3sctl-test": "true"},
		},
	}

	data, err := cfg.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed v1beta1.Cluster
	if err := yaml.UnmarshalStrict(data, &parsed); err != nil {
		t.Fatalf("UnmarshalStrict() error = %v", err)
	}

	parsed.SetDefaults()
	if err := parsed.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !parsed.Spec.Hosts[0].WithNodeID {
		t.Error("withNodeID not set on server")
	}
	if parsed.Spec.K3s.Security == nil || !parsed.Spec.K3s.Security.PreferBundledBin {
		t.Error("preferBundledBin not set")
	}
	if parsed.Spec.K3s.Server == nil || !parsed.Spec.K3s.Server.SupervisorMetrics {
		t.Error("supervisorMetrics not set")
	}
	if parsed.Spec.K3s.Kubeconfig == nil || parsed.Spec.K3s.Kubeconfig.Mode != "0600" {
		t.Error("kubeconfig.mode not set")
	}
	if parsed.Spec.K3s.Agent == nil || parsed.Spec.K3s.Agent.LBServerPort != 6444 {
		t.Error("agent.lbServerPort not set")
	}
	if parsed.Spec.K3s.EmbeddedRegistry == nil || !parsed.Spec.K3s.EmbeddedRegistry.Enabled {
		t.Error("embeddedRegistry not set")
	}
	if parsed.Spec.Hosts[0].Labels["k3sctl-test"] != "true" {
		t.Error("labels not set")
	}
}
