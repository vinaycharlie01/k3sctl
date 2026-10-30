package phase

import (
	"context"

	"fmt"
	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type Unlock struct {
	GenericPhase
}

func (p *Unlock) Title() string {
	return "Release exclusive host lock"
}

func (p *Unlock) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		lockFile := h.Configurer.K3sctlLockFilePath()
		if h.Sudo().FS().FileExist(lockFile) {
			if err := h.Sudo().FS().Remove(lockFile); err != nil {
				slog.Warn(fmt.Sprintf("%s: failed to release lock: %v", h, err))
				return nil
			}
			slog.Debug(fmt.Sprintf("%s: released lock", h))
		}
		return nil
	})
}
