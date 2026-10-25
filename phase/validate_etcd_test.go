package phase

import (
	"testing"

	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
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

func TestFindStaleMembers(t *testing.T) {
	memberList := `8e9e05c52164694d, started, server-1, https://10.0.0.1:2380, https://10.0.0.1:2379, false
3a1b2c3d4e5f6a7b, started, server-2, https://10.0.0.2:2380, https://10.0.0.2:2379, false
a1b2c3d4e5f6a7b8, started, server-old, https://10.0.0.99:2380, https://10.0.0.99:2379, false`

	known := map[string]bool{
		"10.0.0.1": true,
		"10.0.0.2": true,
	}

	t.Run("detects stale member", func(t *testing.T) {
		stale := findStaleMembers(memberList, known)
		assert.Len(t, stale, 1)
		assert.Equal(t, "a1b2c3d4e5f6a7b8", stale[0].id)
		assert.Contains(t, stale[0].peerURL, "10.0.0.99")
	})

	t.Run("no stale when all known", func(t *testing.T) {
		allKnown := map[string]bool{
			"10.0.0.1":  true,
			"10.0.0.2":  true,
			"10.0.0.99": true,
		}
		stale := findStaleMembers(memberList, allKnown)
		assert.Empty(t, stale)
	})

	t.Run("empty output", func(t *testing.T) {
		stale := findStaleMembers("", known)
		assert.Empty(t, stale)
	})

	t.Run("malformed lines skipped", func(t *testing.T) {
		stale := findStaleMembers("garbage\nmore garbage", known)
		assert.Empty(t, stale)
	})
}

func TestValidateEtcdMembers_KnownServerAddresses(t *testing.T) {
	s1 := &cluster.Host{
		Role:           cluster.RoleServer,
		PrivateAddress: "10.0.0.1",
	}
	s1.ConnectionConfig.SSH = &ssh.Config{Address: "192.168.1.1"}
	s1.Metadata.IsLeader = true

	s2 := &cluster.Host{Role: cluster.RoleServer}
	s2.ConnectionConfig.SSH = &ssh.Config{Address: "192.168.1.2"}

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
