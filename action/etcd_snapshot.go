package action

import (
	"github.com/vinaycharlie01/k3sctl/phase"
)

func EtcdSnapshotList() (*phase.Manager, *phase.EtcdSnapshotList) {
	listPhase := &phase.EtcdSnapshotList{}
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

func EtcdSnapshotDelete(name string) *phase.Manager {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.Lock{},
		&phase.PrepareHosts{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		&phase.EtcdSnapshotDelete{SnapshotName: name},
		&phase.Unlock{},
		&phase.Disconnect{},
	)
	return m
}
