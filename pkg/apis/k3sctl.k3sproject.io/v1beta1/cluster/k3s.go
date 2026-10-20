package cluster

import (
	"fmt"
	"strings"

	"github.com/k0sproject/dig"
	"gopkg.in/yaml.v2"
)

type K3s struct {
	Version       string      `yaml:"version,omitempty"`
	Config        dig.Mapping `yaml:"config,omitempty"`
	ExtraArgs     Flags       `yaml:"extraArgs,omitempty"`
	VersionString string      `yaml:"-"`
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
