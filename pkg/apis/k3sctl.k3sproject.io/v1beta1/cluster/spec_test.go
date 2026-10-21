package cluster

import (
	"testing"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecSetDefaults(t *testing.T) {
	s := &Spec{
		Hosts: Hosts{
			{Role: ""},
			{Role: RoleAgent},
		},
	}

	s.SetDefaults()

	assert.NotNil(t, s.K3s)
	assert.Equal(t, RoleServer, s.Hosts[0].Role)
	assert.Equal(t, RoleAgent, s.Hosts[1].Role)
}

func TestSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    Spec
		wantErr string
	}{
		{
			name:    "no hosts",
			spec:    Spec{},
			wantErr: "at least one host is required",
		},
		{
			name: "no servers",
			spec: Spec{
				Hosts: Hosts{{Role: RoleAgent}},
			},
			wantErr: "at least one server host is required",
		},
		{
			name: "no server hosts (agent only)",
			spec: Spec{
				Hosts: Hosts{{Role: "invalid"}},
			},
			wantErr: "at least one server host is required",
		},
		{
			name: "invalid host role with server present",
			spec: Spec{
				Hosts: Hosts{{Role: RoleServer}, {Role: "invalid"}},
			},
			wantErr: "invalid role",
		},
		{
			name: "invalid k3s version",
			spec: Spec{
				Hosts: Hosts{{Role: RoleServer}},
				K3s:   &K3s{Version: "1.28.0"},
			},
			wantErr: "k3s version must start with 'v'",
		},
		{
			name: "valid",
			spec: Spec{
				Hosts: Hosts{{Role: RoleServer}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSpecK3sLeader(t *testing.T) {
	h1 := &Host{Role: RoleServer, ClientWithConfig: rig.ClientWithConfig{
		ConnectionConfig: rig.CompositeConfig{SSH: &ssh.Config{Address: "10.0.0.1"}},
	}}
	h2 := &Host{Role: RoleServer, ClientWithConfig: rig.ClientWithConfig{
		ConnectionConfig: rig.CompositeConfig{SSH: &ssh.Config{Address: "10.0.0.2"}},
	}}
	h2.Metadata.IsLeader = true

	s := &Spec{Hosts: Hosts{h1, h2}}
	leader := s.K3sLeader()
	require.NotNil(t, leader)
	assert.Equal(t, "10.0.0.2", leader.Address())
}

func TestSpecAPIPort(t *testing.T) {
	s := &Spec{}
	assert.Equal(t, 6443, s.APIPort())
}

func TestSpecKubeAPIURL(t *testing.T) {
	t.Run("with private address", func(t *testing.T) {
		s := &Spec{
			Hosts: Hosts{
				{
					Role:           RoleServer,
					PrivateAddress: "192.168.1.10",
					ClientWithConfig: rig.ClientWithConfig{
						ConnectionConfig: rig.CompositeConfig{SSH: &ssh.Config{Address: "10.0.0.1"}},
					},
				},
			},
		}
		assert.Equal(t, "https://192.168.1.10:6443", s.KubeAPIURL())
	})

	t.Run("without private address", func(t *testing.T) {
		s := &Spec{
			Hosts: Hosts{
				{
					Role: RoleServer,
					ClientWithConfig: rig.ClientWithConfig{
						ConnectionConfig: rig.CompositeConfig{SSH: &ssh.Config{Address: "10.0.0.1"}},
					},
				},
			},
		}
		assert.Equal(t, "https://10.0.0.1:6443", s.KubeAPIURL())
	})

	t.Run("no hosts", func(t *testing.T) {
		s := &Spec{Hosts: Hosts{}}
		assert.Equal(t, "", s.KubeAPIURL())
	})
}

func TestSpecCounts(t *testing.T) {
	s := &Spec{
		Hosts: Hosts{
			{Role: RoleServer},
			{Role: RoleServer},
			{Role: RoleAgent},
		},
	}
	assert.Equal(t, 3, s.NodeCount())
	assert.Equal(t, 2, s.ServerCount())
	assert.Equal(t, 1, s.AgentCount())
}
