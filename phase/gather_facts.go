package phase

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type GatherFacts struct {
	GenericPhase
}

func (p *GatherFacts) Title() string {
	return "Gather host facts"
}

func (p *GatherFacts) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		arch, err := h.Arch()
		if err != nil {
			return fmt.Errorf("%s: detect architecture: %w", h, err)
		}
		h.Metadata.Arch = arch
		slog.Info(fmt.Sprintf("%s: architecture: %s", h, arch))

		hostname, err := h.ExecOutput("hostname -s")
		if err != nil {
			slog.Warn(fmt.Sprintf("%s: failed to detect hostname: %v", h, err))
			hostname = h.Address()
		}
		h.Metadata.Hostname = strings.TrimSpace(hostname)
		slog.Info(fmt.Sprintf("%s: hostname: %s", h, h.Metadata.Hostname))

		machineID, err := h.ExecOutput("cat /etc/machine-id 2>/dev/null || cat /var/lib/dbus/machine-id 2>/dev/null || echo unknown")
		if err == nil {
			h.Metadata.MachineID = strings.TrimSpace(machineID)
		}

		if h.PrivateInterface == "" && h.PrivateAddress == "" {
			iface, err := h.Configurer.PrivateInterface(h)
			if err != nil {
				slog.Warn(fmt.Sprintf("%s: could not detect private interface: %v", h, err))
			} else {
				h.PrivateInterface = iface
				addr, err := h.Configurer.PrivateAddress(h, iface, h.Address())
				if err != nil {
					slog.Warn(fmt.Sprintf("%s: could not detect private address: %v", h, err))
				} else {
					h.PrivateAddress = addr
					slog.Info(fmt.Sprintf("%s: private address: %s (interface: %s)", h, addr, iface))
				}
			}
		}

		return nil
	})
}
