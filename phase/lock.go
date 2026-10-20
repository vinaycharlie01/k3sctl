package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type Lock struct {
	GenericPhase
}

func (p *Lock) Title() string {
	return "Acquire exclusive host lock"
}

func (p *Lock) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		lockFile := h.Configurer.K3sctlLockFilePath()
		if h.Sudo().FS().FileExist(lockFile) {
			content, err := h.Sudo().ExecOutput(fmt.Sprintf("cat %s", shellQuote(lockFile)))
			if err == nil && content != "" {
				return fmt.Errorf("%s: host is locked by another k3sctl operation (lock: %s)", h, content)
			}
		}

		if err := h.Sudo().Exec(fmt.Sprintf("mkdir -p %s && echo k3sctl > %s",
			shellQuote(h.Configurer.K3sDataDirDefaultPath()),
			shellQuote(lockFile),
		)); err != nil {
			return fmt.Errorf("%s: acquire lock: %w", h, err)
		}

		log.Debugf("%s: acquired lock", h)
		return nil
	})
}
