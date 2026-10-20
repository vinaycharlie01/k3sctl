package configurer

import (
	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/cmd"
	"github.com/k0sproject/rig/v2/remotefs"
)

type Host interface {
	cmd.SimpleRunner
	Sudo() *rig.Client
	FS() remotefs.FS
}
