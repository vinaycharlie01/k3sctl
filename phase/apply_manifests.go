package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ApplyManifests struct {
	GenericPhase
	leader *cluster.Host
}

func (p *ApplyManifests) Title() string {
	return "Apply additional manifests"
}

func (p *ApplyManifests) Prepare() {
	p.leader = p.Config.Spec.Hosts.Leader()
}

func (p *ApplyManifests) ShouldRun() bool {
	return len(p.Config.Metadata.Manifests) > 0 && p.leader != nil
}

func (p *ApplyManifests) Run(ctx context.Context) error {
	for name, content := range p.Config.Metadata.Manifests {
		if err := p.applyManifest(ctx, name, content); err != nil {
			return err
		}
	}
	return nil
}

func (p *ApplyManifests) applyManifest(ctx context.Context, name string, content []byte) error {
	return p.Wet(p.leader, fmt.Sprintf("would apply manifest %s (%d bytes)", name, len(content)), func() error {
		log.Infof("%s: applying manifest %s (%d bytes)", p.leader, name, len(content))

		tmpPath := fmt.Sprintf("/tmp/k3sctl-manifest-%s.yaml", name)
		sudoFS := p.leader.Sudo().FS()

		if err := sudoFS.WriteFile(tmpPath, content, 0600); err != nil {
			return fmt.Errorf("%s: write manifest %s: %w", p.leader, name, err)
		}
		defer func() { _ = sudoFS.Remove(tmpPath) }()

		kubectlCmd := fmt.Sprintf("%s kubectl apply -f %s",
			shellQuote(p.leader.K3sBinaryDir()), shellQuote(tmpPath))

		output, err := p.leader.Sudo().ExecOutput(kubectlCmd)
		if err != nil {
			return fmt.Errorf("%s: kubectl apply failed for manifest %s: %w", p.leader, name, err)
		}

		log.Infof("%s: %s", p.leader, output)
		return nil
	})
}
