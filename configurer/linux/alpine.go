package linux

import (
	"context"

	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/vinaycharlie01/k3sctl/configurer"
)

type BaseLinux struct {
	configurer.Linux
}

type Alpine struct {
	BaseLinux
}

var _ configurer.Configurer = (*Alpine)(nil)

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "alpine"
		},
		func() any {
			return &Alpine{}
		},
	)
}

func (l *Alpine) Prepare(h configurer.Host) error {
	ctx := context.Background()
	pm := h.Sudo().PackageManager()
	if err := pm.Update(ctx); err != nil {
		return err
	}
	return pm.Install(ctx, "findutils", "coreutils")
}
