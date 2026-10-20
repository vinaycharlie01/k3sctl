package phase

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
)

// fullPhase is the union of every optional phase interface. It exists so
// counterfeiter can generate a single fake that exercises all manager hooks.
type fullPhase interface {
	Phase
	withconfig
	withprepare
	conditional
	withcleanup
	withbefore
	withafter
	withdryrun
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeFullPhase -o ./fake_full_phase_test.go . fullPhase

var (
	_ Phase     = (*FakePhase)(nil)
	_ fullPhase = (*FakeFullPhase)(nil)
)

func newTitledPhase(title string) *FakePhase {
	p := &FakePhase{}
	p.TitleReturns(title)
	return p
}

func TestManager_RunPhasesInOrder(t *testing.T) {
	var order []string
	newPhase := func(title string) *FakePhase {
		p := newTitledPhase(title)
		p.RunCalls(func(context.Context) error {
			order = append(order, title)
			return nil
		})
		return p
	}

	m := &Manager{}
	m.AddPhase(newPhase("first"), newPhase("second"), newPhase("third"))

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := []string{"first", "second", "third"}; !slices.Equal(order, want) {
		t.Fatalf("unexpected execution order: %v", order)
	}
}

func TestManager_ConditionalPhaseSkipped(t *testing.T) {
	skipped := &FakeFullPhase{}
	skipped.TitleReturns("skipped")
	skipped.ShouldRunReturns(false)
	executed := newTitledPhase("executed")

	m := &Manager{}
	m.AddPhase(skipped, executed)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if skipped.ShouldRunCallCount() != 1 {
		t.Errorf("ShouldRun() called %d times, want 1", skipped.ShouldRunCallCount())
	}
	if skipped.RunCallCount() != 0 {
		t.Error("conditional phase with ShouldRun()=false was executed")
	}
	if executed.RunCallCount() != 1 {
		t.Error("phase after skipped phase was not executed")
	}
}

func TestManager_ConditionalPhaseRuns(t *testing.T) {
	p := &FakeFullPhase{}
	p.TitleReturns("conditional")
	p.ShouldRunReturns(true)

	m := &Manager{}
	m.AddPhase(p)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.RunCallCount() != 1 {
		t.Error("conditional phase with ShouldRun()=true was not executed")
	}
}

func TestManager_PrepareCalledBeforeRun(t *testing.T) {
	p := &FakeFullPhase{}
	p.TitleReturns("preparable")
	p.ShouldRunReturns(true)

	preparedBeforeRun := false
	p.RunCalls(func(context.Context) error {
		preparedBeforeRun = p.PrepareCallCount() == 1
		return nil
	})

	m := &Manager{}
	m.AddPhase(p)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.PrepareCallCount() != 1 {
		t.Errorf("Prepare() called %d times, want 1", p.PrepareCallCount())
	}
	if !preparedBeforeRun {
		t.Error("Prepare() was not called before Run()")
	}
}

func TestManager_CleanUpNotCalledOnSuccess(t *testing.T) {
	p := &FakeFullPhase{}
	p.TitleReturns("cleanable")
	p.ShouldRunReturns(true)

	m := &Manager{}
	m.AddPhase(p)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.RunCallCount() != 1 {
		t.Errorf("Run() called %d times, want 1", p.RunCallCount())
	}
	if p.CleanUpCallCount() != 0 {
		t.Error("CleanUp() must not run on success: it is a rollback hook")
	}
}

func TestManager_CleanUpCalledForEnteredPhasesOnFailure(t *testing.T) {
	boom := errors.New("boom")

	staged := &FakeFullPhase{}
	staged.TitleReturns("staged")
	staged.ShouldRunReturns(true)

	failing := &FakeFullPhase{}
	failing.TitleReturns("install")
	failing.ShouldRunReturns(true)
	failing.RunReturns(boom)

	skipped := &FakeFullPhase{}
	skipped.TitleReturns("skipped")
	skipped.ShouldRunReturns(false)

	notReached := &FakeFullPhase{}
	notReached.TitleReturns("not-reached")
	notReached.ShouldRunReturns(true)

	m := &Manager{}
	m.AddPhase(staged, failing, skipped, notReached)

	err := m.Run(context.Background(), &v1beta1.Cluster{})
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped 'boom' error, got: %v", err)
	}

	if staged.CleanUpCallCount() != 1 {
		t.Error("CleanUp() was not called for the phase that ran before the failure")
	}
	if failing.CleanUpCallCount() != 1 {
		t.Error("CleanUp() was not called for the failing phase")
	}
	if skipped.RunCallCount() != 0 || skipped.CleanUpCallCount() != 0 {
		t.Error("a phase skipped by ShouldRun() must not run or be cleaned up")
	}
	if notReached.RunCallCount() != 0 || notReached.CleanUpCallCount() != 0 {
		t.Error("a phase after the failure must not run or be cleaned up")
	}
}

func TestManager_ErrorStopsExecution(t *testing.T) {
	boom := errors.New("boom")
	var order []string

	failing := newTitledPhase("failing")
	failing.RunCalls(func(context.Context) error {
		order = append(order, "failing")
		return boom
	})
	after := newTitledPhase("after")
	after.RunCalls(func(context.Context) error {
		order = append(order, "after")
		return nil
	})

	m := &Manager{}
	m.AddPhase(failing, after)

	err := m.Run(context.Background(), &v1beta1.Cluster{})
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped 'boom' error, got: %v", err)
	}
	if after.RunCallCount() != 0 {
		t.Error("phase after failing phase was executed")
	}
	if want := []string{"failing"}; !slices.Equal(order, want) {
		t.Fatalf("unexpected execution order: %v", order)
	}
}

func TestManager_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	canceller := newTitledPhase("canceller")
	canceller.RunCalls(func(context.Context) error {
		cancel()
		return nil
	})
	checker := newTitledPhase("checker")

	m := &Manager{}
	m.AddPhase(canceller, checker)

	_ = m.Run(ctx, &v1beta1.Cluster{})

	if ctx.Err() == nil {
		t.Fatal("context should be cancelled")
	}
	if checker.RunCallCount() != 1 {
		t.Fatalf("checker Run() called %d times, want 1", checker.RunCallCount())
	}
	if err := checker.RunArgsForCall(0).Err(); !errors.Is(err, context.Canceled) {
		t.Errorf("checker received context error %v, want context.Canceled", err)
	}
}

func TestManager_SetConfigPropagated(t *testing.T) {
	p := &FakeFullPhase{}
	p.TitleReturns("config-aware")
	p.ShouldRunReturns(true)

	m := &Manager{}
	m.AddPhase(p)

	cfg := &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{Name: "test-cluster"},
	}
	if err := m.Run(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.SetConfigCallCount() != 1 {
		t.Fatalf("SetConfig() called %d times, want 1", p.SetConfigCallCount())
	}
	if got := p.SetConfigArgsForCall(0); got != cfg {
		t.Errorf("SetConfig received %p, want %p", got, cfg)
	}
	if name := p.SetConfigArgsForCall(0).Metadata.Name; name != "test-cluster" {
		t.Errorf("expected config name 'test-cluster', got %q", name)
	}
}

func TestManager_BeforeHookError(t *testing.T) {
	beforeErr := errors.New("before failed")

	p := &FakeFullPhase{}
	p.TitleReturns("hook-phase")
	p.ShouldRunReturns(true)
	p.BeforeReturns(beforeErr)

	m := &Manager{}
	m.AddPhase(p)

	err := m.Run(context.Background(), &v1beta1.Cluster{})
	if !errors.Is(err, beforeErr) {
		t.Fatalf("expected wrapped 'before failed' error, got: %v", err)
	}
	if p.BeforeCallCount() != 1 {
		t.Errorf("Before() called %d times, want 1", p.BeforeCallCount())
	}
	if p.RunCallCount() != 0 {
		t.Error("Run() should not be called when Before() fails")
	}
	if p.AfterCallCount() != 0 {
		t.Error("After() should not be called when Before() fails")
	}
	if p.CleanUpCallCount() != 0 {
		t.Error("a phase that was never entered must not be cleaned up")
	}
}

func TestManager_AfterHookError(t *testing.T) {
	afterErr := errors.New("after failed")

	p := &FakeFullPhase{}
	p.TitleReturns("hook-phase")
	p.ShouldRunReturns(true)
	p.AfterReturns(afterErr)

	m := &Manager{}
	m.AddPhase(p)

	err := m.Run(context.Background(), &v1beta1.Cluster{})
	if !errors.Is(err, afterErr) {
		t.Fatalf("expected wrapped 'after failed' error, got: %v", err)
	}
	if p.RunCallCount() != 1 {
		t.Error("Run() should have been called")
	}
	if p.AfterCallCount() != 1 {
		t.Error("After() should have been called")
	}
	if p.CleanUpCallCount() != 1 {
		t.Error("the entered phase should be rolled back when After() fails")
	}
}

func TestManager_DryRunMode(t *testing.T) {
	p := &FakeFullPhase{}
	p.TitleReturns("dry-runnable")
	p.ShouldRunReturns(true)

	m := &Manager{DryRun: true}
	m.AddPhase(p)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.DryRunCallCount() != 1 {
		t.Error("DryRun() was not called in dry-run mode")
	}
	if p.RunCallCount() != 0 {
		t.Error("Run() should not be called in dry-run mode when DryRun() exists")
	}
	if p.BeforeCallCount() != 1 {
		t.Error("Before() should still run in dry-run mode")
	}
	if p.CleanUpCallCount() != 0 {
		t.Error("CleanUp() must not run in dry-run mode")
	}
}

func TestManager_DryRunModeWithoutDryRunInterface(t *testing.T) {
	p := newTitledPhase("not-dry-runnable")

	m := &Manager{DryRun: true}
	m.AddPhase(p)

	if err := m.Run(context.Background(), &v1beta1.Cluster{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.RunCallCount() != 1 {
		t.Error("Run() should be called when DryRun is true but phase lacks DryRun()")
	}
}

func TestManager_FullPhaseLifecycle(t *testing.T) {
	p := &FakeFullPhase{}
	p.TitleReturns("full")
	p.ShouldRunReturns(true)

	m := &Manager{}
	m.AddPhase(p)

	cfg := &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{Name: "lifecycle-test"},
	}
	if err := m.Run(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"Prepare", p.PrepareCallCount(), 1},
		{"ShouldRun", p.ShouldRunCallCount(), 1},
		{"Before", p.BeforeCallCount(), 1},
		{"Run", p.RunCallCount(), 1},
		{"After", p.AfterCallCount(), 1},
		{"DryRun", p.DryRunCallCount(), 0},
		// CleanUp is a rollback hook, so a successful run must not call it.
		{"CleanUp", p.CleanUpCallCount(), 0},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s() called %d times, want %d", tt.name, tt.got, tt.want)
		}
	}

	if got := p.SetConfigCallCount(); got != 1 {
		t.Fatalf("SetConfig() called %d times, want 1", got)
	}
	if got := p.SetConfigArgsForCall(0); got != cfg {
		t.Errorf("SetConfig received %p, want %p", got, cfg)
	}
	if name := p.SetConfigArgsForCall(0).Metadata.Name; name != "lifecycle-test" {
		t.Errorf("expected config name 'lifecycle-test', got %q", name)
	}
}

func TestPhases_Add(t *testing.T) {
	var phases Phases
	p1 := newTitledPhase("a")
	p2 := newTitledPhase("b")
	phases.Add(p1, p2)

	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}
	if phases[0].Title() != "a" || phases[1].Title() != "b" {
		t.Error("phases not in expected order")
	}
}

func TestPhases_AddBefore(t *testing.T) {
	p1 := newTitledPhase("a")
	p2 := newTitledPhase("b")
	insert := newTitledPhase("inserted")

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
	p1 := newTitledPhase("a")
	p2 := newTitledPhase("b")
	insert := newTitledPhase("inserted")

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
	p1 := newTitledPhase("a")
	notInList := newTitledPhase("missing")
	insert := newTitledPhase("appended")

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
	p1 := newTitledPhase("a")
	notInList := newTitledPhase("missing")
	insert := newTitledPhase("appended")

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
