package phase

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/nirantarasi/k3sctl/pkg/k8s"
)

type ApplyManifests struct {
	GenericPhase
	kc k8s.Client
}

func (p *ApplyManifests) Title() string {
	return "Apply additional manifests"
}

func (p *ApplyManifests) Prepare() {
	p.kc = p.Config.Metadata.KubeClient
}

func (p *ApplyManifests) ShouldRun() bool {
	leader := p.Config.Spec.K3sLeader()
	return leader != nil && len(p.Config.Metadata.Manifests) > 0
}

func (p *ApplyManifests) Run(ctx context.Context) error {
	if p.kc == nil {
		leader := p.Config.Spec.K3sLeader()
		var err error
		p.kc, err = kubeClientOrBuild(nil, leader)
		if err != nil {
			return fmt.Errorf("build kube client for manifest apply: %w", err)
		}
	}

	for name, content := range p.Config.Metadata.Manifests {
		if err := p.apply(ctx, name, content); err != nil {
			return err
		}
	}
	return nil
}

func (p *ApplyManifests) apply(ctx context.Context, name string, content []byte) error {
	return p.Wet(p.Config.Spec.K3sLeader(), fmt.Sprintf("apply manifest %s (%d bytes)", name, len(content)), func() error {
		return p.runApply(ctx, name, content)
	})
}

func (p *ApplyManifests) runApply(ctx context.Context, name string, content []byte) error {
	leader := p.Config.Spec.K3sLeader()
	slog.Info(fmt.Sprintf("%s: applying manifest %s (%d bytes)", leader, name, len(content)))

	if err := p.kc.ApplyManifest(ctx, content); err != nil {
		return fmt.Errorf("%s: apply manifest %s: %w", leader, name, err)
	}

	slog.Debug(fmt.Sprintf("%s: manifest %s applied", leader, name))
	return nil
}
