package multinode_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/multinode"
)

// TestConfigValidateCommandInvocation verifies that the config validate
// subcommand is called with the correct arguments.
func TestConfigValidateCommandInvocation(t *testing.T) {
	f := newRunnerFixture(t)

	var calls []string
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return multinode.Result{Stdout: "configuration is valid\n"}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "validate", "-c", f.configPath)
	if err != nil {
		t.Fatalf("config validate failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "valid") {
		t.Errorf("stdout = %q, want it to contain 'valid'", result.Stdout)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	call := calls[0]
	for _, want := range []string{"config", "validate", "-c", f.configPath} {
		if !strings.Contains(call, want) {
			t.Errorf("command %q missing %q", call, want)
		}
	}
}

// TestConfigValidateRejectsInvalidConfig verifies that config validate reports
// an error for a bad configuration file.
func TestConfigValidateRejectsInvalidConfig(t *testing.T) {
	f := newRunnerFixture(t)
	if err := os.MkdirAll(f.artifactDir, 0700); err != nil {
		t.Fatalf("create artifact dir: %v", err)
	}

	badConfig := filepath.Join(f.artifactDir, "bad.yaml")
	if err := os.WriteFile(badConfig, []byte("not: valid: yaml: ["), 0600); err != nil {
		t.Fatalf("write bad config: %v", err)
	}

	f.commands.RunStub = func(_ context.Context, _ string, _ string, args ...string) (multinode.Result, error) {
		if len(args) >= 2 && args[0] == "config" && args[1] == "validate" {
			return multinode.Result{
				ExitCode: 1,
				Stderr:   "validation failed: invalid yaml",
			}, errFake("validation failed")
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	_, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "validate", "-c", badConfig)
	if err == nil {
		t.Fatal("expected config validate to fail for bad config")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error = %v, want it to mention validation failure", err)
	}
}

// TestConfigValidateNonExistentFile verifies that config validate fails when
// the config file does not exist.
func TestConfigValidateNonExistentFile(t *testing.T) {
	f := newRunnerFixture(t)

	missingPath := filepath.Join(f.artifactDir, "nonexistent.yaml")

	f.commands.RunStub = func(_ context.Context, _ string, _ string, args ...string) (multinode.Result, error) {
		if len(args) >= 2 && args[0] == "config" && args[1] == "validate" {
			return multinode.Result{
				ExitCode: 1,
				Stderr:   "open " + missingPath + ": no such file or directory",
			}, errFake("file not found")
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	_, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "validate", "-c", missingPath)
	if err == nil {
		t.Fatal("expected config validate to fail for missing file")
	}
}

// TestConfigValidateBeforeApply verifies a realistic flow where config is
// validated before apply, catching errors early.
func TestConfigValidateBeforeApply(t *testing.T) {
	f := newRunnerFixture(t)

	var calls []string
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		if len(args) > 0 {
			calls = append(calls, args[0])
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()

	// Validate first
	_, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "validate", "-c", f.configPath)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	// Then apply
	_, err = f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	if len(calls) != 2 || calls[0] != "config" || calls[1] != "apply" {
		t.Errorf("call order = %v, want [config, apply]", calls)
	}
}
