package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
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

func TestFindEtcdMemberID(t *testing.T) {
	memberList := `8e9e05c52164694d, started, server-1, https://10.0.0.1:2380, https://10.0.0.1:2379, false
3a1b2c3d4e5f6a7b, started, server-2, https://10.0.0.2:2380, https://10.0.0.2:2379, false
a1b2c3d4e5f6a7b8, started, server-3, https://10.0.0.3:2380, https://10.0.0.3:2379, false`

	t.Run("finds matching member", func(t *testing.T) {
		id := findEtcdMemberID(memberList, "10.0.0.2")
		assert.Equal(t, "3a1b2c3d4e5f6a7b", id)
	})

	t.Run("returns empty for unknown address", func(t *testing.T) {
		id := findEtcdMemberID(memberList, "10.0.0.99")
		assert.Empty(t, id)
	})

	t.Run("handles empty output", func(t *testing.T) {
		id := findEtcdMemberID("", "10.0.0.1")
		assert.Empty(t, id)
	})
}
