package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		got := rewriteKubeconfig(testKubeconfig, "https://10.0.0.1:6443", "", "")
		assert.Contains(t, got, "server: https://10.0.0.1:6443")
		assert.NotContains(t, got, "127.0.0.1")
	})

	t.Run("renames user and cluster", func(t *testing.T) {
		got := rewriteKubeconfig(testKubeconfig, "https://10.0.0.1:6443", "admin", "prod")
		assert.Contains(t, got, "cluster: prod")
		assert.Contains(t, got, "user: admin")
		assert.NotContains(t, got, "cluster: default")
		assert.NotContains(t, got, "user: default")
	})

	t.Run("keeps default names when not requested", func(t *testing.T) {
		got := rewriteKubeconfig(testKubeconfig, "https://10.0.0.1:6443", "", "")
		assert.Contains(t, got, "cluster: default")
		assert.Contains(t, got, "user: default")
	})

	t.Run("leaves unrelated content intact", func(t *testing.T) {
		got := rewriteKubeconfig(testKubeconfig, "https://10.0.0.1:6443", "", "")
		assert.Contains(t, got, "token: secret")
		assert.Contains(t, got, "kind: Config")
	})
}
