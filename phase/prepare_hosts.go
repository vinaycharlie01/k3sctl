package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/configurer"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type PrepareHosts struct {
	GenericPhase
}

func (p *PrepareHosts) Title() string {
	return "Prepare hosts"
}

func (p *PrepareHosts) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		if prep, ok := h.Configurer.(configurer.Preparable); ok {
			if err := prep.Prepare(h); err != nil {
				return fmt.Errorf("%s: prepare host: %w", h, err)
			}
			log.Infof("%s: host prepared", h)
		}
		return nil
	})
}
