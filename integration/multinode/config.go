package multinode

import (
	"fmt"

	rigssh "github.com/k0sproject/rig/v2/protocol/ssh"
	"gopkg.in/yaml.v2"

	"github.com/nirantarasi/k3sctl/integration/environment"
	v1beta1 "github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

// K3sOverrides holds optional K3s configuration that the integration test
// wants to exercise. Each field maps directly to a k3sctl YAML stanza.
type K3sOverrides struct {
	// WithNodeID enables --with-node-id on all hosts.
	WithNodeID bool
	// Security overrides the security stanza.
	Security *cluster.SecurityConfig
	// Runtime overrides the runtime stanza.
	Runtime *cluster.RuntimeConfig
	// Server overrides the server stanza.
	Server *cluster.ServerConfig
	// Kubeconfig overrides the kubeconfig stanza.
	Kubeconfig *cluster.KubeconfigConfig
	// Token overrides the token stanza.
	Token *cluster.TokenConfig
	// EmbeddedRegistry overrides the embeddedRegistry stanza.
	EmbeddedRegistry *cluster.EmbeddedRegistryConfig
	// Agent overrides the agent stanza.
	Agent *cluster.AgentConfig
	// Disable overrides the disable stanza.
	Disable *cluster.DisableConfig
	// Labels sets node labels on all hosts.
	Labels map[string]string
}

// ClusterConfig describes the k3sctl manifest generated for a test run.
type ClusterConfig struct {
	Name       string
	K3sVersion string
	KeyPath    string
	Nodes      []environment.Node
	K3s        *K3sOverrides
}

// Generate renders and validates a k3sctl Cluster manifest using the project's
// own configuration types, so the generated file cannot drift from the schema
// the CLI accepts.
func (c ClusterConfig) Generate() ([]byte, error) {
	if c.K3sVersion == "" {
		return nil, fmt.Errorf("k3s version is required")
	}
	if c.KeyPath == "" {
		return nil, fmt.Errorf("SSH key path is required")
	}
	if len(c.Nodes) == 0 {
		return nil, fmt.Errorf("at least one node is required")
	}

	keyPath := c.KeyPath
	hosts := make(cluster.Hosts, 0, len(c.Nodes))
	for _, node := range c.Nodes {
		h := &cluster.Host{
			ConnectionConfig: cluster.ConnectionConfig{
				SSH: &rigssh.Config{
					Address: node.Address,
					User:    node.SSHUser,
					Port:    node.SSHPort,
					KeyPath: &keyPath,
				},
			},
			Role:           cluster.Role(node.Role),
			PrivateAddress: node.Address,
			Hostname:       node.Name,
		}
		if c.K3s != nil {
			if c.K3s.WithNodeID {
				h.WithNodeID = true
			}
			if c.K3s.Labels != nil {
				h.Labels = c.K3s.Labels
			}
		}
		hosts = append(hosts, h)
	}

	k3sCfg := &cluster.K3s{Version: c.K3sVersion}
	if c.K3s != nil {
		if c.K3s.Security != nil {
			k3sCfg.Security = c.K3s.Security
		}
		if c.K3s.Runtime != nil {
			k3sCfg.Runtime = c.K3s.Runtime
		}
		if c.K3s.Server != nil {
			k3sCfg.Server = c.K3s.Server
		}
		if c.K3s.Kubeconfig != nil {
			k3sCfg.Kubeconfig = c.K3s.Kubeconfig
		}
		if c.K3s.Token != nil {
			k3sCfg.Token = c.K3s.Token
		}
		if c.K3s.EmbeddedRegistry != nil {
			k3sCfg.EmbeddedRegistry = c.K3s.EmbeddedRegistry
		}
		if c.K3s.Agent != nil {
			k3sCfg.Agent = c.K3s.Agent
		}
		if c.K3s.Disable != nil {
			k3sCfg.Disable = c.K3s.Disable
		}
	}

	config := &v1beta1.Cluster{
		APIVersion: v1beta1.APIVersion,
		Kind:       "Cluster",
		Metadata:   v1beta1.ClusterMetadata{Name: c.Name},
		Spec: &cluster.Spec{
			Hosts: hosts,
			K3s:   k3sCfg,
		},
	}
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate generated cluster config: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal cluster config: %w", err)
	}
	return data, nil
}
