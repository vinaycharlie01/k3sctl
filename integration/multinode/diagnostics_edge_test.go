package multinode_test

import (
	"context"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/multinode"
	"github.com/vinaycharlie01/k3sctl/integration/multinode/mocks"
)

func TestCollectDiagnosticsWithEmptyTargets(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}

	output := multinode.CollectDiagnostics(context.Background(), remote, nil, nil)

	if remote.RunCallCount() != 0 {
		t.Errorf("RunCallCount() = %d, want 0", remote.RunCallCount())
	}
	if output != "" {
		t.Errorf("output = %q, want empty for no targets and no platform", output)
	}
}

func TestCollectDiagnosticsWithOnlyPlatform(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}
	platform := func(context.Context) (string, error) { return "test platform output", nil }

	output := multinode.CollectDiagnostics(context.Background(), remote, nil, platform)

	if !strings.Contains(output, "=== Platform ===") {
		t.Errorf("output missing platform section:\n%s", output)
	}
	if !strings.Contains(output, "test platform output") {
		t.Errorf("output missing platform output:\n%s", output)
	}
	if remote.RunCallCount() != 0 {
		t.Errorf("RunCallCount() = %d, want 0 (no targets)", remote.RunCallCount())
	}
}

func TestCollectDiagnosticsWithSingleTarget(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}
	remote.RunStub = func(_ context.Context, _, command string) (multinode.Result, error) {
		return multinode.Result{Stdout: "ok"}, nil
	}

	targets := []multinode.DiagnosticTarget{
		{Name: "lonely-node", Address: "10.0.0.100", Service: "k3s"},
	}
	output := multinode.CollectDiagnostics(context.Background(), remote, targets, nil)

	if !strings.Contains(output, "=== lonely-node (10.0.0.100) ===") {
		t.Errorf("output missing target node:\n%s", output)
	}
}

func TestCollectDiagnosticsRunsAllCommandsPerNode(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}
	commands := make(map[string]int)
	remote.RunStub = func(_ context.Context, addr, command string) (multinode.Result, error) {
		commands[command]++
		return multinode.Result{Stdout: "output"}, nil
	}

	targets := []multinode.DiagnosticTarget{
		{Name: "server", Address: "10.0.0.10", Service: "k3s"},
	}
	multinode.CollectDiagnostics(context.Background(), remote, targets, nil)

	expectedCommands := []string{
		"ip addr",
		"systemctl status k3s --no-pager",
		"sudo journalctl -u k3s --no-pager -n 100",
		"free -m",
		"df -h",
		"ls -la /dev/kmsg",
		"cloud-init status --long",
		"findmnt -o TARGET,PROPAGATION /",
	}
	for _, cmd := range expectedCommands {
		if commands[cmd] == 0 {
			t.Errorf("command %q was not executed", cmd)
		}
	}
}

func TestDiagnosticTargetStruct(t *testing.T) {
	target := multinode.DiagnosticTarget{
		Name:    "server-1",
		Address: "10.0.0.10",
		Service: "k3s",
	}
	if target.Name != "server-1" {
		t.Errorf("Name = %q", target.Name)
	}
	if target.Address != "10.0.0.10" {
		t.Errorf("Address = %q", target.Address)
	}
	if target.Service != "k3s" {
		t.Errorf("Service = %q", target.Service)
	}
}
