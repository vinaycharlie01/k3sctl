//go:build integration

package lxd_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nirantarasi/k3sctl/integration/environment"
	"github.com/nirantarasi/k3sctl/integration/lxd"
	"github.com/nirantarasi/k3sctl/integration/multinode"
	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

// TestLXDConfigReconciliation verifies the full config reconciliation lifecycle:
// disable a component via apply, verify it stops, re-enable it via apply, verify
// it starts again — all without upgrading K3s. This proves that apply is truly
// declarative for config changes.
//
// Scenario:
//  1. Install a single-server+agent cluster with defaults (Traefik enabled).
//  2. Verify Traefik is running.
//  3. Re-apply with disable.components: [traefik, servicelb].
//  4. Verify config.yaml has the disable field.
//  5. Verify Traefik is no longer running.
//  6. Idempotency — re-apply same disabled config, cluster stays healthy.
//  7. Re-apply with Traefik re-enabled (remove disable stanza).
//  8. Verify config.yaml no longer has the disable field.
//  9. Verify Traefik is running again.
//
// 10. Final idempotency — re-apply same enabled config.
func TestLXDConfigReconciliation(t *testing.T) {
	setup := newLXDTestSetup(t, "reconcile")

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
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	// --- Step 1: Initial install with defaults (Traefik enabled) ---
	t.Log("Step 1: initial install with Traefik enabled")
	if err := runner.Run(ctx); err != nil {
		t.Fatalf("initial apply failed: %v", err)
	}

	nodes, err := provider.Create(ctx)
	if err != nil {
		t.Fatalf("get node addresses: %v", err)
	}
	serverNode := findNodeByRole(nodes, environment.RoleServer)
	if serverNode == nil {
		t.Fatal("no server node found")
	}

	// --- Step 2: Verify Traefik is running ---
	t.Log("Step 2: verify Traefik is running")
	waitForPodState(t, sshRunner, ctx, serverNode.Address, "traefik", true)

	// --- Step 3: Disable Traefik and ServiceLB via apply ---
	t.Log("Step 3: re-apply with Traefik and ServiceLB disabled")
	runner.SetK3sOverrides(&multinode.K3sOverrides{
		Disable: &cluster.DisableConfig{
			Components: []string{"traefik", "servicelb"},
		},
	})

	disabledConfigPath, err := runner.WriteConfig(nodes)
	if err != nil {
		t.Fatalf("write disabled config: %v", err)
	}

	if err := runner.Apply(ctx, disabledConfigPath); err != nil {
		t.Fatalf("apply with disabled components failed: %v", err)
	}

	// --- Step 4: Verify config.yaml has the disable field ---
	t.Log("Step 4: verify config.yaml has disable field")
	verifyK3sConfigKey(t, sshRunner, ctx, serverNode.Address, "disable")

	// --- Step 5: Verify Traefik is no longer running ---
	t.Log("Step 5: verify Traefik is stopped")
	waitForPodState(t, sshRunner, ctx, serverNode.Address, "traefik", false)

	// --- Step 6: Idempotency — re-apply same disabled config ---
	t.Log("Step 6: idempotency check — re-apply same disabled config")
	if err := runner.Apply(ctx, disabledConfigPath); err != nil {
		t.Fatalf("idempotent re-apply (disabled) failed: %v", err)
	}

	if err := runner.Verify(ctx, nodes, disabledConfigPath); err != nil {
		t.Fatalf("verification after idempotent re-apply (disabled) failed: %v", err)
	}

	// Traefik should still be gone after the idempotent re-apply.
	waitForPodState(t, sshRunner, ctx, serverNode.Address, "traefik", false)

	// --- Step 7: Re-enable Traefik by removing the disable stanza ---
	t.Log("Step 7: re-apply with Traefik re-enabled (no disable stanza)")
	runner.SetK3sOverrides(nil)

	enabledConfigPath, err := runner.WriteConfig(nodes)
	if err != nil {
		t.Fatalf("write re-enabled config: %v", err)
	}

	if err := runner.Apply(ctx, enabledConfigPath); err != nil {
		t.Fatalf("apply to re-enable Traefik failed: %v", err)
	}

	// --- Step 8: Verify config.yaml no longer has the disable field ---
	t.Log("Step 8: verify config.yaml no longer has disable field")
	verifyK3sConfigKeyAbsent(t, sshRunner, ctx, serverNode.Address, "disable")

	// --- Step 9: Verify Traefik is running again ---
	t.Log("Step 9: verify Traefik is running again")
	waitForPodState(t, sshRunner, ctx, serverNode.Address, "traefik", true)

	// --- Step 10: Final idempotency — re-apply same enabled config ---
	t.Log("Step 10: final idempotency check — re-apply same enabled config")
	if err := runner.Apply(ctx, enabledConfigPath); err != nil {
		t.Fatalf("idempotent re-apply (enabled) failed: %v", err)
	}

	if err := runner.Verify(ctx, nodes, enabledConfigPath); err != nil {
		t.Fatalf("verification after idempotent re-apply (enabled) failed: %v", err)
	}

	waitForPodState(t, sshRunner, ctx, serverNode.Address, "traefik", true)
}

// waitForPodState waits for a pod name substring to be present (running=true) or
// absent (running=false) in the pod list. Polls for up to 5 minutes.
func waitForPodState(t *testing.T, ssh *multinode.SSHRunner, ctx context.Context, address, podSubstr string, wantRunning bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		result, err := ssh.Run(ctx, address,
			"sudo /usr/local/bin/k3s kubectl get pods -A --no-headers 2>/dev/null || true")
		if err != nil {
			t.Logf("get pods failed (will retry): %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		found := false
		for _, line := range strings.Split(result.Stdout, "\n") {
			if strings.Contains(line, podSubstr) && strings.Contains(line, "Running") {
				found = true
				break
			}
		}

		if found == wantRunning {
			return
		}

		time.Sleep(10 * time.Second)
	}

	if wantRunning {
		t.Fatalf("timed out waiting for pod %q to be running on %s", podSubstr, address)
	} else {
		t.Fatalf("timed out waiting for pod %q to stop on %s", podSubstr, address)
	}
}

// verifyK3sConfigKeyAbsent checks that a key is NOT present in /etc/rancher/k3s/config.yaml.
func verifyK3sConfigKeyAbsent(t *testing.T, ssh *multinode.SSHRunner, ctx context.Context, address, key string) {
	t.Helper()
	result, err := ssh.Run(ctx, address, "sudo cat /etc/rancher/k3s/config.yaml")
	if err != nil {
		t.Errorf("failed to read K3s config on %s: %v", address, err)
		return
	}
	if strings.Contains(result.Stdout, key) {
		t.Errorf("K3s config on %s still contains key %q after re-enable:\n%s", address, key, result.Stdout)
	}
}
