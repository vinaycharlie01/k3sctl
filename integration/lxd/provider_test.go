package lxd_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	lxdclient "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
	"github.com/vinaycharlie01/k3sctl/integration/lxd"
	"github.com/vinaycharlie01/k3sctl/integration/lxd/mocks"
)

const testKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyMaterialForUnitTests test@example"

// stubOperation implements the lxdclient.Operation methods the provider calls.
// The embedded interface supplies the rest; the provider never invokes them.
type stubOperation struct {
	lxdclient.Operation

	result       api.Operation
	waitErr      error
	waitFunc     func(context.Context) error
	cancelCalled bool
}

func (o *stubOperation) Get() api.Operation {
	return o.result
}

func (o *stubOperation) Cancel() error {
	o.cancelCalled = true
	return nil
}

func (o *stubOperation) WaitContext(ctx context.Context) error {
	if o.waitFunc != nil {
		return o.waitFunc(ctx)
	}
	return o.waitErr
}

func successOperation() *stubOperation {
	return &stubOperation{result: api.Operation{Metadata: map[string]any{"return": float64(0)}}}
}

func baseConfig() lxd.Config {
	return lxd.Config{
		RunID:            "run-1",
		Nodes:            []lxd.NodeSpec{{Name: "server", Role: environment.RoleServer}},
		SSHAuthorizedKey: testKey,
		PollInterval:     time.Millisecond,
		OperationTimeout: time.Second,
		AddressTimeout:   time.Second,
		BootstrapTimeout: time.Second,
	}
}

func newFakeServer() *mocks.FakeServer {
	fake := &mocks.FakeServer{}
	fake.GetServerReturns(&api.Server{Environment: api.ServerEnvironment{ServerVersion: "5.21.0"}}, "", nil)
	fake.HasExtensionReturns(true)
	return fake
}

func newProvider(t *testing.T, fake *mocks.FakeServer, cfg lxd.Config) *lxd.Provider {
	t.Helper()

	provider, err := lxd.NewWithServer(cfg, fake)
	if err != nil {
		t.Fatalf("NewWithServer() error = %v", err)
	}
	return provider
}

func runningState(address string) *api.InstanceState {
	return &api.InstanceState{
		Status:     "Running",
		StatusCode: api.Running,
		Network: map[string]api.InstanceStateNetwork{
			"eth0": {Addresses: []api.InstanceStateNetworkAddress{
				{Family: "inet6", Address: "fd42::1", Scope: "global"},
				{Family: "inet", Address: address, Scope: "global"},
			}},
		},
	}
}

func TestNewWithServerValidatesServer(t *testing.T) {
	t.Run("requires instances extension", func(t *testing.T) {
		fake := newFakeServer()
		fake.HasExtensionReturns(false)

		_, err := lxd.NewWithServer(baseConfig(), fake)
		if err == nil || !strings.Contains(err.Error(), "does not support") {
			t.Fatalf("NewWithServer() error = %v, want unsupported extension error", err)
		}
	})

	t.Run("server query error", func(t *testing.T) {
		fake := &mocks.FakeServer{}
		fake.GetServerReturns(nil, "", errors.New("socket gone"))

		_, err := lxd.NewWithServer(baseConfig(), fake)
		if err == nil || !strings.Contains(err.Error(), "query LXD server") {
			t.Fatalf("NewWithServer() error = %v, want server query error", err)
		}
	})

	t.Run("valid", func(t *testing.T) {
		newProvider(t, newFakeServer(), baseConfig())
	})
}

func TestCreateProvisionsAndStartsContainers(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))

	var created []api.InstancesPost
	fake.CreateInstanceCalls(func(post api.InstancesPost) (lxdclient.Operation, error) {
		created = append(created, post)
		return successOperation(), nil
	})

	var started []string
	fake.UpdateInstanceStateCalls(func(name string, state api.InstanceStatePut, _ string) (lxdclient.Operation, error) {
		started = append(started, name+":"+state.Action)
		return successOperation(), nil
	})

	addresses := map[string]string{
		"k3sctl-run-1-server": "10.0.0.10",
		"k3sctl-run-1-agent1": "10.0.0.11",
		"k3sctl-run-1-agent2": "10.0.0.12",
	}
	fake.GetInstanceStateCalls(func(name string) (*api.InstanceState, string, error) {
		address, ok := addresses[name]
		if !ok {
			return nil, "", api.StatusErrorf(http.StatusNotFound, "missing instance")
		}
		return runningState(address), "", nil
	})

	cfg := baseConfig()
	cfg.Nodes = []lxd.NodeSpec{
		{Name: "server", Role: environment.RoleServer},
		{Name: "agent1", Role: environment.RoleAgent},
		{Name: "agent2", Role: environment.RoleAgent},
	}
	provider := newProvider(t, fake, cfg)

	nodes, err := provider.Create(context.Background())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(nodes) != 3 {
		t.Fatalf("Create() returned %d nodes, want 3", len(nodes))
	}
	for i, want := range []struct {
		name    string
		role    environment.Role
		address string
	}{
		{"k3sctl-run-1-server", environment.RoleServer, "10.0.0.10"},
		{"k3sctl-run-1-agent1", environment.RoleAgent, "10.0.0.11"},
		{"k3sctl-run-1-agent2", environment.RoleAgent, "10.0.0.12"},
	} {
		got := nodes[i]
		if got.Name != want.name || got.Role != want.role || got.Address != want.address {
			t.Errorf("node[%d] = %+v, want name=%q role=%q address=%q", i, got, want.name, want.role, want.address)
		}
		if got.SSHUser != "ubuntu" || got.SSHPort != 22 {
			t.Errorf("node[%d] SSH = %s:%d, want ubuntu:22", i, got.SSHUser, got.SSHPort)
		}
	}

	if fake.CreateProfileCallCount() != 1 {
		t.Fatalf("CreateProfile called %d times, want 1", fake.CreateProfileCallCount())
	}
	if got := fake.CreateProfileArgsForCall(0).Name; got != "k3sctl-run-1" {
		t.Errorf("profile name = %q, want %q", got, "k3sctl-run-1")
	}

	if len(created) != 3 {
		t.Fatalf("CreateInstance called %d times, want 3", len(created))
	}
	post := created[0]
	if post.Name != "k3sctl-run-1-server" {
		t.Errorf("instance name = %q, want %q", post.Name, "k3sctl-run-1-server")
	}
	if post.Type != api.InstanceTypeContainer {
		t.Errorf("instance type = %q, want %q", post.Type, api.InstanceTypeContainer)
	}
	if !slices.Equal(post.Profiles, []string{"default", "k3sctl-run-1"}) {
		t.Errorf("profiles = %v, want [default k3sctl-run-1]", post.Profiles)
	}
	if post.Source.Type != "image" || post.Source.Alias != "24.04" || post.Source.Protocol != "simplestreams" || post.Source.Mode != "pull" {
		t.Errorf("unexpected image source: %+v", post.Source)
	}

	wantStarted := []string{
		"k3sctl-run-1-server:start",
		"k3sctl-run-1-agent1:start",
		"k3sctl-run-1-agent2:start",
	}
	if !slices.Equal(started, wantStarted) {
		t.Errorf("start actions = %v, want %v", started, wantStarted)
	}
}

func TestCreateWaitsForRunningInstanceState(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
	fake.CreateInstanceReturns(successOperation(), nil)
	fake.UpdateInstanceStateReturns(successOperation(), nil)

	calls := 0
	fake.GetInstanceStateCalls(func(string) (*api.InstanceState, string, error) {
		calls++
		if calls == 1 {
			return &api.InstanceState{Status: "Stopped", StatusCode: api.Stopped}, "", nil
		}
		return runningState("10.0.0.10"), "", nil
	})

	cfg := baseConfig()
	cfg.AddressTimeout = time.Second
	provider := newProvider(t, fake, cfg)

	nodes, err := provider.Create(context.Background())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := nodes[0].Address; got != "10.0.0.10" {
		t.Errorf("address = %q, want 10.0.0.10", got)
	}
	if calls < 2 {
		t.Errorf("GetInstanceState called %d times, want at least 2", calls)
	}
}

func TestCreateTimesOutWaitingForAddress(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
	fake.CreateInstanceReturns(successOperation(), nil)
	fake.UpdateInstanceStateReturns(successOperation(), nil)
	fake.GetInstanceStateReturns(&api.InstanceState{Status: "Stopped", StatusCode: api.Stopped}, "", nil)

	cfg := baseConfig()
	cfg.AddressTimeout = 20 * time.Millisecond
	cfg.PollInterval = time.Millisecond
	provider := newProvider(t, fake, cfg)

	_, err := provider.Create(context.Background())
	if err == nil || !strings.Contains(err.Error(), "wait for instance to become ready") {
		t.Fatalf("Create() error = %v, want readiness timeout", err)
	}
}

func TestCreateFailsWhenProfileCreationFails(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
	fake.CreateProfileReturns(errors.New("permission denied"))

	provider := newProvider(t, fake, baseConfig())

	_, err := provider.Create(context.Background())
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("Create() error = %v, want profile creation error", err)
	}
	if fake.CreateInstanceCallCount() != 0 {
		t.Errorf("CreateInstance called %d times, want 0", fake.CreateInstanceCallCount())
	}
}

func TestEnsureProfileReusesMatchingProfile(t *testing.T) {
	// Capture the configuration the provider considers correct.
	capture := newFakeServer()
	capture.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
	capture.CreateInstanceReturns(successOperation(), nil)
	capture.UpdateInstanceStateReturns(successOperation(), nil)
	capture.GetInstanceStateReturns(runningState("10.0.0.10"), "", nil)

	provider := newProvider(t, capture, baseConfig())
	if _, err := provider.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	desired := capture.CreateProfileArgsForCall(0)

	fake := newFakeServer()
	fake.GetProfileReturns(&api.Profile{Name: desired.Name, Config: desired.Config, Devices: desired.Devices}, "etag-1", nil)
	fake.CreateInstanceReturns(successOperation(), nil)
	fake.UpdateInstanceStateReturns(successOperation(), nil)
	fake.GetInstanceStateReturns(runningState("10.0.0.10"), "", nil)
	fake.GetInstanceReturns(&api.Instance{StatusCode: api.Stopped}, "etag", nil)
	fake.DeleteInstanceReturns(successOperation(), nil)

	provider = newProvider(t, fake, baseConfig())
	if _, err := provider.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if fake.CreateProfileCallCount() != 0 || fake.UpdateProfileCallCount() != 0 {
		t.Errorf("matching profile was rewritten: create=%d update=%d",
			fake.CreateProfileCallCount(), fake.UpdateProfileCallCount())
	}

	if err := provider.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}
	if fake.DeleteProfileCallCount() != 0 {
		t.Error("matching but unowned profile must not be deleted")
	}
}

func TestEnsureProfileUpdatesChangedProfile(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(&api.Profile{
		Name:    "k3sctl-run-1",
		Config:  map[string]string{"security.privileged": "false"},
		Devices: map[string]map[string]string{},
	}, "etag-1", nil)
	fake.CreateInstanceReturns(successOperation(), nil)
	fake.UpdateInstanceStateReturns(successOperation(), nil)
	fake.GetInstanceStateReturns(runningState("10.0.0.10"), "", nil)

	provider := newProvider(t, fake, baseConfig())
	if _, err := provider.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if fake.UpdateProfileCallCount() != 1 {
		t.Fatalf("UpdateProfile called %d times, want 1", fake.UpdateProfileCallCount())
	}
	name, profile, etag := fake.UpdateProfileArgsForCall(0)
	if name != "k3sctl-run-1" || etag != "etag-1" {
		t.Errorf("UpdateProfile(%q, etag=%q), want k3sctl-run-1/etag-1", name, etag)
	}
	if profile.Config["security.privileged"] != "true" {
		t.Errorf("updated profile privileged = %q, want true", profile.Config["security.privileged"])
	}
}

func TestCreatePartialFailureTracksCreatedInstances(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
	fake.CreateInstanceReturns(successOperation(), nil)
	fake.UpdateInstanceStateCalls(func(name string, _ api.InstanceStatePut, _ string) (lxdclient.Operation, error) {
		if name == "k3sctl-run-1-agent1" {
			return nil, errors.New("start failed")
		}
		return successOperation(), nil
	})
	fake.GetInstanceStateCalls(func(string) (*api.InstanceState, string, error) {
		return runningState("10.0.0.10"), "", nil
	})

	cfg := baseConfig()
	cfg.Nodes = []lxd.NodeSpec{
		{Name: "server", Role: environment.RoleServer},
		{Name: "agent1", Role: environment.RoleAgent},
	}
	provider := newProvider(t, fake, cfg)

	nodes, err := provider.Create(context.Background())
	if err == nil {
		t.Fatal("Create() = nil, want start failure")
	}
	if len(nodes) != 1 || nodes[0].Name != "k3sctl-run-1-server" {
		t.Fatalf("Create() nodes = %+v, want only the server", nodes)
	}

	fake.GetInstanceReturns(&api.Instance{Name: "x", Status: "Stopped", StatusCode: api.Stopped}, "etag", nil)
	var deleted []string
	fake.DeleteInstanceCalls(func(name string) (lxdclient.Operation, error) {
		deleted = append(deleted, name)
		return successOperation(), nil
	})

	if err := provider.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}

	want := []string{"k3sctl-run-1-agent1", "k3sctl-run-1-server"}
	if !slices.Equal(deleted, want) {
		t.Errorf("deleted instances = %v, want %v", deleted, want)
	}
	if fake.DeleteProfileCallCount() != 1 {
		t.Errorf("DeleteProfile called %d times, want 1", fake.DeleteProfileCallCount())
	}
}

func TestDestroyStopsRunningInstancesBeforeDeleting(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
	fake.CreateInstanceReturns(successOperation(), nil)
	fake.UpdateInstanceStateReturns(successOperation(), nil)
	fake.GetInstanceStateReturns(runningState("10.0.0.10"), "", nil)

	provider := newProvider(t, fake, baseConfig())
	if _, err := provider.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	fake.GetInstanceReturns(&api.Instance{Name: "k3sctl-run-1-server", Status: "Running", StatusCode: api.Running}, "etag", nil)

	var actions []string
	fake.UpdateInstanceStateCalls(func(name string, state api.InstanceStatePut, _ string) (lxdclient.Operation, error) {
		actions = append(actions, name+":"+state.Action)
		if !state.Force || state.Timeout != -1 {
			t.Errorf("stop request = %+v, want force with timeout -1", state)
		}
		return successOperation(), nil
	})
	fake.DeleteInstanceCalls(func(name string) (lxdclient.Operation, error) {
		actions = append(actions, name+":delete")
		return successOperation(), nil
	})

	if err := provider.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}

	want := []string{"k3sctl-run-1-server:stop", "k3sctl-run-1-server:delete"}
	if !slices.Equal(actions, want) {
		t.Errorf("actions = %v, want %v", actions, want)
	}
}

func TestDestroyIsIdempotentAndIgnoresMissingInstances(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
	fake.CreateInstanceReturns(successOperation(), nil)
	fake.UpdateInstanceStateReturns(successOperation(), nil)
	fake.GetInstanceStateReturns(runningState("10.0.0.10"), "", nil)
	fake.GetInstanceReturns(&api.Instance{Name: "k3sctl-run-1-server", StatusCode: api.Stopped}, "etag", nil)
	fake.DeleteInstanceReturns(successOperation(), nil)

	provider := newProvider(t, fake, baseConfig())
	if _, err := provider.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := provider.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy() error = %v", err)
	}
	deletes, profileDeletes := fake.DeleteInstanceCallCount(), fake.DeleteProfileCallCount()

	if err := provider.Destroy(context.Background()); err != nil {
		t.Fatalf("second Destroy() error = %v", err)
	}
	if fake.DeleteInstanceCallCount() != deletes || fake.DeleteProfileCallCount() != profileDeletes {
		t.Error("second Destroy() repeated already completed work")
	}

	t.Run("instance already gone", func(t *testing.T) {
		missing := newFakeServer()
		missing.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
		missing.CreateInstanceReturns(successOperation(), nil)
		missing.UpdateInstanceStateReturns(successOperation(), nil)
		missing.GetInstanceStateReturns(runningState("10.0.0.10"), "", nil)
		missing.GetInstanceReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "gone"))

		p := newProvider(t, missing, baseConfig())
		if _, err := p.Create(context.Background()); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if err := p.Destroy(context.Background()); err != nil {
			t.Fatalf("Destroy() error = %v", err)
		}
		if missing.DeleteInstanceCallCount() != 0 {
			t.Error("DeleteInstance called for an instance that no longer exists")
		}
	})
}

func TestDestroyContinuesAfterDeleteFailure(t *testing.T) {
	fake := newFakeServer()
	fake.GetProfileReturns(nil, "", api.StatusErrorf(http.StatusNotFound, "not found"))
	fake.CreateInstanceReturns(successOperation(), nil)
	fake.UpdateInstanceStateReturns(successOperation(), nil)
	fake.GetInstanceStateCalls(func(name string) (*api.InstanceState, string, error) {
		return runningState("10.0.0.10"), "", nil
	})
	fake.GetInstanceReturns(&api.Instance{StatusCode: api.Stopped}, "etag", nil)

	cfg := baseConfig()
	cfg.Nodes = []lxd.NodeSpec{
		{Name: "server", Role: environment.RoleServer},
		{Name: "agent1", Role: environment.RoleAgent},
	}
	provider := newProvider(t, fake, cfg)
	if _, err := provider.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var attempted []string
	fake.DeleteInstanceCalls(func(name string) (lxdclient.Operation, error) {
		attempted = append(attempted, name)
		if name == "k3sctl-run-1-agent1" {
			return nil, errors.New("device busy")
		}
		return successOperation(), nil
	})

	err := provider.Destroy(context.Background())
	if err == nil || !strings.Contains(err.Error(), "device busy") {
		t.Fatalf("Destroy() error = %v, want delete failure", err)
	}
	if len(attempted) != 2 {
		t.Errorf("attempted deletions = %v, want 2", attempted)
	}
	if fake.DeleteProfileCallCount() != 1 {
		t.Errorf("DeleteProfile called %d times, want 1 despite instance failure", fake.DeleteProfileCallCount())
	}
}

func TestBootstrapExecutesScriptInContainer(t *testing.T) {
	fake := newFakeServer()

	var execPost api.InstanceExecPost
	fake.ExecInstanceCalls(func(_ string, post api.InstanceExecPost, args *lxdclient.InstanceExecArgs) (lxdclient.Operation, error) {
		execPost = post
		close(args.DataDone)
		return successOperation(), nil
	})

	provider := newProvider(t, fake, baseConfig())

	nodes := []environment.Node{{Name: "k3sctl-run-1-server", Role: environment.RoleServer}}
	if err := provider.Bootstrap(context.Background(), nodes); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if fake.ExecInstanceCallCount() != 1 {
		t.Fatalf("ExecInstance called %d times, want 1", fake.ExecInstanceCallCount())
	}
	if len(execPost.Command) != 3 || execPost.Command[0] != "/bin/sh" || execPost.Command[1] != "-c" {
		t.Fatalf("command = %v, want /bin/sh -c <script>", execPost.Command)
	}
	if !strings.Contains(execPost.Command[2], "hostnamectl set-hostname 'k3sctl-run-1-server'") {
		t.Error("bootstrap script does not set the node hostname")
	}
	if !execPost.WaitForWS {
		t.Error("exec request must wait for the websocket")
	}
}

func TestBootstrapReportsNonZeroExit(t *testing.T) {
	fake := newFakeServer()
	fake.ExecInstanceCalls(func(_ string, _ api.InstanceExecPost, args *lxdclient.InstanceExecArgs) (lxdclient.Operation, error) {
		_, _ = args.Stderr.Write([]byte("apt failed"))
		close(args.DataDone)
		return &stubOperation{result: api.Operation{Metadata: map[string]any{"return": float64(2)}}}, nil
	})

	provider := newProvider(t, fake, baseConfig())

	err := provider.Bootstrap(context.Background(), []environment.Node{{Name: "k3sctl-run-1-server"}})
	if err == nil {
		t.Fatal("Bootstrap() = nil, want exit code error")
	}
	if !strings.Contains(err.Error(), "exited with code 2") || !strings.Contains(err.Error(), "apt failed") {
		t.Fatalf("Bootstrap() error = %v, want exit code and stderr", err)
	}
}

func TestBootstrapTimesOutAndCancelsOperation(t *testing.T) {
	fake := newFakeServer()

	var operation *stubOperation
	fake.ExecInstanceCalls(func(_ string, _ api.InstanceExecPost, _ *lxdclient.InstanceExecArgs) (lxdclient.Operation, error) {
		operation = &stubOperation{
			waitFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		}
		return operation, nil
	})

	cfg := baseConfig()
	cfg.BootstrapTimeout = 10 * time.Millisecond
	provider := newProvider(t, fake, cfg)

	err := provider.Bootstrap(context.Background(), []environment.Node{{Name: "k3sctl-run-1-server"}})
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("Bootstrap() error = %v, want deadline exceeded", err)
	}
	if operation == nil || !operation.cancelCalled {
		t.Error("timed out operation was not cancelled")
	}
}

func TestDiagnosticsIncludesInstanceStateAndLogs(t *testing.T) {
	fake := newFakeServer()
	fake.GetInstanceReturns(&api.Instance{
		Name:           "k3sctl-run-1-server",
		Status:         "Running",
		StatusCode:     api.Running,
		Type:           "container",
		Profiles:       []string{"default", "k3sctl-run-1"},
		ExpandedConfig: map[string]string{"security.privileged": "true"},
	}, "etag", nil)
	fake.GetInstanceStateReturns(runningState("10.0.0.10"), "", nil)
	fake.GetInstanceLogfilesReturns([]string{"lxc.log"}, nil)
	fake.GetInstanceLogfileReturns(io.NopCloser(strings.NewReader("lxc log line")), nil)

	provider := newProvider(t, fake, baseConfig())

	output, err := provider.Diagnostics(context.Background(), []environment.Node{{Name: "k3sctl-run-1-server"}})
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}

	for _, want := range []string{
		"=== LXD instance k3sctl-run-1-server ===",
		"status: Running",
		"profiles: default, k3sctl-run-1",
		"interface eth0",
		"lxc log line",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("diagnostics missing %q:\n%s", want, output)
		}
	}
}

func TestDiagnosticsEmbedsNodeErrors(t *testing.T) {
	fake := newFakeServer()
	fake.GetInstanceReturns(nil, "", errors.New("instance lookup failed"))

	provider := newProvider(t, fake, baseConfig())

	output, err := provider.Diagnostics(context.Background(), []environment.Node{{Name: "k3sctl-run-1-server"}})
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if !strings.Contains(output, "instance lookup failed") {
		t.Fatalf("diagnostics did not embed the node error:\n%s", output)
	}
}

func TestDefaultSocketPathPrefersExplicitEnvironment(t *testing.T) {
	t.Run("LXD_SOCKET", func(t *testing.T) {
		t.Setenv("LXD_SOCKET", "/tmp/custom-lxd.sock")
		t.Setenv("LXD_DIR", "/should/not/be/used")

		if got := lxd.DefaultSocketPath(); got != "/tmp/custom-lxd.sock" {
			t.Fatalf("DefaultSocketPath() = %q, want the LXD_SOCKET value", got)
		}
	})

	t.Run("LXD_DIR", func(t *testing.T) {
		t.Setenv("LXD_SOCKET", "")
		t.Setenv("LXD_DIR", "/tmp/lxd-dir")

		if got := lxd.DefaultSocketPath(); got != "/tmp/lxd-dir/unix.socket" {
			t.Fatalf("DefaultSocketPath() = %q, want the LXD_DIR socket", got)
		}
	})
}
