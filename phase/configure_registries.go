package phase

import (
	"context"
	"fmt"
	"io/fs"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"gopkg.in/yaml.v2"
)

type ConfigureRegistries struct {
	GenericPhase
}

func (p *ConfigureRegistries) Title() string {
	return "Configure private registries"
}

func (p *ConfigureRegistries) ShouldRun() bool {
	cfg := p.Config.Spec.K3s
	return cfg != nil && cfg.Registries != nil &&
		(len(cfg.Registries.Mirrors) > 0 || len(cfg.Registries.Configs) > 0)
}

func (p *ConfigureRegistries) Run(ctx context.Context) error {
	reg := p.Config.Spec.K3s.Registries

	content := make(map[string]interface{})
	if len(reg.Mirrors) > 0 {
		mirrors := make(map[string]interface{})
		for name, mirror := range reg.Mirrors {
			mirrors[name] = map[string]interface{}{
				"endpoint": mirror.Endpoint,
			}
		}
		content["mirrors"] = mirrors
	}

	if len(reg.Configs) > 0 {
		configs := make(map[string]interface{})
		for name, cfg := range reg.Configs {
			entry := make(map[string]interface{})
			if cfg.Auth != nil {
				auth := make(map[string]interface{})
				if cfg.Auth.Username != "" {
					auth["username"] = cfg.Auth.Username
				}
				if cfg.Auth.Password != "" {
					auth["password"] = cfg.Auth.Password
				}
				if cfg.Auth.Token != "" {
					auth["token"] = cfg.Auth.Token
				}
				entry["auth"] = auth
			}
			if cfg.TLS != nil {
				tls := make(map[string]interface{})
				if cfg.TLS.CertFile != "" {
					tls["cert_file"] = cfg.TLS.CertFile
				}
				if cfg.TLS.KeyFile != "" {
					tls["key_file"] = cfg.TLS.KeyFile
				}
				if cfg.TLS.CAFile != "" {
					tls["ca_file"] = cfg.TLS.CAFile
				}
				if cfg.TLS.InsecureSkipVerify {
					tls["insecure_skip_verify"] = true
				}
				entry["tls"] = tls
			}
			configs[name] = entry
		}
		content["configs"] = configs
	}

	data, err := yaml.Marshal(content)
	if err != nil {
		return fmt.Errorf("marshal registries config: %w", err)
	}

	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		registriesPath := fmt.Sprintf("%s/registries.yaml", h.Configurer.K3sConfigDirPath())

		if err := h.Sudo().Exec(fmt.Sprintf("mkdir -p %s", shellQuote(h.Configurer.K3sConfigDirPath()))); err != nil {
			return fmt.Errorf("%s: create config dir: %w", h, err)
		}

		if err := h.Sudo().FS().WriteFile(registriesPath, data, fs.FileMode(0600)); err != nil {
			return fmt.Errorf("%s: write registries.yaml: %w", h, err)
		}
		log.Infof("%s: wrote registries config to %s", h, registriesPath)
		return nil
	})
}
