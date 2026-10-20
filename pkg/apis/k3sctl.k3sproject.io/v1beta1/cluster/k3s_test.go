package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestK3sSetDefaults(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	assert.NotNil(t, k.Config)
}

func TestK3sValidate(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"empty version is valid", "", false},
		{"version with v prefix", "v1.28.3+k3s1", false},
		{"version without v prefix", "1.28.3+k3s1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &K3s{Version: tt.version}
			err := k.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateServerConfig_Leader(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	host := &Host{Role: RoleServer}

	data, err := k.GenerateServerConfig(host, "my-token", true, "")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, true, config["cluster-init"])
	assert.Equal(t, "my-token", config["token"])
	_, hasServer := config["server"]
	assert.False(t, hasServer, "leader should not have server field")
}

func TestGenerateServerConfig_JoinExisting(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	host := &Host{Role: RoleServer}

	data, err := k.GenerateServerConfig(host, "my-token", false, "10.0.0.1")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, "https://10.0.0.1:6443", config["server"])
	assert.Equal(t, "my-token", config["token"])
	_, hasClusterInit := config["cluster-init"]
	assert.False(t, hasClusterInit, "joining server should not have cluster-init")
}

func TestGenerateServerConfig_PrivateAddress(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	host := &Host{
		Role:           RoleServer,
		PrivateAddress: "192.168.1.10",
	}

	data, err := k.GenerateServerConfig(host, "", true, "")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, "192.168.1.10", config["node-ip"])
}

func TestGenerateServerConfig_NoFlannel(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	host := &Host{
		Role:      RoleServer,
		NoFlannel: true,
	}

	data, err := k.GenerateServerConfig(host, "", true, "")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, "none", config["flannel-backend"])
}

func TestGenerateServerConfig_ExtraArgs(t *testing.T) {
	k := &K3s{
		ExtraArgs: Flags{"--tls-san=10.0.0.100", "--disable=traefik", "--write-kubeconfig-mode=644"},
	}
	k.SetDefaults()
	host := &Host{Role: RoleServer}

	data, err := k.GenerateServerConfig(host, "", true, "")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, "10.0.0.100", config["tls-san"])
	assert.Equal(t, "traefik", config["disable"])
	assert.Equal(t, "644", config["write-kubeconfig-mode"])
}

func TestGenerateServerConfig_ExtraArgsBoolFlag(t *testing.T) {
	k := &K3s{
		ExtraArgs: Flags{"--secrets-encryption"},
	}
	k.SetDefaults()
	host := &Host{Role: RoleServer}

	data, err := k.GenerateServerConfig(host, "", true, "")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, true, config["secrets-encryption"])
}

func TestGenerateServerConfig_BaseConfigPassedThrough(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	k.Config["write-kubeconfig-mode"] = "644"
	k.Config["cluster-cidr"] = "10.42.0.0/16"
	host := &Host{Role: RoleServer}

	data, err := k.GenerateServerConfig(host, "", true, "")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, "644", config["write-kubeconfig-mode"])
	assert.Equal(t, "10.42.0.0/16", config["cluster-cidr"])
}

func TestGenerateAgentConfig_Basic(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	host := &Host{Role: RoleAgent}

	data, err := k.GenerateAgentConfig(host, "10.0.0.1", "agent-token")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, "https://10.0.0.1:6443", config["server"])
	assert.Equal(t, "agent-token", config["token"])
}

func TestGenerateAgentConfig_PrivateAddress(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	host := &Host{
		Role:           RoleAgent,
		PrivateAddress: "192.168.1.20",
	}

	data, err := k.GenerateAgentConfig(host, "10.0.0.1", "token")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, "192.168.1.20", config["node-ip"])
}

func TestGenerateAgentConfig_FiltersServerOnlyFields(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	k.Config["cluster-init"] = true
	k.Config["cluster-cidr"] = "10.42.0.0/16"
	k.Config["service-cidr"] = "10.43.0.0/16"
	k.Config["cluster-dns"] = "10.43.0.10"
	k.Config["disable"] = "traefik"
	k.Config["tls-san"] = "my.domain.com"
	k.Config["write-kubeconfig-mode"] = "644"
	k.Config["kube-apiserver-arg"] = "some-arg"
	k.Config["kube-controller-manager-arg"] = "some-arg"
	k.Config["kube-scheduler-arg"] = "some-arg"
	k.Config["etcd-expose-metrics"] = true
	k.Config["node-label"] = "foo=bar"
	host := &Host{Role: RoleAgent}

	data, err := k.GenerateAgentConfig(host, "10.0.0.1", "token")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	serverOnlyKeys := []string{
		"cluster-init", "cluster-cidr", "service-cidr", "cluster-dns",
		"disable", "tls-san", "etcd-expose-metrics",
		"write-kubeconfig-mode", "kube-apiserver-arg",
		"kube-controller-manager-arg", "kube-scheduler-arg",
	}
	for _, key := range serverOnlyKeys {
		_, present := config[key]
		assert.False(t, present, "agent config should not contain server-only field %q", key)
	}

	assert.Equal(t, "foo=bar", config["node-label"], "non-server fields should pass through")
}

func TestGenerateAgentConfig_EmptyToken(t *testing.T) {
	k := &K3s{}
	k.SetDefaults()
	host := &Host{Role: RoleAgent}

	data, err := k.GenerateAgentConfig(host, "10.0.0.1", "")
	require.NoError(t, err)

	var config map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &config))

	assert.Equal(t, "", config["token"])
}
