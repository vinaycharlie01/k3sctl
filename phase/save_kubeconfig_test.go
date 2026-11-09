package phase

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
)

const testKubeconfigWithCA = `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ` + "dGVzdC1jYS1jZXJ0LWRhdGE=" + `
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
    client-certificate-data: ` + "dGVzdC1jbGllbnQtY2VydA==" + `
    client-key-data: ` + "dGVzdC1jbGllbnQta2V5" + `
`

func TestSaveKubeconfig_ShouldRun(t *testing.T) {
	t.Run("true when kubeconfig is set", func(t *testing.T) {
		p := &SaveKubeconfig{}
		p.Config = &v1beta1.Cluster{
			Metadata: v1beta1.ClusterMetadata{Kubeconfig: testKubeconfig},
		}
		assert.True(t, p.ShouldRun())
	})

	t.Run("false when kubeconfig is empty", func(t *testing.T) {
		p := &SaveKubeconfig{}
		p.Config = &v1beta1.Cluster{}
		assert.False(t, p.ShouldRun())
	})
}

func TestSaveKubeconfig_Run(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".kube", "config")

	p := &SaveKubeconfig{Path: path}
	p.Config = &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{Kubeconfig: testKubeconfig},
	}

	err := p.Run(t.Context())
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "https://127.0.0.1:6443")
}

func TestSaveKubeconfig_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config")

	p := &SaveKubeconfig{Path: path}
	p.Config = &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{Kubeconfig: testKubeconfig},
	}

	err := p.Run(t.Context())
	require.NoError(t, err)

	info, err := os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestSaveKubeconfig_PreservesCAData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	p := &SaveKubeconfig{Path: path}
	p.Config = &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{Kubeconfig: testKubeconfigWithCA},
	}

	err := p.Run(t.Context())
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	config, err := clientcmd.Load(data)
	require.NoError(t, err)

	cluster := config.Clusters["default"]
	require.NotNil(t, cluster, "cluster 'default' should exist")
	assert.Equal(t, "test-ca-cert-data", string(cluster.CertificateAuthorityData))

	user := config.AuthInfos["default"]
	require.NotNil(t, user, "user 'default' should exist")
	assert.Equal(t, "test-client-cert", string(user.ClientCertificateData))
	assert.Equal(t, "test-client-key", string(user.ClientKeyData))
}

func TestSaveKubeconfig_FullPipeline(t *testing.T) {
	raw := []byte(testKubeconfigWithCA)

	rewritten, err := func() ([]byte, error) {
		config, err := clientcmd.Load(raw)
		if err != nil {
			return nil, err
		}
		for _, c := range config.Clusters {
			c.Server = "https://10.0.0.1:6443"
		}
		return clientcmd.Write(*config)
	}()
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	p := &SaveKubeconfig{Path: path}
	p.Config = &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{Kubeconfig: string(rewritten)},
	}

	err = p.Run(t.Context())
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	config, err := clientcmd.Load(data)
	require.NoError(t, err)

	cluster := config.Clusters["default"]
	require.NotNil(t, cluster)
	assert.Equal(t, "https://10.0.0.1:6443", cluster.Server)
	assert.Equal(t, "test-ca-cert-data", string(cluster.CertificateAuthorityData))

	raw64 := base64.StdEncoding.EncodeToString([]byte("test-ca-cert-data"))
	assert.Contains(t, string(data), raw64, "saved file should contain base64-encoded CA data")
}
