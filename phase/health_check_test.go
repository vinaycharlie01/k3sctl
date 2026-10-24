package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestHealthCheck_ShouldRun(t *testing.T) {
	tests := []struct {
		name  string
		hosts cluster.Hosts
		want  bool
	}{
		{"no hosts", cluster.Hosts{}, false},
		{"with hosts", cluster.Hosts{&cluster.Host{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &HealthCheck{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{Hosts: tt.hosts}}
			assert.Equal(t, tt.want, p.ShouldRun())
		})
	}
}

func TestHealthInfo_DefaultValues(t *testing.T) {
	info := HealthInfo{}
	assert.Empty(t, info.Host)
	assert.Empty(t, info.Role)
	assert.Empty(t, info.K3sStatus)
	assert.Empty(t, info.K3sVersion)
	assert.Empty(t, info.APIServer)
	assert.Empty(t, info.Etcd)
}
