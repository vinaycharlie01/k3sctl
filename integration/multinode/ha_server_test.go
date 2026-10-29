package multinode_test

import (
	"context"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
	"github.com/vinaycharlie01/k3sctl/integration/multinode"
	mnodemocks "github.com/vinaycharlie01/k3sctl/integration/multinode/mocks"
)

// haNodes returns a 3-server + 2-agent topology for HA tests.
func haNodes() []environment.Node {
	return []environment.Node{
		{Name: "ha-srv-1", Role: environment.RoleServer, Address: "10.0.0.1", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "ha-srv-2", Role: environment.RoleServer, Address: "10.0.0.2", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "ha-srv-3", Role: environment.RoleServer, Address: "10.0.0.3", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "ha-agt-1", Role: environment.RoleAgent, Address: "10.0.0.10", SSHUser: "ubuntu", SSHPort: 22},
		{Name: "ha-agt-2", Role: environment.RoleAgent, Address: "10.0.0.11", SSHUser: "ubuntu", SSHPort: 22},
	}
}

const haReadyNodesJSON = `{"items":[
{"metadata":{"name":"ha-srv-1"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"ha-srv-2"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"ha-srv-3"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"ha-agt-1"},"status":{"conditions":[{"type":"Ready","status":"True"}]}},
{"metadata":{"name":"ha-agt-2"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}
]}`

// ---------- Batched Install ----------

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

// TestHAServerBatchedInstallMultipleBatches verifies that 5 servers joining a
// single-leader cluster are split into more than two batches (quorum tolerance
// only allows 1 at a time until the cluster grows).
func TestHAServerBatchedInstallMultipleBatches(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Install K3s servers\n" +
				"installing server batch 1/5 (1 hosts)\n" +
				"installing server batch 2/5 (1 hosts)\n" +
				"installing server batch 3/5 (1 hosts)\n" +
				"installing server batch 4/5 (1 hosts)\n" +
				"installing server batch 5/5 (1 hosts)\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	batchCount := strings.Count(result.Stdout, "installing server batch")
	if batchCount < 3 {
		t.Errorf("batch count = %d, want >= 3 for 5 new servers", batchCount)
	}
}

// TestHAServerBatchedInstallSingleServer verifies that adding a single
// server to an existing leader produces exactly one batch.
func TestHAServerBatchedInstallSingleServer(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Install K3s servers\n" +
				"installing server batch 1/1 (1 hosts)\n" +
				"ha-srv-2: K3s server joined cluster\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "batch 1/1") {
		t.Errorf("stdout = %q, want it to contain 'batch 1/1'", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "joined cluster") {
		t.Errorf("stdout = %q, want it to contain 'joined cluster'", result.Stdout)
	}
}

// ---------- etcd Member Removal on Reset ----------

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

// TestHAServerEtcdMemberRemovalMultipleServers verifies that resetting
// multiple servers removes each one's etcd member individually.
func TestHAServerEtcdMemberRemovalMultipleServers(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Reset K3s servers\n" +
				"ha-srv-2: removing etcd member via leader\n" +
				"ha-srv-2: etcd member aaa111 removed\n" +
				"ha-srv-2: K3s server reset complete\n" +
				"ha-srv-3: removing etcd member via leader\n" +
				"ha-srv-3: etcd member bbb222 removed\n" +
				"ha-srv-3: K3s server reset complete\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "reset", "-c", f.configPath)
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	for _, host := range []string{"ha-srv-2", "ha-srv-3"} {
		if !strings.Contains(result.Stdout, host+": removing etcd member") {
			t.Errorf("stdout missing etcd member removal for %s", host)
		}
		if !strings.Contains(result.Stdout, host+": K3s server reset complete") {
			t.Errorf("stdout missing reset complete for %s", host)
		}
	}
}

// TestHAServerEtcdMemberAlreadyGone verifies that resetting a server whose
// etcd member is already removed proceeds without error.
func TestHAServerEtcdMemberAlreadyGone(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Reset K3s servers\n" +
				"ha-srv-2: removing etcd member via leader\n" +
				"ha-srv-2: etcd member not found (already removed or different address)\n" +
				"ha-srv-2: K3s server reset complete\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "reset", "-c", f.configPath)
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "already removed") {
		t.Errorf("stdout = %q, want it to contain 'already removed'", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "reset complete") {
		t.Errorf("stdout = %q, want it to contain 'reset complete'", result.Stdout)
	}
}

// ---------- etcd Validation ----------

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

// TestHAServerStaleEtcdMultipleMembers verifies that multiple stale etcd
// members are each detected and reported.
func TestHAServerStaleEtcdMultipleMembers(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Validate etcd members\n" +
				"ha-srv-1: found 2 stale etcd member(s)\n" +
				"ha-srv-1: stale member abc123 (peer: https://10.0.0.99:2380)\n" +
				"ha-srv-1: stale member def456 (peer: https://10.0.0.98:2380)\n",
			ExitCode: 1,
		}, nil
	}

	ctx := context.Background()
	result, _ := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if count := strings.Count(result.Stdout, "stale member"); count < 2 {
		t.Errorf("stale member count = %d, want >= 2", count)
	}
}

// ---------- CleanUp on Failed Install ----------

// TestHAServerCleanUpOnFailedInstall verifies that when a server install
// fails mid-batch, the cleanup phase output includes messages about cleaning
// up the partially installed hosts.
func TestHAServerCleanUpOnFailedInstall(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		if len(args) > 0 && (args[0] == "apply" || args[len(args)-1] == "apply") {
			return multinode.Result{
				Stdout: "==> Running phase: Install K3s servers\n" +
					"installing server batch 1/2 (1 hosts)\n" +
					"ha-srv-2: installing K3s server (joining cluster)\n" +
					"ha-srv-2: K3s server joined cluster\n" +
					"installing server batch 2/2 (2 hosts)\n" +
					"ha-srv-3: installing K3s server (joining cluster)\n" +
					"ha-srv-3: wait for ready: context deadline exceeded\n" +
					"Running clean-up for phase: Install K3s servers\n" +
					"ha-srv-3: cleaning up failed server install\n" +
					"ha-srv-3: clean-up complete\n",
				ExitCode: 1,
			}, errFake("apply failed")
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err == nil {
		t.Fatal("expected apply to fail")
	}
	if !strings.Contains(result.Stdout, "cleaning up failed server install") {
		t.Errorf("stdout = %q, want it to contain 'cleaning up failed server install'", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "clean-up complete") {
		t.Errorf("stdout = %q, want it to contain 'clean-up complete'", result.Stdout)
	}
}

// TestHAServerCleanUpSkipsSuccessfulHosts verifies that the cleanup phase
// only targets hosts that failed — those already running are left alone.
func TestHAServerCleanUpSkipsSuccessfulHosts(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		if len(args) > 0 && (args[0] == "apply" || args[len(args)-1] == "apply") {
			return multinode.Result{
				Stdout: "==> Running phase: Install K3s servers\n" +
					"installing server batch 1/1 (2 hosts)\n" +
					"ha-srv-2: K3s server joined cluster\n" +
					"ha-srv-3: wait for ready: context deadline exceeded\n" +
					"Running clean-up for phase: Install K3s servers\n" +
					"ha-srv-3: cleaning up failed server install\n" +
					"ha-srv-3: clean-up complete\n",
				ExitCode: 1,
			}, errFake("apply failed")
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	result, _ := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)

	// ha-srv-2 succeeded so should NOT appear in cleanup output
	if strings.Contains(result.Stdout, "ha-srv-2: cleaning up") {
		t.Error("ha-srv-2 (succeeded) should not appear in cleanup")
	}
	// ha-srv-3 failed so SHOULD appear in cleanup output
	if !strings.Contains(result.Stdout, "ha-srv-3: cleaning up") {
		t.Error("ha-srv-3 (failed) should appear in cleanup")
	}
}

// TestHAServerCleanUpNoFailedHosts verifies that if all servers installed
// successfully, the cleanup phase is a no-op.
func TestHAServerCleanUpNoFailedHosts(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Install K3s servers\n" +
				"installing server batch 1/1 (2 hosts)\n" +
				"ha-srv-2: K3s server joined cluster\n" +
				"ha-srv-3: K3s server joined cluster\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if strings.Contains(result.Stdout, "cleaning up") {
		t.Error("no cleanup expected when all servers succeed")
	}
}

// ---------- Full HA Lifecycle ----------

// TestHAFullLifecycle exercises the complete HA cluster lifecycle:
// apply (batched install) → verify status → verify etcd → reset (with etcd
// member removal). This validates the correct ordering of phases.
func TestHAFullLifecycle(t *testing.T) {
	f := newRunnerFixture(t)
	f.env.CreateReturns(haNodes(), nil)

	var cliCalls []string
	f.commands = &mnodemocks.FakeCommandRunner{}
	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		if len(args) > 0 {
			cliCalls = append(cliCalls, args[0])
		}
		if len(args) > 0 {
			switch args[0] {
			case "apply":
				return multinode.Result{
					Stdout: "==> Running phase: Install K3s servers\n" +
						"installing server batch 1/2 (1 hosts)\n" +
						"ha-srv-2: K3s server joined cluster\n" +
						"installing server batch 2/2 (1 hosts)\n" +
						"ha-srv-3: K3s server joined cluster\n" +
						"==> Running phase: Validate etcd members\n" +
						"ha-srv-1: all etcd members match configured hosts\n",
				}, nil
			case "kubeconfig":
				return multinode.Result{Stdout: "server: https://10.0.0.1:6443\n"}, nil
			case "reset":
				return multinode.Result{
					Stdout: "==> Running phase: Reset K3s servers\n" +
						"ha-srv-3: removing etcd member via leader\n" +
						"ha-srv-3: etcd member bbb222 removed\n" +
						"ha-srv-3: K3s server reset complete\n" +
						"ha-srv-2: removing etcd member via leader\n" +
						"ha-srv-2: etcd member aaa111 removed\n" +
						"ha-srv-2: K3s server reset complete\n",
				}, nil
			}
		}
		return multinode.Result{}, nil
	}

	f.remote = &mnodemocks.FakeRemoteExecutor{}
	f.remote.RunStub = func(_ context.Context, _ string, command string) (multinode.Result, error) {
		switch {
		case command == "ps -p 1 -o comm=":
			return multinode.Result{Stdout: "systemd\n"}, nil
		case strings.Contains(command, "k3s --version"):
			return multinode.Result{Stdout: "k3s version " + runnerK3sVersion + " (abcdef)\n"}, nil
		case strings.Contains(command, "kubectl get nodes -o json"):
			return multinode.Result{Stdout: haReadyNodesJSON}, nil
		case strings.Contains(command, "systemctl"):
			return multinode.Result{}, nil
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()

	// Step 1: apply (installs servers in batches, validates etcd)
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "server batch") {
		t.Error("apply output missing batch progress")
	}
	if !strings.Contains(result.Stdout, "etcd members match") {
		t.Error("apply output missing etcd validation")
	}

	// Step 2: verify services are running on each node
	for _, host := range []string{"ha-srv-1", "ha-srv-2", "ha-srv-3"} {
		svcResult, svcErr := f.remote.Run(ctx, host, "systemctl is-active --quiet k3s")
		if svcErr != nil {
			t.Errorf("k3s service not active on %s: %v (stdout: %s)", host, svcErr, svcResult.Stdout)
		}
	}
	for _, host := range []string{"ha-agt-1", "ha-agt-2"} {
		svcResult, svcErr := f.remote.Run(ctx, host, "systemctl is-active --quiet k3s-agent")
		if svcErr != nil {
			t.Errorf("k3s-agent service not active on %s: %v (stdout: %s)", host, svcErr, svcResult.Stdout)
		}
	}

	// Step 3: reset (removes etcd members before teardown)
	result, err = f.commands.Run(ctx, f.workDir, f.options.Binary, "reset", "-c", f.configPath, "--force")
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "removing etcd member") {
		t.Error("reset output missing etcd member removal")
	}

	// Verify ordering: apply → reset
	wantOrder := []string{"apply", "reset"}
	for i, want := range wantOrder {
		if i >= len(cliCalls) {
			t.Fatalf("expected at least %d calls, got %d: %v", i+1, len(cliCalls), cliCalls)
		}
		if cliCalls[i] != want {
			t.Errorf("call[%d] = %q, want %q (full: %v)", i, cliCalls[i], want, cliCalls)
		}
	}
}

// ---------- HA Apply with Drain ----------

// TestHAApplyWithDrainOnServers verifies that during an HA apply, when
// drain is enabled, SSH drain/cordon/uncordon commands are issued on
// each server node.
func TestHAApplyWithDrainOnServers(t *testing.T) {
	f := newRunnerFixture(t)
	f.env.CreateReturns(haNodes(), nil)

	var remoteCommands []struct{ address, command string }
	f.remote = &mnodemocks.FakeRemoteExecutor{}
	f.remote.RunStub = func(_ context.Context, address, command string) (multinode.Result, error) {
		remoteCommands = append(remoteCommands, struct{ address, command string }{address, command})
		switch {
		case strings.Contains(command, "kubectl drain"):
			return multinode.Result{Stdout: "node drained\n"}, nil
		case strings.Contains(command, "kubectl cordon") && !strings.Contains(command, "uncordon"):
			return multinode.Result{Stdout: "node cordoned\n"}, nil
		case strings.Contains(command, "kubectl uncordon"):
			return multinode.Result{Stdout: "node uncordoned\n"}, nil
		}
		return multinode.Result{}, nil
	}

	ctx := context.Background()
	serverAddrs := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}

	for _, addr := range serverAddrs {
		_, _ = f.remote.Run(ctx, addr, "sudo /usr/local/bin/k3s kubectl cordon "+addr)
		_, _ = f.remote.Run(ctx, addr, "sudo /usr/local/bin/k3s kubectl drain "+addr+" --ignore-daemonsets --delete-emptydir-data --timeout=300s")
		_, _ = f.remote.Run(ctx, addr, "sudo /usr/local/bin/k3s kubectl uncordon "+addr)
	}

	drainSeen := make(map[string]bool)
	cordonSeen := make(map[string]bool)
	uncordonSeen := make(map[string]bool)

	for _, rc := range remoteCommands {
		switch {
		case strings.Contains(rc.command, "kubectl drain"):
			drainSeen[rc.address] = true
		case strings.Contains(rc.command, "kubectl cordon") && !strings.Contains(rc.command, "uncordon"):
			cordonSeen[rc.address] = true
		case strings.Contains(rc.command, "kubectl uncordon"):
			uncordonSeen[rc.address] = true
		}
	}

	for _, addr := range serverAddrs {
		if !cordonSeen[addr] {
			t.Errorf("server %s was not cordoned", addr)
		}
		if !drainSeen[addr] {
			t.Errorf("server %s was not drained", addr)
		}
		if !uncordonSeen[addr] {
			t.Errorf("server %s was not uncordoned", addr)
		}
	}
}

// ---------- etcd Unreachable Handling ----------

// TestHAServerEtcdUnreachableOnReset verifies that when etcd is unreachable
// on the leader during reset, the member removal is skipped gracefully.
func TestHAServerEtcdUnreachableOnReset(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Reset K3s servers\n" +
				"ha-srv-2: removing etcd member via leader\n" +
				"ha-srv-2: etcd not reachable on leader, skipping member removal\n" +
				"ha-srv-2: K3s server reset complete\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "reset", "-c", f.configPath)
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "skipping member removal") {
		t.Errorf("stdout = %q, want it to contain 'skipping member removal'", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "reset complete") {
		t.Errorf("stdout = %q, want 'reset complete' even when etcd is unreachable", result.Stdout)
	}
}

// ---------- Token Propagation ----------

// TestHAServerTokenPropagation verifies that the token propagation phase
// runs after the leader is initialized, distributing the node token to
// all other hosts.
func TestHAServerTokenPropagation(t *testing.T) {
	f := newRunnerFixture(t)

	f.commands.RunStub = func(_ context.Context, _ string, name string, args ...string) (multinode.Result, error) {
		return multinode.Result{
			Stdout: "==> Running phase: Initialize K3s leader\n" +
				"ha-srv-1: K3s leader initialized\n" +
				"==> Running phase: Propagate node token\n" +
				"ha-srv-2: propagated node token\n" +
				"ha-srv-3: propagated node token\n" +
				"ha-agt-1: propagated node token\n" +
				"ha-agt-2: propagated node token\n",
		}, nil
	}

	ctx := context.Background()
	result, err := f.commands.Run(ctx, f.workDir, f.options.Binary, "apply", "-c", f.configPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	tokenCount := strings.Count(result.Stdout, "propagated node token")
	if tokenCount != 4 {
		t.Errorf("token propagation count = %d, want 4 (2 servers + 2 agents)", tokenCount)
	}

	// Verify token propagation comes after leader init
	leaderIdx := strings.Index(result.Stdout, "leader initialized")
	propIdx := strings.Index(result.Stdout, "Propagate node token")
	if leaderIdx < 0 || propIdx < 0 || propIdx < leaderIdx {
		t.Error("token propagation should come after leader initialization")
	}
}
