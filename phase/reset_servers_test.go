package phase

import (
	"testing"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/stretchr/testify/assert"
)

func TestResetServers_Prepare(t *testing.T) {
	leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer}
	leader.Metadata.IsLeader = true

	resetHost := &cluster.Host{Hostname: "server-2", Role: cluster.RoleServer, Reset: true}
	noReset := &cluster.Host{Hostname: "server-3", Role: cluster.RoleServer}
	agent := &cluster.Host{Hostname: "agent-1", Role: cluster.RoleAgent, Reset: true}

	p := &ResetServers{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		Hosts: cluster.Hosts{leader, resetHost, noReset, agent},
	}}
	p.Prepare()

	assert.Equal(t, leader, p.leader)
	assert.Len(t, p.hosts, 1)
	assert.Equal(t, resetHost, p.hosts[0])
}

func TestResetServers_ShouldRun(t *testing.T) {
	t.Run("true when hosts to reset", func(t *testing.T) {
		p := &ResetServers{hosts: cluster.Hosts{&cluster.Host{}}}
		assert.True(t, p.ShouldRun())
	})

	t.Run("false when no hosts", func(t *testing.T) {
		p := &ResetServers{}
		assert.False(t, p.ShouldRun())
	})
}

func TestResetServers_LeaderExcluded(t *testing.T) {
	leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer, Reset: true}
	leader.Metadata.IsLeader = true

	p := &ResetServers{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		Hosts: cluster.Hosts{leader},
	}}
	p.Prepare()

	assert.Empty(t, p.hosts, "leader should not be included in reset hosts")
}
