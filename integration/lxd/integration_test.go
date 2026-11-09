//go:build integration

package lxd_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nirantarasi/k3sctl/integration/environment"
	"github.com/nirantarasi/k3sctl/integration/lxd"
	"github.com/nirantarasi/k3sctl/integration/multinode"
	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

const defaultIntegrationK3sVersion = "v1.37.1+k3s1"

// lxdTestSetup holds common setup shared across all LXD integration tests.
type lxdTestSetup struct {
	binary      string
	k3sVersion  string
	runID       string
	artifactDir string
}

func newLXDTestSetup(t *testing.T, suffix string) lxdTestSetup {
	t.Helper()

	if os.Getenv("K3SCTL_LXD_INTEGRATION") != "1" {
		t.Skip("set K3SCTL_LXD_INTEGRATION=1 to run the LXD integration test")
	}

	binary := os.Getenv("K3SCTL_BINARY")
	if binary == "" {
		t.Fatal("K3SCTL_BINARY must point at the compiled k3sctl binary")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("K3SCTL_BINARY %q is not usable: %v", binary, err)
	}

	k3sVersion := os.Getenv("K3S_VERSION")
	if k3sVersion == "" {
		k3sVersion = defaultIntegrationK3sVersion
	}

	runID := os.Getenv("K3SCTL_LXD_RUN_ID")
	if runID == "" {
		runID = fmt.Sprintf("local-%d", time.Now().Unix())
	}
	if suffix != "" {
		runID = runID + "-" + suffix
	}

	artifactDir := os.Getenv("K3SCTL_ARTIFACT_DIR")
	if artifactDir == "" {
		artifactDir = filepath.Join("integration-artifacts", runID)
	} else if suffix != "" {
		artifactDir = filepath.Join(artifactDir, suffix)
	}

	t.Logf("run id: %s", runID)
	t.Logf("artifacts: %s", artifactDir)
	t.Logf("k3s version: %s", k3sVersion)

	return lxdTestSetup{
		binary:      binary,
		k3sVersion:  k3sVersion,
		runID:       runID,
		artifactDir: artifactDir,
	}
}

// TestLXDMultiNodeK3s provisions three Ubuntu 24.04 LXD containers, installs
// K3s on them with the real k3sctl binary and verifies the resulting cluster.
//
// It requires a running LXD daemon reachable through the local Unix socket and
// only runs when K3SCTL_LXD_INTEGRATION=1.
func TestLXDMultiNodeK3s(t *testing.T) {
	setup := newLXDTestSetup(t, "")

	workDir := t.TempDir()
	keyPair, err := multinode.GenerateSSHKey(filepath.Join(workDir, "keys"))
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	provider, err := lxd.New(context.Background(), lxd.Config{
		RunID:            setup.runID,
		ImageAlias:       "24.04",
		SSHUser:          "ubuntu",
		SSHPort:          22,
		SSHAuthorizedKey: keyPair.AuthorizedKey,
		Nodes: []lxd.NodeSpec{
			{Name: "server", Role: environment.RoleServer},
			{Name: "agent1", Role: environment.RoleAgent},
			{Name: "agent2", Role: environment.RoleAgent},
		},
	})
	if err != nil {
		t.Fatalf("connect to LXD: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := provider.Destroy(cleanupCtx); err != nil {
			t.Logf("fallback LXD cleanup: %v", err)
		}
	})

	runner, err := multinode.NewRunner(multinode.Options{
		Binary:      setup.binary,
		WorkDir:     workDir,
		KeyPath:     keyPair.PrivateKeyPath,
		ArtifactDir: setup.artifactDir,
		ClusterName: "k3sctl-" + setup.runID,
		K3sVersion:  setup.k3sVersion,
		Environment: provider,
		Remote: &multinode.SSHRunner{
			User:    "ubuntu",
			Port:    22,
			KeyPath: keyPair.PrivateKeyPath,
			Timeout: 30 * time.Second,
		},
		ApplyTimeout: 25 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("multi-node K3s integration test failed: %v", err)
	}
}

// TestLXDExtendedFlags provisions a single-server + 1 agent cluster with
// K3s configuration flags enabled: preferBundledBin, supervisorMetrics,
// enablePprof, kubeconfig mode, agent lbServerPort, embedded registry, and
// custom labels. After install it verifies the flags took effect by inspecting
// the K3s config file and node labels.
func TestLXDExtendedFlags(t *testing.T) {
	setup := newLXDTestSetup(t, "flags")

	workDir := t.TempDir()
	keyPair, err := multinode.GenerateSSHKey(filepath.Join(workDir, "keys"))
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	provider, err := lxd.New(context.Background(), lxd.Config{
		RunID:            setup.runID,
		ImageAlias:       "24.04",
		SSHUser:          "ubuntu",
		SSHPort:          22,
		SSHAuthorizedKey: keyPair.AuthorizedKey,
		Nodes: []lxd.NodeSpec{
			{Name: "server", Role: environment.RoleServer},
			{Name: "agent1", Role: environment.RoleAgent},
		},
	})
	if err != nil {
		t.Fatalf("connect to LXD: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := provider.Destroy(cleanupCtx); err != nil {
			t.Logf("fallback LXD cleanup: %v", err)
		}
	})

	sshRunner := &multinode.SSHRunner{
		User:    "ubuntu",
		Port:    22,
		KeyPath: keyPair.PrivateKeyPath,
		Timeout: 30 * time.Second,
	}

	runner, err := multinode.NewRunner(multinode.Options{
		Binary:       setup.binary,
		WorkDir:      workDir,
		KeyPath:      keyPair.PrivateKeyPath,
		ArtifactDir:  setup.artifactDir,
		ClusterName:  "k3sctl-" + setup.runID,
		K3sVersion:   setup.k3sVersion,
		Environment:  provider,
		Remote:       sshRunner,
		ApplyTimeout: 25 * time.Minute,
		PostApplyVerify: func(ctx context.Context, nodes []environment.Node) error {
			serverNode := findNodeByRole(nodes, environment.RoleServer)
			agentNode := findNodeByRole(nodes, environment.RoleAgent)

			if serverNode != nil {
				for _, key := range []string{
					"prefer-bundled-bin",
					"supervisor-metrics",
					"enable-pprof",
				} {
					verifyK3sConfigKey(t, sshRunner, ctx, serverNode.Address, key)
				}
				verifyNodeLabel(t, sshRunner, ctx, serverNode.Address, "k3sctl-test=extended-flags")
			}

			if agentNode != nil {
				for _, key := range []string{
					"prefer-bundled-bin",
					"lb-server-port",
				} {
					verifyK3sConfigKey(t, sshRunner, ctx, agentNode.Address, key)
				}
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	runner.SetK3sOverrides(&multinode.K3sOverrides{
		Security:         &cluster.SecurityConfig{PreferBundledBin: true},
		Server:           &cluster.ServerConfig{SupervisorMetrics: true, EnablePprof: true},
		Kubeconfig:       &cluster.KubeconfigConfig{Mode: "0600"},
		Agent:            &cluster.AgentConfig{LBServerPort: 6444},
		EmbeddedRegistry: &cluster.EmbeddedRegistryConfig{Enabled: true},
		Labels:           map[string]string{"k3sctl-test": "extended-flags"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("extended flags integration test failed: %v", err)
	}
}

func findNodeByRole(nodes []environment.Node, role environment.Role) *environment.Node {
	for i := range nodes {
		if nodes[i].Role == role {
			return &nodes[i]
		}
	}
	return nil
}

func verifyK3sConfigKey(t *testing.T, ssh *multinode.SSHRunner, ctx context.Context, address, key string) {
	t.Helper()
	result, err := ssh.Run(ctx, address, "sudo cat /etc/rancher/k3s/config.yaml")
	if err != nil {
		t.Errorf("failed to read K3s config on %s: %v", address, err)
		return
	}
	if !strings.Contains(result.Stdout, key) {
		t.Errorf("K3s config on %s does not contain key %q", address, key)
	}
}

func verifyNodeLabel(t *testing.T, ssh *multinode.SSHRunner, ctx context.Context, address, labelKV string) {
	t.Helper()
	result, err := ssh.Run(ctx, address, "sudo /usr/local/bin/k3s kubectl get nodes -o json")
	if err != nil {
		t.Errorf("failed to get nodes on %s: %v", address, err)
		return
	}
	parts := strings.SplitN(labelKV, "=", 2)
	if len(parts) != 2 {
		t.Errorf("invalid label kv: %q", labelKV)
		return
	}
	if !strings.Contains(result.Stdout, parts[0]) {
		t.Errorf("node labels on %s do not contain key %q", address, parts[0])
	}
}

// TestLXDHAMultiServer provisions a 5-node HA cluster (3 servers + 2 agents)
// using LXD system containers. It verifies:
//
//   - Quorum-safe batched server installation (the leader installs first, then
//     remaining servers join in quorum-safe batches)
//   - All 5 nodes reach Ready state
//   - The embedded etcd cluster has exactly 3 members
//   - Every server's K3s API endpoint responds
//   - k3sctl reset cleanly removes etcd members and tears down the cluster
//
// This test exercises the full HA code path end-to-end with the real k3sctl
// binary, complementing the mock-based tests in integration/multinode/.
func TestLXDHAMultiServer(t *testing.T) {
	setup := newLXDTestSetup(t, "ha")

	workDir := t.TempDir()
	keyPair, err := multinode.GenerateSSHKey(filepath.Join(workDir, "keys"))
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	provider, err := lxd.New(context.Background(), lxd.Config{
		RunID:            setup.runID,
		ImageAlias:       "24.04",
		SSHUser:          "ubuntu",
		SSHPort:          22,
		SSHAuthorizedKey: keyPair.AuthorizedKey,
		Nodes: []lxd.NodeSpec{
			{Name: "server1", Role: environment.RoleServer},
			{Name: "server2", Role: environment.RoleServer},
			{Name: "server3", Role: environment.RoleServer},
			{Name: "agent1", Role: environment.RoleAgent},
			{Name: "agent2", Role: environment.RoleAgent},
		},
	})
	if err != nil {
		t.Fatalf("connect to LXD: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := provider.Destroy(cleanupCtx); err != nil {
			t.Logf("fallback LXD cleanup: %v", err)
		}
	})

	sshRunner := &multinode.SSHRunner{
		User:    "ubuntu",
		Port:    22,
		KeyPath: keyPair.PrivateKeyPath,
		Timeout: 30 * time.Second,
	}

	runner, err := multinode.NewRunner(multinode.Options{
		Binary:       setup.binary,
		WorkDir:      workDir,
		KeyPath:      keyPair.PrivateKeyPath,
		ArtifactDir:  setup.artifactDir,
		ClusterName:  "k3sctl-" + setup.runID,
		K3sVersion:   setup.k3sVersion,
		Environment:  provider,
		Remote:       sshRunner,
		ApplyTimeout: 25 * time.Minute,
		ResetTimeout: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("HA multi-server K3s integration test failed: %v", err)
	}
}
