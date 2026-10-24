package phase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestCertificateCheck_ShouldRun(t *testing.T) {
	server := &cluster.Host{Role: cluster.RoleServer}
	agent := &cluster.Host{Role: cluster.RoleAgent}

	tests := []struct {
		name  string
		hosts cluster.Hosts
		want  bool
	}{
		{"has servers", cluster.Hosts{server}, true},
		{"only agents", cluster.Hosts{agent}, false},
		{"no hosts", cluster.Hosts{}, false},
		{"mixed", cluster.Hosts{server, agent}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &CertificateCheck{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{Hosts: tt.hosts}}
			assert.Equal(t, tt.want, p.ShouldRun())
		})
	}
}

func TestParseOpenSSLDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"standard format", "Oct 20 12:00:00 2025 GMT", false},
		{"single digit day space-padded", "Oct  5 12:00:00 2025 GMT", false},
		{"invalid format", "2025-10-20", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOpenSSLDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 2025, got.Year())
				assert.Equal(t, time.October, got.Month())
			}
		})
	}
}

func TestTokenList_ShouldRun(t *testing.T) {
	server := &cluster.Host{Role: cluster.RoleServer}
	agent := &cluster.Host{Role: cluster.RoleAgent}

	tests := []struct {
		name  string
		hosts cluster.Hosts
		want  bool
	}{
		{"has servers", cluster.Hosts{server}, true},
		{"only agents", cluster.Hosts{agent}, false},
		{"no hosts", cluster.Hosts{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &TokenList{}
			p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{Hosts: tt.hosts}}
			assert.Equal(t, tt.want, p.ShouldRun())
		})
	}
}
