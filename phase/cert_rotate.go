package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type CertRotate struct {
	GenericPhase
}

func (p *CertRotate) Title() string {
	return "Rotate K3s certificates"
}

func (p *CertRotate) ShouldRun() bool {
	cfg := p.Config.Spec.K3s
	if cfg == nil || cfg.CertRotate == nil {
		return false
	}
	return cfg.CertRotate.Enabled
}

func (p *CertRotate) Run(ctx context.Context) error {
	servers := p.Config.Spec.Hosts.Servers()
	return p.parallelDo(ctx, servers, func(ctx context.Context, h *cluster.Host) error {
		if !h.Metadata.K3sServiceRunning {
			log.Infof("%s: k3s not running, skipping certificate rotation", h)
			return nil
		}

		log.Infof("%s: rotating certificates", h)

		cmd := h.Configurer.K3sCmdf("certificate rotate --force")
		if err := h.Sudo().Exec(cmd); err != nil {
			return fmt.Errorf("%s: certificate rotation failed: %w", h, err)
		}

		log.Infof("%s: restarting k3s after certificate rotation", h)
		svc, err := h.Sudo().Service(h.K3sServiceName())
		if err != nil {
			return fmt.Errorf("%s: cannot find service: %w", h, err)
		}
		if err := svc.Restart(ctx); err != nil {
			return fmt.Errorf("%s: restart after cert rotation failed: %w", h, err)
		}

		log.Infof("%s: certificates rotated successfully", h)
		return nil
	})
}
