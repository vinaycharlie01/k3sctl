package phase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	clusterToken, err := p.resolveClusterToken()
	if err != nil {
		return fmt.Errorf("resolve cluster token: %w", err)
	}

	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		configDir := h.Configurer.K3sConfigDirPath()
		if err := h.Sudo().Exec(fmt.Sprintf("mkdir -p %s", shellQuote(configDir))); err != nil {
			return fmt.Errorf("%s: create config dir: %w", h, err)
		}

		var configData []byte
		if h.IsServer() {
			isLeader := p.leader != nil && h == p.leader
			var joinAddr string
			if !isLeader && p.leader != nil {
				joinAddr = p.leader.Address()
			}
			configData, err = p.Config.Spec.K3s.GenerateServerConfig(h, clusterToken, isLeader, joinAddr)
		} else {
			serverAddr := ""
			if p.leader != nil {
				serverAddr = p.leader.Address()
			}
			configData, err = p.Config.Spec.K3s.GenerateAgentConfig(h, serverAddr, clusterToken)
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

// resolveClusterToken determines the token every node will use. On an existing
// cluster the token is read from the leader during fact gathering. When the
// cluster is being initialized it is either taken from the user config or
// generated up front, so follower and agent configs can be written with a
// usable token before the leader starts.
func (p *ConfigureK3s) resolveClusterToken() (string, error) {
	if p.leader == nil {
		return "", fmt.Errorf("no leader server configured")
	}
	if p.leader.Metadata.NodeToken != "" {
		return p.leader.Metadata.NodeToken, nil
	}

	if p.Config.Spec.K3s == nil {
		return "", fmt.Errorf("k3s spec is not configured")
	}
	if token, ok := p.Config.Spec.K3s.Config["token"].(string); ok && token != "" {
		p.leader.Metadata.NodeToken = token
		return token, nil
	}

	if p.leader.Metadata.K3sServiceRunning {
		return "", fmt.Errorf("leader %s is running but its node token is unknown", p.leader)
	}

	token, err := generateClusterToken()
	if err != nil {
		return "", err
	}
	p.leader.Metadata.NodeToken = token
	log.Infof("%s: generated cluster token", p.leader)
	return token, nil
}

func generateClusterToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
