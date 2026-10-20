package phase

import (
	"context"
	"fmt"
	"strings"

	"github.com/k0sproject/rig/v2/remotefs"
	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type Restore struct {
	GenericPhase
	RestoreFrom string
	leader      *cluster.Host
}

func (p *Restore) Title() string {
	return "Restore K3s from snapshot"
}

func (p *Restore) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *Restore) ShouldRun() bool {
	if p.RestoreFrom == "" {
		return false
	}
	if p.leader == nil {
		return false
	}
	return !p.leader.Metadata.K3sServiceRunning && !p.leader.Reset
}

func (p *Restore) Run(_ context.Context) error {
	h := p.leader

	tmpDir, err := h.Sudo().ExecOutput("mktemp -d")
	if err != nil {
		return fmt.Errorf("%s: create temp dir: %w", h, err)
	}
	tmpDir = strings.TrimSpace(tmpDir)

	defer func() {
		_ = h.Sudo().Exec(fmt.Sprintf("rm -rf %s", shellQuote(tmpDir)))
	}()

	remotePath := fmt.Sprintf("%s/restore-snapshot.db", tmpDir)

	log.Infof("%s: uploading snapshot from %s", h, p.RestoreFrom)
	if err := remotefs.Upload(h.Sudo().FS(), p.RestoreFrom, remotePath); err != nil {
		return fmt.Errorf("%s: upload snapshot: %w", h, err)
	}

	cmd := fmt.Sprintf("%s server --cluster-reset --cluster-reset-restore-path=%s",
		shellQuote(h.Configurer.K3sBinaryPath()),
		shellQuote(remotePath),
	)

	log.Infof("%s: restoring from snapshot", h)
	if err := h.Sudo().Exec(cmd); err != nil {
		return fmt.Errorf("%s: restore snapshot: %w", h, err)
	}

	log.Infof("%s: K3s restored from snapshot", h)
	return nil
}
