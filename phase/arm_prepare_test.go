package phase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestPrepareArm_Title(t *testing.T) {
	p := &PrepareArm{}
	assert.Equal(t, "Prepare ARM nodes", p.Title())
}

func TestPrepareArm_Prepare_FiltersArm32(t *testing.T) {
	tests := []struct {
		name     string
		arch     string
		reset    bool
		included bool
	}{
		{"armv7l included", "armv7l", false, true},
		{"armhf included", "armhf", false, true},
		{"aarch64 excluded (64-bit)", "aarch64", false, false},
		{"arm64 excluded (64-bit)", "arm64", false, false},
		{"x86_64 excluded", "x86_64", false, false},
		{"amd64 excluded", "amd64", false, false},
		{"empty arch excluded", "", false, false},
		{"armv7l reset excluded", "armv7l", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &cluster.Host{
				Role:     cluster.RoleServer,
				Reset:    tt.reset,
				Metadata: cluster.HostMetadata{Arch: tt.arch},
			}
			cfg := &v1beta1.Cluster{
				Spec: &cluster.Spec{
					Hosts: cluster.Hosts{h},
				},
			}
			p := &PrepareArm{}
			p.Config = cfg
			p.Prepare()
			if tt.included {
				assert.Len(t, p.hosts, 1, "expected host to be included")
			} else {
				assert.Len(t, p.hosts, 0, "expected host to be excluded")
			}
		})
	}
}

func TestPrepareArm_ShouldRun_NoArmHosts(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer, Metadata: cluster.HostMetadata{Arch: "x86_64"}},
			},
		},
	}
	p := &PrepareArm{}
	p.Config = cfg
	p.Prepare()
	assert.False(t, p.ShouldRun())
}

func TestPrepareArm_ShouldRun_WithArmHosts(t *testing.T) {
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				{Role: cluster.RoleServer, Metadata: cluster.HostMetadata{Arch: "armv7l"}},
			},
		},
	}
	p := &PrepareArm{}
	p.Config = cfg
	p.Prepare()
	assert.True(t, p.ShouldRun())
}

func TestPrepareArm_Run_SetsEnvironment(t *testing.T) {
	h := &cluster.Host{
		Role:     cluster.RoleServer,
		Metadata: cluster.HostMetadata{Arch: "armv7l"},
	}
	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{h},
		},
	}
	p := &PrepareArm{}
	p.Config = cfg
	p.Manager = &Manager{Concurrency: 1}
	p.Prepare()

	err := p.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "armv7l", h.Environment["ETCD_UNSUPPORTED_ARCH"])
}

func TestPrepareArm_Run_InitializesNilEnvironment(t *testing.T) {
	h := &cluster.Host{
		Role:     cluster.RoleServer,
		Metadata: cluster.HostMetadata{Arch: "armhf"},
	}
	assert.Nil(t, h.Environment)

	cfg := &v1beta1.Cluster{
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{h},
		},
	}
	p := &PrepareArm{}
	p.Config = cfg
	p.Manager = &Manager{Concurrency: 1}
	p.Prepare()

	err := p.Run(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, h.Environment)
	assert.Equal(t, "armhf", h.Environment["ETCD_UNSUPPORTED_ARCH"])
}
