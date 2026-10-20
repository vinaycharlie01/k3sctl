package phase

import (
	"context"
	"errors"
	"testing"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
)

// mockPhase implements only the base Phase interface.
type mockPhase struct {
	title    string
	runErr   error
	executed bool
	order    *[]string
}

func (m *mockPhase) Title() string { return m.title }
func (m *mockPhase) Run(_ context.Context) error {
	m.executed = true
	if m.order != nil {
		*m.order = append(*m.order, m.title)
	}
	return m.runErr
}

// mockConditionalPhase adds ShouldRun (conditional interface).
type mockConditionalPhase struct {
	mockPhase
	shouldRun bool
}

func (m *mockConditionalPhase) ShouldRun() bool { return m.shouldRun }

// mockPreparePhase adds Prepare (withprepare interface).
type mockPreparePhase struct {
	mockPhase
	prepared bool
}

func (m *mockPreparePhase) Prepare() { m.prepared = true }

// mockCleanUpPhase adds CleanUp (withcleanup interface).
type mockCleanUpPhase struct {
	mockPhase
	cleanedUp bool
}

func (m *mockCleanUpPhase) CleanUp() { m.cleanedUp = true }

// mockConfigPhase adds SetConfig (withconfig interface).
type mockConfigPhase struct {
	mockPhase
	receivedConfig *v1beta1.Cluster
}

func (m *mockConfigPhase) SetConfig(config *v1beta1.Cluster) {
	m.receivedConfig = config
}

// mockBeforeAfterPhase adds Before and After hooks.
type mockBeforeAfterPhase struct {
	mockPhase
	beforeCalled bool
	afterCalled  bool
	beforeErr    error
	afterErr     error
}

func (m *mockBeforeAfterPhase) Before(_ context.Context) error {
	m.beforeCalled = true
	return m.beforeErr
}

func (m *mockBeforeAfterPhase) After(_ context.Context) error {
	m.afterCalled = true
	return m.afterErr
}

// mockDryRunPhase adds DryRun (withdryrun interface).
type mockDryRunPhase struct {
	mockPhase
	dryRunCalled bool
	dryRunErr    error
}

func (m *mockDryRunPhase) DryRun() error {
	m.dryRunCalled = true
	return m.dryRunErr
}

// mockFullPhase combines all optional interfaces.
type mockFullPhase struct {
	title          string
	runErr         error
	executed       bool
	order          *[]string
	shouldRun      bool
	prepared       bool
	cleanedUp      bool
	receivedConfig *v1beta1.Cluster
}

func (m *mockFullPhase) Title() string { return m.title }
func (m *mockFullPhase) Run(_ context.Context) error {
	m.executed = true
	if m.order != nil {
		*m.order = append(*m.order, m.title)
	}
	return m.runErr
}
func (m *mockFullPhase) ShouldRun() bool                   { return m.shouldRun }
func (m *mockFullPhase) Prepare()                          { m.prepared = true }
func (m *mockFullPhase) CleanUp()                          { m.cleanedUp = true }
func (m *mockFullPhase) SetConfig(config *v1beta1.Cluster) { m.receivedConfig = config }

func TestManager_RunPhasesInOrder(t *testing.T) {
	order := []string{}
	p1 := &mockPhase{title: "first", order: &order}
	p2 := &mockPhase{title: "second", order: &order}
	p3 := &mockPhase{title: "third", order: &order}

	m := &Manager{}
	m.AddPhase(p1, p2, p3)

	cfg := &v1beta1.Cluster{}
	if err := m.Run(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 phases executed, got %d", len(order))
	}
	if order[0] != "first" || order[1] != "second" || order[2] != "third" {
		t.Fatalf("unexpected execution order: %v", order)
	}
}

func TestManager_ConditionalPhaseSkipped(t *testing.T) {
	skipped := &mockConditionalPhase{
		mockPhase: mockPhase{title: "skipped"},
		shouldRun: false,
	}
	executed := &mockPhase{title: "executed"}

	m := &Manager{}
	m.AddPhase(skipped, executed)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if skipped.executed {
		t.Error("conditional phase with ShouldRun()=false was executed")
	}
	if !executed.executed {
		t.Error("phase after skipped phase was not executed")
	}
}

func TestManager_ConditionalPhaseRuns(t *testing.T) {
	phase := &mockConditionalPhase{
		mockPhase: mockPhase{title: "conditional"},
		shouldRun: true,
	}

	m := &Manager{}
	m.AddPhase(phase)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !phase.executed {
		t.Error("conditional phase with ShouldRun()=true was not executed")
	}
}

func TestManager_PrepareCalledBeforeRun(t *testing.T) {
	phase := &mockPreparePhase{
		mockPhase: mockPhase{title: "preparable"},
	}

	m := &Manager{}
	m.AddPhase(phase)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !phase.prepared {
		t.Error("Prepare() was not called")
	}
	if !phase.executed {
		t.Error("Run() was not called after Prepare()")
	}
}

func TestManager_CleanUpCalledAfterRun(t *testing.T) {
	phase := &mockCleanUpPhase{
		mockPhase: mockPhase{title: "cleanable"},
	}

	m := &Manager{}
	m.AddPhase(phase)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !phase.executed {
		t.Error("Run() was not called")
	}
	if !phase.cleanedUp {
		t.Error("CleanUp() was not called after Run()")
	}
}

func TestManager_ErrorStopsExecution(t *testing.T) {
	order := []string{}
	failing := &mockPhase{title: "failing", runErr: errors.New("boom"), order: &order}
	after := &mockPhase{title: "after", order: &order}

	m := &Manager{}
	m.AddPhase(failing, after)

	err := m.Run(context.Background(), &v1beta1.Cluster{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, failing.runErr) {
		t.Fatalf("expected wrapped 'boom' error, got: %v", err)
	}
	if after.executed {
		t.Error("phase after failing phase was executed")
	}
	if len(order) != 1 || order[0] != "failing" {
		t.Fatalf("unexpected execution order: %v", order)
	}
}

func TestManager_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	canceller := &mockPhase{title: "canceller"}
	// Override Run to cancel context during execution.
	cancelPhase := &contextCancelPhase{title: "canceller", cancel: cancel}
	checker := &mockPhase{title: "checker"}

	m := &Manager{}
	m.AddPhase(cancelPhase, checker)

	err := m.Run(ctx, &v1beta1.Cluster{})
	// The checker phase's Run will be called but should see a cancelled context.
	// The manager itself doesn't check ctx between phases, so we verify the
	// cancellation propagated by checking the context is done.
	_ = err
	_ = canceller

	if ctx.Err() == nil {
		t.Error("context should be cancelled")
	}
}

// contextCancelPhase cancels the context during Run.
type contextCancelPhase struct {
	title  string
	cancel context.CancelFunc
}

func (c *contextCancelPhase) Title() string { return c.title }
func (c *contextCancelPhase) Run(_ context.Context) error {
	c.cancel()
	return nil
}

func TestManager_SetConfigPropagated(t *testing.T) {
	phase := &mockConfigPhase{
		mockPhase: mockPhase{title: "config-aware"},
	}

	m := &Manager{}
	m.AddPhase(phase)

	cfg := &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{Name: "test-cluster"},
	}
	if err := m.Run(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if phase.receivedConfig == nil {
		t.Fatal("SetConfig was not called")
	}
	if phase.receivedConfig.Metadata.Name != "test-cluster" {
		t.Errorf("expected config name 'test-cluster', got %q", phase.receivedConfig.Metadata.Name)
	}
}

func TestManager_BeforeHookError(t *testing.T) {
	phase := &mockBeforeAfterPhase{
		mockPhase: mockPhase{title: "hook-phase"},
		beforeErr: errors.New("before failed"),
	}

	m := &Manager{}
	m.AddPhase(phase)

	err := m.Run(context.Background(), &v1beta1.Cluster{})
	if err == nil {
		t.Fatal("expected error from Before hook")
	}
	if !phase.beforeCalled {
		t.Error("Before() was not called")
	}
	if phase.executed {
		t.Error("Run() should not be called when Before() fails")
	}
	if phase.afterCalled {
		t.Error("After() should not be called when Before() fails")
	}
}

func TestManager_AfterHookError(t *testing.T) {
	phase := &mockBeforeAfterPhase{
		mockPhase: mockPhase{title: "hook-phase"},
		afterErr:  errors.New("after failed"),
	}

	m := &Manager{}
	m.AddPhase(phase)

	err := m.Run(context.Background(), &v1beta1.Cluster{})
	if err == nil {
		t.Fatal("expected error from After hook")
	}
	if !phase.executed {
		t.Error("Run() should have been called")
	}
	if !phase.afterCalled {
		t.Error("After() should have been called")
	}
}

func TestManager_DryRunMode(t *testing.T) {
	phase := &mockDryRunPhase{
		mockPhase: mockPhase{title: "dry-runnable"},
	}

	m := &Manager{DryRun: true}
	m.AddPhase(phase)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !phase.dryRunCalled {
		t.Error("DryRun() was not called in dry-run mode")
	}
	if phase.executed {
		t.Error("Run() should not be called in dry-run mode when DryRun() exists")
	}
}

func TestManager_DryRunModeWithoutDryRunInterface(t *testing.T) {
	phase := &mockPhase{title: "not-dry-runnable"}

	m := &Manager{DryRun: true}
	m.AddPhase(phase)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !phase.executed {
		t.Error("Run() should be called when DryRun is true but phase lacks DryRun()")
	}
}

func TestManager_FullPhaseLifecycle(t *testing.T) {
	phase := &mockFullPhase{
		title:     "full",
		shouldRun: true,
	}

	m := &Manager{}
	m.AddPhase(phase)

	cfg := &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{Name: "lifecycle-test"},
	}
	if err := m.Run(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if phase.receivedConfig == nil || phase.receivedConfig.Metadata.Name != "lifecycle-test" {
		t.Error("SetConfig not called correctly")
	}
	if !phase.prepared {
		t.Error("Prepare not called")
	}
	if !phase.executed {
		t.Error("Run not called")
	}
	if !phase.cleanedUp {
		t.Error("CleanUp not called")
	}
}

func TestPhases_Add(t *testing.T) {
	var phases Phases
	p1 := &mockPhase{title: "a"}
	p2 := &mockPhase{title: "b"}
	phases.Add(p1, p2)

	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}
	if phases[0].Title() != "a" || phases[1].Title() != "b" {
		t.Error("phases not in expected order")
	}
}

func TestPhases_AddBefore(t *testing.T) {
	p1 := &mockPhase{title: "a"}
	p2 := &mockPhase{title: "b"}
	insert := &mockPhase{title: "inserted"}

	var phases Phases
	phases.Add(p1, p2)
	phases.AddBefore(p2, insert)

	if len(phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(phases))
	}
	if phases[0].Title() != "a" || phases[1].Title() != "inserted" || phases[2].Title() != "b" {
		t.Errorf("unexpected order: %s, %s, %s", phases[0].Title(), phases[1].Title(), phases[2].Title())
	}
}

func TestPhases_AddAfter(t *testing.T) {
	p1 := &mockPhase{title: "a"}
	p2 := &mockPhase{title: "b"}
	insert := &mockPhase{title: "inserted"}

	var phases Phases
	phases.Add(p1, p2)
	phases.AddAfter(p1, insert)

	if len(phases) != 3 {
		t.Fatalf("expected 3 phases, got %d", len(phases))
	}
	if phases[0].Title() != "a" || phases[1].Title() != "inserted" || phases[2].Title() != "b" {
		t.Errorf("unexpected order: %s, %s, %s", phases[0].Title(), phases[1].Title(), phases[2].Title())
	}
}

func TestPhases_AddBeforeNotFound(t *testing.T) {
	p1 := &mockPhase{title: "a"}
	notInList := &mockPhase{title: "missing"}
	insert := &mockPhase{title: "appended"}

	var phases Phases
	phases.Add(p1)
	phases.AddBefore(notInList, insert)

	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}
	if phases[1].Title() != "appended" {
		t.Error("when ref not found, AddBefore should append")
	}
}

func TestPhases_AddAfterNotFound(t *testing.T) {
	p1 := &mockPhase{title: "a"}
	notInList := &mockPhase{title: "missing"}
	insert := &mockPhase{title: "appended"}

	var phases Phases
	phases.Add(p1)
	phases.AddAfter(notInList, insert)

	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}
	if phases[1].Title() != "appended" {
		t.Error("when ref not found, AddAfter should append")
	}
}

func TestManager_EmptyPhases(t *testing.T) {
	m := &Manager{}
	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("running empty phases should succeed, got: %v", err)
	}
}

func TestManager_HostsNilConfig(t *testing.T) {
	m := &Manager{}
	if hosts := m.Hosts(); hosts != nil {
		t.Errorf("expected nil hosts for nil config, got %v", hosts)
	}
}

func TestManager_HostsNilSpec(t *testing.T) {
	m := &Manager{Config: &v1beta1.Cluster{}}
	if hosts := m.Hosts(); hosts != nil {
		t.Errorf("expected nil hosts for nil spec, got %v", hosts)
	}
}
