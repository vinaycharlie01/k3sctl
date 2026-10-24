package action

import (
	"io"

	"github.com/vinaycharlie01/k3sctl/phase"
)

type ApplyOptions struct {
	DisableDowngradeCheck bool
	NoWait                bool
	NoDrain               bool
	RestoreFrom           string
	KubeconfigOut         io.Writer
	KubeconfigAPIAddress  string
	KubeconfigUser        string
	KubeconfigCluster     string
}

func Apply(opts ApplyOptions) *phase.Manager {
	lockPhase := &phase.Lock{}

	m := &phase.Manager{}
	m.AddPhase(
		&phase.DefaultK3sVersion{},
		&phase.Connect{},
		&phase.DetectOS{},
		&phase.PrepareHosts{},
		&phase.GatherFacts{},
		&phase.ValidateHosts{},
		&phase.Preflight{},
		&phase.GatherK3sFacts{},
		&phase.ValidateFacts{DisableDowngradeCheck: opts.DisableDowngradeCheck},
		&phase.ValidateEtcdMembers{},
		lockPhase,
		&phase.ResolveBinaryProviders{},
		&phase.StageBinaries{},
		&phase.UploadFiles{},
		&phase.InstallBinaries{},
		&phase.PrepareArm{},
		&phase.ConfigureK3s{},
		&phase.CertRotate{},
		&phase.Restore{RestoreFrom: opts.RestoreFrom},
		&phase.RunHooks{Action: "apply", Stage: "before"},
		&phase.InitializeK3s{},
		&phase.PropagateToken{},
		&phase.InstallServers{},
		&phase.InstallAgents{},
		&phase.UpgradeServers{NoDrain: opts.NoDrain},
		&phase.UpgradeAgents{NoDrain: opts.NoDrain},
		&phase.Reinstall{NoDrain: opts.NoDrain},
		&phase.ResetAgents{NoDrain: opts.NoDrain},
		&phase.ResetServers{NoDrain: opts.NoDrain},
		&phase.RunHooks{Action: "apply", Stage: "after"},
		&phase.ApplyManifests{},
	)

	if opts.KubeconfigOut != nil {
		m.AddPhase(&phase.GetKubeconfig{
			Out:        opts.KubeconfigOut,
			APIAddress: opts.KubeconfigAPIAddress,
			User:       opts.KubeconfigUser,
			Cluster:    opts.KubeconfigCluster,
		})
	}

	m.AddPhase(
		&phase.Unlock{},
		&phase.Disconnect{},
	)

	return m
}
