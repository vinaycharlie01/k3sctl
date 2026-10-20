package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/configurer"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"

	_ "github.com/vinaycharlie01/k3sctl/configurer/linux"
	_ "github.com/vinaycharlie01/k3sctl/configurer/linux/enterpriselinux"
)

type DetectOS struct {
	GenericPhase
}

func (p *DetectOS) Title() string {
	return "Detect host operating systems"
}

func (p *DetectOS) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		if h.Client == nil {
			return fmt.Errorf("%s: not connected", h)
		}

		osRelease, err := h.OS()
		if err != nil {
			return fmt.Errorf("%s: detect OS: %w", h, err)
		}

		log.Infof("%s: detected OS: %s %s", h, osRelease.ID, osRelease.Version)

		if h.OSIDOverride != "" {
			osRelease.ID = h.OSIDOverride
		}

		c, err := configurer.ResolveOSModule(osRelease)
		if err != nil {
			return fmt.Errorf("%s: %w", h, err)
		}

		h.Configurer = c
		log.Debugf("%s: using configurer %T", h, c)
		return nil
	})
}
