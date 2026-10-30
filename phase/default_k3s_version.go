package phase

import (
	"context"

	"fmt"
	"log/slog"

	"github.com/vinaycharlie01/k3sctl/integration/github"
)

type DefaultK3sVersion struct {
	GenericPhase
}

func (p *DefaultK3sVersion) Title() string {
	return "Determine K3s version"
}

func (p *DefaultK3sVersion) Run(ctx context.Context) error {
	if p.Config.Spec.K3s.Version != "" {
		slog.Info(fmt.Sprintf("using configured K3s version %s", p.Config.Spec.K3s.Version))
		return nil
	}

	ver, err := github.LatestK3sVersion(ctx, false)
	if err != nil {
		return err
	}

	p.Config.Spec.K3s.Version = ver
	slog.Info(fmt.Sprintf("using latest K3s version: %s", ver))
	return nil
}
