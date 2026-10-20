package enterpriselinux

import (
	"strings"

	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/vinaycharlie01/k3sctl/configurer"
	k3slinux "github.com/vinaycharlie01/k3sctl/configurer/linux"
)

type Fedora struct {
	k3slinux.EnterpriseLinux
}

var _ configurer.Configurer = (*Fedora)(nil)

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "fedora" && !strings.Contains(r.Name, "CoreOS")
		},
		func() any {
			return &Fedora{}
		},
	)
}
