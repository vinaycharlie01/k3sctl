package phase

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakePhase -o ./mocks/fake_phase.go . Phase
type Phase interface {
	Run(ctx context.Context) error
	Title() string
}

type withconfig interface {
	SetConfig(config *v1beta1.Cluster)
}

type withprepare interface {
	Prepare()
}

type conditional interface {
	ShouldRun() bool
}

type withcleanup interface {
	CleanUp()
}

type withdryrun interface {
	DryRun() error
}

type withbefore interface {
	Before(ctx context.Context) error
}

type withafter interface {
	After(ctx context.Context) error
}

type withmanager interface {
	SetManager(*Manager)
}

// FullPhase is the union of every optional phase interface. A phase that
// implements all of them exercises every hook the Manager can invoke. It is
// exported so a single fake can be generated for it.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeFullPhase -o ./mocks/fake_full_phase.go . FullPhase
type FullPhase interface {
	Phase
	withconfig
	withprepare
	conditional
	withcleanup
	withbefore
	withafter
	withdryrun
	withmanager
}

type Phases []Phase

func (p *Phases) Add(phases ...Phase) {
	*p = append(*p, phases...)
}

func (p *Phases) AddBefore(ref Phase, phases ...Phase) {
	for i, phase := range *p {
		if phase == ref {
			newPhases := make(Phases, 0, len(*p)+len(phases))
			newPhases = append(newPhases, (*p)[:i]...)
			newPhases = append(newPhases, phases...)
			newPhases = append(newPhases, (*p)[i:]...)
			*p = newPhases
			return
		}
	}
	p.Add(phases...)
}

func (p *Phases) AddAfter(ref Phase, phases ...Phase) {
	for i, phase := range *p {
		if phase == ref {
			newPhases := make(Phases, 0, len(*p)+len(phases))
			newPhases = append(newPhases, (*p)[:i+1]...)
			newPhases = append(newPhases, phases...)
			newPhases = append(newPhases, (*p)[i+1:]...)
			*p = newPhases
			return
		}
	}
	p.Add(phases...)
}

type Manager struct {
	Config            *v1beta1.Cluster
	Phases            Phases
	DryRun            bool
	Concurrency       int
	ConcurrentUploads int
}

func (m *Manager) AddPhase(phases ...Phase) {
	m.Phases.Add(phases...)
}

func (m *Manager) Hosts() cluster.Hosts {
	if m.Config == nil || m.Config.Spec == nil {
		return nil
	}
	return m.Config.Spec.Hosts
}

// Run executes the phases in order. CleanUp is a rollback hook, not a success
// hook: when a phase fails, it is invoked for every phase that was entered
// (including the failing one) so that partially applied changes are undone.
func (m *Manager) Run(ctx context.Context, config *v1beta1.Cluster) error {
	m.Config = config

	var ran Phases
	var result error
	defer func() {
		// Dry runs mutate nothing, and successful runs keep their results.
		if result == nil || m.DryRun {
			return
		}
		for _, phase := range ran {
			if c, ok := phase.(withcleanup); ok {
				log.Infof("Running clean-up for phase: %s", phase.Title())
				c.CleanUp()
			}
		}
	}()

	for _, phase := range m.Phases {
		entered, err := m.runPhase(ctx, phase)
		if entered {
			ran = append(ran, phase)
		}
		if err != nil {
			result = err
			return err
		}
	}
	return nil
}

// runPhase reports whether the phase was entered, so the caller can roll back
// exactly the phases that started.
func (m *Manager) runPhase(ctx context.Context, phase Phase) (bool, error) {
	if wc, ok := phase.(withconfig); ok {
		wc.SetConfig(m.Config)
	}

	if sm, ok := phase.(withmanager); ok {
		sm.SetManager(m)
	}

	if wp, ok := phase.(withprepare); ok {
		wp.Prepare()
	}

	if cond, ok := phase.(conditional); ok {
		if !cond.ShouldRun() {
			return false, nil
		}
	}

	title := phase.Title()
	log.Infof("==> Running phase: %s", title)
	start := time.Now()

	if wb, ok := phase.(withbefore); ok {
		if err := wb.Before(ctx); err != nil {
			return false, fmt.Errorf("phase %s before hook: %w", title, err)
		}
	}

	if m.DryRun {
		if dr, ok := phase.(withdryrun); ok {
			if err := dr.DryRun(); err != nil {
				return false, fmt.Errorf("phase %s dry run: %w", title, err)
			}
			return false, nil
		}
	}

	if err := phase.Run(ctx); err != nil {
		return true, fmt.Errorf("phase %s: %w", title, err)
	}

	if wa, ok := phase.(withafter); ok {
		if err := wa.After(ctx); err != nil {
			return true, fmt.Errorf("phase %s after hook: %w", title, err)
		}
	}

	log.Infof("==> Phase %s completed in %s", title, time.Since(start).Round(time.Millisecond))
	return true, nil
}
