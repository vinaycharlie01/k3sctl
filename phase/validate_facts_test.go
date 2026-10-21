package phase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestValidateFacts_Title(t *testing.T) {
	p := &ValidateFacts{}
	assert.Equal(t, "Validate facts", p.Title())
}

func TestValidateFacts_NoVersion(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			K3s:   &cluster.K3s{},
			Hosts: cluster.Hosts{},
		},
	}
	p := &ValidateFacts{}
	p.Config = cfg
	err := p.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target K3s version not set")
}

func TestValidateFacts_NoDowngrade(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			K3s: &cluster.K3s{Version: "v1.28.0+k3s1"},
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer, Metadata: cluster.HostMetadata{K3sRunningVersion: "v1.29.0+k3s1"}},
			},
		},
	}
	p := &ValidateFacts{}
	p.Config = cfg
	err := p.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "downgrade")
}

func TestValidateFacts_DowngradeCheckDisabled(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			K3s: &cluster.K3s{Version: "v1.28.0+k3s1"},
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer, Metadata: cluster.HostMetadata{K3sRunningVersion: "v1.29.0+k3s1"}},
			},
		},
	}
	p := &ValidateFacts{DisableDowngradeCheck: true}
	p.Config = cfg
	err := p.Run(context.Background())
	require.NoError(t, err)
}

func TestValidateFacts_UpgradeNeeded(t *testing.T) {
	h := &cluster.Host{
		Role:     cluster.RoleServer,
		Metadata: cluster.HostMetadata{K3sRunningVersion: "v1.28.0+k3s1"},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			K3s:   &cluster.K3s{Version: "v1.29.0+k3s1"},
			Hosts: cluster.Hosts{h},
		},
	}
	p := &ValidateFacts{}
	p.Config = cfg
	err := p.Run(context.Background())
	require.NoError(t, err)
	assert.True(t, h.Metadata.NeedsUpgrade)
}

func TestValidateFacts_SameVersionNoUpgrade(t *testing.T) {
	h := &cluster.Host{
		Role:     cluster.RoleServer,
		Metadata: cluster.HostMetadata{K3sRunningVersion: "v1.29.0+k3s1"},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			K3s:   &cluster.K3s{Version: "v1.29.0+k3s1"},
			Hosts: cluster.Hosts{h},
		},
	}
	p := &ValidateFacts{}
	p.Config = cfg
	err := p.Run(context.Background())
	require.NoError(t, err)
	assert.False(t, h.Metadata.NeedsUpgrade)
}

func TestValidateFacts_ResetHostSkipped(t *testing.T) {
	h := &cluster.Host{
		Role:     cluster.RoleServer,
		Reset:    true,
		Metadata: cluster.HostMetadata{K3sRunningVersion: "v1.30.0+k3s1"},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			K3s:   &cluster.K3s{Version: "v1.28.0+k3s1"},
			Hosts: cluster.Hosts{h},
		},
	}
	p := &ValidateFacts{}
	p.Config = cfg
	err := p.Run(context.Background())
	require.NoError(t, err)
}

func TestValidateFacts_FreshInstallSkipped(t *testing.T) {
	h := &cluster.Host{
		Role:     cluster.RoleServer,
		Metadata: cluster.HostMetadata{K3sRunningVersion: ""},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			K3s:   &cluster.K3s{Version: "v1.29.0+k3s1"},
			Hosts: cluster.Hosts{h},
		},
	}
	p := &ValidateFacts{}
	p.Config = cfg
	err := p.Run(context.Background())
	require.NoError(t, err)
	assert.False(t, h.Metadata.NeedsUpgrade)
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.28.0+k3s1", "000000000100000000280000000000"},
		{"1.28.0", "000000000100000000280000000000"},
		{"v1.29.1+k3s2", "000000000100000000290000000001"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeVersion(tt.input))
		})
	}
}

func TestIsDowngrade(t *testing.T) {
	tests := []struct {
		current  string
		target   string
		expected bool
	}{
		{"v1.29.0+k3s1", "v1.28.0+k3s1", true},
		{"v1.28.0+k3s1", "v1.29.0+k3s1", false},
		{"v1.28.0+k3s1", "v1.28.0+k3s1", false},
		{"v1.28.1+k3s1", "v1.28.0+k3s1", true},
	}
	for _, tt := range tests {
		t.Run(tt.current+"->"+tt.target, func(t *testing.T) {
			assert.Equal(t, tt.expected, isDowngrade(tt.current, tt.target))
		})
	}
}
