package linux

import (
	"strings"

	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/vinaycharlie01/k3sctl/configurer"
)

type CoreOS struct {
	BaseLinux
}

var _ configurer.Configurer = (*CoreOS)(nil)

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return strings.Contains(r.Name, "CoreOS") && (r.ID == "fedora" || r.ID == "rhel")
		},
		func() any {
			return &CoreOS{}
		},
	)
}
