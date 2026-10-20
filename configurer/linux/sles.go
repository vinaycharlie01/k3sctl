package linux

import (
	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/vinaycharlie01/k3sctl/configurer"
)

type SLES struct {
	BaseLinux
}

var _ configurer.Configurer = (*SLES)(nil)

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "sles"
		},
		func() any {
			return &SLES{}
		},
	)
}
