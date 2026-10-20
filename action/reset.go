package action

import (
	"github.com/vinaycharlie01/k3sctl/phase"
)

type ResetOptions struct {
	NoDrain  bool
	NoDelete bool
}

func Reset(opts ResetOptions) *phase.Manager {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.DefaultK3sVersion{},
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.Lock{},
		&phase.PrepareHosts{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		&phase.ResetAgents{NoDrain: opts.NoDrain, NoDelete: opts.NoDelete},
		&phase.ResetServers{NoDrain: opts.NoDrain, NoDelete: opts.NoDelete},
		&phase.ResetLeader{},
		&phase.DaemonReload{},
		&phase.Unlock{},
		&phase.Disconnect{},
	)
	return m
}
