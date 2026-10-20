package action

import (
	"io"

	"github.com/vinaycharlie01/k3sctl/phase"
)

type KubeconfigOptions struct {
	Out        io.Writer
	APIAddress string
	User       string
	Cluster    string
}

func Kubeconfig(opts KubeconfigOptions) *phase.Manager {
	m := &phase.Manager{}
	m.AddPhase(
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.GatherFacts{},
		&phase.GetKubeconfig{
			Out:        opts.Out,
			APIAddress: opts.APIAddress,
			User:       opts.User,
			Cluster:    opts.Cluster,
		},
		&phase.Disconnect{},
	)
	return m
}
