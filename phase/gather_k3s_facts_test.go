package phase

import (
	"testing"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestParseExecStartFlags(t *testing.T) {
	tests := []struct {
		name string
		unit string
		want cluster.Flags
	}{
		{
			"server flags",
			"[Service]\nExecStart=/usr/local/bin/k3s server --disable=traefik --node-label=a=b\n",
			cluster.Flags{"--disable=traefik", "--node-label=a=b"},
		},
		{
			"agent flags",
			"ExecStart=/usr/local/bin/k3s agent --node-taint=x=y\n",
			cluster.Flags{"--node-taint=x=y"},
		},
		{
			"no flags",
			"ExecStart=/usr/local/bin/k3s server\n",
			cluster.Flags{},
		},
		{
			"ignores other directives",
			"ExecStartPre=-/sbin/modprobe overlay\nExecStart=/usr/local/bin/k3s agent --foo\n",
			cluster.Flags{"--foo"},
		},
		{
			"ignores unknown subcommand",
			"ExecStart=/usr/bin/env k3s\n",
			cluster.Flags{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseExecStartFlags(tt.unit); !got.Equals(tt.want) {
				t.Errorf("parseExecStartFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}
