package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ResetLeader struct {
	GenericPhase
	leader *cluster.Host
}

func (p *ResetLeader) Title() string {
	return "Reset K3s leader"
}

func (p *ResetLeader) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *ResetLeader) ShouldRun() bool {
	return p.leader != nil && p.leader.Reset
}

func (p *ResetLeader) Run(ctx context.Context) error {
	h := p.leader
	log.Infof("%s: resetting K3s leader", h)

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

	log.Infof("%s: K3s leader reset complete", h)
	return nil
}
