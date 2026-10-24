package phase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type EtcdSnapshot struct {
	Name    string
	Size    string
	Created string
}

type EtcdSnapshotList struct {
	GenericPhase
	leader    *cluster.Host
	Snapshots []EtcdSnapshot
}

func (p *EtcdSnapshotList) Title() string {
	return "List etcd snapshots"
}

func (p *EtcdSnapshotList) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *EtcdSnapshotList) ShouldRun() bool {
	return p.leader != nil && p.leader.Metadata.K3sServiceRunning
}

func (p *EtcdSnapshotList) Run(_ context.Context) error {
	h := p.leader

	cmd := h.Configurer.K3sCmdf("etcd-snapshot list")
	log.Infof("%s: listing etcd snapshots", h)

	output, err := h.Sudo().ExecOutput(cmd)
	if err != nil {
		return fmt.Errorf("%s: list etcd snapshots: %w", h, err)
	}

	p.Snapshots = parseSnapshotList(output)
	log.Infof("%s: found %d snapshot(s)", h, len(p.Snapshots))
	return nil
}

func parseSnapshotList(output string) []EtcdSnapshot {
	var snapshots []EtcdSnapshot
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip the header line
		if strings.HasPrefix(line, "Name") {
			continue
		}
		fields := strings.Fields(line)
		// Expect at least: name, location, size, created
		if len(fields) < 4 {
			continue
		}
		snapshots = append(snapshots, EtcdSnapshot{
			Name:    fields[0],
			Size:    fields[len(fields)-2],
			Created: fields[len(fields)-1],
		})
	}
	return snapshots
}
