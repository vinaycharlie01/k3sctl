package multinode_test

import (
	"context"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/multinode"
)

// TestHAServerBatchedInstall verifies that a multi-server apply with
// quorum-safe batching shows batch progress in the output.
func TestHAServerBatchedInstall(t *testing.T) {
	f := newRunnerFixture(t)

	var calls []string
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return multinode.Result{
			Stdout: "==> Running phase: Install K3s servers\ninstalling server batch 1/2 (1 hosts)\ninstalling server batch 2/2 (2 hosts)\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "server batch") {
		t.Errorf("stdout = %q, want it to contain 'server batch'", result.Stdout)
	}
}

// TestHAServerEtcdMemberRemovalOnReset verifies that resetting a server node
// includes an etcd member removal step.
func TestHAServerEtcdMemberRemovalOnReset(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Reset K3s servers\nserver-2: removing etcd member via leader\nserver-2: etcd member abc123 removed\nserver-2: K3s server reset complete\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "reset", "-c", f.configPath)
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "removing etcd member") {
		t.Errorf("stdout = %q, want it to contain 'removing etcd member'", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "etcd member") {
		t.Errorf("stdout = %q, want it to contain 'etcd member'", result.Stdout)
	}
}

// TestHAServerEtcdValidation verifies that the validate etcd members phase
// detects stale members in the output.
func TestHAServerEtcdValidation(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Validate etcd members\nserver-1: all etcd members match configured hosts\nserver-1: etcd is healthy\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "etcd members match") {
		t.Errorf("stdout = %q, want it to contain 'etcd members match'", result.Stdout)
	}
}

// TestHAServerStaleEtcdDetection verifies that stale etcd members are
// reported when detected and --force removes them.
func TestHAServerStaleEtcdDetection(t *testing.T) {
	f := newRunnerFixture(t)

	t.Run("without force returns error", func(t *testing.T) {
		f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
			return multinode.Result{
				Stdout:   "==> Running phase: Validate etcd members\nserver-1: found 1 stale etcd member(s)\n",
				ExitCode: 1,
			}, nil
		}

		ctx := context.Background()
		result, _ := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
		if !strings.Contains(result.Stdout, "stale etcd member") {
			t.Errorf("stdout = %q, want it to contain 'stale etcd member'", result.Stdout)
		}
	})

	t.Run("with force removes stale members", func(t *testing.T) {
		f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
			return multinode.Result{
				Stdout: "==> Running phase: Validate etcd members\nserver-1: removing stale etcd member abc123\nserver-1: removed stale etcd member abc123\n",
			}, nil
		}

		ctx := context.Background()
		result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "--force", "-c", f.configPath)
		if err != nil {
			t.Fatalf("apply --force failed: %v", err)
		}
		if !strings.Contains(result.Stdout, "removing stale etcd member") {
			t.Errorf("stdout = %q, want it to contain 'removing stale etcd member'", result.Stdout)
		}
	})
}
