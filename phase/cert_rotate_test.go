package phase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestCertRotate_ShouldRun(t *testing.T) {
	tests := []struct {
		name string
		k3s  *cluster.K3s
		want bool
	}{
		{"nil k3s", nil, false},
		{"nil certRotate", &cluster.K3s{}, false},
		{"disabled", &cluster.K3s{CertRotate: &cluster.CertRotateConfig{Enabled: false}}, false},
		{"enabled", &cluster.K3s{CertRotate: &cluster.CertRotateConfig{Enabled: true}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CertRotate{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{K3s: tt.k3s}}
			assert.Equal(t, tt.want, p.ShouldRun())
		})
	}
}
