package phase

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/k0sproject/rig/v2/remotefs"
	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type UploadAirgapBundle struct {
	GenericPhase
	hosts cluster.Hosts
}

func (p *UploadAirgapBundle) Title() string {
	return "Upload air-gap image bundle"
}

func (p *UploadAirgapBundle) Prepare() {
	if p.Config.Spec.K3s == nil || p.Config.Spec.K3s.Airgap == nil || !p.Config.Spec.K3s.Airgap.Enabled {
		return
	}
	if p.Config.Spec.K3s.Airgap.ImageBundle == "" {
		return
	}
	p.hosts = p.Config.Spec.Hosts
}

func (p *UploadAirgapBundle) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *UploadAirgapBundle) Run(ctx context.Context) error {
	bundle := p.Config.Spec.K3s.Airgap.ImageBundle
	return p.parallelDo(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		return p.uploadBundle(h, bundle)
	})
}

func (p *UploadAirgapBundle) uploadBundle(h *cluster.Host, localPath string) error {
	imagesDir := filepath.Join(h.K3sDataDir(), "agent", "images")
	remotePath := filepath.Join(imagesDir, filepath.Base(localPath))

	log.Infof("%s: uploading air-gap image bundle to %s", h, remotePath)

	return p.Wet(h, fmt.Sprintf("upload air-gap bundle %s to %s", localPath, remotePath), func() error {
		if err := h.Sudo().Exec(fmt.Sprintf("mkdir -p %s", shellQuote(imagesDir))); err != nil {
			return fmt.Errorf("%s: create images dir: %w", h, err)
		}

		if err := remotefs.Upload(h.Sudo().FS(), localPath, remotePath); err != nil {
			return fmt.Errorf("%s: upload air-gap bundle: %w", h, err)
		}

		if err := h.Sudo().Exec(fmt.Sprintf("chmod 0644 %s", shellQuote(remotePath))); err != nil {
			return fmt.Errorf("%s: chmod air-gap bundle: %w", h, err)
		}

		log.Infof("%s: air-gap image bundle uploaded to %s", h, remotePath)
		return nil
	})
}
