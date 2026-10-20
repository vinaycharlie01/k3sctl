package cluster

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHost(address string, role Role) *Host {
	h := &Host{Role: role}
	if address != "" {
		h.SSH = &ssh.Config{Address: address}
	}
	return h
}

func TestHostsFiltering(t *testing.T) {
	server := newTestHost("10.0.0.1", RoleServer)
	agent := newTestHost("10.0.0.2", RoleAgent)
	hosts := Hosts{server, agent}

	assert.Equal(t, Hosts{server}, hosts.Servers())
	assert.Equal(t, Hosts{agent}, hosts.Agents())
	assert.Equal(t, 2, hosts.Count())
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, hosts.Addresses())
	assert.Equal(t, server, hosts.First())
	assert.Nil(t, Hosts{}.First())
}

func TestHostsLeader(t *testing.T) {
	t.Run("prefers the host marked as leader", func(t *testing.T) {
		first := newTestHost("10.0.0.1", RoleServer)
		flagged := newTestHost("10.0.0.2", RoleServer)
		flagged.Metadata.IsLeader = true

		assert.Equal(t, flagged, (Hosts{first, flagged}).Leader())
	})

	t.Run("falls back to the first server", func(t *testing.T) {
		agent := newTestHost("10.0.0.1", RoleAgent)
		server := newTestHost("10.0.0.2", RoleServer)

		assert.Equal(t, server, (Hosts{agent, server}).Leader())
	})

	t.Run("no servers", func(t *testing.T) {
		assert.Nil(t, (Hosts{newTestHost("10.0.0.1", RoleAgent)}).Leader())
	})
}

func TestHostsFind(t *testing.T) {
	a := newTestHost("10.0.0.1", RoleServer)
	b := newTestHost("10.0.0.2", RoleAgent)

	assert.Equal(t, b, (Hosts{a, b}).Find(func(h *Host) bool { return h.IsAgent() }))
	assert.Nil(t, (Hosts{a, b}).Find(func(h *Host) bool { return false }))
}

func TestHostsEachStopsOnError(t *testing.T) {
	var visited int
	wantErr := errors.New("boom")

	err := (Hosts{
		newTestHost("10.0.0.1", RoleServer),
		newTestHost("10.0.0.2", RoleServer),
		newTestHost("10.0.0.3", RoleServer),
	}).Each(func(h *Host) error {
		visited++
		if visited == 2 {
			return wantErr
		}
		return nil
	})

	require.ErrorIs(t, err, wantErr)
	assert.Equal(t, 2, visited)
}

func TestHostsParallelEach(t *testing.T) {
	t.Run("processes every host", func(t *testing.T) {
		hosts := Hosts{
			newTestHost("10.0.0.1", RoleServer),
			newTestHost("10.0.0.2", RoleAgent),
			newTestHost("10.0.0.3", RoleAgent),
		}
		var count atomic.Int32

		err := hosts.ParallelEach(context.Background(), 2, func(_ context.Context, _ *Host) error {
			count.Add(1)
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, int32(3), count.Load())
	})

	t.Run("returns the first error", func(t *testing.T) {
		wantErr := errors.New("boom")
		hosts := Hosts{newTestHost("10.0.0.1", RoleServer)}

		err := hosts.ParallelEach(context.Background(), 1, func(_ context.Context, _ *Host) error {
			return wantErr
		})

		require.ErrorIs(t, err, wantErr)
	})
}

func TestHostsBatchedParallelEach(t *testing.T) {
	var hosts Hosts
	for i := 0; i < 5; i++ {
		hosts = append(hosts, newTestHost("10.0.0.1", RoleServer))
	}
	var count atomic.Int32

	err := hosts.BatchedParallelEach(context.Background(), 2, 2, func(_ context.Context, _ *Host) error {
		count.Add(1)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(5), count.Load())
}
