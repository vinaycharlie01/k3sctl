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

type UpgradeAgents struct {
	GenericPhase
	NoDrain bool
	hosts   cluster.Hosts
	leader  *cluster.Host
}

func (p *UpgradeAgents) Title() string {
	return "Upgrade K3s agents"
}

func (p *UpgradeAgents) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.IsAgent() && !h.Reset && h.Metadata.NeedsUpgrade
	})
}

func (p *UpgradeAgents) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *UpgradeAgents) Run(ctx context.Context) error {
	concurrency := 1
	if p.Config.Spec.Options != nil && p.Config.Spec.Options.Concurrency != nil && p.Config.Spec.Options.Concurrency.Limit > 0 {
		concurrency = p.Config.Spec.Options.Concurrency.Limit
	}

	batchSize := len(p.hosts)
	if batchSize > concurrency {
		batchSize = concurrency
	}

	return p.hosts.BatchedParallelEach(ctx, concurrency, batchSize, func(ctx context.Context, h *cluster.Host) error {
		return p.upgradeAgent(ctx, h)
	})
}

func (p *UpgradeAgents) upgradeAgent(ctx context.Context, h *cluster.Host) error {
	slog.Info(fmt.Sprintf("%s: upgrading K3s agent from %s to %s", h, h.Metadata.K3sRunningVersion, p.Config.Spec.K3s.Version))

	if !p.NoDrain && p.leader != nil {
		slog.Info(fmt.Sprintf("%s: cordoning node", h))
		if err := h.CordonNode(p.leader); err != nil {
			slog.Warn(fmt.Sprintf("%s: cordon failed: %v", h, err))
		}

		slog.Info(fmt.Sprintf("%s: draining node", h))
		if err := h.DrainNode(p.leader, p.Config.Spec.Options.GetDrainOptions()); err != nil {
			slog.Warn(fmt.Sprintf("%s: drain failed: %v", h, err))
		}
	}

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: get service: %w", h, err)
	}

	slog.Info(fmt.Sprintf("%s: stopping K3s agent", h))
	if err := svc.Stop(ctx); err != nil {
		return fmt.Errorf("%s: stop service: %w", h, err)
	}

	if err := retry.Context(ctx, node.ServiceStoppedFunc(h, h.K3sServiceName()),
		retry.Timeout(2*time.Minute), retry.Interval(5*time.Second)); err != nil {
		return fmt.Errorf("%s: wait for stop: %w", h, err)
	}

	binaryDir := h.Configurer.K3sBinaryPath()
	if provider := h.Metadata.BinaryProvider; provider != nil {
		stagedPath := provider.StagedPath()
		if stagedPath != "" && h.Sudo().FS().FileExist(stagedPath) {
			if err := h.Sudo().Exec(fmt.Sprintf("install -m 0755 %s %s",
				shellQuote(stagedPath),
				shellQuote(binaryDir),
			)); err != nil {
				return fmt.Errorf("%s: install binary: %w", h, err)
			}
			_ = h.Sudo().FS().Remove(stagedPath)
		}
	}

	slog.Info(fmt.Sprintf("%s: starting K3s agent", h))
	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("%s: start service: %w", h, err)
	}

	if !p.NoDrain && p.leader != nil {
		slog.Info(fmt.Sprintf("%s: uncordoning node", h))
		if err := h.UncordonNode(p.leader); err != nil {
			slog.Warn(fmt.Sprintf("%s: uncordon failed: %v", h, err))
		}
	}

	h.Metadata.K3sRunningVersion = p.Config.Spec.K3s.Version
	h.Metadata.NeedsUpgrade = false
	slog.Info(fmt.Sprintf("%s: K3s agent upgraded", h))

	return nil
}

func (p *UpgradeAgents) CleanUp() {
	if p.leader == nil {
		return
	}
	for _, h := range p.hosts {
		if err := h.UncordonNode(p.leader); err != nil {
			slog.Debug(fmt.Sprintf("%s: cleanup uncordon: %v", h, err))
		}
	}
}
