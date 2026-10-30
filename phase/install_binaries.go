package phase

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type InstallBinaries struct {
	GenericPhase
	hosts cluster.Hosts
}

func (p *InstallBinaries) Title() string {
	return "Install K3s binaries"
}

func (p *InstallBinaries) Prepare() {
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		if h.Metadata.BinaryProvider == nil {
			return false
		}
		return h.Metadata.NeedsUpgrade
	})
}

func (p *InstallBinaries) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *InstallBinaries) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		provider := h.Metadata.BinaryProvider
		if provider == nil {
			return fmt.Errorf("%s: no binary provider configured", h)
		}

		binaryDir := h.Configurer.K3sBinaryPath()
		stagedPath := provider.StagedPath()
		if stagedPath == "" {
			return fmt.Errorf("%s: no staged binary path", h)
		}

		if !h.Sudo().FS().FileExist(stagedPath) {
			return fmt.Errorf("%s: staged binary not found at %s", h, stagedPath)
		}

		if err := h.Sudo().Exec(fmt.Sprintf("install -m 0755 %s %s",
			shellQuote(stagedPath),
			shellQuote(binaryDir),
		)); err != nil {
			return fmt.Errorf("%s: install binary: %w", h, err)
		}

		if err := h.Sudo().FS().Remove(stagedPath); err != nil {
			slog.Warn(fmt.Sprintf("%s: failed to clean staged binary: %v", h, err))
		}

		slog.Info(fmt.Sprintf("%s: K3s binary installed to %s", h, binaryDir))
		return nil
	})
}
