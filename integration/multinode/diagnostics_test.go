package multinode_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/multinode"
	"github.com/vinaycharlie01/k3sctl/integration/multinode/mocks"
)

const diagnosticsCommandsPerNode = 8

func diagnosticTargets() []multinode.DiagnosticTarget {
	return []multinode.DiagnosticTarget{
		{Name: "server", Address: "10.0.0.10", Service: "k3s"},
		{Name: "agent1", Address: "10.0.0.11", Service: "k3s-agent"},
	}
}

func TestCollectDiagnosticsIncludesPlatformAndEveryNode(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}
	remote.RunStub = func(_ context.Context, address, command string) (multinode.Result, error) {
		return multinode.Result{Stdout: command + " output"}, nil
	}

	platform := func(context.Context) (string, error) { return "platform info", nil }

	output := multinode.CollectDiagnostics(context.Background(), remote, diagnosticTargets(), platform)

	for _, want := range []string{
		"=== Platform ===",
		"platform info",
		"=== server (10.0.0.10) ===",
		"=== agent1 (10.0.0.11) ===",
		"--- ip addr ---",
		"--- systemctl status k3s --no-pager ---",
		"--- systemctl status k3s-agent --no-pager ---",
		"--- sudo journalctl -u k3s-agent --no-pager -n 100 ---",
		"--- ls -la /dev/kmsg ---",
		"ip addr output",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}

	if got, want := remote.RunCallCount(), len(diagnosticTargets())*diagnosticsCommandsPerNode; got != want {
		t.Fatalf("RunCallCount() = %d, want %d", got, want)
	}
}

func TestCollectDiagnosticsAddressesTheOwningNode(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}

	multinode.CollectDiagnostics(context.Background(), remote, diagnosticTargets(), nil)

	for i := 0; i < remote.RunCallCount(); i++ {
		_, address, _ := remote.RunArgsForCall(i)
		if address != "10.0.0.10" && address != "10.0.0.11" {
			t.Fatalf("call %d addressed %q, want one of the target addresses", i, address)
		}
		if i < diagnosticsCommandsPerNode && address != "10.0.0.10" {
			t.Fatalf("call %d addressed %q, want the server address", i, address)
		}
		if i >= diagnosticsCommandsPerNode && address != "10.0.0.11" {
			t.Fatalf("call %d addressed %q, want the agent address", i, address)
		}
	}
}

func TestCollectDiagnosticsEmbedsCommandErrorsAndContinues(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}
	remote.RunStub = func(_ context.Context, _, command string) (multinode.Result, error) {
		if command == "ip addr" {
			return multinode.Result{Stdout: "partial", Stderr: "boom"}, errors.New("ssh command failed")
		}
		return multinode.Result{}, nil
	}

	output := multinode.CollectDiagnostics(context.Background(), remote, diagnosticTargets(), nil)

	for _, want := range []string{"partial", "boom", "error: ssh command failed", "=== agent1 (10.0.0.11) ==="} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}

	if got := strings.Count(output, "error: ssh command failed"); got != 2 {
		t.Errorf("error count = %d, want 2 (one per node)", got)
	}
}

func TestCollectDiagnosticsEmbedsPlatformError(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}
	platform := func(context.Context) (string, error) { return "", errors.New("lxc unreachable") }

	output := multinode.CollectDiagnostics(context.Background(), remote, diagnosticTargets(), platform)

	if !strings.Contains(output, "error: lxc unreachable") {
		t.Errorf("output missing the platform error:\n%s", output)
	}
	if !strings.Contains(output, "=== server (10.0.0.10) ===") {
		t.Errorf("node diagnostics were dropped after a platform error:\n%s", output)
	}
}

func TestCollectDiagnosticsOmitsPlatformSectionWhenNoCallback(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}

	output := multinode.CollectDiagnostics(context.Background(), remote, diagnosticTargets(), nil)

	if strings.Contains(output, "=== Platform ===") {
		t.Errorf("unexpected platform section:\n%s", output)
	}
}

func TestCollectDiagnosticsHandlesEmptyCommandOutput(t *testing.T) {
	remote := &mocks.FakeRemoteExecutor{}

	output := multinode.CollectDiagnostics(context.Background(), remote, []multinode.DiagnosticTarget{{Name: "server", Address: "10.0.0.10", Service: "k3s"}}, nil)

	if got := strings.Count(output, "--- "); got != diagnosticsCommandsPerNode {
		t.Errorf("command sections = %d, want %d", got, diagnosticsCommandsPerNode)
	}
	if strings.Contains(output, "error:") {
		t.Errorf("unexpected error line for a clean run:\n%s", output)
	}
}
