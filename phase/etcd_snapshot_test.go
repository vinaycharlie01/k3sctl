package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestEtcdSnapshotList_ShouldRun(t *testing.T) {
	running := &cluster.Host{Role: cluster.RoleServer}
	running.Metadata.K3sServiceRunning = true
	running.Metadata.IsLeader = true

	stopped := &cluster.Host{Role: cluster.RoleServer}
	stopped.Metadata.IsLeader = true

	tests := []struct {
		name  string
		hosts cluster.Hosts
		want  bool
	}{
		{"running leader", cluster.Hosts{running}, true},
		{"stopped leader", cluster.Hosts{stopped}, false},
		{"no hosts", cluster.Hosts{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &EtcdSnapshotList{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{Hosts: tt.hosts}}
			p.Prepare()
			assert.Equal(t, tt.want, p.ShouldRun())
		})
	}
}

func TestEtcdSnapshotDelete_ShouldRun(t *testing.T) {
	leader := &cluster.Host{Role: cluster.RoleServer}
	leader.Metadata.K3sServiceRunning = true
	leader.Metadata.IsLeader = true

	tests := []struct {
		name         string
		hosts        cluster.Hosts
		snapshotName string
		want         bool
	}{
		{"running leader with name", cluster.Hosts{leader}, "snap-001", true},
		{"running leader without name", cluster.Hosts{leader}, "", false},
		{"no hosts", cluster.Hosts{}, "snap-001", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &EtcdSnapshotDelete{SnapshotName: tt.snapshotName}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{Hosts: tt.hosts}}
			p.Prepare()
			assert.Equal(t, tt.want, p.ShouldRun())
		})
	}
}
