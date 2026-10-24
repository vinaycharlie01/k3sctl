package action

import "github.com/vinaycharlie01/k3sctl/phase"

type RestoreOptions struct {
	RestoreFrom string
}

func Restore(opts RestoreOptions) *phase.Manager {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.Lock{},
		&phase.PrepareHosts{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		&phase.StopK3s{},
		&phase.Restore{RestoreFrom: opts.RestoreFrom},
		&phase.StartK3s{},
		&phase.Unlock{},
		&phase.Disconnect{},
	)
	return m
}
