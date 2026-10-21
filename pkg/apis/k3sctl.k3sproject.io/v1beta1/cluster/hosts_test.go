package cluster

import (
	"context"
	"fmt"
	"testing"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHost(role Role, addr string) *Host {
	return &Host{
		Role: role,
		ClientWithConfig: rig.ClientWithConfig{
			ConnectionConfig: rig.CompositeConfig{
				SSH: &ssh.Config{Address: addr},
			},
		},
	}
}

func TestHostsFilter(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
		newTestHost(RoleServer, "10.0.0.3"),
	}

	servers := hosts.Filter(func(h *Host) bool { return h.IsServer() })
	assert.Len(t, servers, 2)
}

func TestHostsServers(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
		newTestHost(RoleServer, "10.0.0.3"),
	}
	assert.Len(t, hosts.Servers(), 2)
}

func TestHostsAgents(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
		newTestHost(RoleAgent, "10.0.0.3"),
	}
	assert.Len(t, hosts.Agents(), 2)
}

func TestHostsFirst(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleServer, "10.0.0.2"),
	}
	assert.Equal(t, "10.0.0.1", hosts.First().Address())

	empty := Hosts{}
	assert.Nil(t, empty.First())
}

func TestHostsLeader(t *testing.T) {
	t.Run("explicit leader", func(t *testing.T) {
		h1 := newTestHost(RoleServer, "10.0.0.1")
		h2 := newTestHost(RoleServer, "10.0.0.2")
		h2.Metadata.IsLeader = true
		hosts := Hosts{h1, h2}

		leader := hosts.Leader()
		require.NotNil(t, leader)
		assert.Equal(t, "10.0.0.2", leader.Address())
	})

	t.Run("first server when no explicit leader", func(t *testing.T) {
		hosts := Hosts{
			newTestHost(RoleServer, "10.0.0.1"),
			newTestHost(RoleServer, "10.0.0.2"),
		}
		leader := hosts.Leader()
		require.NotNil(t, leader)
		assert.Equal(t, "10.0.0.1", leader.Address())
	})

	t.Run("nil when no servers", func(t *testing.T) {
		hosts := Hosts{newTestHost(RoleAgent, "10.0.0.1")}
		assert.Nil(t, hosts.Leader())
	})
}

func TestHostsFind(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
	}

	found := hosts.Find(func(h *Host) bool { return h.Address() == "10.0.0.2" })
	require.NotNil(t, found)
	assert.Equal(t, RoleAgent, found.Role)

	notFound := hosts.Find(func(h *Host) bool { return h.Address() == "10.0.0.99" })
	assert.Nil(t, notFound)
}

func TestHostsEach(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
	}

	var visited []string
	err := hosts.Each(func(h *Host) error {
		visited = append(visited, h.Address())
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, visited)
}

func TestHostsEachError(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
	}

	err := hosts.Each(func(h *Host) error {
		return fmt.Errorf("fail")
	})
	assert.Error(t, err)
}

func TestHostsParallelEach(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
		newTestHost(RoleAgent, "10.0.0.3"),
	}

	err := hosts.ParallelEach(context.Background(), 2, func(ctx context.Context, h *Host) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestHostsParallelEachError(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
	}

	err := hosts.ParallelEach(context.Background(), 0, func(ctx context.Context, h *Host) error {
		if h.Address() == "10.0.0.1" {
			return fmt.Errorf("test error")
		}
		return nil
	})
	assert.Error(t, err)
}

func TestHostsCount(t *testing.T) {
	hosts := Hosts{newTestHost(RoleServer, "10.0.0.1")}
	assert.Equal(t, 1, hosts.Count())
	assert.Equal(t, 0, Hosts{}.Count())
}

func TestHostsAddresses(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
	}
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, hosts.Addresses())
}

func TestHostsBatchedParallelEach(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
		newTestHost(RoleAgent, "10.0.0.3"),
		newTestHost(RoleAgent, "10.0.0.4"),
	}

	count := 0
	err := hosts.BatchedParallelEach(context.Background(), 2, 2, func(ctx context.Context, h *Host) error {
		count++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestHostsBatchedParallelEach_ZeroBatchSize(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
	}

	count := 0
	err := hosts.BatchedParallelEach(context.Background(), 2, 0, func(ctx context.Context, h *Host) error {
		count++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestHostsParallelEach_ZeroConcurrency(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
	}

	count := 0
	err := hosts.ParallelEach(context.Background(), 0, func(ctx context.Context, h *Host) error {
		count++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestHostsEach_CancelOnError(t *testing.T) {
	hosts := Hosts{
		newTestHost(RoleServer, "10.0.0.1"),
		newTestHost(RoleAgent, "10.0.0.2"),
		newTestHost(RoleAgent, "10.0.0.3"),
	}

	count := 0
	err := hosts.Each(func(h *Host) error {
		count++
		return fmt.Errorf("stop")
	})
	require.Error(t, err)
	assert.Equal(t, 1, count, "Each should stop at first error")
}
