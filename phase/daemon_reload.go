package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type DaemonReload struct {
	GenericPhase
}

func (p *DaemonReload) Title() string {
	return "Reload systemd daemon"
}

func (p *DaemonReload) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		if err := h.Sudo().Exec("systemctl daemon-reload"); err != nil {
			return fmt.Errorf("%s: daemon-reload failed: %w", h, err)
		}
		log.Debugf("%s: systemd daemon reloaded", h)
		return nil
	})
}
