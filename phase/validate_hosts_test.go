package phase

import (
	"context"
	"fmt"
	"testing"

	"github.com/vinaycharlie01/k3sctl/configurer"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockconfigurer struct {
	configurer.Linux
}

type mockValidator struct {
	configurer.Linux
	err error
}

func (c *mockValidator) ValidateHost(_ configurer.Host) error {
	return c.err
}

func TestValidateHosts_Title(t *testing.T) {
	p := &ValidateHosts{}
	assert.Equal(t, "Validate hosts", p.Title())
}

func TestValidateHosts_DuplicateAddress(t *testing.T) {
	hosts := cluster.Hosts{
		{
			ClientWithConfig: rig.ClientWithConfig{
				ConnectionConfig: rig.CompositeConfig{
					SSH: &ssh.Config{Address: "10.0.0.1"},
				},
			},
			Role: cluster.RoleServer,
		},
		{
			ClientWithConfig: rig.ClientWithConfig{
				ConnectionConfig: rig.CompositeConfig{
					SSH: &ssh.Config{Address: "10.0.0.1"},
				},
			},
			Role: cluster.RoleAgent,
		},
	}

	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: hosts,
		},
	}

	p := &ValidateHosts{}
	p.Config = cfg
	p.Manager = &Manager{Concurrency: 1}

	err := p.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate host address")
}

func TestValidateHosts_UniqueAddresses(t *testing.T) {
	hosts := cluster.Hosts{
		{
			ClientWithConfig: rig.ClientWithConfig{
				ConnectionConfig: rig.CompositeConfig{
					SSH: &ssh.Config{Address: "10.0.0.1"},
				},
			},
			Role: cluster.RoleServer,
		},
		{
			ClientWithConfig: rig.ClientWithConfig{
				ConnectionConfig: rig.CompositeConfig{
					SSH: &ssh.Config{Address: "10.0.0.2"},
				},
			},
			Role: cluster.RoleAgent,
		},
	}

	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: hosts,
		},
	}

	p := &ValidateHosts{}
	p.Config = cfg
	p.Manager = &Manager{Concurrency: 1}

	err := p.Run(context.Background())
	if err != nil {
		assert.NotContains(t, err.Error(), "duplicate host address")
	}
}

func TestValidateConfigurer(t *testing.T) {
	t.Run("non validating configurer passes", func(t *testing.T) {
		h := &cluster.Host{Configurer: &mockconfigurer{}}
		if v, ok := h.Configurer.(configurer.HostValidator); ok {
			err := v.ValidateHost(h)
			assert.NoError(t, err)
		}
	})

	t.Run("validating configurer with error", func(t *testing.T) {
		h := &cluster.Host{Configurer: &mockValidator{err: fmt.Errorf("missing feature")}}
		v, ok := h.Configurer.(configurer.HostValidator)
		require.True(t, ok)
		err := v.ValidateHost(h)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing feature")
	})

	t.Run("validating configurer success", func(t *testing.T) {
		h := &cluster.Host{Configurer: &mockValidator{err: nil}}
		v, ok := h.Configurer.(configurer.HostValidator)
		require.True(t, ok)
		err := v.ValidateHost(h)
		assert.NoError(t, err)
	})
}
