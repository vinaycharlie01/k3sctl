package phase

import (
	"bytes"
	"context"
	"fmt"

	rigcmd "github.com/k0sproject/rig/v2/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

// ApplyManifests applies additional Kubernetes manifests bundled in the
// cluster configuration to the cluster.
type ApplyManifests struct {
	GenericPhase
	leader *cluster.Host
}

func (p *ApplyManifests) Title() string {
	return "Apply additional manifests"
}

func (p *ApplyManifests) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *ApplyManifests) ShouldRun() bool {
	return p.leader != nil && len(p.Config.Metadata.Manifests) > 0
}

func (p *ApplyManifests) Run(ctx context.Context) error {
	for name, content := range p.Config.Metadata.Manifests {
		if err := p.apply(ctx, name, content); err != nil {
			return err
		}
	}
	return nil
}

func (p *ApplyManifests) apply(ctx context.Context, name string, content []byte) error {
	return p.Wet(p.leader, fmt.Sprintf("apply manifest %s (%d bytes)", name, len(content)), func() error {
		return p.runApply(ctx, name, content)
	})
}

func (p *ApplyManifests) runApply(ctx context.Context, name string, content []byte) error {
	log.Infof("%s: applying manifest %s (%d bytes)", p.leader, name, len(content))

	var stdout, stderr bytes.Buffer
	err := p.leader.Sudo().ExecContext(ctx,
		p.leader.Configurer.KubectlCmdf(p.leader, p.leader.K3sDataDir(), "apply -f -"),
		rigcmd.Stdin(bytes.NewReader(content)),
		rigcmd.Stdout(&stdout),
		rigcmd.Stderr(&stderr),
	)
	if err != nil {
		return fmt.Errorf("%s: apply manifest %s: %w (stderr: %s)", p.leader, name, err, stderr.String())
	}
	if out := stdout.String(); out != "" {
		log.Debugf("%s: kubectl apply: %s", p.leader, out)
	}
	return nil
}
