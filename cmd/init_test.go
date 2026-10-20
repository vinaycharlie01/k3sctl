package cmd

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func renderInitTemplate(t *testing.T, data initData) string {
	t.Helper()

	tmpl, err := template.New("init").Parse(initTemplate)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, data))
	return buf.String()
}

func TestInitTemplateRoundTrips(t *testing.T) {
	t.Run("server only", func(t *testing.T) {
		path := writeConfigFile(t, renderInitTemplate(t, initData{
			Name:    "demo",
			Address: "192.168.1.10",
			User:    "ubuntu",
			Port:    2222,
			KeyPath: "~/.ssh/id_ed25519",
			Version: "v1.30.0+k3s1",
		}))

		cfg, err := loadConfig(path)
		require.NoError(t, err)

		assert.Equal(t, "demo", cfg.Metadata.Name)
		require.Len(t, cfg.Spec.Hosts, 1)
		host := cfg.Spec.Hosts[0]
		assert.Equal(t, cluster.RoleServer, host.Role)
		require.NotNil(t, host.SSH)
		assert.Equal(t, "192.168.1.10", host.SSH.Address)
		assert.Equal(t, "ubuntu", host.SSH.User)
		assert.Equal(t, 2222, host.SSH.Port)
		require.NotNil(t, host.SSH.KeyPath)
		assert.Equal(t, "~/.ssh/id_ed25519", *host.SSH.KeyPath)
		assert.Equal(t, "v1.30.0+k3s1", cfg.Spec.K3s.Version)
	})

	t.Run("server and agent", func(t *testing.T) {
		path := writeConfigFile(t, renderInitTemplate(t, initData{
			Name:         "demo",
			Address:      "192.168.1.10",
			Agent:        true,
			AgentAddress: "192.168.1.11",
			User:         "root",
			Port:         22,
			KeyPath:      "~/.ssh/id_rsa",
			Version:      "v1.30.0+k3s1",
		}))

		cfg, err := loadConfig(path)
		require.NoError(t, err)

		require.Len(t, cfg.Spec.Hosts, 2)
		assert.Equal(t, cluster.RoleServer, cfg.Spec.Hosts[0].Role)
		assert.Equal(t, cluster.RoleAgent, cfg.Spec.Hosts[1].Role)
		assert.Equal(t, "192.168.1.11", cfg.Spec.Hosts[1].SSH.Address)
	})
}
