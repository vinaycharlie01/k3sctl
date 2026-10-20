package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "k3sctl.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

const clusterDocument = `apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: test
spec:
  hosts:
    - role: server
`

func TestLoadConfig_CollectsAdditionalDocumentsAsManifests(t *testing.T) {
	path := writeConfigFile(t, clusterDocument+`---
apiVersion: v1
kind: ConfigMap
metadata:
  name: extra
data:
  key: value
`)

	cfg, err := loadConfig(path)
	require.NoError(t, err)
	require.Len(t, cfg.Metadata.Manifests, 1)

	raw, ok := cfg.Metadata.Manifests["k3sctl.yaml"]
	require.True(t, ok, "manifest keyed by its source file")
	assert.Contains(t, string(raw), "kind: ConfigMap")
}

func TestLoadConfig_DuplicateDocumentNamesAreKept(t *testing.T) {
	path := writeConfigFile(t, clusterDocument+`---
apiVersion: v1
kind: ConfigMap
metadata:
  name: first
---
apiVersion: v1
kind: Secret
metadata:
  name: second
`)

	cfg, err := loadConfig(path)
	require.NoError(t, err)
	require.Len(t, cfg.Metadata.Manifests, 2)
	assert.Contains(t, cfg.Metadata.Manifests, "k3sctl.yaml")
	assert.Contains(t, cfg.Metadata.Manifests, "k3sctl.yaml-2")
}

func TestLoadConfig_RequiresExactlyOneClusterDocument(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		path := writeConfigFile(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: extra
`)
		_, err := loadConfig(path)
		require.Error(t, err)
	})

	t.Run("multiple", func(t *testing.T) {
		path := writeConfigFile(t, clusterDocument+"---\n"+clusterDocument)
		_, err := loadConfig(path)
		require.Error(t, err)
	})
}
