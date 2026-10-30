package phase

import (
	"context"

	"fmt"
	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type Connect struct {
	GenericPhase
}

func (p *Connect) Title() string {
	return "Connect to hosts"
}

func (p *Connect) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		slog.Info(fmt.Sprintf("%s: connecting", h))
		return h.Connect(ctx)
	})
}
