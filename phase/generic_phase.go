package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type GenericPhase struct {
	Config  *v1beta1.Cluster
	Manager *Manager
}

func (p *GenericPhase) SetConfig(config *v1beta1.Cluster) {
	p.Config = config
}

func (p *GenericPhase) SetManager(m *Manager) {
	p.Manager = m
}

func (p *GenericPhase) Wet(host fmt.Stringer, dryMsg string, fns ...func() error) error {
	if p.Manager != nil && p.Manager.DryRun {
		log.Infof("%s: %s", host, dryMsg)
		return nil
	}
	for _, fn := range fns {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

func (p *GenericPhase) DryMsg(host fmt.Stringer, msg string) {
	if p.Manager != nil && p.Manager.DryRun {
		log.Infof("%s: %s", host, msg)
	}
}

func (p *GenericPhase) DryMsgf(host fmt.Stringer, format string, args ...any) {
	p.DryMsg(host, fmt.Sprintf(format, args...))
}

func (p *GenericPhase) parallelDo(ctx context.Context, hosts cluster.Hosts, fn func(ctx context.Context, h *cluster.Host) error) error {
	concurrency := 30
	if p.Manager != nil && p.Manager.Concurrency > 0 {
		concurrency = p.Manager.Concurrency
	}
	return hosts.ParallelEach(ctx, concurrency, fn)
}

func (p *GenericPhase) parallelDoUpload(ctx context.Context, hosts cluster.Hosts, fn func(ctx context.Context, h *cluster.Host) error) error {
	concurrency := 5
	if p.Manager != nil && p.Manager.ConcurrentUploads > 0 {
		concurrency = p.Manager.ConcurrentUploads
	}
	return hosts.ParallelEach(ctx, concurrency, fn)
}
