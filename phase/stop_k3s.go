package phase

import (
	"context"
	"fmt"
	"time"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/vinaycharlie01/k3sctl/pkg/node"
	"github.com/vinaycharlie01/k3sctl/pkg/retry"
)

type StopK3s struct {
	GenericPhase
	leader *cluster.Host
}

func (p *StopK3s) Title() string {
	return "Stop K3s service"
}

func (p *StopK3s) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *StopK3s) ShouldRun() bool {
	return p.leader != nil && p.leader.Metadata.K3sServiceRunning
}

func (p *StopK3s) Run(ctx context.Context) error {
	h := p.leader

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: get service: %w", h, err)
	}

	slog.Info(fmt.Sprintf("%s: stopping K3s service for restore", h))
	if err := svc.Stop(ctx); err != nil {
		return fmt.Errorf("%s: stop service: %w", h, err)
	}

	if err := retry.Context(ctx, node.ServiceStoppedFunc(h, h.K3sServiceName()),
		retry.Timeout(2*time.Minute), retry.Interval(5*time.Second)); err != nil {
		return fmt.Errorf("%s: wait for stop: %w", h, err)
	}

	h.Metadata.K3sServiceRunning = false
	return nil
}
