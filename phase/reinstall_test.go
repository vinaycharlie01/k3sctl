package phase

import (
	"testing"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestReinstall_ShouldRun(t *testing.T) {
	tests := []struct {
		name               string
		running            bool
		needsReinstall     bool
		needsConfigRestart bool
		wantShouldRun      bool
	}{
		{"no changes", true, false, false, false},
		{"flags changed", true, true, false, true},
		{"config changed", true, false, true, true},
		{"both changed", true, true, true, true},
		{"config changed but not running", false, false, true, false},
		{"flags changed but not running", false, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &cluster.Host{}
			host.Role = cluster.RoleServer
			host.Metadata.K3sServiceRunning = tt.running
			host.Metadata.NeedsReinstall = tt.needsReinstall
			host.Metadata.NeedsConfigRestart = tt.needsConfigRestart

			p := &Reinstall{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
				K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
				Hosts: cluster.Hosts{host},
			}}
			p.Prepare()
			got := p.ShouldRun()
			if got != tt.wantShouldRun {
				t.Errorf("ShouldRun() = %v, want %v", got, tt.wantShouldRun)
			}
		})
	}
}
