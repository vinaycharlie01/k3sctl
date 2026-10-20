package phase

import (
	"context"
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type Backup struct {
	GenericPhase
	Out    io.Writer
	leader *cluster.Host
}

func (p *Backup) Title() string {
	return "Backup K3s etcd"
}

func (p *Backup) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *Backup) ShouldRun() bool {
	return p.leader != nil && p.leader.Metadata.K3sServiceRunning
}

func (p *Backup) Run(_ context.Context) error {
	h := p.leader

	tmpDir, err := h.Sudo().ExecOutput("mktemp -d")
	if err != nil {
		return fmt.Errorf("%s: create temp dir: %w", h, err)
	}
	tmpDir = strings.TrimSpace(tmpDir)

	defer func() {
		_ = h.Sudo().Exec(fmt.Sprintf("rm -rf %s", shellQuote(tmpDir)))
	}()

	snapshotPath := fmt.Sprintf("%s/snapshot.db", tmpDir)
	cmd := fmt.Sprintf("%s etcd-snapshot save --etcd-snapshot-dir %s --snapshot-name snapshot",
		shellQuote(h.Configurer.K3sBinaryPath()),
		shellQuote(tmpDir),
	)

	log.Infof("%s: creating etcd snapshot", h)
	if err := h.Sudo().Exec(cmd); err != nil {
		return fmt.Errorf("%s: etcd snapshot: %w", h, err)
	}

	if !h.Sudo().FS().FileExist(snapshotPath) {
		entries, listErr := h.Sudo().ExecOutput(fmt.Sprintf("ls -la %s/", shellQuote(tmpDir)))
		if listErr == nil {
			for _, line := range strings.Split(entries, "\n") {
				if strings.Contains(line, "snapshot") {
					parts := strings.Fields(line)
					if len(parts) > 0 {
						snapshotPath = fmt.Sprintf("%s/%s", tmpDir, parts[len(parts)-1])
						break
					}
				}
			}
		}
	}

	if p.Out != nil {
		log.Infof("%s: downloading snapshot", h)
		f, err := h.Sudo().FS().Open(snapshotPath)
		if err != nil {
			return fmt.Errorf("%s: open snapshot: %w", h, err)
		}
		defer f.Close()

		n, err := io.Copy(p.Out, f)
		if err != nil {
			return fmt.Errorf("%s: download snapshot: %w", h, err)
		}
		log.Infof("%s: downloaded %d bytes", h, n)
	}

	return nil
}
