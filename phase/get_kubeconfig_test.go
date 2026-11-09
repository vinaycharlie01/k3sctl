package phase

import (
	"testing"

	"github.com/nirantarasi/k3sctl/pkg/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKubeconfig = `apiVersion: v1
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
kind: Config
users:
- name: default
  user:
    token: secret
`

func TestRewriteKubeconfig(t *testing.T) {
	t.Run("replaces the server address", func(t *testing.T) {
		got, err := k8s.RewriteKubeconfig([]byte(testKubeconfig), "https://10.0.0.1:6443", "", "")
		require.NoError(t, err)
		assert.Contains(t, string(got), "https://10.0.0.1:6443")
		assert.NotContains(t, string(got), "127.0.0.1")
	})

	t.Run("renames user and cluster", func(t *testing.T) {
		got, err := k8s.RewriteKubeconfig([]byte(testKubeconfig), "https://10.0.0.1:6443", "admin", "prod")
		require.NoError(t, err)
		assert.Contains(t, string(got), "prod")
		assert.Contains(t, string(got), "admin")
	})

	t.Run("keeps default names when not requested", func(t *testing.T) {
		got, err := k8s.RewriteKubeconfig([]byte(testKubeconfig), "https://10.0.0.1:6443", "", "")
		require.NoError(t, err)
		assert.Contains(t, string(got), "default")
	})

	t.Run("leaves unrelated content intact", func(t *testing.T) {
		got, err := k8s.RewriteKubeconfig([]byte(testKubeconfig), "https://10.0.0.1:6443", "", "")
		require.NoError(t, err)
		assert.Contains(t, string(got), "secret")
	})
}
