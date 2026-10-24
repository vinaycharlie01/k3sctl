package action

import (
	"io"

	"github.com/vinaycharlie01/k3sctl/phase"
)

type ConfigDiffOptions struct {
	Writer io.Writer
}

func ConfigDiff(opts ConfigDiffOptions) *phase.Manager {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.GatherFacts{},
		&phase.GatherK3sFacts{},
		&phase.ConfigDiff{Writer: opts.Writer},
		&phase.Disconnect{},
	)
	return m
}
