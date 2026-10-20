package phase

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type PrepareArm struct {
	GenericPhase
	hosts cluster.Hosts
}

func (p *PrepareArm) Title() string {
	return "Prepare ARM nodes"
}

func (p *PrepareArm) Prepare() {
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		if h.Reset {
			return false
		}

		arch := h.Metadata.Arch
		if arch == "" {
			return false
		}

		if !strings.HasPrefix(arch, "arm") && !strings.HasPrefix(arch, "aarch") {
			return false
		}

		if strings.HasSuffix(arch, "64") {
			return false
		}

		return true
	})
}

func (p *PrepareArm) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *PrepareArm) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		arch := h.Metadata.Arch
		log.Warnf("%s: enabling ETCD_UNSUPPORTED_ARCH=%s override for 32-bit ARM", h, arch)
		if h.Environment == nil {
			h.Environment = make(map[string]string)
		}
		h.Environment["ETCD_UNSUPPORTED_ARCH"] = arch
		return nil
	})
}
