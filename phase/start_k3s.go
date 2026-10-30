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

type StartK3s struct {
	GenericPhase
	leader *cluster.Host
}

func (p *StartK3s) Title() string {
	return "Start K3s service"
}

func (p *StartK3s) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *StartK3s) ShouldRun() bool {
	return p.leader != nil && !p.leader.Metadata.K3sServiceRunning
}

func (p *StartK3s) Run(ctx context.Context) error {
	h := p.leader

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: get service: %w", h, err)
	}

	slog.Info(fmt.Sprintf("%s: starting K3s service", h))
	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("%s: start service: %w", h, err)
	}

	slog.Info(fmt.Sprintf("%s: waiting for K3s to be ready", h))
	if err := retry.Context(ctx, node.KubeNodeReadyFunc(h, h.KubernetesNodeName()),
		retry.Timeout(5*time.Minute), retry.Interval(10*time.Second)); err != nil {
		return fmt.Errorf("%s: wait for ready: %w", h, err)
	}

	h.Metadata.K3sServiceRunning = true
	slog.Info(fmt.Sprintf("%s: K3s service started", h))
	return nil
}
