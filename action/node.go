package action

import (
	"github.com/vinaycharlie01/k3sctl/phase"
)

type NodeRemoveOptions struct {
	Target  string
	NoDrain bool
	Force   bool
}

func NodeRemove(opts NodeRemoveOptions) *phase.Manager {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.Lock{},
		&phase.PrepareHosts{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		&phase.NodeRemove{
			Target:  opts.Target,
			NoDrain: opts.NoDrain,
			Force:   opts.Force,
		},
		&phase.DaemonReload{},
		&phase.Unlock{},
		&phase.Disconnect{},
	)
	return m
}
