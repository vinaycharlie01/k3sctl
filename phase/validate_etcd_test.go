package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestValidateEtcdMembers_Title(t *testing.T) {
	p := &ValidateEtcdMembers{}
	assert.Equal(t, "Validate etcd members", p.Title())
}

func TestValidateEtcdMembers_ShouldRun_MultiServer(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &ValidateEtcdMembers{}
	p.Config = cfg
	p.Prepare()
	assert.True(t, p.ShouldRun())
}

func TestValidateEtcdMembers_ShouldRun_SingleServer(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &ValidateEtcdMembers{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestValidateEtcdMembers_ShouldRun_NoHosts(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{},
		},
	}
	p := &ValidateEtcdMembers{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestValidateEtcdMembers_ShouldRun_AgentsOnly(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleAgent},
				{Role: cluster.RoleAgent},
			},
		},
	}
	p := &ValidateEtcdMembers{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestValidateEtcdMembers_Prepare_SetsLeader(t *testing.T) {
	leader := &cluster.Host{Role: cluster.RoleServer, Metadata: cluster.HostMetadata{IsLeader: true}}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				leader,
				{Role: cluster.RoleServer},
			},
		},
	}
	p := &ValidateEtcdMembers{}
	p.Config = cfg
	p.Prepare()
	assert.Equal(t, leader, p.leader)
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello\nworld", "hello"},
		{"single line", "single line"},
		{"", ""},
		{"first\nsecond\nthird", "first"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, firstLine(tt.input))
		})
	}
}
