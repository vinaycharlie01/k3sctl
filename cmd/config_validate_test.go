package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidateCommandRegistered(t *testing.T) {
	app := NewK3sctl()
	var found bool
	for _, c := range app.Commands {
		if c.Name == "config" {
			for _, sub := range c.Subcommands {
				if sub.Name == "validate" {
					found = true
					break
				}
			}
			break
		}
	}
	assert.True(t, found, "config validate subcommand must be registered")
}

func TestConfigValidate_ValidConfig(t *testing.T) {
	path := writeConfigFile(t, `apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: test
spec:
  hosts:
    - role: server
      ssh:
        address: 10.0.0.1
        user: root
        keyPath: /tmp/test-key
  k3s:
    version: v1.30.0+k3s1
`)

	_, err := loadConfig(path)
	require.NoError(t, err)
}

func TestConfigValidate_InvalidConfig(t *testing.T) {
	t.Run("missing hosts", func(t *testing.T) {
		path := writeConfigFile(t, `apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: test
spec: {}
`)
		_, err := loadConfig(path)
		require.Error(t, err)
	})

	t.Run("no cluster document", func(t *testing.T) {
		path := writeConfigFile(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: extra
`)
		_, err := loadConfig(path)
		require.Error(t, err)
	})
}

func TestConfigValidate_NonExistentFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	_, err := loadConfig(path)
	require.Error(t, err)
}
