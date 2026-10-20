package phase

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/vinaycharlie01/k3sctl/pkg/k3s"
)

type StageBinaries struct {
	GenericPhase
	hosts cluster.Hosts
}

func (p *StageBinaries) Title() string {
	return "Stage K3s binaries"
}

func (p *StageBinaries) Prepare() {
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		if h.Metadata.BinaryProvider == nil {
			return false
		}
		return h.Metadata.NeedsUpgrade
	})
}

func (p *StageBinaries) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *StageBinaries) Run(ctx context.Context) error {
	return p.parallelDoUpload(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		provider := h.Metadata.BinaryProvider
		if provider == nil {
			return fmt.Errorf("%s: no binary provider configured", h)
		}

		log.Infof("%s: staging K3s binary", h)
		if err := provider.Stage(ctx); err != nil {
			return fmt.Errorf("%s: stage binary: %w", h, err)
		}
		log.Infof("%s: K3s binary staged", h)
		return nil
	})
}

func (p *StageBinaries) CleanUp() {
	for _, h := range p.hosts {
		if h.Metadata.BinaryProvider != nil {
			h.Metadata.BinaryProvider.CleanUp(context.Background())
		}
	}
}

type ResolveBinaryProviders struct {
	GenericPhase
}

func (p *ResolveBinaryProviders) Title() string {
	return "Resolve K3s binary providers"
}

func (p *ResolveBinaryProviders) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		h.Metadata.BinaryProvider = k3s.NewBinaryProvider(h, p.Config.Spec.K3s)
		return nil
	})
}
