package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestClusterSetDefaults(t *testing.T) {
	h := &cluster.Host{}
	cfg := &Cluster{Spec: &cluster.Spec{Hosts: cluster.Hosts{h}}}

	cfg.SetDefaults()

	assert.Equal(t, APIVersion, cfg.APIVersion)
	assert.Equal(t, "Cluster", cfg.Kind)
	assert.Equal(t, "k3s-cluster", cfg.Metadata.Name)
	require.NotNil(t, cfg.Spec.K3s)
	assert.Equal(t, cluster.RoleServer, h.Role)
}

func TestClusterSetDefaultsWithoutSpec(t *testing.T) {
	cfg := &Cluster{}
	require.NotPanics(t, cfg.SetDefaults)
	assert.Equal(t, APIVersion, cfg.APIVersion)
}

func TestClusterValidate(t *testing.T) {
	validSpec := &cluster.Spec{Hosts: cluster.Hosts{{Role: cluster.RoleServer}}}

	t.Run("accepts a valid cluster", func(t *testing.T) {
		cfg := &Cluster{APIVersion: APIVersion, Kind: "Cluster", Spec: validSpec}
		require.NoError(t, cfg.Validate())
	})

	t.Run("rejects a mismatched apiVersion", func(t *testing.T) {
		cfg := &Cluster{APIVersion: "v1", Kind: "Cluster", Spec: validSpec}
		require.Error(t, cfg.Validate())
	})

	t.Run("rejects a mismatched kind", func(t *testing.T) {
		cfg := &Cluster{APIVersion: APIVersion, Kind: "Node", Spec: validSpec}
		require.Error(t, cfg.Validate())
	})

	t.Run("requires a spec", func(t *testing.T) {
		cfg := &Cluster{APIVersion: APIVersion, Kind: "Cluster"}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "spec is required")
	})

	t.Run("delegates to the spec", func(t *testing.T) {
		cfg := &Cluster{APIVersion: APIVersion, Kind: "Cluster", Spec: &cluster.Spec{}}
		require.Error(t, cfg.Validate())
	})
}

func TestClusterResolve(t *testing.T) {
	cfg := &Cluster{Spec: &cluster.Spec{Hosts: cluster.Hosts{{Role: cluster.RoleServer}}}}
	require.NoError(t, cfg.Resolve())
	assert.Equal(t, APIVersion, cfg.APIVersion)
}
