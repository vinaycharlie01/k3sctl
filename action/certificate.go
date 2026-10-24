package action

import (
	"github.com/vinaycharlie01/k3sctl/phase"
)

func CertificateCheck() (*phase.Manager, *phase.CertificateCheck) {
	checkPhase := &phase.CertificateCheck{}
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.Lock{},
		&phase.PrepareHosts{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		checkPhase,
		&phase.Unlock{},
		&phase.Disconnect{},
	)
	return m, checkPhase
}
