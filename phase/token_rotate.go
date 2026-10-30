package phase

import (
	"context"
	"fmt"
	"time"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type TokenRotate struct {
	GenericPhase
	leader *cluster.Host
}

func (p *TokenRotate) Title() string {
	return "Rotate K3s token"
}

func (p *TokenRotate) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *TokenRotate) ShouldRun() bool {
	cfg := p.Config.Spec.K3s
	return cfg != nil && cfg.Token != nil && cfg.Token.Rotate
}

func (p *TokenRotate) Run(ctx context.Context) error {
	h := p.leader
	if h == nil || !h.Metadata.K3sServiceRunning {
		return fmt.Errorf("leader server not available for token rotation")
	}

	slog.Info(fmt.Sprintf("%s: rotating cluster token", h))

	cmd := h.Configurer.K3sCmdf("token rotate --new-token %s", shellQuote(generateNewToken()))
	if err := h.Sudo().Exec(cmd); err != nil {
		return fmt.Errorf("%s: token rotation failed: %w", h, err)
	}

	slog.Info(fmt.Sprintf("%s: restarting all servers after token rotation", h))
	servers := p.Config.Spec.Hosts.Servers()
	if err := p.parallelDo(ctx, servers, func(ctx context.Context, h *cluster.Host) error {
		if !h.Metadata.K3sServiceRunning {
			return nil
		}
		svc, err := h.Sudo().Service(h.K3sServiceName())
		if err != nil {
			return fmt.Errorf("%s: find service: %w", h, err)
		}
		if err := svc.Restart(ctx); err != nil {
			return fmt.Errorf("%s: restart: %w", h, err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("restart servers after token rotation: %w", err)
	}

	time.Sleep(10 * time.Second)

	slog.Info(fmt.Sprintf("%s: restarting all agents after token rotation", h))
	agents := p.Config.Spec.Hosts.Agents()
	if err := p.parallelDo(ctx, agents, func(ctx context.Context, h *cluster.Host) error {
		if !h.Metadata.K3sServiceRunning {
			return nil
		}
		svc, err := h.Sudo().Service(h.K3sServiceName())
		if err != nil {
			return fmt.Errorf("%s: find service: %w", h, err)
		}
		if err := svc.Restart(ctx); err != nil {
			return fmt.Errorf("%s: restart: %w", h, err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("restart agents after token rotation: %w", err)
	}

	slog.Info(fmt.Sprintf("%s: token rotation complete", h))
	return nil
}

func generateNewToken() string {
	token, err := generateClusterToken()
	if err != nil {
		return fmt.Sprintf("rotated-%d", time.Now().Unix())
	}
	return token
}
