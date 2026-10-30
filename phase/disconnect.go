package phase

import (
	"context"

	"fmt"
	"log/slog"
)

type Disconnect struct {
	GenericPhase
}

func (p *Disconnect) Title() string {
	return "Disconnect from hosts"
}

func (p *Disconnect) Run(_ context.Context) error {
	for _, h := range p.Config.Spec.Hosts {
		slog.Info(fmt.Sprintf("%s: disconnecting", h))
		h.Disconnect()
	}
	return nil
}
