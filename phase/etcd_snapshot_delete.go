package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type EtcdSnapshotDelete struct {
	GenericPhase
	leader       *cluster.Host
	SnapshotName string
}

func (p *EtcdSnapshotDelete) Title() string {
	return "Delete etcd snapshot"
}

func (p *EtcdSnapshotDelete) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *EtcdSnapshotDelete) ShouldRun() bool {
	return p.leader != nil && p.leader.Metadata.K3sServiceRunning && p.SnapshotName != ""
}

func (p *EtcdSnapshotDelete) Run(_ context.Context) error {
	h := p.leader

	cmd := h.Configurer.K3sCmdf("etcd-snapshot delete %s", shellQuote(p.SnapshotName))
	log.Infof("%s: deleting etcd snapshot %s", h, p.SnapshotName)

	if err := h.Sudo().Exec(cmd); err != nil {
		return fmt.Errorf("%s: delete etcd snapshot %s: %w", h, p.SnapshotName, err)
	}

	log.Infof("%s: snapshot %s deleted", h, p.SnapshotName)
	return nil
}
