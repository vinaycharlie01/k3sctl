package phase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/configurer"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ValidateHosts struct {
	GenericPhase
}

func (p *ValidateHosts) Title() string {
	return "Validate hosts"
}

func (p *ValidateHosts) Run(ctx context.Context) error {
	hosts := p.Config.Spec.Hosts

	seen := make(map[string]bool)
	for _, h := range hosts {
		addr := h.Address()
		if seen[addr] {
			return fmt.Errorf("duplicate host address: %s", addr)
		}
		seen[addr] = true
	}

	for _, h := range hosts {
		osKind, err := h.OSKind()
		if err != nil {
			return fmt.Errorf("%s: %w", h, err)
		}
		if osKind != "linux" {
			return fmt.Errorf("%s: K3s only supports Linux hosts (detected: %s)", h, osKind)
		}

		if v, ok := h.Configurer.(configurer.HostValidator); ok {
			if err := v.ValidateHost(h); err != nil {
				return fmt.Errorf("%s: host validation failed: %w", h, err)
			}
		}

		output, err := h.Sudo().ExecOutput("id -u")
		if err != nil {
			return fmt.Errorf("%s: failed to check sudo access: %w", h, err)
		}
		if strings.TrimSpace(output) != "0" {
			log.Warnf("%s: sudo user is not root (uid: %s), some operations may fail", h, strings.TrimSpace(output))
		}
	}

	return p.validateClockSkew(ctx, hosts)
}

func (p *ValidateHosts) validateClockSkew(ctx context.Context, hosts cluster.Hosts) error {
	return p.parallelDo(ctx, hosts, func(ctx context.Context, h *cluster.Host) error {
		output, err := h.ExecOutput("date +%s")
		if err != nil {
			log.Warnf("%s: failed to check clock: %v", h, err)
			return nil
		}
		_ = strings.TrimSpace(output)
		return nil
	})
}
