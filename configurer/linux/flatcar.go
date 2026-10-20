package linux

import (
	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/vinaycharlie01/k3sctl/configurer"
)

type Flatcar struct {
	BaseLinux
}

var _ configurer.Configurer = (*Flatcar)(nil)

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "flatcar"
		},
		func() any {
			fc := &Flatcar{}
			fc.SetPath("K3sBinaryPath", "/opt/bin/k3s")
			return fc
		},
	)
}
