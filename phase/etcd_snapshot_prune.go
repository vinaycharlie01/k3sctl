package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type EtcdSnapshotPrune struct {
	GenericPhase
	leader *cluster.Host
}

func (p *EtcdSnapshotPrune) Title() string {
	return "Prune etcd snapshots"
}

func (p *EtcdSnapshotPrune) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *EtcdSnapshotPrune) ShouldRun() bool {
	cfg := p.Config.Spec.K3s
	if cfg == nil || cfg.EtcdSnapshots == nil || cfg.EtcdSnapshots.Prune == nil {
		return false
	}
	return cfg.EtcdSnapshots.Prune.Enabled && p.leader != nil && p.leader.Metadata.K3sServiceRunning
}

func (p *EtcdSnapshotPrune) Run(_ context.Context) error {
	h := p.leader
	prune := p.Config.Spec.K3s.EtcdSnapshots.Prune

	args := fmt.Sprintf("etcd-snapshot prune --snapshot-retention %d", prune.Retention)
	if prune.Name != "" {
		args += fmt.Sprintf(" --name %s", shellQuote(prune.Name))
	}

	log.Infof("%s: pruning etcd snapshots (retention: %d)", h, prune.Retention)
	cmd := h.Configurer.K3sCmdf(args)
	if err := h.Sudo().Exec(cmd); err != nil {
		return fmt.Errorf("%s: etcd snapshot prune failed: %w", h, err)
	}

	log.Infof("%s: etcd snapshot pruning complete", h)
	return nil
}
