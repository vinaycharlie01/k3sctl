package phase

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type SecretsEncryptManage struct {
	GenericPhase
	leader *cluster.Host
}

func (p *SecretsEncryptManage) Title() string {
	return "Manage secrets encryption"
}

func (p *SecretsEncryptManage) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *SecretsEncryptManage) ShouldRun() bool {
	cfg := p.Config.Spec.K3s
	if cfg == nil || cfg.SecretsEncrypt == nil {
		return false
	}
	se := cfg.SecretsEncrypt
	return se.Enabled && (se.RotateKeys || se.Reencrypt)
}

func (p *SecretsEncryptManage) Run(ctx context.Context) error {
	h := p.leader
	if h == nil || !h.Metadata.K3sServiceRunning {
		return fmt.Errorf("leader server not available for secrets encryption management")
	}

	se := p.Config.Spec.K3s.SecretsEncrypt

	if se.RotateKeys {
		log.Infof("%s: rotating secrets encryption keys dynamically", h)
		cmd := h.Configurer.K3sCmdf("secrets-encrypt rotate-keys")
		if err := h.Sudo().Exec(cmd); err != nil {
			return fmt.Errorf("%s: secrets-encrypt rotate-keys failed: %w", h, err)
		}
		log.Infof("%s: waiting for key rotation to propagate", h)
		time.Sleep(5 * time.Second)
	}

	if se.Reencrypt {
		log.Infof("%s: triggering re-encryption of secrets", h)

		log.Infof("%s: preparing encryption key rotation", h)
		cmd := h.Configurer.K3sCmdf("secrets-encrypt prepare")
		if err := h.Sudo().Exec(cmd); err != nil {
			return fmt.Errorf("%s: secrets-encrypt prepare failed: %w", h, err)
		}

		if err := p.restartAllServers(ctx); err != nil {
			return fmt.Errorf("restart servers after prepare: %w", err)
		}

		log.Infof("%s: rotating encryption keys", h)
		cmd = h.Configurer.K3sCmdf("secrets-encrypt rotate")
		if err := h.Sudo().Exec(cmd); err != nil {
			return fmt.Errorf("%s: secrets-encrypt rotate failed: %w", h, err)
		}

		if err := p.restartAllServers(ctx); err != nil {
			return fmt.Errorf("restart servers after rotate: %w", err)
		}

		log.Infof("%s: re-encrypting all secrets", h)
		cmd = h.Configurer.K3sCmdf("secrets-encrypt reencrypt")
		if err := h.Sudo().Exec(cmd); err != nil {
			return fmt.Errorf("%s: secrets-encrypt reencrypt failed: %w", h, err)
		}

		log.Infof("%s: secrets re-encryption initiated", h)
	}

	return nil
}

func (p *SecretsEncryptManage) restartAllServers(ctx context.Context) error {
	servers := p.Config.Spec.Hosts.Servers()
	return p.parallelDo(ctx, servers, func(ctx context.Context, h *cluster.Host) error {
		if !h.Metadata.K3sServiceRunning {
			return nil
		}
		log.Infof("%s: restarting k3s service", h)
		svc, err := h.Sudo().Service(h.K3sServiceName())
		if err != nil {
			return fmt.Errorf("%s: find service: %w", h, err)
		}
		if err := svc.Restart(ctx); err != nil {
			return fmt.Errorf("%s: restart service: %w", h, err)
		}
		time.Sleep(5 * time.Second)
		return nil
	})
}
