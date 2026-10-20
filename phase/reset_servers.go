package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ResetServers struct {
	GenericPhase
	NoDrain  bool
	NoDelete bool
	hosts    cluster.Hosts
	leader   *cluster.Host
}

func (p *ResetServers) Title() string {
	return "Reset K3s servers"
}

func (p *ResetServers) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.IsServer() && h.Reset && h != p.leader
	})
}

func (p *ResetServers) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *ResetServers) Run(ctx context.Context) error {
	for _, h := range p.hosts {
		if err := p.resetServer(ctx, h); err != nil {
			return err
		}
	}
	return nil
}

func (p *ResetServers) resetServer(ctx context.Context, h *cluster.Host) error {
	log.Infof("%s: resetting K3s server", h)

	if !p.NoDrain && p.leader != nil {
		if err := h.CordonNode(p.leader); err != nil {
			log.Warnf("%s: cordon failed: %v", h, err)
		}
		if err := h.DrainNode(p.leader, p.Config.Spec.Options.GetDrainOptions()); err != nil {
			log.Warnf("%s: drain failed: %v", h, err)
		}
	}

	if !p.NoDelete && p.leader != nil {
		if err := h.DeleteNode(p.leader); err != nil {
			log.Warnf("%s: delete node failed: %v", h, err)
		}
	}

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err == nil {
		if err := svc.Stop(ctx); err != nil {
			log.Warnf("%s: stop service failed: %v", h, err)
		}
	}

	if err := h.Sudo().Exec(fmt.Sprintf("rm -rf %s", shellQuote(h.K3sDataDir()))); err != nil {
		log.Warnf("%s: remove data dir failed: %v", h, err)
	}

	configPath := h.Configurer.K3sConfigPath()
	if h.Sudo().FS().FileExist(configPath) {
		if err := h.Sudo().FS().Remove(configPath); err != nil {
			log.Warnf("%s: remove config failed: %v", h, err)
		}
	}

	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", h.K3sServiceName())
	if h.Sudo().FS().FileExist(unitPath) {
		if err := h.Sudo().Exec(fmt.Sprintf("systemctl disable %s", h.K3sServiceName())); err != nil {
			log.Warnf("%s: disable service failed: %v", h, err)
		}
		if err := h.Sudo().FS().Remove(unitPath); err != nil {
			log.Warnf("%s: remove service unit failed: %v", h, err)
		}
	}

	binaryPath := h.Configurer.K3sBinaryPath()
	if h.Sudo().FS().FileExist(binaryPath) {
		if err := h.Sudo().FS().Remove(binaryPath); err != nil {
			log.Warnf("%s: remove binary failed: %v", h, err)
		}
	}

	log.Infof("%s: K3s server reset complete", h)
	return nil
}
