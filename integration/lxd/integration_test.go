//go:build integration

package lxd_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
	"github.com/vinaycharlie01/k3sctl/integration/lxd"
	"github.com/vinaycharlie01/k3sctl/integration/multinode"
)

const defaultIntegrationK3sVersion = "v1.37.1+k3s1"

// TestLXDMultiNodeK3s provisions three Ubuntu 24.04 LXD containers, installs
// K3s on them with the real k3sctl binary and verifies the resulting cluster.
//
// It requires a running LXD daemon reachable through the local Unix socket and
// only runs when K3SCTL_LXD_INTEGRATION=1.
func TestLXDMultiNodeK3s(t *testing.T) {
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

	artifactDir := os.Getenv("K3SCTL_ARTIFACT_DIR")
	if artifactDir == "" {
		artifactDir = filepath.Join("integration-artifacts", runID)
	}

	t.Logf("run id: %s", runID)
	t.Logf("artifacts: %s", artifactDir)
	t.Logf("k3s version: %s", k3sVersion)

	workDir := t.TempDir()
	keyPair, err := multinode.GenerateSSHKey(filepath.Join(workDir, "keys"))
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	provider, err := lxd.New(context.Background(), lxd.Config{
		RunID:            runID,
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

	// The runner destroys the environment itself. This is a last-resort cleanup
	// so a panic or an early failure cannot leak containers.
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := provider.Destroy(cleanupCtx); err != nil {
			t.Logf("fallback LXD cleanup: %v", err)
		}
	})

	runner, err := multinode.NewRunner(multinode.Options{
		Binary:      binary,
		WorkDir:     workDir,
		KeyPath:     keyPair.PrivateKeyPath,
		ArtifactDir: artifactDir,
		ClusterName: "k3sctl-" + runID,
		K3sVersion:  k3sVersion,
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
