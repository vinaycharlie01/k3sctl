package action

import (
	"io"

	"github.com/vinaycharlie01/k3sctl/phase"
)

type BackupOptions struct {
	Out io.Writer
}

func Backup(opts BackupOptions) *phase.Manager {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.Lock{},
		&phase.PrepareHosts{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		&phase.Backup{Out: opts.Out},
		&phase.Unlock{},
		&phase.Disconnect{},
	)
	return m
}
