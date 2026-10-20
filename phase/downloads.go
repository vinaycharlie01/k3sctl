package phase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type DownloadBinaries struct {
	GenericPhase
	hosts cluster.Hosts
}

func (p *DownloadBinaries) Title() string {
	return "Download K3s binaries"
}

func (p *DownloadBinaries) Prepare() {
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.Metadata.BinaryProvider != nil && h.Metadata.NeedsUpgrade
	})
}

func (p *DownloadBinaries) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *DownloadBinaries) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		if h.Metadata.BinaryProvider == nil {
			return nil
		}
		return h.Metadata.BinaryProvider.Stage(ctx)
	})
}

func downloadFile(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file %s: %w", dest, err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("write file %s: %w", dest, err)
	}

	log.Debugf("downloaded %d bytes to %s", n, dest)
	return nil
}
