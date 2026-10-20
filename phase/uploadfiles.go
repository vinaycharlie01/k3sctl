package phase

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/k0sproject/rig/v2/remotefs"
	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type UploadFiles struct {
	GenericPhase
	hosts cluster.Hosts
}

func (p *UploadFiles) Title() string {
	return "Upload files to hosts"
}

func (p *UploadFiles) Prepare() {
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return len(h.UploadFiles) > 0
	})
}

func (p *UploadFiles) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *UploadFiles) Run(ctx context.Context) error {
	return p.parallelDo(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		for _, uf := range h.UploadFiles {
			resolved, err := uf.Resolve()
			if err != nil {
				return fmt.Errorf("%s: resolve upload file: %w", h, err)
			}

			for _, f := range resolved {
				dst := f.Destination()
				if dst == "" {
					return fmt.Errorf("%s: no destination for upload file %s", h, f.Source)
				}

				dstDir := filepath.Dir(dst)
				log.Infof("%s: uploading %s to %s", h, f.Source, dst)

				if err := h.Sudo().Exec(fmt.Sprintf("mkdir -p %s", shellQuote(dstDir))); err != nil {
					return fmt.Errorf("%s: create dir %s: %w", h, dstDir, err)
				}

				if err := remotefs.Upload(h.Sudo().FS(), f.Source, dst); err != nil {
					return fmt.Errorf("%s: upload %s: %w", h, f.Source, err)
				}

				if f.Perm != "" {
					if err := h.Sudo().Exec(fmt.Sprintf("chmod %s %s", f.Perm, shellQuote(dst))); err != nil {
						return fmt.Errorf("%s: chmod %s: %w", h, dst, err)
					}
				}

				if f.DirPerm != "" {
					if err := h.Sudo().Exec(fmt.Sprintf("chmod %s %s", f.DirPerm, shellQuote(dstDir))); err != nil {
						return fmt.Errorf("%s: chmod dir %s: %w", h, dstDir, err)
					}
				}

				if f.User != "" || f.Group != "" {
					ownership := f.User
					if f.Group != "" {
						ownership += ":" + f.Group
					}
					if err := h.Sudo().Exec(fmt.Sprintf("chown %s %s", ownership, shellQuote(dst))); err != nil {
						return fmt.Errorf("%s: chown %s: %w", h, dst, err)
					}
				}

				log.Infof("%s: uploaded %s to %s", h, f.Source, dst)
			}
		}
		return nil
	})
}
