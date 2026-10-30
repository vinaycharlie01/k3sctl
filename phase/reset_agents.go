package phase

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ResetAgents struct {
	GenericPhase
	NoDrain  bool
	NoDelete bool
	hosts    cluster.Hosts
	leader   *cluster.Host
}

func (p *ResetAgents) Title() string {
	return "Reset K3s agents"
}

func (p *ResetAgents) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.IsAgent() && h.Reset
	})
}

func (p *ResetAgents) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *ResetAgents) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		return p.resetAgent(ctx, h)
	})
}

func (p *ResetAgents) resetAgent(ctx context.Context, h *cluster.Host) error {
	slog.Info(fmt.Sprintf("%s: resetting K3s agent", h))

	if !p.NoDrain && p.leader != nil {
		if err := h.CordonNode(p.leader); err != nil {
			slog.Warn(fmt.Sprintf("%s: cordon failed: %v", h, err))
		}
		if err := h.DrainNode(p.leader, p.Config.Spec.Options.GetDrainOptions()); err != nil {
			slog.Warn(fmt.Sprintf("%s: drain failed: %v", h, err))
		}
	}

	if !p.NoDelete && p.leader != nil {
		if err := h.DeleteNode(p.leader); err != nil {
			slog.Warn(fmt.Sprintf("%s: delete node failed: %v", h, err))
		}
	}

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err == nil {
		if err := svc.Stop(ctx); err != nil {
			slog.Warn(fmt.Sprintf("%s: stop service failed: %v", h, err))
		}
	}

	if err := h.Sudo().Exec(fmt.Sprintf("rm -rf %s", shellQuote(h.K3sDataDir()))); err != nil {
		slog.Warn(fmt.Sprintf("%s: remove data dir failed: %v", h, err))
	}

	configPath := h.Configurer.K3sConfigPath()
	if h.Sudo().FS().FileExist(configPath) {
		if err := h.Sudo().FS().Remove(configPath); err != nil {
			slog.Warn(fmt.Sprintf("%s: remove config failed: %v", h, err))
		}
	}

	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", h.K3sServiceName())
	if h.Sudo().FS().FileExist(unitPath) {
		if err := h.Sudo().Exec(fmt.Sprintf("systemctl disable %s", h.K3sServiceName())); err != nil {
			slog.Warn(fmt.Sprintf("%s: disable service failed: %v", h, err))
		}
		if err := h.Sudo().FS().Remove(unitPath); err != nil {
			slog.Warn(fmt.Sprintf("%s: remove service unit failed: %v", h, err))
		}
	}

	binaryPath := h.Configurer.K3sBinaryPath()
	if h.Sudo().FS().FileExist(binaryPath) {
		if err := h.Sudo().FS().Remove(binaryPath); err != nil {
			slog.Warn(fmt.Sprintf("%s: remove binary failed: %v", h, err))
		}
	}

	slog.Info(fmt.Sprintf("%s: K3s agent reset complete", h))
	return nil
}
