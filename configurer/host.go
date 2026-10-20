package configurer

import (
	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/cmd"
	"github.com/k0sproject/rig/v2/remotefs"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeHost -o ./mocks/fake_host.go . Host
type Host interface {
	cmd.SimpleRunner
	Sudo() *rig.Client
	FS() remotefs.FS
}
