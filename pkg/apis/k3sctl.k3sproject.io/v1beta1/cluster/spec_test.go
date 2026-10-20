package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecSetDefaults(t *testing.T) {
	host := newTestHost("10.0.0.1", "")
	spec := &Spec{Hosts: Hosts{host}}

	spec.SetDefaults()

	require.NotNil(t, spec.K3s)
	assert.Equal(t, RoleServer, host.Role)
}

func TestSpecValidate(t *testing.T) {
	t.Run("requires hosts", func(t *testing.T) {
		err := (&Spec{}).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one host")
	})

	t.Run("requires a server", func(t *testing.T) {
		spec := &Spec{Hosts: Hosts{newTestHost("10.0.0.1", RoleAgent)}}
		err := spec.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one server")
	})

	t.Run("rejects an invalid role", func(t *testing.T) {
		spec := &Spec{Hosts: Hosts{newTestHost("10.0.0.1", "database")}}
		require.Error(t, spec.Validate())
	})

	t.Run("accepts a server", func(t *testing.T) {
		spec := &Spec{Hosts: Hosts{newTestHost("10.0.0.1", RoleServer)}}
		require.NoError(t, spec.Validate())
	})
}

func TestSpecKubeAPIURL(t *testing.T) {
	t.Run("no leader", func(t *testing.T) {
		assert.Empty(t, (&Spec{Hosts: Hosts{newTestHost("10.0.0.1", RoleAgent)}}).KubeAPIURL())
	})

	t.Run("prefers the private address", func(t *testing.T) {
		leader := newTestHost("10.0.0.1", RoleServer)
		leader.PrivateAddress = "192.168.0.1"

		assert.Equal(t, "https://192.168.0.1:6443", (&Spec{Hosts: Hosts{leader}}).KubeAPIURL())
	})

	t.Run("falls back to the connection address", func(t *testing.T) {
		leader := newTestHost("10.0.0.1", RoleServer)

		assert.Equal(t, "https://10.0.0.1:6443", (&Spec{Hosts: Hosts{leader}}).KubeAPIURL())
	})
}

func TestSpecCounts(t *testing.T) {
	spec := &Spec{Hosts: Hosts{
		newTestHost("10.0.0.1", RoleServer),
		newTestHost("10.0.0.2", RoleServer),
		newTestHost("10.0.0.3", RoleAgent),
	}}

	assert.Equal(t, 3, spec.NodeCount())
	assert.Equal(t, 2, spec.ServerCount())
	assert.Equal(t, 1, spec.AgentCount())
	assert.Equal(t, 6443, spec.APIPort())
}
