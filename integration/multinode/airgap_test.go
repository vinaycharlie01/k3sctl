package multinode_test

import (
	"context"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/multinode"
)

// TestAirgapApplyUploadsBundle verifies that an apply with airgap config
// runs the air-gap image bundle upload step.
func TestAirgapApplyUploadsBundle(t *testing.T) {
	f := newRunnerFixture(t)

	var calls []string
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return multinode.Result{
			Stdout: "==> Running phase: Upload air-gap image bundle\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "Upload air-gap image bundle") {
		t.Errorf("stdout = %q, want it to contain 'Upload air-gap image bundle'", result.Stdout)
	}
}

// TestAirgapApplyDryRun verifies that dry-run mode does not actually upload
// the air-gap bundle but still shows the phase.
func TestAirgapApplyDryRun(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Upload air-gap image bundle\n[dry-run] upload air-gap bundle /tmp/images.tar.zst to /var/lib/rancher/k3s/agent/images/images.tar.zst\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "--dry-run", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply --dry-run failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "dry-run") {
		t.Errorf("stdout = %q, want it to contain 'dry-run'", result.Stdout)
	}
}

// TestAirgapSkippedWhenDisabled verifies that the air-gap upload phase is
// skipped when airgap is not configured.
func TestAirgapSkippedWhenDisabled(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Upload files\n==> Running phase: Install binaries\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if strings.Contains(result.Stdout, "Upload air-gap image bundle") {
		t.Errorf("stdout should not contain 'Upload air-gap image bundle' when airgap disabled")
	}
}
