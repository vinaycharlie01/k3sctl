package cluster

import (
	"testing"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/openssh"
	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sshHost(addr string) Host {
	h := Host{}
	h.ClientWithConfig.ConnectionConfig.SSH = &ssh.Config{Address: addr}
	return h
}

func opensshHost(addr string) Host {
	h := Host{}
	h.ClientWithConfig.ConnectionConfig.OpenSSH = &openssh.Config{Address: addr}
	return h
}

func localhostHost() Host {
	h := Host{}
	h.ClientWithConfig.ConnectionConfig.Localhost = true
	return h
}

func TestHostString(t *testing.T) {
	tests := []struct {
		name     string
		host     Host
		expected string
	}{
		{
			name:     "returns hostname when set",
			host:     Host{Hostname: "node1"},
			expected: "node1",
		},
		{
			name:     "returns SSH address when no hostname",
			host:     sshHost("10.0.0.1"),
			expected: "10.0.0.1",
		},
		{
			name:     "returns unknown-host when nothing set",
			host:     Host{},
			expected: "unknown-host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.host.String())
		})
	}
}

func TestHostAddress(t *testing.T) {
	tests := []struct {
		name     string
		host     Host
		expected string
	}{
		{
			name:     "SSH address",
			host:     sshHost("10.0.0.1"),
			expected: "10.0.0.1",
		},
		{
			name:     "OpenSSH address",
			host:     opensshHost("10.0.0.2"),
			expected: "10.0.0.2",
		},
		{
			name:     "localhost",
			host:     localhostHost(),
			expected: "localhost",
		},
		{
			name:     "empty when nothing set",
			host:     Host{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.host.Address())
		})
	}
}

func TestHostProtocol(t *testing.T) {
	tests := []struct {
		name     string
		host     Host
		expected string
	}{
		{
			name:     "ssh",
			host:     sshHost("10.0.0.1"),
			expected: "ssh",
		},
		{
			name:     "openssh",
			host:     opensshHost(""),
			expected: "openssh",
		},
		{
			name:     "localhost",
			host:     localhostHost(),
			expected: "localhost",
		},
		{
			name:     "unknown",
			host:     Host{},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.host.Protocol())
		})
	}
}

func TestHostIsServerIsAgent(t *testing.T) {
	server := Host{Role: RoleServer}
	assert.True(t, server.IsServer())
	assert.False(t, server.IsAgent())

	agent := Host{Role: RoleAgent}
	assert.False(t, agent.IsServer())
	assert.True(t, agent.IsAgent())
}

func TestHostK3sDataDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		h := Host{}
		assert.Equal(t, "/var/lib/rancher/k3s", h.K3sDataDir())
	})

	t.Run("custom", func(t *testing.T) {
		h := Host{DataDir: "/custom/data"}
		assert.Equal(t, "/custom/data", h.K3sDataDir())
	})
}

func TestHostK3sBinaryDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		h := Host{}
		assert.Equal(t, "/usr/local/bin/k3s", h.K3sBinaryDir())
	})

	t.Run("custom", func(t *testing.T) {
		h := Host{K3sBinaryPath: "/opt/k3s"}
		assert.Equal(t, "/opt/k3s", h.K3sBinaryDir())
	})
}

func TestHostK3sConfigDir(t *testing.T) {
	h := Host{}
	assert.Equal(t, "/etc/rancher/k3s/config.yaml", h.K3sConfigDir())
}

func TestHostKubernetesNodeName(t *testing.T) {
	tests := []struct {
		name     string
		host     Host
		expected string
	}{
		{
			name:     "from metadata kubernetes node name",
			host:     Host{Metadata: HostMetadata{KubernetesNodeName: "K8S-NODE"}},
			expected: "K8S-NODE",
		},
		{
			name:     "from metadata hostname lowered",
			host:     Host{Metadata: HostMetadata{Hostname: "MyHost"}},
			expected: "myhost",
		},
		{
			name:     "from hostname lowered",
			host:     Host{Hostname: "TestNode"},
			expected: "testnode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.host.KubernetesNodeName())
		})
	}
}

func TestHostK3sServiceName(t *testing.T) {
	server := Host{Role: RoleServer}
	assert.Equal(t, "k3s", server.K3sServiceName())

	agent := Host{Role: RoleAgent}
	assert.Equal(t, "k3s-agent", agent.K3sServiceName())
}

func TestHostK3sInstallFlags(t *testing.T) {
	h := Host{
		InstallFlags: Flags{"--disable=traefik"},
		Labels:       map[string]string{"env": "prod"},
		Taints:       []string{"special:NoSchedule"},
	}

	flags := h.K3sInstallFlags()
	assert.Contains(t, flags, "--disable=traefik")

	found := false
	for _, f := range flags {
		if f == "--node-label=env=prod" {
			found = true
		}
	}
	assert.True(t, found, "should contain node-label flag")

	taintFound := false
	for _, f := range flags {
		if f == "--node-taint=special:NoSchedule" {
			taintFound = true
		}
	}
	assert.True(t, taintFound, "should contain node-taint flag")
}

func TestHostValidate(t *testing.T) {
	tests := []struct {
		name    string
		host    Host
		wantErr bool
	}{
		{"valid server", Host{Role: RoleServer}, false},
		{"valid agent", Host{Role: RoleAgent}, false},
		{"invalid role", Host{Role: "worker"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.host.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHostExpandTokens(t *testing.T) {
	h := sshHost("10.0.0.5")
	h.Hostname = "myhost"
	h.Role = RoleServer
	h.PrivateAddress = "192.168.1.5"
	h.Metadata = HostMetadata{
		Hostname: "real-hostname",
		Arch:     "amd64",
	}

	result := h.ExpandTokens("node-{HOSTNAME}-{ADDRESS}-{ROLE}-{ARCH}-{PRIVATE_ADDRESS}")
	assert.Equal(t, "node-real-hostname-10.0.0.5-server-amd64-192.168.1.5", result)
}

func TestHostInstalledAndRunningVersion(t *testing.T) {
	h := Host{
		Metadata: HostMetadata{
			K3sInstalledVersion: "v1.28.2+k3s1",
			K3sRunningVersion:   "v1.28.1+k3s1",
		},
	}
	assert.Equal(t, "v1.28.2+k3s1", h.InstalledK3sVersion())
	assert.Equal(t, "v1.28.1+k3s1", h.RunningK3sVersion())
}

func TestHostSudoNilClient(t *testing.T) {
	h := Host{}
	assert.Nil(t, h.Sudo())
}

func TestHostFSNilClient(t *testing.T) {
	h := Host{}
	assert.Nil(t, h.FS())
}

func TestHostRunHooksNilHooks(t *testing.T) {
	h := Host{}
	assert.NoError(t, h.RunHooks("apply", "before"))
}

func TestHostRunHooksNoMatch(t *testing.T) {
	h := Host{
		Hooks: Hooks{
			"apply": {"before": {"echo hello"}},
		},
	}
	assert.NoError(t, h.RunHooks("reset", "after"))
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, "'hello'", shellQuote("hello"))
	assert.Equal(t, "'it'\\''s'", shellQuote("it's"))
	assert.Equal(t, "''", shellQuote(""))
}

func TestHostK3sInstallFlags_Empty(t *testing.T) {
	h := Host{}
	flags := h.K3sInstallFlags()
	assert.Empty(t, flags)
}

func TestHostK3sInstallFlags_CombinesAll(t *testing.T) {
	h := Host{
		InstallFlags: Flags{"--disable=traefik", "--disable=servicelb"},
		Labels:       map[string]string{"env": "prod", "tier": "frontend"},
		Taints:       []string{"special:NoSchedule", "gpu:NoExecute"},
	}

	flags := h.K3sInstallFlags()
	assert.Contains(t, flags, "--disable=traefik")
	assert.Contains(t, flags, "--disable=servicelb")

	// Verify labels and taints are present
	var labelFlags, taintFlags []string
	for _, f := range flags {
		if len(f) > len("--node-label=") && f[:len("--node-label=")] == "--node-label=" {
			labelFlags = append(labelFlags, f)
		}
		if len(f) > len("--node-taint=") && f[:len("--node-taint=")] == "--node-taint=" {
			taintFlags = append(taintFlags, f)
		}
	}
	assert.Len(t, labelFlags, 2, "should have 2 label flags")
	assert.Len(t, taintFlags, 2, "should have 2 taint flags")
}

func TestHostOSKind_NilConfigurer(t *testing.T) {
	h := Host{}
	_, err := h.OSKind()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configurer not resolved")
}

func TestHostExpandTokens_AllTokens(t *testing.T) {
	h := sshHost("10.0.0.5")
	h.Hostname = "myhost"
	h.Role = RoleServer
	h.PrivateAddress = "192.168.1.5"
	h.Metadata = HostMetadata{
		Hostname: "real-hostname",
		Arch:     "amd64",
	}

	result := h.ExpandTokens("{HOSTNAME}-{ADDRESS}-{ROLE}-{ARCH}-{PRIVATE_ADDRESS}")
	assert.Equal(t, "real-hostname-10.0.0.5-server-amd64-192.168.1.5", result)
}

func TestHostExpandTokens_EmptyMetadata(t *testing.T) {
	h := Host{Role: RoleAgent}
	result := h.ExpandTokens("role-{ROLE}")
	assert.Equal(t, "role-agent", result)
}

func TestHostDestination_WithDstDir(t *testing.T) {
	u := UploadFile{Source: "/local/script.sh", DstDir: "/opt/bin"}
	assert.Contains(t, u.Destination(), "script.sh")
}

func TestHostValidateInvalidRole(t *testing.T) {
	h := Host{Role: "worker"}
	err := h.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
	assert.Contains(t, err.Error(), "worker")
}

func TestHostValidateServerRole(t *testing.T) {
	h := Host{Role: RoleServer}
	assert.NoError(t, h.Validate())
}

func TestHostValidateAgentRole(t *testing.T) {
	h := Host{Role: RoleAgent}
	assert.NoError(t, h.Validate())
}
