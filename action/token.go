package action

import (
	"github.com/vinaycharlie01/k3sctl/phase"
)

func TokenList() (*phase.Manager, *phase.TokenList) {
	listPhase := &phase.TokenList{}
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.Lock{},
		&phase.PrepareHosts{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		listPhase,
		&phase.Unlock{},
		&phase.Disconnect{},
	)
	return m, listPhase
}
