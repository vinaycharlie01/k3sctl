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

	running := p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.IsServer() && h.Metadata.K3sServiceRunning
	})

	batches := quorumSafeBatches(len(running), p.hosts)
	for i, batch := range batches {
		log.Infof("installing server batch %d/%d (%d hosts)", i+1, len(batches), len(batch))
		for _, h := range batch {
			if err := p.installServer(ctx, h); err != nil {
				return err
			}
		}
	}
	return nil
}

// quorumSafeBatches splits hosts into batches that never exceed quorum
// tolerance. With N running servers, quorum is (N/2)+1, so we can add at
// most (N-1)/2 new members before quorum shifts. The first batch is always
// size 1 when only a single server is running (the leader), because adding
// two members to a single-node cluster can lose quorum if both are slow to
// join. After each batch the running count increases by the batch size.
func quorumSafeBatches(numRunning int, hosts cluster.Hosts) []cluster.Hosts {
	if len(hosts) == 0 {
		return nil
	}

	var batches []cluster.Hosts
	remaining := hosts
	running := numRunning

	for len(remaining) > 0 {
		batchSize := quorumBatchSize(running)
		if batchSize > len(remaining) {
			batchSize = len(remaining)
		}
		batches = append(batches, remaining[:batchSize])
		remaining = remaining[batchSize:]
		running += batchSize
	}

	return batches
}

// quorumBatchSize returns how many servers can safely join at once given
// the current number of running servers. For a cluster of N nodes, quorum
// is (N/2)+1 and the tolerance is N - quorum = (N-1)/2.
func quorumBatchSize(running int) int {
	if running <= 1 {
		return 1
	}
	tolerance := (running - 1) / 2
	if tolerance < 1 {
		return 1
	}
	return tolerance
}

// CleanUp rolls back partially installed servers when the phase fails.
// Hosts that were attempted but never reached the running state are stopped,
// their systemd unit removed, and data directory cleaned up to prevent
// leaving a half-configured node in the cluster.
func (p *InstallServers) CleanUp() {
	failed := p.hosts.Filter(func(h *cluster.Host) bool {
		return !h.Metadata.K3sServiceRunning
	})
	if len(failed) == 0 {
		return
	}

	_ = failed.ParallelEach(context.Background(), 5, func(_ context.Context, h *cluster.Host) error {
		log.Infof("%s: cleaning up failed server install", h)

		if p.Manager != nil && p.Manager.DryRun {
			return nil
		}

		svc, err := h.Sudo().Service(h.K3sServiceName())
		if err == nil {
			_ = svc.Stop(context.Background())
		}

		unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", h.K3sServiceName())
		if h.Sudo().FS().FileExist(unitPath) {
			_ = h.Sudo().Exec(fmt.Sprintf("systemctl disable %s", h.K3sServiceName()))
			_ = h.Sudo().FS().Remove(unitPath)
		}

		_ = h.Sudo().Exec(fmt.Sprintf("rm -rf %s", shellQuote(h.K3sDataDir())))

		configPath := h.Configurer.K3sConfigPath()
		if h.Sudo().FS().FileExist(configPath) {
			_ = h.Sudo().FS().Remove(configPath)
		}

		log.Infof("%s: clean-up complete", h)
		return nil
	})
}

func (p *InstallServers) installServer(ctx context.Context, h *cluster.Host) error {
	log.Infof("%s: installing K3s server (joining cluster)", h)

	installFlags := h.K3sInstallFlags()
	unitContent := k3sServerUnit(h.Configurer.K3sBinaryPath(), installFlags, h.Environment)
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
	if err := retry.Context(ctx, node.KubeNodeReadyFunc(p.leader, h.KubernetesNodeName()),
		retry.Timeout(5*time.Minute), retry.Interval(10*time.Second)); err != nil {
		return fmt.Errorf("%s: wait for ready: %w", h, err)
	}

	h.Metadata.K3sServiceRunning = true
	h.Metadata.K3sRunningVersion = p.Config.Spec.K3s.Version
	h.Metadata.NeedsUpgrade = false
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
