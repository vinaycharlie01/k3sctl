package phase

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/vinaycharlie01/k3sctl/pkg/node"
	"github.com/vinaycharlie01/k3sctl/pkg/retry"
)

type Reinstall struct {
	GenericPhase
	NoDrain bool
	hosts   cluster.Hosts
	leader  *cluster.Host
}

func (p *Reinstall) Title() string {
	return "Reinstall K3s on hosts with changed flags"
}

func (p *Reinstall) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.Metadata.K3sServiceRunning && h.Metadata.NeedsReinstall
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
	log.Infof("%s: reinstalling K3s service with updated flags", h)

	if h.IsAgent() && !p.NoDrain && p.leader != nil {
		if err := h.CordonNode(p.leader); err != nil {
			log.Warnf("%s: cordon failed: %v", h, err)
		}
		if err := h.DrainNode(p.leader, p.Config.Spec.Options.GetDrainOptions()); err != nil {
			log.Warnf("%s: drain failed: %v", h, err)
		}
		defer func() {
			if err := h.UncordonNode(p.leader); err != nil {
				log.Warnf("%s: uncordon failed: %v", h, err)
			}
		}()
	}

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
	log.Infof("%s: K3s service reinstalled", h)
	return nil
}
