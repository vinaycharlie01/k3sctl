package multinode

import (
	"fmt"

	rigssh "github.com/k0sproject/rig/v2/protocol/ssh"
	"gopkg.in/yaml.v2"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
	v1beta1 "github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

// ClusterConfig describes the k3sctl manifest generated for a test run.
type ClusterConfig struct {
	Name       string
	K3sVersion string
	KeyPath    string
	Nodes      []environment.Node
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
		hosts = append(hosts, &cluster.Host{
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
		})
	}

	config := &v1beta1.Cluster{
		APIVersion: v1beta1.APIVersion,
		Kind:       "Cluster",
		Metadata:   v1beta1.ClusterMetadata{Name: c.Name},
		Spec: &cluster.Spec{
			Hosts: hosts,
			K3s:   &cluster.K3s{Version: c.K3sVersion},
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
