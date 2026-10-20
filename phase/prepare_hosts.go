package phase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	rig "github.com/k0sproject/rig/v2"
	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/configurer"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/vinaycharlie01/k3sctl/pkg/retry"
)

type PrepareHosts struct {
	GenericPhase
}

func (p *PrepareHosts) Title() string {
	return "Prepare hosts"
}

func (p *PrepareHosts) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		return p.prepareHost(ctx, h)
	})
}

func (p *PrepareHosts) prepareHost(ctx context.Context, h *cluster.Host) error {
	if prep, ok := h.Configurer.(configurer.Preparable); ok {
		if err := prep.Prepare(h); err != nil {
			return fmt.Errorf("%s: prepare host: %w", h, err)
		}
		log.Infof("%s: host prepared", h)
	}

	if len(h.Environment) > 0 {
		log.Infof("%s: updating environment", h)
		if err := p.updateEnvironment(ctx, h); err != nil {
			return fmt.Errorf("%s: update environment: %w", h, err)
		}
	}

	return nil
}

// updateEnvironment writes the host's environment to /etc/environment and, for
// SSH connections, reconnects so that subsequent commands see the new values.
func (p *PrepareHosts) updateEnvironment(ctx context.Context, h *cluster.Host) error {
	if err := h.Configurer.UpdateEnvironment(h, h.Environment); err != nil {
		return err
	}
	if h.Protocol() != "ssh" {
		return nil
	}

	log.Infof("%s: reconnecting to apply new environment", h)
	h.Disconnect()
	return retry.Context(ctx, func(ctx context.Context) error {
		if err := h.Connect(ctx); err != nil {
			if errors.Is(err, rig.ErrAbort) || strings.Contains(err.Error(), "host key mismatch") {
				return errors.Join(retry.ErrAbort, err)
			}
			return fmt.Errorf("reconnect: %w", err)
		}
		return nil
	}, retry.Timeout(10*time.Minute))
}
