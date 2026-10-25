package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestUpgradeServers_Prepare(t *testing.T) {
	leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer}
	leader.Metadata.IsLeader = true
	leader.Metadata.NeedsUpgrade = true

	noUpgrade := &cluster.Host{Hostname: "server-2", Role: cluster.RoleServer}
	noUpgrade.Metadata.NeedsUpgrade = false

	agent := &cluster.Host{Hostname: "agent-1", Role: cluster.RoleAgent}
	agent.Metadata.NeedsUpgrade = true

	p := &UpgradeServers{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		Hosts: cluster.Hosts{leader, noUpgrade, agent},
	}}
	p.Prepare()

	assert.Equal(t, leader, p.leader)
	assert.Len(t, p.hosts, 1, "only the server needing upgrade should be selected")
	assert.Equal(t, leader, p.hosts[0])
}

func TestUpgradeServers_ShouldRun(t *testing.T) {
	t.Run("true when hosts need upgrade", func(t *testing.T) {
		p := &UpgradeServers{hosts: cluster.Hosts{&cluster.Host{}}}
		assert.True(t, p.ShouldRun())
	})

	t.Run("false when no hosts", func(t *testing.T) {
		p := &UpgradeServers{}
		assert.False(t, p.ShouldRun())
	})
}

func TestUpgradeServers_NoDrainField(t *testing.T) {
	t.Run("defaults to false", func(t *testing.T) {
		p := &UpgradeServers{}
		assert.False(t, p.NoDrain)
	})

	t.Run("can be set to true", func(t *testing.T) {
		p := &UpgradeServers{NoDrain: true}
		assert.True(t, p.NoDrain)
	})
}

func TestUpgradeServers_CleanUp(t *testing.T) {
	t.Run("nil leader does not panic", func(t *testing.T) {
		p := &UpgradeServers{
			hosts:  cluster.Hosts{&cluster.Host{Hostname: "server-1"}},
			leader: nil,
		}
		assert.NotPanics(t, func() { p.CleanUp() })
	})
}
