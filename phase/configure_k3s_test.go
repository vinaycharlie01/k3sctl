package phase

import (
	"encoding/hex"
	"testing"

	"github.com/k0sproject/dig"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestConfigureK3s_ResolveClusterToken(t *testing.T) {
	t.Run("keeps token discovered on the leader", func(t *testing.T) {
		leader := &cluster.Host{}
		leader.Metadata.NodeToken = "existing-token"

		p := configurePhase(leader, &cluster.K3s{})
		got, err := p.resolveClusterToken()
		if err != nil {
			t.Fatalf("resolveClusterToken() error = %v", err)
		}
		if got != "existing-token" {
			t.Errorf("resolveClusterToken() = %q, want %q", got, "existing-token")
		}
	})

	t.Run("uses a token from the config", func(t *testing.T) {
		leader := &cluster.Host{}

		p := configurePhase(leader, &cluster.K3s{Config: dig.Mapping{"token": "configured-token"}})
		got, err := p.resolveClusterToken()
		if err != nil {
			t.Fatalf("resolveClusterToken() error = %v", err)
		}
		if got != "configured-token" {
			t.Errorf("resolveClusterToken() = %q, want %q", got, "configured-token")
		}
		if leader.Metadata.NodeToken != "configured-token" {
			t.Errorf("leader token = %q, want %q", leader.Metadata.NodeToken, "configured-token")
		}
	})

	t.Run("generates a token for a fresh cluster", func(t *testing.T) {
		leader := &cluster.Host{}

		p := configurePhase(leader, &cluster.K3s{})
		got, err := p.resolveClusterToken()
		if err != nil {
			t.Fatalf("resolveClusterToken() error = %v", err)
		}
		if len(got) != 64 {
			t.Fatalf("generated token length = %d, want 64", len(got))
		}
		if _, err := hex.DecodeString(got); err != nil {
			t.Errorf("generated token is not hex: %v", err)
		}
		if leader.Metadata.NodeToken != got {
			t.Errorf("leader token = %q, want %q", leader.Metadata.NodeToken, got)
		}
	})

	t.Run("fails when a running leader has no known token", func(t *testing.T) {
		leader := &cluster.Host{}
		leader.Metadata.K3sServiceRunning = true

		p := configurePhase(leader, &cluster.K3s{})
		if _, err := p.resolveClusterToken(); err == nil {
			t.Fatal("resolveClusterToken() error = nil, want error")
		}
	})

	t.Run("fails without a leader", func(t *testing.T) {
		p := &ConfigureK3s{}
		p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{K3s: &cluster.K3s{}}}

		if _, err := p.resolveClusterToken(); err == nil {
			t.Fatal("resolveClusterToken() error = nil, want error")
		}
	})
}

func configurePhase(leader *cluster.Host, k3s *cluster.K3s) *ConfigureK3s {
	p := &ConfigureK3s{leader: leader}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   k3s,
		Hosts: cluster.Hosts{leader},
	}}
	return p
}
