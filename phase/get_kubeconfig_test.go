package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestGetKubeconfig_Title(t *testing.T) {
	p := &GetKubeconfig{}
	assert.Equal(t, "Get admin kubeconfig", p.Title())
}

func TestGetKubeconfig_ShouldRun_WithLeader(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &GetKubeconfig{}
	p.Config = cfg
	p.Prepare()
	assert.True(t, p.ShouldRun())
}

func TestGetKubeconfig_ShouldRun_NoLeader(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{},
		},
	}
	p := &GetKubeconfig{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestGetKubeconfig_ShouldRun_AgentsOnly(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleAgent},
			},
		},
	}
	p := &GetKubeconfig{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestGetKubeconfig_Prepare_SetsLeader(t *testing.T) {
	leader := &cluster.Host{
		Role:     cluster.RoleServer,
		Metadata: cluster.HostMetadata{IsLeader: true},
	}
	agent := &cluster.Host{
		Role: cluster.RoleAgent,
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{agent, leader},
		},
	}
	p := &GetKubeconfig{}
	p.Config = cfg
	p.Prepare()
	assert.Equal(t, leader, p.leader)
}

func TestGetKubeconfig_Prepare_FallsBackToFirstServer(t *testing.T) {
	server1 := &cluster.Host{Role: cluster.RoleServer}
	server2 := &cluster.Host{Role: cluster.RoleServer}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{server1, server2},
		},
	}
	p := &GetKubeconfig{}
	p.Config = cfg
	p.Prepare()
	assert.Equal(t, server1, p.leader)
}

func TestGetKubeconfig_APIAddressAndOverrides(t *testing.T) {
	p := &GetKubeconfig{
		APIAddress: "https://custom.example.com:6443",
		Cluster:    "my-cluster",
		User:       "admin",
	}

	assert.Equal(t, "https://custom.example.com:6443", p.APIAddress)
	assert.Equal(t, "my-cluster", p.Cluster)
	assert.Equal(t, "admin", p.User)
}
