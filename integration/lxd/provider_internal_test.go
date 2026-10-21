package lxd

import (
	"strings"
	"testing"
	"time"

	"github.com/canonical/lxd/shared/api"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
)

const testAuthorizedKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyMaterialForUnitTests test@example"

func validConfig() Config {
	return Config{
		RunID: "run-1",
		Nodes: []NodeSpec{
			{Name: "server", Role: environment.RoleServer},
			{Name: "agent1", Role: environment.RoleAgent},
		},
		SSHAuthorizedKey: testAuthorizedKey,
	}.withDefaults()
}

func TestConfigWithDefaults(t *testing.T) {
	cfg := Config{}.withDefaults()

	if cfg.InstancePrefix != defaultInstancePrefix {
		t.Errorf("InstancePrefix = %q, want %q", cfg.InstancePrefix, defaultInstancePrefix)
	}
	if cfg.ProfilePrefix != defaultProfilePrefix {
		t.Errorf("ProfilePrefix = %q, want %q", cfg.ProfilePrefix, defaultProfilePrefix)
	}
	if cfg.ImageAlias != defaultImageAlias {
		t.Errorf("ImageAlias = %q, want %q", cfg.ImageAlias, defaultImageAlias)
	}
	if cfg.ImageProtocol != defaultImageProtocol {
		t.Errorf("ImageProtocol = %q, want %q", cfg.ImageProtocol, defaultImageProtocol)
	}
	if len(cfg.BaseProfiles) != 1 || cfg.BaseProfiles[0] != "default" {
		t.Errorf("BaseProfiles = %v, want [default]", cfg.BaseProfiles)
	}
	if cfg.SSHUser != defaultSSHUser {
		t.Errorf("SSHUser = %q, want %q", cfg.SSHUser, defaultSSHUser)
	}
	if cfg.SSHPort != defaultSSHPort {
		t.Errorf("SSHPort = %d, want %d", cfg.SSHPort, defaultSSHPort)
	}
	if cfg.OperationTimeout <= 0 || cfg.BootstrapTimeout <= 0 || cfg.AddressTimeout <= 0 || cfg.PollInterval <= 0 {
		t.Errorf("timeouts must be positive: %+v", cfg)
	}
}

func TestConfigWithDefaultsPreservesExplicitValues(t *testing.T) {
	cfg := Config{
		InstancePrefix:   "custom",
		ProfilePrefix:    "profile",
		ImageAlias:       "22.04",
		BaseProfiles:     []string{"base"},
		SSHUser:          "admin",
		SSHPort:          2222,
		OperationTimeout: time.Minute,
		PollInterval:     10 * time.Millisecond,
	}.withDefaults()

	if cfg.InstancePrefix != "custom" || cfg.ProfilePrefix != "profile" || cfg.ImageAlias != "22.04" {
		t.Errorf("explicit prefixes/image were overwritten: %+v", cfg)
	}
	if len(cfg.BaseProfiles) != 1 || cfg.BaseProfiles[0] != "base" {
		t.Errorf("BaseProfiles = %v, want [base]", cfg.BaseProfiles)
	}
	if cfg.SSHUser != "admin" || cfg.SSHPort != 2222 || cfg.OperationTimeout != time.Minute || cfg.PollInterval != 10*time.Millisecond {
		t.Errorf("explicit values were overwritten: %+v", cfg)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:   "valid",
			mutate: func(*Config) {},
		},
		{
			name:    "missing run ID",
			mutate:  func(c *Config) { c.RunID = "   " },
			wantErr: "run ID is required",
		},
		{
			name:    "no nodes",
			mutate:  func(c *Config) { c.Nodes = nil },
			wantErr: "at least one node is required",
		},
		{
			name:    "invalid node name",
			mutate:  func(c *Config) { c.Nodes[0].Name = "Server_1" },
			wantErr: "invalid node name",
		},
		{
			name:    "invalid role",
			mutate:  func(c *Config) { c.Nodes[0].Role = "worker" },
			wantErr: "invalid role",
		},
		{
			name: "duplicate node name",
			mutate: func(c *Config) {
				c.Nodes = append(c.Nodes, NodeSpec{Name: "server", Role: environment.RoleAgent})
			},
			wantErr: "duplicate node name",
		},
		{
			name:    "invalid SSH user",
			mutate:  func(c *Config) { c.SSHUser = "bad user" },
			wantErr: "invalid SSH user",
		},
		{
			name:    "invalid SSH port",
			mutate:  func(c *Config) { c.SSHPort = 70000 },
			wantErr: "invalid SSH port",
		},
		{
			name:    "instance name too long",
			mutate:  func(c *Config) { c.Nodes[0].Name = strings.Repeat("a", maxNameLength) },
			wantErr: "exceeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validate() = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validate() = %q, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeRunID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"1234-2", "1234-2"},
		{"Run ID/With Spaces", "run-id-with-spaces"},
		{"--trim--", "trim"},
		{"", "run"},
		{"!!!", "run"},
		{strings.Repeat("a", 40), strings.Repeat("a", 32)},
		{strings.Repeat("a", 31) + "-" + strings.Repeat("b", 10), strings.Repeat("a", 31)},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := sanitizeRunID(tt.in); got != tt.want {
				t.Fatalf("sanitizeRunID(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResourceNames(t *testing.T) {
	cfg := validConfig()
	cfg.RunID = "1234-2"

	if got, want := cfg.instanceName("server"), "k3sctl-1234-2-server"; got != want {
		t.Errorf("instanceName = %q, want %q", got, want)
	}
	if got, want := cfg.profileName(), "k3sctl-1234-2"; got != want {
		t.Errorf("profileName = %q, want %q", got, want)
	}
}

func TestPickIPv4(t *testing.T) {
	network := func(addresses ...api.InstanceStateNetworkAddress) api.InstanceStateNetwork {
		return api.InstanceStateNetwork{Addresses: addresses}
	}
	addr := func(family, address, scope string) api.InstanceStateNetworkAddress {
		return api.InstanceStateNetworkAddress{Family: family, Address: address, Scope: scope}
	}

	tests := []struct {
		name    string
		state   *api.InstanceState
		want    string
		wantErr bool
	}{
		{
			name: "single global IPv4",
			state: &api.InstanceState{Network: map[string]api.InstanceStateNetwork{
				"eth0": network(addr("inet6", "fd42::1", "global"), addr("inet", "10.0.0.5", "global")),
			}},
			want: "10.0.0.5",
		},
		{
			name: "ignores loopback, link-local and unspecified",
			state: &api.InstanceState{Network: map[string]api.InstanceStateNetwork{
				"lo": network(addr("inet", "127.0.0.1", "global")),
				"eth0": network(
					addr("inet", "169.254.1.1", "link"),
					addr("inet", "0.0.0.0", "global"),
					addr("inet", "192.168.1.10", "global"),
				),
			}},
			want: "192.168.1.10",
		},
		{
			name: "ignores non-global scope",
			state: &api.InstanceState{Network: map[string]api.InstanceStateNetwork{
				"eth0": network(addr("inet", "10.0.0.5", "link")),
			}},
			wantErr: true,
		},
		{
			name: "deterministic interface order",
			state: &api.InstanceState{Network: map[string]api.InstanceStateNetwork{
				"eth1": network(addr("inet", "10.0.0.9", "global")),
				"eth0": network(addr("inet", "10.0.0.5", "global")),
			}},
			want: "10.0.0.5",
		},
		{
			name: "no IPv4 addresses",
			state: &api.InstanceState{Network: map[string]api.InstanceStateNetwork{
				"eth0": network(addr("inet6", "fd42::1", "global")),
			}},
			wantErr: true,
		},
		{
			name:    "no interfaces",
			state:   &api.InstanceState{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pickIPv4(tt.state)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("pickIPv4() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("pickIPv4() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("pickIPv4() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBootstrapScript(t *testing.T) {
	script, err := bootstrapScript("server", "ubuntu", testAuthorizedKey)
	if err != nil {
		t.Fatalf("bootstrapScript() error = %v", err)
	}

	for _, want := range []string{
		"hostnamectl set-hostname 'server'",
		testAuthorizedKey,
		"openssh-server",
		"systemctl enable --now ssh",
		"/dev/kmsg",
		"mount --make-rshared /",
		"apt-get update",
		"/etc/sudoers.d/ubuntu",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("bootstrap script does not contain %q", want)
		}
	}

	tests := []struct {
		name         string
		hostname     string
		user         string
		key          string
		wantErrMatch string
	}{
		{"invalid hostname", "Bad_Host", "ubuntu", testAuthorizedKey, "invalid hostname"},
		{"invalid user", "server", "bad user", testAuthorizedKey, "invalid SSH user"},
		{"rsa key", "server", "ubuntu", "ssh-rsa AAAAB3NzaC1yc2E test", "invalid SSH authorized key"},
		{"garbage key", "server", "ubuntu", "not-a-key", "invalid SSH authorized key"},
		{"empty key", "server", "ubuntu", "", "invalid SSH authorized key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bootstrapScript(tt.hostname, tt.user, tt.key)
			if err == nil {
				t.Fatalf("bootstrapScript() = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErrMatch) {
				t.Fatalf("bootstrapScript() = %q, want error containing %q", err, tt.wantErrMatch)
			}
		})
	}
}

func TestProfileConfig(t *testing.T) {
	cfg := profileConfig()

	required := map[string]string{
		"security.privileged": "true",
		"security.nesting":    "true",
	}
	for key, want := range required {
		if got := cfg[key]; got != want {
			t.Errorf("profileConfig()[%q] = %q, want %q", key, got, want)
		}
	}
	if !strings.Contains(cfg["linux.kernel_modules"], "br_netfilter") {
		t.Errorf("linux.kernel_modules = %q, want br_netfilter", cfg["linux.kernel_modules"])
	}
	if !strings.Contains(cfg["raw.lxc"], "lxc.apparmor.profile=unconfined") {
		t.Errorf("raw.lxc = %q, want unconfined AppArmor", cfg["raw.lxc"])
	}
	if !strings.Contains(cfg["raw.lxc"], "lxc.mount.auto=proc:rw sys:rw") {
		t.Errorf("raw.lxc = %q, want mount auto settings", cfg["raw.lxc"])
	}
}

func TestProfileMatches(t *testing.T) {
	desired := api.ProfilePut{Config: profileConfig(), Devices: map[string]map[string]string{}}

	tests := []struct {
		name    string
		profile *api.Profile
		want    bool
	}{
		{
			name:    "identical",
			profile: &api.Profile{Config: desired.Config, Devices: desired.Devices},
			want:    true,
		},
		{
			name: "different config",
			profile: &api.Profile{
				Config:  map[string]string{"security.privileged": "false"},
				Devices: map[string]map[string]string{},
			},
		},
		{
			name: "different devices",
			profile: &api.Profile{
				Config:  profileConfig(),
				Devices: map[string]map[string]string{"root": {"type": "disk"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := profileMatches(tt.profile, desired); got != tt.want {
				t.Fatalf("profileMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
