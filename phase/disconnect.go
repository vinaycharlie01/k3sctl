package phase

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type Disconnect struct {
	GenericPhase
}

func (p *Disconnect) Title() string {
	return "Disconnect from hosts"
}

func (p *Disconnect) Run(_ context.Context) error {
	for _, h := range p.Config.Spec.Hosts {
		log.Infof("%s: disconnecting", h)
		h.Disconnect()
	}
	return nil
}
