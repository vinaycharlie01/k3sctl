package phase

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

func TestConfigDiff_NoDifferences(t *testing.T) {
	host := &cluster.Host{}
	host.Metadata.K3sInstalledVersion = "v1.30.0+k3s1"
	host.Metadata.K3sServiceInstalled = true
	host.Metadata.K3sServiceRunning = true

	var buf bytes.Buffer
	p := &ConfigDiff{Writer: &buf}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
		Hosts: cluster.Hosts{host},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(buf.String(), "No differences found") {
		t.Errorf("output = %q, want 'No differences found'", buf.String())
	}
	if len(p.Entries) != 0 {
		t.Errorf("Entries = %d, want 0", len(p.Entries))
	}
}

func TestConfigDiff_VersionMismatch(t *testing.T) {
	host := &cluster.Host{}
	host.Metadata.K3sInstalledVersion = "v1.29.0+k3s1"
	host.Metadata.K3sServiceInstalled = true
	host.Metadata.K3sServiceRunning = true

	var buf bytes.Buffer
	p := &ConfigDiff{Writer: &buf}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
		Hosts: cluster.Hosts{host},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(p.Entries) != 1 {
		t.Fatalf("Entries = %d, want 1", len(p.Entries))
	}
	e := p.Entries[0]
	if e.Field != "k3s-version" {
		t.Errorf("Field = %q, want 'k3s-version'", e.Field)
	}
	if e.Current != "v1.29.0+k3s1" {
		t.Errorf("Current = %q, want 'v1.29.0+k3s1'", e.Current)
	}
	if e.Desired != "v1.30.0+k3s1" {
		t.Errorf("Desired = %q, want 'v1.30.0+k3s1'", e.Desired)
	}
}

func TestConfigDiff_NotInstalled(t *testing.T) {
	host := &cluster.Host{}
	host.Metadata.K3sServiceInstalled = false

	var buf bytes.Buffer
	p := &ConfigDiff{Writer: &buf}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
		Hosts: cluster.Hosts{host},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(p.Entries) != 1 {
		t.Fatalf("Entries = %d, want 1", len(p.Entries))
	}
	e := p.Entries[0]
	if e.Field != "k3s" {
		t.Errorf("Field = %q, want 'k3s'", e.Field)
	}
	if e.Current != "(not installed)" {
		t.Errorf("Current = %q, want '(not installed)'", e.Current)
	}
}

func TestConfigDiff_ServiceStopped(t *testing.T) {
	host := &cluster.Host{}
	host.Metadata.K3sInstalledVersion = "v1.30.0+k3s1"
	host.Metadata.K3sServiceInstalled = true
	host.Metadata.K3sServiceRunning = false

	var buf bytes.Buffer
	p := &ConfigDiff{Writer: &buf}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
		Hosts: cluster.Hosts{host},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var found bool
	for _, e := range p.Entries {
		if e.Field == "service" && e.Current == "stopped" && e.Desired == "running" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected service stopped→running entry, got %v", p.Entries)
	}
}

func TestConfigDiff_InstallFlagsMismatch(t *testing.T) {
	host := &cluster.Host{}
	host.Metadata.K3sInstalledVersion = "v1.30.0+k3s1"
	host.Metadata.K3sServiceInstalled = true
	host.Metadata.K3sServiceRunning = true
	host.Metadata.K3sInstallFlagsKnown = true
	host.Metadata.K3sCurrentInstallFlags = cluster.Flags{"--disable=traefik"}
	host.InstallFlags = cluster.Flags{"--disable=traefik", "--disable=servicelb"}

	var buf bytes.Buffer
	p := &ConfigDiff{Writer: &buf}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
		Hosts: cluster.Hosts{host},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var found bool
	for _, e := range p.Entries {
		if e.Field == "install-flags" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected install-flags entry, got %v", p.Entries)
	}
}

func TestConfigDiff_ConfigKeyChanges(t *testing.T) {
	host := &cluster.Host{}
	host.Metadata.K3sInstalledVersion = "v1.30.0+k3s1"
	host.Metadata.K3sServiceInstalled = true
	host.Metadata.K3sServiceRunning = true
	host.Metadata.K3sExistingConfig = "cluster-cidr: 10.42.0.0/16\nservice-cidr: 10.43.0.0/16\nold-key: old-value"

	var buf bytes.Buffer
	p := &ConfigDiff{Writer: &buf}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s: &cluster.K3s{
			Version: "v1.30.0+k3s1",
			Config: map[string]interface{}{
				"cluster-cidr": "10.42.0.0/16",
				"service-cidr": "10.44.0.0/16",
				"new-key":      "new-value",
			},
		},
		Hosts: cluster.Hosts{host},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	entryMap := make(map[string]DiffEntry)
	for _, e := range p.Entries {
		entryMap[e.Field] = e
	}

	// cluster-cidr unchanged — should not appear
	if _, ok := entryMap["config:cluster-cidr"]; ok {
		t.Error("cluster-cidr should not appear (unchanged)")
	}

	// service-cidr changed
	if e, ok := entryMap["config:service-cidr"]; !ok {
		t.Error("expected config:service-cidr entry")
	} else {
		if e.Current != "10.43.0.0/16" || e.Desired != "10.44.0.0/16" {
			t.Errorf("service-cidr: current=%q desired=%q", e.Current, e.Desired)
		}
	}

	// old-key removed
	if e, ok := entryMap["config:old-key"]; !ok {
		t.Error("expected config:old-key entry")
	} else if e.Desired != "(removed)" {
		t.Errorf("old-key desired = %q, want '(removed)'", e.Desired)
	}

	// new-key added
	if e, ok := entryMap["config:new-key"]; !ok {
		t.Error("expected config:new-key entry")
	} else if e.Current != "(unset)" {
		t.Errorf("new-key current = %q, want '(unset)'", e.Current)
	}
}

func TestConfigDiff_MultipleHosts(t *testing.T) {
	h1 := &cluster.Host{Hostname: "node-1"}
	h1.Metadata.K3sInstalledVersion = "v1.29.0+k3s1"
	h1.Metadata.K3sServiceInstalled = true
	h1.Metadata.K3sServiceRunning = true

	h2 := &cluster.Host{Hostname: "node-2"}
	h2.Metadata.K3sInstalledVersion = "v1.30.0+k3s1"
	h2.Metadata.K3sServiceInstalled = true
	h2.Metadata.K3sServiceRunning = true

	var buf bytes.Buffer
	p := &ConfigDiff{Writer: &buf}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
		Hosts: cluster.Hosts{h1, h2},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Only h1 should have a diff (version mismatch)
	if len(p.Entries) != 1 {
		t.Fatalf("Entries = %d, want 1", len(p.Entries))
	}
	if p.Entries[0].Host != "node-1" {
		t.Errorf("Host = %q, want 'node-1'", p.Entries[0].Host)
	}
}

func TestConfigDiff_TableOutput(t *testing.T) {
	host := &cluster.Host{Hostname: "node-1"}
	host.Metadata.K3sInstalledVersion = "v1.29.0+k3s1"
	host.Metadata.K3sServiceInstalled = true
	host.Metadata.K3sServiceRunning = true

	var buf bytes.Buffer
	p := &ConfigDiff{Writer: &buf}
	p.Config = &v1beta1.Cluster{Spec: &cluster.Spec{
		K3s:   &cluster.K3s{Version: "v1.30.0+k3s1"},
		Hosts: cluster.Hosts{host},
	}}

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "1 difference(s) found") {
		t.Errorf("missing count header in output: %q", out)
	}
	if !strings.Contains(out, "HOST") || !strings.Contains(out, "FIELD") {
		t.Errorf("missing table headers in output: %q", out)
	}
	if !strings.Contains(out, "→") {
		t.Errorf("missing arrow separator in output: %q", out)
	}
}

func TestParseConfigKeys(t *testing.T) {
	raw := `cluster-cidr: 10.42.0.0/16
service-cidr: 10.43.0.0/16
# a comment
disable: traefik

with-colon: http://example.com:8080`

	got := parseConfigKeys(raw)
	want := map[string]string{
		"cluster-cidr": "10.42.0.0/16",
		"service-cidr": "10.43.0.0/16",
		"disable":      "traefik",
		"with-colon":   "http://example.com:8080",
	}

	if len(got) != len(want) {
		t.Fatalf("parseConfigKeys returned %d keys, want %d: %v", len(got), len(want), got)
	}
	for k, wantV := range want {
		if gotV, ok := got[k]; !ok {
			t.Errorf("missing key %q", k)
		} else if gotV != wantV {
			t.Errorf("key %q = %q, want %q", k, gotV, wantV)
		}
	}
}

func TestFormatFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags cluster.Flags
		want  string
	}{
		{"empty", nil, "(none)"},
		{"single", cluster.Flags{"--disable=traefik"}, "--disable=traefik"},
		{"sorted", cluster.Flags{"--b", "--a"}, "--a --b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFlags(tt.flags)
			if got != tt.want {
				t.Errorf("formatFlags(%v) = %q, want %q", tt.flags, got, tt.want)
			}
		})
	}
}
