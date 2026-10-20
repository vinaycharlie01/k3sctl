package phase

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
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
	log.Infof("%s: upgrading K3s agent from %s to %s", h, h.Metadata.K3sRunningVersion, p.Config.Spec.K3s.Version)

	if !p.NoDrain && p.leader != nil {
		log.Infof("%s: cordoning node", h)
		if err := h.CordonNode(p.leader); err != nil {
			log.Warnf("%s: cordon failed: %v", h, err)
		}

		log.Infof("%s: draining node", h)
		if err := h.DrainNode(p.leader, p.Config.Spec.Options.Drain); err != nil {
			log.Warnf("%s: drain failed: %v", h, err)
		}
	}

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: get service: %w", h, err)
	}

	log.Infof("%s: stopping K3s agent", h)
	if err := svc.Stop(ctx); err != nil {
		return fmt.Errorf("%s: stop service: %w", h, err)
	}

	if err := retry.Context(ctx, node.ServiceStoppedFunc(h, h.K3sServiceName()),
		retry.Timeout(2*time.Minute), retry.Interval(5*time.Second)); err != nil {
		return fmt.Errorf("%s: wait for stop: %w", h, err)
	}

	binaryDir := h.Configurer.K3sBinaryPath()
	stagedPath := fmt.Sprintf("/tmp/k3sctl-staged-k3s-%s", h.Metadata.Arch)
	if h.Sudo().FS().FileExist(stagedPath) {
		if err := h.Sudo().Exec(fmt.Sprintf("install -m 0755 %s %s",
			shellQuote(stagedPath),
			shellQuote(binaryDir),
		)); err != nil {
			return fmt.Errorf("%s: install binary: %w", h, err)
		}
		_ = h.Sudo().FS().Remove(stagedPath)
	}

	log.Infof("%s: starting K3s agent", h)
	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("%s: start service: %w", h, err)
	}

	if !p.NoDrain && p.leader != nil {
		log.Infof("%s: uncordoning node", h)
		if err := h.UncordonNode(p.leader); err != nil {
			log.Warnf("%s: uncordon failed: %v", h, err)
		}
	}

	h.Metadata.K3sRunningVersion = p.Config.Spec.K3s.Version
	log.Infof("%s: K3s agent upgraded", h)

	return nil
}

func (p *UpgradeAgents) CleanUp() {
	if p.leader == nil {
		return
	}
	for _, h := range p.hosts {
		if err := h.UncordonNode(p.leader); err != nil {
			log.Debugf("%s: cleanup uncordon: %v", h, err)
		}
	}
}
