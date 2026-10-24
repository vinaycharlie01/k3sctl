package phase

import (
	"testing"

	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestNodeRemove_MatchesTarget(t *testing.T) {
	host := &cluster.Host{
		ConnectionConfig: cluster.ConnectionConfig{
			SSH: &ssh.Config{Address: "10.0.0.5"},
		},
		Hostname: "worker-1",
	}
	host.Metadata.Hostname = "worker-1.local"
	host.Metadata.KubernetesNodeName = "worker-1"

	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{"match by hostname", "worker-1", true},
		{"match by hostname case insensitive", "Worker-1", true},
		{"match by SSH address", "10.0.0.5", true},
		{"match by metadata hostname", "worker-1.local", true},
		{"match by metadata hostname case insensitive", "WORKER-1.LOCAL", true},
		{"match by kubernetes node name", "worker-1", true},
		{"no match", "worker-99", false},
		{"partial match does not count", "worker", false},
		{"empty target does not match", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &NodeRemove{Target: tt.target}
			assert.Equal(t, tt.want, p.matchesTarget(host))
		})
	}
}

func TestNodeRemove_Prepare(t *testing.T) {
	leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer}
	leader.Metadata.IsLeader = true
	agent := &cluster.Host{
		ConnectionConfig: cluster.ConnectionConfig{
			SSH: &ssh.Config{Address: "10.0.0.10"},
		},
		Hostname: "agent-1",
		Role:     cluster.RoleAgent,
	}

	t.Run("finds matching host", func(t *testing.T) {
		p := &NodeRemove{Target: "agent-1"}
		p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
			Hosts: cluster.Hosts{leader, agent},
		}}
		p.Prepare()
		assert.Equal(t, agent, p.host)
		assert.Equal(t, leader, p.leader)
	})

	t.Run("no match leaves host nil", func(t *testing.T) {
		p := &NodeRemove{Target: "nonexistent"}
		p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
			Hosts: cluster.Hosts{leader, agent},
		}}
		p.Prepare()
		assert.Nil(t, p.host)
	})
}

func TestNodeRemove_ShouldRun(t *testing.T) {
	t.Run("true when host is set", func(t *testing.T) {
		p := &NodeRemove{host: &cluster.Host{}}
		assert.True(t, p.ShouldRun())
	})

	t.Run("false when host is nil", func(t *testing.T) {
		p := &NodeRemove{}
		assert.False(t, p.ShouldRun())
	})
}
