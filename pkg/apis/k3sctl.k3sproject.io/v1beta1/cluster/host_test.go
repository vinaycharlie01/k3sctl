package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestHostUnmarshalYAML(t *testing.T) {
	t.Run("connection and host fields coexist", func(t *testing.T) {
		input := `
role: agent
privateAddress: 10.0.0.6
installFlags:
  - --node-label=foo=bar
ssh:
  address: 10.0.0.5
  user: root
`
		var host Host
		require.NoError(t, yaml.UnmarshalStrict([]byte(input), &host))

		assert.Equal(t, RoleAgent, host.Role)
		assert.Equal(t, "10.0.0.6", host.PrivateAddress)
		require.Len(t, host.InstallFlags, 1)
		assert.Equal(t, "--node-label=foo=bar", host.InstallFlags[0])
		require.NotNil(t, host.SSH)
		assert.Equal(t, "10.0.0.5", host.SSH.Address)
		assert.Equal(t, "10.0.0.5", host.Address())
		assert.Equal(t, "ssh", host.Protocol())
	})

	t.Run("localhost", func(t *testing.T) {
		var host Host
		require.NoError(t, yaml.UnmarshalStrict([]byte("localhost: true\n"), &host))
		assert.True(t, host.Localhost)
		assert.Equal(t, "localhost", host.Address())
		assert.Equal(t, "localhost", host.Protocol())
	})

	t.Run("unknown fields are still rejected", func(t *testing.T) {
		var host Host
		err := yaml.UnmarshalStrict([]byte("nonsense: true\n"), &host)
		require.Error(t, err)
	})
}

func TestHostValidate(t *testing.T) {
	require.NoError(t, (&Host{Role: RoleServer}).Validate())
	require.NoError(t, (&Host{Role: RoleAgent}).Validate())

	err := (&Host{Role: "database"}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func TestHostK3sInstallFlags(t *testing.T) {
	host := &Host{
		InstallFlags: Flags{"--debug"},
		Labels:       map[string]string{"zone": "a"},
		Taints:       []string{"dedicated=infra:NoSchedule"},
	}

	flags := host.K3sInstallFlags()

	assert.Contains(t, flags, "--debug")
	assert.Contains(t, flags, "--node-label=zone=a")
	assert.Contains(t, flags, "--node-taint=dedicated=infra:NoSchedule")
}

func TestHostK3sServiceName(t *testing.T) {
	assert.Equal(t, "k3s", (&Host{Role: RoleServer}).K3sServiceName())
	assert.Equal(t, "k3s-agent", (&Host{Role: RoleAgent}).K3sServiceName())
}
