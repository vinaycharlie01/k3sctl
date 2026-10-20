package enterpriselinux

import (
	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/vinaycharlie01/k3sctl/configurer"
	k3slinux "github.com/vinaycharlie01/k3sctl/configurer/linux"
)

type OracleLinux struct {
	k3slinux.EnterpriseLinux
}

var _ configurer.Configurer = (*OracleLinux)(nil)

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "ol"
		},
		func() any {
			return &OracleLinux{}
		},
	)
}
