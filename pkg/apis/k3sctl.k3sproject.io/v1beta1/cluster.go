package v1beta1

import (
	"fmt"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

const APIVersion = "k3sctl.k3sproject.io/v1beta1"

type ClusterMetadata struct {
	Name       string            `yaml:"name" default:"k3s-cluster"`
	Kubeconfig string            `yaml:"-"`
	Manifests  map[string][]byte `yaml:"-"`
}

type Cluster struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   ClusterMetadata `yaml:"metadata"`
	Spec       *cluster.Spec   `yaml:"spec"`
}

func (c *Cluster) SetDefaults() {
	if c.APIVersion == "" {
		c.APIVersion = APIVersion
	}
	if c.Kind == "" {
		c.Kind = "Cluster"
	}
	if c.Metadata.Name == "" {
		c.Metadata.Name = "k3s-cluster"
	}
	if c.Spec != nil {
		c.Spec.SetDefaults()
	}
}

func (c *Cluster) Validate() error {
	if c.APIVersion != APIVersion {
		return fmt.Errorf("unsupported apiVersion: %s (expected %s)", c.APIVersion, APIVersion)
	}
	if c.Kind != "Cluster" {
		return fmt.Errorf("unsupported kind: %s (expected Cluster)", c.Kind)
	}
	if c.Spec == nil {
		return fmt.Errorf("spec is required")
	}
	return c.Spec.Validate()
}

func (c *Cluster) Resolve() error {
	c.SetDefaults()
	return c.Validate()
}
