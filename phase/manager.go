package phase

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

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

func (m *Manager) Run(ctx context.Context, config *v1beta1.Cluster) error {
	m.Config = config
	for _, phase := range m.Phases {
		if err := m.runPhase(ctx, phase); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) runPhase(ctx context.Context, phase Phase) error {
	if wc, ok := phase.(withconfig); ok {
		wc.SetConfig(m.Config)
	}

	if p, ok := phase.(interface{ SetManager(*Manager) }); ok {
		p.SetManager(m)
	}

	if wp, ok := phase.(withprepare); ok {
		wp.Prepare()
	}

	if cond, ok := phase.(conditional); ok {
		if !cond.ShouldRun() {
			return nil
		}
	}

	title := phase.Title()
	log.Infof("==> Running phase: %s", title)
	start := time.Now()

	if wb, ok := phase.(withbefore); ok {
		if err := wb.Before(ctx); err != nil {
			return fmt.Errorf("phase %s before hook: %w", title, err)
		}
	}

	if m.DryRun {
		if dr, ok := phase.(withdryrun); ok {
			if err := dr.DryRun(); err != nil {
				return fmt.Errorf("phase %s dry run: %w", title, err)
			}
			return nil
		}
	}

	if err := phase.Run(ctx); err != nil {
		return fmt.Errorf("phase %s: %w", title, err)
	}

	if wa, ok := phase.(withafter); ok {
		if err := wa.After(ctx); err != nil {
			return fmt.Errorf("phase %s after hook: %w", title, err)
		}
	}

	if wc, ok := phase.(withcleanup); ok {
		defer wc.CleanUp()
	}

	log.Infof("==> Phase %s completed in %s", title, time.Since(start).Round(time.Millisecond))
	return nil
}
