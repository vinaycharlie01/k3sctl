package phase

import (
	"context"
	"testing"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestValidateFacts_NeedsUpgrade(t *testing.T) {
	const target = "v1.30.0+k3s1"

	tests := []struct {
		name        string
		installed   string
		running     string
		reset       bool
		wantUpgrade bool
		wantErr     bool
	}{
		{"fresh host needs install", "", "", false, true, false},
		{"same version does not need upgrade", target, target, false, false, false},
		{"older version needs upgrade", "v1.29.0+k3s1", "v1.29.0+k3s1", false, true, false},
		{"installed but stopped still needs upgrade", "v1.29.0+k3s1", "", false, true, false},
		{"downgrade is rejected", "v1.31.0+k3s1", "v1.31.0+k3s1", false, false, true},
		{"reset host is skipped", "", "", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &cluster.Host{}
			host.Metadata.K3sInstalledVersion = tt.installed
			host.Metadata.K3sRunningVersion = tt.running
			host.Reset = tt.reset

			p := &ValidateFacts{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
				K3s:   &cluster.K3s{Version: target},
				Hosts: cluster.Hosts{host},
			}}

			err := p.Run(context.Background())
			if (err != nil) != tt.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got := host.Metadata.NeedsUpgrade; got != tt.wantUpgrade {
				t.Errorf("NeedsUpgrade = %v, want %v", got, tt.wantUpgrade)
			}
		})
	}
}

func TestValidateFacts_DisableDowngradeCheck(t *testing.T) {
	const target = "v1.30.0+k3s1"

	host := &cluster.Host{}
	host.Metadata.K3sInstalledVersion = "v1.31.0+k3s1"
	host.Metadata.K3sRunningVersion = "v1.31.0+k3s1"

	p := &ValidateFacts{DisableDowngradeCheck: true}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: target},
		Hosts: cluster.Hosts{host},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if !host.Metadata.NeedsUpgrade {
		t.Error("NeedsUpgrade = false, want true")
	}
}

func TestValidateFacts_NeedsReinstall(t *testing.T) {
	const target = "v1.30.0+k3s1"

	tests := []struct {
		name    string
		running bool
		known   bool
		current cluster.Flags
		desired cluster.Flags
		want    bool
	}{
		{"matching flags", true, true, cluster.Flags{"--disable=traefik"}, cluster.Flags{"--disable=traefik"}, false},
		{"changed flags", true, true, cluster.Flags{"--disable=traefik"}, cluster.Flags{"--disable=servicelb"}, true},
		{"flag order is irrelevant", true, true, cluster.Flags{"--a", "--b"}, cluster.Flags{"--b", "--a"}, false},
		{"service not running", false, true, cluster.Flags{"--a"}, cluster.Flags{"--b"}, false},
		{"unit unreadable", true, false, nil, cluster.Flags{"--a"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &cluster.Host{InstallFlags: tt.desired}
			host.Metadata.K3sInstalledVersion = target
			host.Metadata.K3sRunningVersion = target
			host.Metadata.K3sServiceRunning = tt.running
			host.Metadata.K3sInstallFlagsKnown = tt.known
			host.Metadata.K3sCurrentInstallFlags = tt.current

			p := &ValidateFacts{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
				K3s:   &cluster.K3s{Version: target},
				Hosts: cluster.Hosts{host},
			}}

			if err := p.Run(context.Background()); err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}
			if got := host.Metadata.NeedsReinstall; got != tt.want {
				t.Errorf("NeedsReinstall = %v, want %v", got, tt.want)
			}
		})
	}
}
