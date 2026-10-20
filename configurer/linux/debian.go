package linux

import (
	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/vinaycharlie01/k3sctl/configurer"
)

type Debian struct {
	configurer.Linux
}

var _ configurer.Configurer = (*Debian)(nil)

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "debian"
		},
		func() any {
			return &Debian{}
		},
	)
}
