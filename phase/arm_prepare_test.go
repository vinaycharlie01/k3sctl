package phase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestNeedsEtcdUnsupportedArch(t *testing.T) {
	tests := []struct {
		name    string
		arch    string
		version string
		want    bool
	}{
		{"amd64", "amd64", "v1.28.3+k3s1", false},
		{"arm64 on modern k3s", "arm64", "v1.28.3+k3s1", false},
		{"aarch64 on modern k3s", "aarch64", "v1.28.3+k3s1", false},
		{"arm64 on etcd 3.4", "arm64", "v1.21.5+k3s1", true},
		{"aarch64 exactly at etcd 3.5", "aarch64", "v1.22.1+k3s1", false},
		{"32-bit arm always needs override", "arm", "v1.28.3+k3s1", true},
		{"armv7l always needs override", "armv7l", "v1.28.3+k3s1", true},
		{"unknown architecture", "", "v1.28.3+k3s1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, needsEtcdUnsupportedArch(tt.arch, tt.version))
		})
	}
}

func TestPrepareArmRunSetsEnvironment(t *testing.T) {
	host := &cluster.Host{Role: cluster.RoleServer}
	host.Metadata.Arch = "arm"
	p := &PrepareArm{hosts: cluster.Hosts{host}}

	require.NoError(t, p.Run(context.Background()))
	assert.Equal(t, "arm", host.Environment["ETCD_UNSUPPORTED_ARCH"])
}
