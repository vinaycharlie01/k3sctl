package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestCluster_SetDefaults(t *testing.T) {
	c := &Cluster{}
	c.SetDefaults()
	assert.Equal(t, APIVersion, c.APIVersion)
	assert.Equal(t, "Cluster", c.Kind)
	assert.Equal(t, "k3s-cluster", c.Metadata.Name)
}

func TestCluster_SetDefaults_PreservesValues(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
		Metadata:   ClusterMetadata{Name: "my-cluster"},
	}
	c.SetDefaults()
	assert.Equal(t, "my-cluster", c.Metadata.Name)
}

func TestCluster_SetDefaults_WithSpec(t *testing.T) {
	c := &Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	c.SetDefaults()
	assert.Equal(t, APIVersion, c.APIVersion)
}

func TestCluster_Validate_Valid(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	err := c.Validate()
	require.NoError(t, err)
}

func TestCluster_Validate_WrongAPIVersion(t *testing.T) {
	c := &Cluster{
		APIVersion: "wrong/v1",
		Kind:       "Cluster",
		Spec:       &cluster.Spec{},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported apiVersion")
}

func TestCluster_Validate_WrongKind(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Wrong",
		Spec:       &cluster.Spec{},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported kind")
}

func TestCluster_Validate_NilSpec(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec is required")
}

func TestCluster_Resolve(t *testing.T) {
	c := &Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	err := c.Resolve()
	require.NoError(t, err)
	assert.Equal(t, APIVersion, c.APIVersion)
	assert.Equal(t, "Cluster", c.Kind)
}

func TestCluster_Resolve_Invalid(t *testing.T) {
	c := &Cluster{
		APIVersion: "wrong",
	}
	err := c.Resolve()
	require.Error(t, err)
}

func TestAPIVersion_Constant(t *testing.T) {
	assert.Equal(t, "k3sctl.k3sproject.io/v1beta1", APIVersion)
}

func TestCluster_Validate_NoHosts(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{},
		},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one host")
}

func TestCluster_Validate_NoServers(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleAgent},
			},
		},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one server")
}

func TestCluster_Validate_InvalidHostRole(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
				{Role: "invalid"},
			},
		},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func TestCluster_Validate_K3sVersionInvalid(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
			K3s: &cluster.K3s{Version: "1.28.0"},
		},
	}
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must start with 'v'")
}

func TestCluster_Validate_K3sVersionValid(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
			K3s: &cluster.K3s{Version: "v1.28.0+k3s1"},
		},
	}
	err := c.Validate()
	require.NoError(t, err)
}

func TestCluster_Validate_MultipleHosts(t *testing.T) {
	c := &Cluster{
		APIVersion: APIVersion,
		Kind:       "Cluster",
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
				{Role: cluster.RoleServer},
				{Role: cluster.RoleAgent},
			},
		},
	}
	err := c.Validate()
	require.NoError(t, err)
}
