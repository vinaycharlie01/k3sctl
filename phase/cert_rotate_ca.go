package phase

import (
	"context"
	"fmt"
	"time"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type CertRotateCA struct {
	GenericPhase
	leader *cluster.Host
}

func (p *CertRotateCA) Title() string {
	return "Rotate K3s CA certificates"
}

func (p *CertRotateCA) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *CertRotateCA) ShouldRun() bool {
	cfg := p.Config.Spec.K3s
	return cfg != nil && cfg.CertRotate != nil && cfg.CertRotate.RotateCA
}

func (p *CertRotateCA) Run(ctx context.Context) error {
	h := p.leader
	if h == nil || !h.Metadata.K3sServiceRunning {
		return fmt.Errorf("leader server not available for CA rotation")
	}

	slog.Info(fmt.Sprintf("%s: initiating CA certificate rotation", h))

	cmd := h.Configurer.K3sCmdf("certificate rotate-ca --force")
	if err := h.Sudo().Exec(cmd); err != nil {
		return fmt.Errorf("%s: CA certificate rotation failed: %w", h, err)
	}

	slog.Info(fmt.Sprintf("%s: restarting k3s service after CA rotation", h))
	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: find service: %w", h, err)
	}
	if err := svc.Restart(ctx); err != nil {
		return fmt.Errorf("%s: restart after CA rotation: %w", h, err)
	}

	time.Sleep(15 * time.Second)

	slog.Info(fmt.Sprintf("%s: restarting all other servers for CA propagation", h))
	servers := p.Config.Spec.Hosts.Servers()
	if err := p.parallelDo(ctx, servers, func(ctx context.Context, s *cluster.Host) error {
		if s == h || !s.Metadata.K3sServiceRunning {
			return nil
		}
		slog.Info(fmt.Sprintf("%s: restarting for CA propagation", s))
		svc, err := s.Sudo().Service(s.K3sServiceName())
		if err != nil {
			return fmt.Errorf("%s: find service: %w", s, err)
		}
		if err := svc.Restart(ctx); err != nil {
			return fmt.Errorf("%s: restart: %w", s, err)
		}
		return nil
	}); err != nil {
		return err
	}

	time.Sleep(10 * time.Second)

	slog.Info(fmt.Sprintf("%s: restarting all agents for CA propagation", h))
	agents := p.Config.Spec.Hosts.Agents()
	if err := p.parallelDo(ctx, agents, func(ctx context.Context, a *cluster.Host) error {
		if !a.Metadata.K3sServiceRunning {
			return nil
		}
		slog.Info(fmt.Sprintf("%s: restarting for CA propagation", a))
		svc, err := a.Sudo().Service(a.K3sServiceName())
		if err != nil {
			return fmt.Errorf("%s: find service: %w", a, err)
		}
		if err := svc.Restart(ctx); err != nil {
			return fmt.Errorf("%s: restart: %w", a, err)
		}
		return nil
	}); err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("%s: CA certificate rotation complete", h))
	return nil
}
