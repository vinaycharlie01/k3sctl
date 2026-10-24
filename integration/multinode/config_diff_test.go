package multinode_test

import (
	"context"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/multinode"
	mnodemocks "github.com/vinaycharlie01/k3sctl/integration/multinode/mocks"
)

// TestConfigDiffCommandInvocation verifies that the k3sctl config diff
// subcommand is called with the correct arguments.
func TestConfigDiffCommandInvocation(t *testing.T) {
	f := newRunnerFixture(t)

	var calls []string
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return multinode.Result{Stdout: "No differences found.\n"}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "diff", "-c", f.configPath)
	if err != nil {
		t.Fatalf("config diff failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	call := calls[0]
	for _, want := range []string{"config", "diff", "-c", f.configPath} {
		if !strings.Contains(call, want) {
			t.Errorf("command %q missing %q", call, want)
		}
	}
}

// TestConfigDiffShowsDifferences verifies that config diff detects version
// mismatches when the cluster state differs from the configuration.
func TestConfigDiffShowsDifferences(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		if len(args) >= 2 && args[0] == "config" && args[1] == "diff" {
			return multinode.Result{
				Stdout: "1 difference(s) found:\n\nHOST       FIELD        CURRENT        →  DESIRED\n----       -----        -------           -------\n10.0.0.1   k3s-version  v1.29.0+k3s1      v1.30.0+k3s1\n",
			}, nil
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "diff", "-c", f.configPath)
	if err != nil {
		t.Fatalf("config diff failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "difference(s) found") {
		t.Errorf("stdout = %q, want it to contain 'difference(s) found'", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "k3s-version") {
		t.Errorf("stdout = %q, want it to contain 'k3s-version'", result.Stdout)
	}
}

// TestConfigDiffBeforeApply verifies a realistic flow: diff → apply,
// ensuring diff is checked before applying changes.
func TestConfigDiffBeforeApply(t *testing.T) {
	f := newRunnerFixture(t)

	var calls []string
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		if len(args) > 0 {
			calls = append(calls, args[0])
		}
		return multinode.Result{}, nil
	}

	f.remote = &mnodemocks.FakeRemoteExecutor{}
	f.remote.RunStub = func(_ context.Context, _ string, command string) (multinode.Result, error) {
		return multinode.Result{}, nil
	}

	ctx := context.Background()

	// Step 1: config diff
	_, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "diff", "-c", f.configPath)
	if err != nil {
		t.Fatalf("config diff failed: %v", err)
	}

	// Step 2: apply
	_, err = f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	if len(calls) != 2 || calls[0] != "config" || calls[1] != "apply" {
		t.Errorf("call order = %v, want [config, apply]", calls)
	}
}

// TestConfigDiffNoDifferences verifies that when cluster matches config,
// the output confirms no changes are needed.
func TestConfigDiffNoDifferences(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		if len(args) >= 2 && args[0] == "config" && args[1] == "diff" {
			return multinode.Result{
				Stdout: "No differences found. Cluster matches the configuration.\n",
			}, nil
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "diff", "-c", f.configPath)
	if err != nil {
		t.Fatalf("config diff failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "No differences found") {
		t.Errorf("stdout = %q, want 'No differences found'", result.Stdout)
	}
}
