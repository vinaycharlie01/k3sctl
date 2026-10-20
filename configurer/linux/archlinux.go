package linux

import (
	"slices"

	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/vinaycharlie01/k3sctl/configurer"
)

type Archlinux struct {
	configurer.Linux
}

var _ configurer.Configurer = (*Archlinux)(nil)

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "arch" || slices.Contains(r.IDLike, "arch")
		},
		func() any {
			return &Archlinux{}
		},
	)
}
