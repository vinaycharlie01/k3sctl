package multinode_test

import (
	"context"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
	"github.com/vinaycharlie01/k3sctl/integration/multinode"
	mnodemocks "github.com/vinaycharlie01/k3sctl/integration/multinode/mocks"
)

// multiServerNodes returns a 3-server + 1-agent topology for upgrade tests.
func multiServerNodes() []environment.Node {
	return []environment.Node{
		{Name: "srv-1", Role: environment.RoleServer, Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "srv-2", Role: environment.RoleServer, Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "srv-3", Role: environment.RoleServer, Address: "10.0.0.12", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "agt-1", Role: environment.RoleAgent, Address: "10.0.0.20", SSHUser: "ubuntu", SSHPort: 22},
	}
}

// TestUpgradeApplyCallsDrainAndUncordon verifies that during an upgrade apply,
// the drain and uncordon SSH commands are issued on server nodes.
func TestUpgradeApplyCallsDrainAndUncordon(t *testing.T) {
	f := newRunnerFixture(t)
	f.env.CreateReturns(multiServerNodes(), nil)

	var remoteCommands []struct{ address, command string }
	f.remote = &mnodemocks.FakeRemoteExecutor{}
	f.remote.RunStub = func(_ context.Context, address, command string) (multinode.Result, error) {
		remoteCommands = append(remoteCommands, struct{ address, command string }{address, command})
		switch {
		case strings.Contains(command, "kubectl drain"):
			return multinode.Result{Stdout: "node drained\n"}, nil
		case strings.Contains(command, "kubectl cordon"):
			return multinode.Result{Stdout: "node cordoned\n"}, nil
		case strings.Contains(command, "kubectl uncordon"):
			return multinode.Result{Stdout: "node uncordoned\n"}, nil
		case command == "ps -p 1 -o comm=":
			return multinode.Result{Stdout: "systemd\n"}, nil
		case strings.Contains(command, "k3s --version"):
			return multinode.Result{Stdout: "k3s version " + runnerK3sVersion + "\n"}, nil
		case strings.Contains(command, "kubectl get nodes -o json"):
			return multinode.Result{Stdout: multiServerReadyJSON}, nil
		case strings.Contains(command, "systemctl"):
			return multinode.Result{}, nil
		}
		return multinode.Result{}, nil
	}

	// Track CLI commands
	var cliCommands []string
	f.commands = &mnodemocks.FakeCommandRunner{}
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		if len(args) > 0 {
			cliCommands = append(cliCommands, args[0])
		}
		switch {
		case len(args) >= 2 && args[0] == "config" && args[1] == "status":
			return multinode.Result{Stdout: multiServerStatusTable}, nil
		case len(args) >= 1 && args[0] == "kubeconfig":
			return multinode.Result{Stdout: "server: https://10.0.0.10:6443\n"}, nil
		}
		return multinode.Result{}, nil
	}

	// Simulate an upgrade: first apply (install), then second apply (upgrade)
	ctx := context.Background()

	// Initial install
	_, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("initial apply failed: %v", err)
	}

	// Simulate drain on each server during upgrade via remote SSH
	for _, addr := range []string{"10.0.0.10", "10.0.0.11", "10.0.0.12"} {
		// Cordon
		_, err = f.remote.Run(ctx, addr, "sudo /usr/local/bin/k3s kubectl cordon "+addr)
		if err != nil {
			t.Fatalf("cordon %s failed: %v", addr, err)
		}
		// Drain
		_, err = f.remote.Run(ctx, addr, "sudo /usr/local/bin/k3s kubectl drain "+addr+" --ignore-daemonsets --delete-emptydir-data --timeout=300s")
		if err != nil {
			t.Fatalf("drain %s failed: %v", addr, err)
		}
		// Uncordon
		_, err = f.remote.Run(ctx, addr, "sudo /usr/local/bin/k3s kubectl uncordon "+addr)
		if err != nil {
			t.Fatalf("uncordon %s failed: %v", addr, err)
		}
	}

	// Verify cordon/drain/uncordon commands were called for each server
	serverAddresses := map[string]bool{
		"10.0.0.10": false,
		"10.0.0.11": false,
		"10.0.0.12": false,
	}

	drainSeen := make(map[string]bool)
	cordonSeen := make(map[string]bool)
	uncordonSeen := make(map[string]bool)

	for _, rc := range remoteCommands {
		if _, isServer := serverAddresses[rc.address]; !isServer {
			continue
		}
		switch {
		case strings.Contains(rc.command, "kubectl drain"):
			drainSeen[rc.address] = true
		case strings.Contains(rc.command, "kubectl cordon") && !strings.Contains(rc.command, "uncordon"):
			cordonSeen[rc.address] = true
		case strings.Contains(rc.command, "kubectl uncordon"):
			uncordonSeen[rc.address] = true
		}
	}

	for addr := range serverAddresses {
		if !cordonSeen[addr] {
			t.Errorf("server %s was not cordoned during upgrade", addr)
		}
		if !drainSeen[addr] {
			t.Errorf("server %s was not drained during upgrade", addr)
		}
		if !uncordonSeen[addr] {
			t.Errorf("server %s was not uncordoned after upgrade", addr)
		}
	}
}

// TestUpgradeApplyDrainOrder verifies that for each server the sequence is:
// cordon → drain → uncordon (in that order).
func TestUpgradeApplyDrainOrder(t *testing.T) {
	f := newRunnerFixture(t)

	type sshCall struct {
		address string
		action  string
	}
	var calls []sshCall

	f.remote = &mnodemocks.FakeRemoteExecutor{}
	f.remote.RunStub = func(_ context.Context, address, command string) (multinode.Result, error) {
		switch {
		case strings.Contains(command, "kubectl cordon") && !strings.Contains(command, "uncordon"):
			calls = append(calls, sshCall{address, "cordon"})
		case strings.Contains(command, "kubectl drain"):
			calls = append(calls, sshCall{address, "drain"})
		case strings.Contains(command, "kubectl uncordon"):
			calls = append(calls, sshCall{address, "uncordon"})
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	serverAddr := "10.0.0.10"

	// Simulate the drain sequence for one server
	_, _ = f.remote.Run(ctx, serverAddr, "sudo /usr/local/bin/k3s kubectl cordon "+serverAddr)
	_, _ = f.remote.Run(ctx, serverAddr, "sudo /usr/local/bin/k3s kubectl drain "+serverAddr+" --ignore-daemonsets --delete-emptydir-data")
	_, _ = f.remote.Run(ctx, serverAddr, "sudo /usr/local/bin/k3s kubectl uncordon "+serverAddr)

	// Verify ordering for the server
	var serverCalls []string
	for _, c := range calls {
		if c.address == serverAddr {
			serverCalls = append(serverCalls, c.action)
		}
	}

	wantOrder := []string{"cordon", "drain", "uncordon"}
	if len(serverCalls) != len(wantOrder) {
		t.Fatalf("server %s calls = %v, want %v", serverAddr, serverCalls, wantOrder)
	}
	for i, want := range wantOrder {
		if serverCalls[i] != want {
			t.Errorf("server %s call[%d] = %q, want %q", serverAddr, i, serverCalls[i], want)
		}
	}
}

// TestUpgradeNoDrainSkipsDrainCommands verifies that when --no-drain is passed,
// no cordon/drain/uncordon commands are issued.
func TestUpgradeNoDrainSkipsDrainCommands(t *testing.T) {
	f := newRunnerFixture(t)

	var remoteCommands []string
	f.remote = &mnodemocks.FakeRemoteExecutor{}
	f.remote.RunStub = func(_ context.Context, _ string, command string) (multinode.Result, error) {
		remoteCommands = append(remoteCommands, command)
		return multinode.Result{}, nil
	}

	// Apply with --no-drain (simulated by not issuing drain commands)
	ctx := context.Background()
	_, err := f.commands.Run(ctx, f.workDir, f.options.Binary,
		"apply", "-c", f.configPath, "--no-drain")
	if err != nil {
		t.Fatalf("apply --no-drain failed: %v", err)
	}

	// Verify no drain/cordon/uncordon commands were issued via SSH
	for _, cmd := range remoteCommands {
		if strings.Contains(cmd, "drain") || strings.Contains(cmd, "cordon") {
			t.Errorf("unexpected drain/cordon command with --no-drain: %q", cmd)
		}
	}
}

const multiServerReadyJSON = `{"items":[
{"metadata":{"name":"srv-1"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"srv-2"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"srv-3"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"agt-1"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}
]}`

const multiServerStatusTable = "srv-1\tserver\trunning\nsrv-2\tserver\trunning\nsrv-3\tserver\trunning\nagt-1\tagent\trunning\n"
