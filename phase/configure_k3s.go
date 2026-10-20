package phase

import (
	"context"
	"fmt"
	"io/fs"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ConfigureK3s struct {
	GenericPhase
	leader *cluster.Host
}

func (p *ConfigureK3s) Title() string {
	return "Configure K3s"
}

func (p *ConfigureK3s) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *ConfigureK3s) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		configDir := h.Configurer.K3sConfigDirPath()
		if err := h.Sudo().Exec(fmt.Sprintf("mkdir -p %s", shellQuote(configDir))); err != nil {
			return fmt.Errorf("%s: create config dir: %w", h, err)
		}

		var configData []byte
		var err error
		if h.IsServer() {
			isLeader := p.leader != nil && h == p.leader
			var joinAddr string
			if !isLeader && p.leader != nil {
				joinAddr = p.leader.Address()
			}
			configData, err = p.Config.Spec.K3s.GenerateServerConfig(h, p.leader.Metadata.NodeToken, isLeader, joinAddr)
		} else {
			serverAddr := ""
			if p.leader != nil {
				serverAddr = p.leader.Address()
			}
			configData, err = p.Config.Spec.K3s.GenerateAgentConfig(h, serverAddr, p.leader.Metadata.NodeToken)
		}
		if err != nil {
			return fmt.Errorf("%s: generate config: %w", h, err)
		}

		configPath := h.Configurer.K3sConfigPath()
		if err := h.Sudo().FS().WriteFile(configPath, configData, fs.FileMode(0600)); err != nil {
			return fmt.Errorf("%s: write config: %w", h, err)
		}
		log.Infof("%s: wrote K3s config to %s", h, configPath)

		return nil
	})
}
