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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
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

	var config map[string]any
	require.NoError(t, yaml.Unmarshal(data, &config))

	_, hasToken := config["token"]
	assert.False(t, hasToken, "empty token should be omitted from agent config")
}

func TestK3sValidate_EtcdSnapshotS3RequiresBucket(t *testing.T) {
	tests := []struct {
		name    string
		k3s     K3s
		wantErr bool
	}{
		{
			"S3 enabled without bucket",
			K3s{EtcdSnapshots: &EtcdSnapshotConfig{S3: &EtcdS3Config{Enabled: true}}},
			true,
		},
		{
			"S3 enabled with bucket",
			K3s{EtcdSnapshots: &EtcdSnapshotConfig{S3: &EtcdS3Config{Enabled: true, Bucket: "my-bucket"}}},
			false,
		},
		{
			"S3 disabled without bucket",
			K3s{EtcdSnapshots: &EtcdSnapshotConfig{S3: &EtcdS3Config{Enabled: false}}},
			false,
		},
		{
			"nil S3",
			K3s{EtcdSnapshots: &EtcdSnapshotConfig{}},
			false,
		},
		{
			"nil etcdSnapshots",
			K3s{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.k3s.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "bucket")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateServerConfig_EtcdSnapshots(t *testing.T) {
	t.Run("scheduled snapshots with cron and retention", func(t *testing.T) {
		k := &K3s{
			EtcdSnapshots: &EtcdSnapshotConfig{
				Scheduled:    true,
				ScheduleCron: "0 */12 * * *",
				Retention:    5,
				Compress:     true,
				Dir:          "/snapshots",
			},
		}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		assert.Equal(t, "0 */12 * * *", config["etcd-snapshot-schedule-cron"])
		assert.Equal(t, 5, config["etcd-snapshot-retention"])
		assert.Equal(t, true, config["etcd-snapshot-compress"])
		assert.Equal(t, "/snapshots", config["etcd-snapshot-dir"])
	})

	t.Run("unscheduled does not set cron fields", func(t *testing.T) {
		k := &K3s{
			EtcdSnapshots: &EtcdSnapshotConfig{
				Scheduled:    false,
				ScheduleCron: "0 */12 * * *",
				Retention:    5,
			},
		}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		_, hasCron := config["etcd-snapshot-schedule-cron"]
		assert.False(t, hasCron, "cron should not be set when scheduled=false")
		_, hasRetention := config["etcd-snapshot-retention"]
		assert.False(t, hasRetention, "retention should not be set when scheduled=false")
	})

	t.Run("S3 config", func(t *testing.T) {
		k := &K3s{
			EtcdSnapshots: &EtcdSnapshotConfig{
				S3: &EtcdS3Config{
					Enabled:       true,
					Endpoint:      "s3.example.com",
					EndpointCA:    "/ca.crt",
					SkipSSLVerify: true,
					AccessKey:     "AKID",
					SecretKey:     "secret",
					Bucket:        "backups",
					Region:        "us-east-1",
					Folder:        "k3s",
					Insecure:      true,
				},
			},
		}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		assert.Equal(t, true, config["etcd-s3"])
		assert.Equal(t, "s3.example.com", config["etcd-s3-endpoint"])
		assert.Equal(t, "/ca.crt", config["etcd-s3-endpoint-ca"])
		assert.Equal(t, true, config["etcd-s3-skip-ssl-verify"])
		assert.Equal(t, "AKID", config["etcd-s3-access-key"])
		assert.Equal(t, "secret", config["etcd-s3-secret-key"])
		assert.Equal(t, "backups", config["etcd-s3-bucket"])
		assert.Equal(t, "us-east-1", config["etcd-s3-region"])
		assert.Equal(t, "k3s", config["etcd-s3-folder"])
		assert.Equal(t, true, config["etcd-s3-insecure"])
	})

	t.Run("S3 disabled produces no S3 keys", func(t *testing.T) {
		k := &K3s{
			EtcdSnapshots: &EtcdSnapshotConfig{
				S3: &EtcdS3Config{
					Enabled: false,
					Bucket:  "backups",
				},
			},
		}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		_, hasS3 := config["etcd-s3"]
		assert.False(t, hasS3, "etcd-s3 should not be set when S3 is disabled")
	})

	t.Run("nil etcdSnapshots produces no snapshot keys", func(t *testing.T) {
		k := &K3s{}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		for key := range config {
			assert.False(t, len(key) > 4 && key[:4] == "etcd", "no etcd keys expected, got %q", key)
		}
	})
}

func TestGenerateServerConfig_SecretsEncrypt(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		k := &K3s{SecretsEncrypt: &SecretsEncryptConfig{Enabled: true}}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		assert.Equal(t, true, config["secrets-encryption"])
	})

	t.Run("disabled", func(t *testing.T) {
		k := &K3s{SecretsEncrypt: &SecretsEncryptConfig{Enabled: false}}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		_, has := config["secrets-encryption"]
		assert.False(t, has)
	})

	t.Run("nil", func(t *testing.T) {
		k := &K3s{}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		_, has := config["secrets-encryption"]
		assert.False(t, has)
	})
}

func TestGenerateServerConfig_TokenConfig(t *testing.T) {
	t.Run("both tokens", func(t *testing.T) {
		k := &K3s{Token: &TokenConfig{
			ServerToken: "server-secret",
			AgentToken:  "agent-secret",
		}}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		assert.Equal(t, "server-secret", config["token"])
		assert.Equal(t, "agent-secret", config["agent-token"])
	})

	t.Run("server token overrides cluster token", func(t *testing.T) {
		k := &K3s{Token: &TokenConfig{ServerToken: "from-yaml"}}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "from-cluster", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		assert.Equal(t, "from-yaml", config["token"], "token config should override clusterToken")
	})

	t.Run("agent token only", func(t *testing.T) {
		k := &K3s{Token: &TokenConfig{AgentToken: "agent-only"}}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "cluster-tok", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		assert.Equal(t, "cluster-tok", config["token"], "cluster token should remain when no server token in config")
		assert.Equal(t, "agent-only", config["agent-token"])
	})

	t.Run("nil token config", func(t *testing.T) {
		k := &K3s{}
		k.SetDefaults()
		host := &Host{Role: RoleServer}

		data, err := k.GenerateServerConfig(host, "cluster-tok", true, "")
		require.NoError(t, err)

		var config map[string]any
		require.NoError(t, yaml.Unmarshal(data, &config))

		assert.Equal(t, "cluster-tok", config["token"])
		_, hasAgent := config["agent-token"]
		assert.False(t, hasAgent)
	})
}

func TestK3sYAMLRoundTrip(t *testing.T) {
	input := `
version: v1.30.0+k3s1
etcdSnapshots:
  scheduled: true
  scheduleCron: "0 */6 * * *"
  retention: 3
  s3:
    enabled: true
    bucket: my-bucket
    region: us-west-2
secretsEncrypt:
  enabled: true
certRotate:
  enabled: true
token:
  serverToken: srv-tok
  agentToken: agt-tok
preflight:
  enabled: true
  checkCgroups: true
  checkDisk: true
  minDiskGB: 10
`
	var k K3s
	require.NoError(t, yaml.Unmarshal([]byte(input), &k))

	assert.Equal(t, "v1.30.0+k3s1", k.Version)

	require.NotNil(t, k.EtcdSnapshots)
	assert.True(t, k.EtcdSnapshots.Scheduled)
	assert.Equal(t, "0 */6 * * *", k.EtcdSnapshots.ScheduleCron)
	assert.Equal(t, 3, k.EtcdSnapshots.Retention)
	require.NotNil(t, k.EtcdSnapshots.S3)
	assert.True(t, k.EtcdSnapshots.S3.Enabled)
	assert.Equal(t, "my-bucket", k.EtcdSnapshots.S3.Bucket)
	assert.Equal(t, "us-west-2", k.EtcdSnapshots.S3.Region)

	require.NotNil(t, k.SecretsEncrypt)
	assert.True(t, k.SecretsEncrypt.Enabled)

	require.NotNil(t, k.CertRotate)
	assert.True(t, k.CertRotate.Enabled)

	require.NotNil(t, k.Token)
	assert.Equal(t, "srv-tok", k.Token.ServerToken)
	assert.Equal(t, "agt-tok", k.Token.AgentToken)

	require.NotNil(t, k.Preflight)
	assert.True(t, k.Preflight.Enabled)
	assert.True(t, k.Preflight.CheckCgroups)
	assert.True(t, k.Preflight.CheckDisk)
	assert.Equal(t, 10, k.Preflight.MinDiskGB)
}
