package cluster

import (
	"fmt"
)

type Spec struct {
	Hosts     Hosts             `yaml:"hosts" validate:"required,min=1"`
	K3s       *K3s              `yaml:"k3s,omitempty"`
	Options   *Options          `yaml:"options,omitempty"`
	Manifests map[string]string `yaml:"manifests,omitempty"`
}

func (s *Spec) SetDefaults() {
	if s.K3s == nil {
		s.K3s = &K3s{}
	}
	s.K3s.SetDefaults()

	for _, h := range s.Hosts {
		if h.Role == "" {
			h.Role = RoleServer
		}
	}
}

func (s *Spec) Validate() error {
	if len(s.Hosts) == 0 {
		return fmt.Errorf("at least one host is required")
	}

	servers := s.Hosts.Servers()
	if len(servers) == 0 {
		return fmt.Errorf("at least one server host is required")
	}

	for _, h := range s.Hosts {
		if err := h.Validate(); err != nil {
			return fmt.Errorf("host %s: %w", h, err)
		}
	}

	if s.K3s != nil {
		if err := s.K3s.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Spec) K3sLeader() *Host {
	return s.Hosts.Leader()
}

func (s *Spec) APIPort() int {
	return 6443
}

func (s *Spec) KubeAPIURL() string {
	leader := s.K3sLeader()
	if leader == nil {
		return ""
	}
	addr := leader.PrivateAddress
	if addr == "" {
		addr = leader.Address()
	}
	return fmt.Sprintf("https://%s:%d", addr, s.APIPort())
}

func (s *Spec) NodeCount() int {
	return len(s.Hosts)
}

func (s *Spec) ServerCount() int {
	return len(s.Hosts.Servers())
}

func (s *Spec) AgentCount() int {
	return len(s.Hosts.Agents())
}
