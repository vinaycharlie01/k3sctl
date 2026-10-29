package multinode_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/multinode"
	mnodemocks "github.com/vinaycharlie01/k3sctl/integration/multinode/mocks"
)

// TestRestoreCommandInvocation verifies that the k3sctl binary is called with
// the correct arguments for a restore operation. It uses the same mock fixture
// as the runner tests but invokes the CLI directly through FakeCommandRunner.
func TestRestoreCommandInvocation(t *testing.T) {
	f := newRunnerFixture(t)
	if err := os.MkdirAll(f.artifactDir, 0700); err != nil {
		t.Fatalf("create artifact dir: %v", err)
	}

	// Create a fake snapshot file that the restore command can stat.
	snapshotPath := filepath.Join(f.artifactDir, "etcd-snapshot.db")
	if err := os.WriteFile(snapshotPath, []byte("fake-snapshot"), 0600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	// Track which commands are run.
	var calls []string
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		return multinode.Result{}, nil
	}

	// Simulate: k3sctl restore --from <snapshot> -c <config> --force
	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary,
		"restore", "--from", snapshotPath, "-c", f.configPath, "--force")
	if err != nil {
		t.Fatalf("restore command failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("restore exit code = %d, want 0", result.ExitCode)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 command call, got %d", len(calls))
	}

	call := calls[0]
	for _, want := range []string{"restore", "--from", snapshotPath, "-c", f.configPath, "--force"} {
		if !strings.Contains(call, want) {
			t.Errorf("command %q missing argument %q", call, want)
		}
	}
}

// TestRestoreCommandFailureReportsError verifies that a failed restore
// propagates the error correctly.
func TestRestoreCommandFailureReportsError(t *testing.T) {
	f := newRunnerFixture(t)

	restoreErr := errFake("restore failed: etcd data corrupted")
	f.commands.RunStub = func(_ context.Context, _ string, _ string, args ...string) (multinode.Result, error) {
		if len(args) > 0 && args[0] == "restore" {
			return multinode.Result{ExitCode: 1, Stderr: "etcd data corrupted"}, restoreErr
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	_, err := f.commands.Run(ctx, f.workDir, f.options.Binary,
		"restore", "--from", "/tmp/snap.db", "-c", f.configPath, "--force")
	if err == nil {
		t.Fatal("expected restore to fail")
	}
	if !strings.Contains(err.Error(), "etcd data corrupted") {
		t.Errorf("error = %v, want it to mention the restore failure", err)
	}
}

// TestRestoreFullCycle exercises a realistic restore flow: apply → backup → restore → verify.
// Each step is tracked through FakeCommandRunner to assert correct ordering.
func TestRestoreFullCycle(t *testing.T) {
	f := newRunnerFixture(t)
	if err := os.MkdirAll(f.artifactDir, 0700); err != nil {
		t.Fatalf("create artifact dir: %v", err)
	}

	snapshotPath := filepath.Join(f.artifactDir, "etcd-snapshot.db")
	if err := os.WriteFile(snapshotPath, []byte("fake-snapshot"), 0600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	var calls []string
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		cmd := args[0]
		calls = append(calls, cmd)
		switch cmd {
		case "config":
			if len(args) > 1 && args[1] == "validate" {
				return multinode.Result{Stdout: "configuration is valid\n"}, nil
			}
		case "kubeconfig":
			return multinode.Result{Stdout: "server: https://10.0.0.10:6443\n"}, nil
		}
		return multinode.Result{}, nil
	}

	f.remote = &mnodemocks.FakeRemoteExecutor{}
	f.remote.RunStub = func(_ context.Context, _ string, command string) (multinode.Result, error) {
		switch command {
		case "ps -p 1 -o comm=":
			return multinode.Result{Stdout: "systemd\n"}, nil
		case "sudo /usr/local/bin/k3s --version":
			return multinode.Result{Stdout: "k3s version " + runnerK3sVersion + " (abcdef)\n"}, nil
		case "sudo /usr/local/bin/k3s kubectl get nodes -o json":
			return multinode.Result{Stdout: readyNodesJSON}, nil
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()

	// Step 1: apply
	_, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Step 2: backup
	_, err = f.commands.Run(ctx, f.workDir, f.options.Binary, "backup", "-c", f.configPath)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Step 3: restore
	_, err = f.commands.Run(ctx, f.workDir, f.options.Binary,
		"restore", "--from", snapshotPath, "-c", f.configPath, "--force")
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Step 4: config validate (post-restore sanity check)
	_, err = f.commands.Run(ctx, f.workDir, f.options.Binary, "config", "validate", "-c", f.configPath)
	if err != nil {
		t.Fatalf("config validate failed: %v", err)
	}

	// Verify ordering: apply → backup → restore → config
	wantOrder := []string{"apply", "backup", "restore", "config"}
	if len(calls) < len(wantOrder) {
		t.Fatalf("expected at least %d calls, got %d: %v", len(wantOrder), len(calls), calls)
	}
	for i, want := range wantOrder {
		if calls[i] != want {
			t.Errorf("call[%d] = %q, want %q (full sequence: %v)", i, calls[i], want, calls)
		}
	}
}

// errFake is a simple error type for test assertions.
type errFake string

func (e errFake) Error() string { return string(e) }
