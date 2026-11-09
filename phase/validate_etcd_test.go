package phase

import (
	"testing"

	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/stretchr/testify/assert"
)

func TestValidateEtcdMembers_Prepare(t *testing.T) {
	leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer}
	leader.Metadata.IsLeader = true

	p := &ValidateEtcdMembers{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		Hosts: cluster.Hosts{leader},
	}}
	p.Prepare()

	assert.Equal(t, leader, p.leader)
}

func TestValidateEtcdMembers_ShouldRun(t *testing.T) {
	t.Run("false with no leader", func(t *testing.T) {
		p := &ValidateEtcdMembers{}
		p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
			Hosts: cluster.Hosts{},
		}}
		p.Prepare()
		assert.False(t, p.ShouldRun())
	})

	t.Run("false with single server", func(t *testing.T) {
		leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer}
		leader.Metadata.IsLeader = true
		p := &ValidateEtcdMembers{}
		p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
			Hosts: cluster.Hosts{leader},
		}}
		p.Prepare()
		assert.False(t, p.ShouldRun())
	})

	t.Run("true with multiple servers", func(t *testing.T) {
		leader := &cluster.Host{Hostname: "server-1", Role: cluster.RoleServer}
		leader.Metadata.IsLeader = true
		other := &cluster.Host{Hostname: "server-2", Role: cluster.RoleServer}
		p := &ValidateEtcdMembers{}
		p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
			Hosts: cluster.Hosts{leader, other},
		}}
		p.Prepare()
		assert.True(t, p.ShouldRun())
	})
}

func TestValidateEtcdMembers_KnownServerAddresses(t *testing.T) {
	s1 := &cluster.Host{
		Role:           cluster.RoleServer,
		PrivateAddress: "10.0.0.1",
	}
	s1.SSH = &ssh.Config{Address: "192.168.1.1"}
	s1.Metadata.IsLeader = true

	s2 := &cluster.Host{Role: cluster.RoleServer}
	s2.SSH = &ssh.Config{Address: "192.168.1.2"}

	agent := &cluster.Host{
		Role:           cluster.RoleAgent,
		PrivateAddress: "10.0.0.10",
	}

	p := &ValidateEtcdMembers{}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		Hosts: cluster.Hosts{s1, s2, agent},
	}}
	p.Prepare()

	addrs := p.knownServerAddresses()
	assert.True(t, addrs["10.0.0.1"])
	assert.True(t, addrs["192.168.1.1"])
	assert.True(t, addrs["192.168.1.2"])
	assert.False(t, addrs["10.0.0.10"], "agent addresses should not be included")
}

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "hello", firstLine("hello\nworld"))
	assert.Equal(t, "single", firstLine("single"))
	assert.Equal(t, "", firstLine(""))
}
