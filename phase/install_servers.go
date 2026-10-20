package phase

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/vinaycharlie01/k3sctl/pkg/node"
	"github.com/vinaycharlie01/k3sctl/pkg/retry"
)

type InstallServers struct {
	GenericPhase
	hosts  cluster.Hosts
	leader *cluster.Host
}

func (p *InstallServers) Title() string {
	return "Install K3s servers"
}

func (p *InstallServers) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.IsServer() && h != p.leader && !h.Metadata.K3sServiceRunning
	})
}

func (p *InstallServers) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *InstallServers) Run(ctx context.Context) error {
	if p.leader.Metadata.NodeToken == "" {
		return fmt.Errorf("leader node token not available")
	}

	for _, h := range p.hosts {
		if err := p.installServer(ctx, h); err != nil {
			return err
		}
	}
	return nil
}

func (p *InstallServers) installServer(ctx context.Context, h *cluster.Host) error {
	log.Infof("%s: installing K3s server (joining cluster)", h)

	installFlags := h.K3sInstallFlags()
	unitContent := k3sServerUnit(h.Configurer.K3sBinaryPath(), installFlags)
	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", h.K3sServiceName())

	if err := h.Sudo().FS().WriteFile(unitPath, []byte(unitContent), fs.FileMode(0644)); err != nil {
		return fmt.Errorf("%s: write service unit: %w", h, err)
	}

	if err := h.Sudo().Exec("systemctl daemon-reload"); err != nil {
		return fmt.Errorf("%s: daemon-reload: %w", h, err)
	}

	if err := h.Sudo().Exec(fmt.Sprintf("systemctl enable %s", h.K3sServiceName())); err != nil {
		return fmt.Errorf("%s: enable service: %w", h, err)
	}

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: get service: %w", h, err)
	}

	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("%s: start service: %w", h, err)
	}

	log.Infof("%s: waiting for K3s server to be ready", h)
	if err := retry.Context(ctx, node.KubeNodeReadyFunc(h),
		retry.Timeout(5*time.Minute), retry.Interval(10*time.Second)); err != nil {
		return fmt.Errorf("%s: wait for ready: %w", h, err)
	}

	h.Metadata.K3sServiceRunning = true
	h.Metadata.K3sRunningVersion = p.Config.Spec.K3s.Version
	log.Infof("%s: K3s server joined cluster", h)

	return nil
}

type PropagateToken struct {
	GenericPhase
	hosts  cluster.Hosts
	leader *cluster.Host
}

func (p *PropagateToken) Title() string {
	return "Propagate node token"
}

func (p *PropagateToken) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h != p.leader && h.Metadata.NodeToken == ""
	})
}

func (p *PropagateToken) ShouldRun() bool {
	return len(p.hosts) > 0 && p.leader != nil && p.leader.Metadata.NodeToken != ""
}

func (p *PropagateToken) Run(ctx context.Context) error {
	token := strings.TrimSpace(p.leader.Metadata.NodeToken)
	return p.parallelDo(ctx, p.hosts, func(ctx context.Context, h *cluster.Host) error {
		h.Metadata.NodeToken = token
		log.Debugf("%s: propagated node token", h)
		return nil
	})
}
