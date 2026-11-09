package phase

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"log/slog"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/nirantarasi/k3sctl/pkg/node"
	"github.com/nirantarasi/k3sctl/pkg/retry"
)

type Reinstall struct {
	GenericPhase
	NoDrain bool
	hosts   cluster.Hosts
	leader  *cluster.Host
}

func (p *Reinstall) Title() string {
	return "Reconcile K3s on hosts with changed flags or config"
}

func (p *Reinstall) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.Metadata.K3sServiceRunning &&
			(h.Metadata.NeedsReinstall || h.Metadata.NeedsConfigRestart)
	})
}

func (p *Reinstall) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *Reinstall) Run(ctx context.Context) error {
	for _, h := range p.hosts {
		if err := p.reinstall(ctx, h); err != nil {
			return err
		}
	}
	return nil
}

func (p *Reinstall) reinstall(ctx context.Context, h *cluster.Host) error {
	reason := "config changed"
	if h.Metadata.NeedsReinstall {
		reason = "install flags changed"
	}
	slog.Info(fmt.Sprintf("%s: reconciling K3s service (%s)", h, reason))

	kc := p.Config.Metadata.KubeClient
	if h.IsAgent() && !p.NoDrain && kc != nil {
		if err := h.CordonNode(ctx, kc); err != nil {
			slog.Warn(fmt.Sprintf("%s: cordon failed: %v", h, err))
		}
		if err := h.DrainNode(ctx, kc, p.Config.Spec.Options.GetDrainOptions()); err != nil {
			slog.Warn(fmt.Sprintf("%s: drain failed: %v", h, err))
		}
		defer func() {
			if err := h.UncordonNode(ctx, kc); err != nil {
				slog.Warn(fmt.Sprintf("%s: uncordon failed: %v", h, err))
			}
		}()
	}

	if h.Metadata.NeedsReinstall {
		installFlags := h.K3sInstallFlags()

		var unitContent string
		if h.IsServer() {
			unitContent = k3sServerUnit(h.Configurer.K3sBinaryPath(), installFlags, h.Environment)
		} else {
			unitContent = k3sAgentUnit(h.Configurer.K3sBinaryPath(), installFlags, h.Environment)
		}

		unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", h.K3sServiceName())
		if err := h.Sudo().FS().WriteFile(unitPath, []byte(unitContent), fs.FileMode(0644)); err != nil {
			return fmt.Errorf("%s: write service unit: %w", h, err)
		}

		if err := h.Sudo().Exec("systemctl daemon-reload"); err != nil {
			return fmt.Errorf("%s: daemon-reload: %w", h, err)
		}
	}

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: get service: %w", h, err)
	}

	if err := svc.Restart(ctx); err != nil {
		return fmt.Errorf("%s: restart service: %w", h, err)
	}

	if err := retry.Context(ctx, node.ServiceRunningFunc(h, h.K3sServiceName()),
		retry.Timeout(3*time.Minute), retry.Interval(5*time.Second)); err != nil {
		return fmt.Errorf("%s: wait for service: %w", h, err)
	}

	h.Metadata.NeedsReinstall = false
	h.Metadata.NeedsConfigRestart = false
	slog.Info(fmt.Sprintf("%s: K3s service reconciled", h))
	return nil
}
