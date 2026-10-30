package phase

import (
	"context"
	"strings"

	"fmt"
	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

// etcdArm64SupportedSince is the first k3s release bundling etcd 3.5, which
// added arm64 support. Older releases need the override on arm64 as well.
const etcdArm64SupportedSince = "v1.22.1"

// PrepareArm enables the ETCD_UNSUPPORTED_ARCH override on ARM servers whose
// embedded etcd build does not natively support the host architecture.
type PrepareArm struct {
	GenericPhase
	hosts cluster.Hosts
}

func (p *PrepareArm) Title() string {
	return "Prepare ARM nodes"
}

func (p *PrepareArm) Prepare() {
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return !h.Reset && h.IsServer() && needsEtcdUnsupportedArch(h.Metadata.Arch, p.Config.Spec.K3s.Version)
	})
}

func (p *PrepareArm) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *PrepareArm) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.hosts, func(_ context.Context, h *cluster.Host) error {
		arch := h.Metadata.Arch
		slog.Warn(fmt.Sprintf("%s: enabling ETCD_UNSUPPORTED_ARCH=%s; the embedded etcd does not support this architecture natively", h, arch))
		if h.Environment == nil {
			h.Environment = make(map[string]string)
		}
		h.Environment["ETCD_UNSUPPORTED_ARCH"] = arch
		return nil
	})
}

// needsEtcdUnsupportedArch reports whether the embedded etcd requires the
// ETCD_UNSUPPORTED_ARCH escape hatch for the given architecture and k3s
// version. 32-bit ARM is never supported natively; arm64 is supported since
// etcd 3.5 (k3s v1.22.1).
func needsEtcdUnsupportedArch(arch, version string) bool {
	arch = strings.TrimSpace(arch)
	if !strings.HasPrefix(arch, "arm") && !strings.HasPrefix(arch, "aarch") {
		return false
	}
	if strings.HasSuffix(arch, "64") {
		return normalizeVersion(version) < normalizeVersion(etcdArm64SupportedSince)
	}
	return true
}
