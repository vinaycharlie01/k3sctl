package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestApplyManifests_Title(t *testing.T) {
	p := &ApplyManifests{}
	assert.Equal(t, "Apply additional manifests", p.Title())
}

func TestApplyManifests_ShouldRun_WithManifestsAndLeader(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{
			Manifests: map[string][]byte{
				"test": []byte("apiVersion: v1\nkind: ConfigMap"),
			},
		},
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &ApplyManifests{}
	p.Config = cfg
	p.Prepare()
	assert.True(t, p.ShouldRun())
}

func TestApplyManifests_ShouldRun_NoManifests(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{},
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &ApplyManifests{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestApplyManifests_ShouldRun_NoLeader(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{
			Manifests: map[string][]byte{
				"test": []byte("data"),
			},
		},
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{},
		},
	}
	p := &ApplyManifests{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestApplyManifests_ShouldRun_EmptyManifests(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Metadata: v1beta1.ClusterMetadata{
			Manifests: map[string][]byte{},
		},
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &ApplyManifests{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestApplyManifests_Prepare_SetsLeader(t *testing.T) {
	leader := &cluster.Host{Role: cluster.RoleServer, Metadata: cluster.HostMetadata{IsLeader: true}}
	agent := &cluster.Host{Role: cluster.RoleAgent}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{agent, leader},
		},
	}
	p := &ApplyManifests{}
	p.Config = cfg
	p.Prepare()
	assert.Equal(t, leader, p.leader)
}
