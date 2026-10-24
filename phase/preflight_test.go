package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestPreflight_ShouldRun(t *testing.T) {
	tests := []struct {
		name string
		k3s  *cluster.K3s
		want bool
	}{
		{"nil k3s", nil, false},
		{"nil preflight", &cluster.K3s{}, false},
		{"preflight disabled", &cluster.K3s{Preflight: &cluster.PreflightConfig{Enabled: false}}, false},
		{"preflight enabled", &cluster.K3s{Preflight: &cluster.PreflightConfig{Enabled: true}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preflight{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{K3s: tt.k3s}}
			assert.Equal(t, tt.want, p.ShouldRun())
		})
	}
}

func TestPreflight_RequiredPorts(t *testing.T) {
	assert.Contains(t, requiredPorts, "server")
	assert.Contains(t, requiredPorts, "agent")
	assert.Contains(t, requiredPorts["server"], 6443)
	assert.Contains(t, requiredPorts["server"], 10250)
	assert.Contains(t, requiredPorts["server"], 2379)
	assert.Contains(t, requiredPorts["server"], 2380)
	assert.Contains(t, requiredPorts["agent"], 10250)
}

func TestPreflight_RequiredModules(t *testing.T) {
	assert.Contains(t, requiredModules, "br_netfilter")
	assert.Contains(t, requiredModules, "overlay")
	assert.Contains(t, requiredModules, "ip_tables")
	assert.Contains(t, requiredModules, "nf_conntrack")
}
