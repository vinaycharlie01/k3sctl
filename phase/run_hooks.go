package phase

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type RunHooks struct {
	GenericPhase
	Action string
	Stage  string
	hosts  cluster.Hosts
}

func (p *RunHooks) Title() string {
	return fmt.Sprintf("Run %s %s hooks", p.Stage, p.Action)
}

func (p *RunHooks) Prepare() {
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		hooks := h.Hooks.ForActionAndStage(p.Action, p.Stage)
		return len(hooks) > 0
	})
}

func (p *RunHooks) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *RunHooks) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		hooks := h.Hooks.ForActionAndStage(p.Action, p.Stage)
		for _, cmd := range hooks {
			slog.Info(fmt.Sprintf("%s: running hook: %s", h, cmd))
			if err := h.Sudo().Exec(cmd); err != nil {
				return fmt.Errorf("%s: hook failed: %w", h, err)
			}
		}
		return nil
	})
}
