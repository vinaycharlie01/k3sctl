package cluster

import (
	"fmt"
	"os"
	"strings"

	"github.com/k0sproject/dig"
	"gopkg.in/yaml.v2"
)

func fileExistsLocal(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type EtcdSnapshotConfig struct {
	Scheduled    bool          `yaml:"scheduled,omitempty"`
	ScheduleCron string        `yaml:"scheduleCron,omitempty"`
	Retention    int           `yaml:"retention,omitempty"`
	Dir          string        `yaml:"dir,omitempty"`
	Compress     bool          `yaml:"compress,omitempty"`
	S3           *EtcdS3Config `yaml:"s3,omitempty"`
}

type EtcdS3Config struct {
	Enabled       bool   `yaml:"enabled,omitempty"`
	Endpoint      string `yaml:"endpoint,omitempty"`
	EndpointCA    string `yaml:"endpointCa,omitempty"`
	SkipSSLVerify bool   `yaml:"skipSslVerify,omitempty"`
	AccessKey     string `yaml:"accessKey,omitempty"`
	SecretKey     string `yaml:"secretKey,omitempty"`
	Bucket        string `yaml:"bucket,omitempty"`
	Region        string `yaml:"region,omitempty"`
	Folder        string `yaml:"folder,omitempty"`
	Insecure      bool   `yaml:"insecure,omitempty"`
}

type SecretsEncryptConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

type CertRotateConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

type TokenConfig struct {
	AgentToken  string `yaml:"agentToken,omitempty"`
	ServerToken string `yaml:"serverToken,omitempty"`
}

type PreflightConfig struct {
	Enabled      bool `yaml:"enabled" default:"true"`
	CheckCgroups bool `yaml:"checkCgroups" default:"true"`
	CheckModules bool `yaml:"checkModules" default:"true"`
	CheckPorts   bool `yaml:"checkPorts" default:"true"`
	CheckDisk    bool `yaml:"checkDisk" default:"true"`
	MinDiskGB    int  `yaml:"minDiskGB,omitempty"`
}

type AirgapConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ImageBundle string `yaml:"imageBundle,omitempty"`
}

type K3s struct {
	Version        string                `yaml:"version,omitempty"`
	Config         dig.Mapping           `yaml:"config,omitempty"`
	ExtraArgs      Flags                 `yaml:"extraArgs,omitempty"`
	EtcdSnapshots  *EtcdSnapshotConfig   `yaml:"etcdSnapshots,omitempty"`
	SecretsEncrypt *SecretsEncryptConfig `yaml:"secretsEncrypt,omitempty"`
	CertRotate     *CertRotateConfig     `yaml:"certRotate,omitempty"`
	Token          *TokenConfig          `yaml:"token,omitempty"`
	Preflight      *PreflightConfig      `yaml:"preflight,omitempty"`
	Airgap         *AirgapConfig         `yaml:"airgap,omitempty"`
	VersionString  string                `yaml:"-"`
}

func (k *K3s) SetDefaults() {
	if k.Config == nil {
		k.Config = dig.Mapping{}
	}
}

func (k *K3s) Validate() error {
	if k.Version != "" && !strings.HasPrefix(k.Version, "v") {
		return fmt.Errorf("k3s version must start with 'v': got %q", k.Version)
	}
	if es := k.EtcdSnapshots; es != nil {
		if es.S3 != nil && es.S3.Enabled {
			if es.S3.Bucket == "" {
				return fmt.Errorf("etcdSnapshots.s3.bucket is required when S3 is enabled")
			}
		}
	}
	if ag := k.Airgap; ag != nil && ag.Enabled {
		if ag.ImageBundle != "" {
			if !fileExistsLocal(ag.ImageBundle) {
				return fmt.Errorf("airgap.imageBundle: file not found: %s", ag.ImageBundle)
			}
		}
	}
	return nil
}

func (k *K3s) GenerateServerConfig(host *Host, clusterToken string, isInit bool, joinAddr string) ([]byte, error) {
	config := make(dig.Mapping)

	for key, val := range k.Config {
		config[key] = val
	}

	if host.PrivateAddress != "" {
		config["node-ip"] = host.PrivateAddress
	}

	if isInit {
		config["cluster-init"] = true
	} else if joinAddr != "" {
		config["server"] = fmt.Sprintf("https://%s:6443", joinAddr)
	}

	if clusterToken != "" {
		config["token"] = clusterToken
	}

	if host.NoFlannel {
		config["flannel-backend"] = "none"
	}

	k.applyEtcdSnapshotConfig(config)
	k.applySecretsEncryptConfig(config)
	k.applyTokenConfig(config)

	for _, arg := range k.ExtraArgs {
		parts := strings.SplitN(arg, "=", 2)
		key := strings.TrimPrefix(parts[0], "--")
		if len(parts) == 2 {
			config[key] = parts[1]
		} else {
			config[key] = true
		}
	}

	return yaml.Marshal(config)
}

func (k *K3s) applyEtcdSnapshotConfig(config dig.Mapping) {
	es := k.EtcdSnapshots
	if es == nil {
		return
	}
	if es.Scheduled {
		if es.ScheduleCron != "" {
			config["etcd-snapshot-schedule-cron"] = es.ScheduleCron
		}
		if es.Retention > 0 {
			config["etcd-snapshot-retention"] = es.Retention
		}
		if es.Compress {
			config["etcd-snapshot-compress"] = true
		}
	}
	if es.Dir != "" {
		config["etcd-snapshot-dir"] = es.Dir
	}

	s3 := es.S3
	if s3 == nil || !s3.Enabled {
		return
	}
	config["etcd-s3"] = true
	if s3.Endpoint != "" {
		config["etcd-s3-endpoint"] = s3.Endpoint
	}
	if s3.EndpointCA != "" {
		config["etcd-s3-endpoint-ca"] = s3.EndpointCA
	}
	if s3.SkipSSLVerify {
		config["etcd-s3-skip-ssl-verify"] = true
	}
	if s3.AccessKey != "" {
		config["etcd-s3-access-key"] = s3.AccessKey
	}
	if s3.SecretKey != "" {
		config["etcd-s3-secret-key"] = s3.SecretKey
	}
	if s3.Bucket != "" {
		config["etcd-s3-bucket"] = s3.Bucket
	}
	if s3.Region != "" {
		config["etcd-s3-region"] = s3.Region
	}
	if s3.Folder != "" {
		config["etcd-s3-folder"] = s3.Folder
	}
	if s3.Insecure {
		config["etcd-s3-insecure"] = true
	}
}

func (k *K3s) applySecretsEncryptConfig(config dig.Mapping) {
	if k.SecretsEncrypt == nil || !k.SecretsEncrypt.Enabled {
		return
	}
	config["secrets-encryption"] = true
}

func (k *K3s) applyTokenConfig(config dig.Mapping) {
	if k.Token == nil {
		return
	}
	if k.Token.AgentToken != "" {
		config["agent-token"] = k.Token.AgentToken
	}
	if k.Token.ServerToken != "" {
		config["token"] = k.Token.ServerToken
	}
}

func (k *K3s) GenerateAgentConfig(host *Host, serverAddr, clusterToken string) ([]byte, error) {
	config := make(dig.Mapping)

	for key, val := range k.Config {
		switch key {
		case "cluster-init", "cluster-cidr", "service-cidr", "cluster-dns",
			"disable", "tls-san", "etcd-expose-metrics",
			"write-kubeconfig-mode", "kube-apiserver-arg",
			"kube-controller-manager-arg", "kube-scheduler-arg":
			continue
		default:
			config[key] = val
		}
	}

	if serverAddr != "" {
		config["server"] = fmt.Sprintf("https://%s:6443", serverAddr)
	}
	if clusterToken != "" {
		config["token"] = clusterToken
	}

	if host.PrivateAddress != "" {
		config["node-ip"] = host.PrivateAddress
	}

	return yaml.Marshal(config)
}
