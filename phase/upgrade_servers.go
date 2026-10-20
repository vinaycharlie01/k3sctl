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

type UpgradeServers struct {
	GenericPhase
	hosts cluster.Hosts
}

func (p *UpgradeServers) Title() string {
	return "Upgrade K3s servers"
}

func (p *UpgradeServers) Prepare() {
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.IsServer() && !h.Reset && h.Metadata.NeedsUpgrade
	})
}

func (p *UpgradeServers) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *UpgradeServers) Run(ctx context.Context) error {
	for _, h := range p.hosts {
		if err := p.upgradeServer(ctx, h); err != nil {
			return err
		}
	}
	return nil
}

func (p *UpgradeServers) upgradeServer(ctx context.Context, h *cluster.Host) error {
	log.Infof("%s: upgrading K3s server from %s to %s", h, h.Metadata.K3sRunningVersion, p.Config.Spec.K3s.Version)

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: get service: %w", h, err)
	}

	log.Infof("%s: stopping K3s service", h)
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

	log.Infof("%s: starting K3s service", h)
	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("%s: start service: %w", h, err)
	}

	log.Infof("%s: waiting for K3s server to be ready", h)
	if err := retry.Context(ctx, node.KubeNodeReadyFunc(h),
		retry.Timeout(5*time.Minute), retry.Interval(10*time.Second)); err != nil {
		return fmt.Errorf("%s: wait for ready: %w", h, err)
	}

	h.Metadata.K3sRunningVersion = p.Config.Spec.K3s.Version
	log.Infof("%s: K3s server upgraded", h)

	return nil
}
