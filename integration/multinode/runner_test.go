package multinode_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
	envmocks "github.com/vinaycharlie01/k3sctl/integration/environment/mocks"
	"github.com/vinaycharlie01/k3sctl/integration/multinode"
	mnodemocks "github.com/vinaycharlie01/k3sctl/integration/multinode/mocks"
)

const (
	runnerK3sVersion = "v1.30.0+k3s1"

	readyNodesJSON = `{"items":[
{"metadata":{"name":"run-server"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"run-agent1"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"run-agent2"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}
]}`

	notReadyNodesJSON = `{"items":[
{"metadata":{"name":"run-server"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"run-agent1"},"status":{"conditions":[{"type":"Ready","status":"False"}]}},
{"metadata":{"name":"run-agent2"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}
]}`

)

func runnerNodes() []environment.Node {
	return []environment.Node{
		{Name: "run-server", Role: environment.RoleServer, Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "run-agent1", Role: environment.RoleAgent, Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "run-agent2", Role: environment.RoleAgent, Address: "10.0.0.12", SSHUser: "ubuntu", SSHPort: 22},
	}
}

type runnerFixture struct {
	env         *envmocks.FakeEnvironment
	remote      *mnodemocks.FakeRemoteExecutor
	commands    *mnodemocks.FakeCommandRunner
	options     multinode.Options
	workDir     string
	artifactDir string
	configPath  string
}

func newRunnerFixture(t *testing.T) *runnerFixture {
	t.Helper()

	base := t.TempDir()
	workDir := filepath.Join(base, "work")
	artifactDir := filepath.Join(base, "artifacts")

	f := &runnerFixture{
		workDir:     workDir,
		artifactDir: artifactDir,
		configPath:  filepath.Join(artifactDir, "k3sctl.yaml"),
	}

	f.env = &envmocks.FakeEnvironment{}
	f.env.CreateReturns(runnerNodes(), nil)

	f.remote = &mnodemocks.FakeRemoteExecutor{}
	f.remote.RunStub = func(_ context.Context, _ string, command string) (multinode.Result, error) {
		switch command {
		case "ps -p 1 -o comm=":
			return multinode.Result{Stdout: "systemd\n"}, nil
		case "sudo /usr/local/bin/k3s --version":
			return multinode.Result{Stdout: "k3s version " + runnerK3sVersion + " (abcdef)\n"}, nil
		case "sudo /usr/local/bin/k3s kubectl get nodes -o json":
			return multinode.Result{Stdout: readyNodesJSON}, nil
		default:
			return multinode.Result{}, nil
		}
	}

	f.commands = &mnodemocks.FakeCommandRunner{}
	f.commands.RunStub = func(_ context.Context, _ string, _ string, args ...string) (multinode.Result, error) {
		switch {
		case len(args) >= 1 && args[0] == "kubeconfig":
			return multinode.Result{Stdout: "server: " + args[len(args)-1] + "\n"}, nil
		default:
			return multinode.Result{}, nil
		}
	}

	f.options = multinode.Options{
		Binary:      "/usr/local/bin/k3sctl",
		WorkDir:     workDir,
		KeyPath:     filepath.Join(workDir, "id_ed25519"),
		ArtifactDir: artifactDir,
		ClusterName: "run",
		K3sVersion:  runnerK3sVersion,
		Environment: f.env,
		Remote:      f.remote,
		Commands:    f.commands,
		APICheck:    func(context.Context, string) error { return nil },
	}

	return f
}

func TestRunnerRunInstallsAndVerifiesCluster(t *testing.T) {
	f := newRunnerFixture(t)

	runner, err := multinode.NewRunner(f.options)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := f.env.CreateCallCount(); got != 1 {
		t.Errorf("CreateCallCount() = %d, want 1", got)
	}
	if _, nodes := f.env.BootstrapArgsForCall(0); len(nodes) != 3 {
		t.Errorf("Bootstrap() received %d nodes, want 3", len(nodes))
	}
	if got := f.env.DestroyCallCount(); got != 1 {
		t.Errorf("DestroyCallCount() = %d, want 1", got)
	}

	if got := f.commands.RunCallCount(); got != 3 {
		t.Fatalf("RunCallCount() = %d, want 3 (apply, kubeconfig, reset)", got)
	}

	assertCommand := func(index int, wantArgs ...string) {
		t.Helper()
		_, dir, name, args := f.commands.RunArgsForCall(index)
		if name != f.options.Binary {
			t.Errorf("call %d binary = %q, want %q", index, name, f.options.Binary)
		}
		if dir != f.workDir {
			t.Errorf("call %d dir = %q, want %q", index, dir, f.workDir)
		}
		if strings.Join(args, " ") != strings.Join(wantArgs, " ") {
			t.Errorf("call %d args = %v, want %v", index, args, wantArgs)
		}
	}

	assertCommand(0, "--debug", "apply", "-c", f.configPath)
	assertCommand(1, "kubeconfig", "-c", f.configPath, "--api-address", "https://10.0.0.10:6443")
	assertCommand(2, "reset", "-c", f.configPath, "--force")

	for _, name := range []string{
		"k3sctl.yaml",
		"apply.log",
		"k3s-version.log",
		"nodes.json",
		"kubeconfig.yaml",
		"kubeconfig.log",
		"reset.log",
	} {
		if _, err := os.Stat(filepath.Join(f.artifactDir, name)); err != nil {
			t.Errorf("artifact %s missing: %v", name, err)
		}
	}

	if _, err := os.Stat(f.workDir); !os.IsNotExist(err) {
		t.Errorf("work directory still present after cleanup: err = %v", err)
	}

	var sawPing, sawServerService, sawAgentService bool
	for i := 0; i < f.remote.RunCallCount(); i++ {
		_, address, command := f.remote.RunArgsForCall(i)
		switch command {
		case "ping -c 1 -W 3 10.0.0.10 >/dev/null":
			sawPing = true
		case "systemctl is-active --quiet k3s":
			if address != "10.0.0.10" {
				t.Errorf("k3s service checked on %q, want the server", address)
			}
			sawServerService = true
		case "systemctl is-active --quiet k3s-agent":
			sawAgentService = true
		}
	}
	if !sawPing {
		t.Error("agents were not checked for connectivity to the server")
	}
	if !sawServerService {
		t.Error("k3s service was not verified")
	}
	if !sawAgentService {
		t.Error("k3s-agent service was not verified")
	}
}

func TestRunnerRunCollectsDiagnosticsAndCleansUpAfterApplyFailure(t *testing.T) {
	f := newRunnerFixture(t)
	applyErr := errors.New("apply failed")
	f.commands.RunStub = func(_ context.Context, _ string, _ string, args ...string) (multinode.Result, error) {
		if len(args) > 1 && args[1] == "apply" {
			return multinode.Result{Args: args, Stderr: "k3sctl: boom", ExitCode: 1}, applyErr
		}
		return multinode.Result{Args: args}, nil
	}

	runner, err := multinode.NewRunner(f.options)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if err := runner.Run(context.Background()); err == nil {
		t.Fatal("expected Run() to fail")
	} else {
		if !strings.Contains(err.Error(), "k3sctl apply") {
			t.Errorf("error = %v, want an apply failure", err)
		}
		if !errors.Is(err, applyErr) {
			t.Errorf("error = %v, want it to preserve the original failure", err)
		}
	}

	diagnostics, err := os.ReadFile(filepath.Join(f.artifactDir, "diagnostics.log"))
	if err != nil {
		t.Fatalf("read diagnostics.log: %v", err)
	}
	if !strings.Contains(string(diagnostics), "=== run-server (10.0.0.10) ===") {
		t.Errorf("diagnostics.log does not contain node diagnostics:\n%s", diagnostics)
	}

	resetCalled := false
	for i := 0; i < f.commands.RunCallCount(); i++ {
		_, _, _, args := f.commands.RunArgsForCall(i)
		if len(args) > 0 && args[0] == "reset" {
			resetCalled = true
		}
	}
	if !resetCalled {
		t.Error("k3sctl reset was not invoked after a failed apply")
	}

	if got := f.env.DestroyCallCount(); got != 1 {
		t.Errorf("DestroyCallCount() = %d, want 1", got)
	}
	if _, statErr := os.Stat(f.workDir); !os.IsNotExist(statErr) {
		t.Errorf("work directory still present after cleanup: err = %v", statErr)
	}
}

func TestRunnerRunResetsClusterWhenVerificationFails(t *testing.T) {
	f := newRunnerFixture(t)
	f.remote.RunStub = func(_ context.Context, _ string, command string) (multinode.Result, error) {
		switch command {
		case "ps -p 1 -o comm=":
			return multinode.Result{Stdout: "systemd\n"}, nil
		case "sudo /usr/local/bin/k3s --version":
			return multinode.Result{Stdout: "k3s version " + runnerK3sVersion + "\n"}, nil
		case "sudo /usr/local/bin/k3s kubectl get nodes -o json":
			return multinode.Result{Stdout: notReadyNodesJSON}, nil
		default:
			return multinode.Result{}, nil
		}
	}

	runner, err := multinode.NewRunner(f.options)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if err := runner.Run(context.Background()); err == nil {
		t.Fatal("expected Run() to fail")
	} else if !strings.Contains(err.Error(), "expected 3 Ready nodes") {
		t.Errorf("error = %v, want a readiness failure", err)
	}

	resetCalled := false
	for i := 0; i < f.commands.RunCallCount(); i++ {
		_, _, _, args := f.commands.RunArgsForCall(i)
		if len(args) > 0 && args[0] == "reset" {
			resetCalled = true
		}
	}
	if !resetCalled {
		t.Error("k3sctl reset was not invoked after a failed verification")
	}
	if got := f.env.DestroyCallCount(); got != 1 {
		t.Errorf("DestroyCallCount() = %d, want 1", got)
	}
}

func TestRunnerRunSkipsApplyWhenCreateFails(t *testing.T) {
	f := newRunnerFixture(t)
	f.env.CreateReturns(nil, errors.New("lxd unavailable"))

	runner, err := multinode.NewRunner(f.options)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if err := runner.Run(context.Background()); err == nil {
		t.Fatal("expected Run() to fail")
	} else if !strings.Contains(err.Error(), "create environment") {
		t.Errorf("error = %v, want a create failure", err)
	}

	if got := f.commands.RunCallCount(); got != 0 {
		t.Errorf("RunCallCount() = %d, want 0", got)
	}
	if got := f.env.DestroyCallCount(); got != 1 {
		t.Errorf("DestroyCallCount() = %d, want 1", got)
	}
	if _, err := os.Stat(filepath.Join(f.artifactDir, "diagnostics.log")); err != nil {
		t.Errorf("diagnostics.log missing: %v", err)
	}
}

func TestRunnerRunReportsCleanupFailures(t *testing.T) {
	f := newRunnerFixture(t)
	f.env.DestroyReturns(errors.New("destroy failed"))

	runner, err := multinode.NewRunner(f.options)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	err = runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected Run() to report the cleanup failure")
	}
	if !strings.Contains(err.Error(), "cleanup") {
		t.Errorf("error = %v, want it to mention cleanup", err)
	}
	if !strings.Contains(err.Error(), "destroy failed") {
		t.Errorf("error = %v, want it to preserve the destroy failure", err)
	}
}

func TestNewRunnerRejectsIncompleteOptions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*multinode.Options)
		want   string
	}{
		{
			name:   "missing binary",
			mutate: func(o *multinode.Options) { o.Binary = "" },
			want:   "k3sctl binary path is required",
		},
		{
			name:   "missing work dir",
			mutate: func(o *multinode.Options) { o.WorkDir = "" },
			want:   "work directory is required",
		},
		{
			name:   "missing key path",
			mutate: func(o *multinode.Options) { o.KeyPath = "" },
			want:   "SSH key path is required",
		},
		{
			name:   "missing artifact dir",
			mutate: func(o *multinode.Options) { o.ArtifactDir = "" },
			want:   "artifact directory is required",
		},
		{
			name:   "missing environment",
			mutate: func(o *multinode.Options) { o.Environment = nil },
			want:   "environment is required",
		},
		{
			name:   "missing remote executor",
			mutate: func(o *multinode.Options) { o.Remote = nil },
			want:   "remote executor is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture(t)
			options := f.options
			tt.mutate(&options)

			_, err := multinode.NewRunner(options)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}
