package phase

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type ResetServers struct {
	GenericPhase
	NoDrain  bool
	NoDelete bool
	hosts    cluster.Hosts
	leader   *cluster.Host
}

func (p *ResetServers) Title() string {
	return "Reset K3s servers"
}

func (p *ResetServers) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	p.hosts = p.Config.Spec.Hosts.Filter(func(h *cluster.Host) bool {
		return h.IsServer() && h.Reset && h != p.leader
	})
}

func (p *ResetServers) ShouldRun() bool {
	return len(p.hosts) > 0
}

func (p *ResetServers) Run(ctx context.Context) error {
	for _, h := range p.hosts {
		if err := p.resetServer(ctx, h); err != nil {
			return err
		}
	}
	return nil
}

func (p *ResetServers) resetServer(ctx context.Context, h *cluster.Host) error {
	slog.Info(fmt.Sprintf("%s: resetting K3s server", h))

	if !p.NoDrain && p.leader != nil {
		if err := h.CordonNode(p.leader); err != nil {
			slog.Warn(fmt.Sprintf("%s: cordon failed: %v", h, err))
		}
		if err := h.DrainNode(p.leader, p.Config.Spec.Options.GetDrainOptions()); err != nil {
			slog.Warn(fmt.Sprintf("%s: drain failed: %v", h, err))
		}
	}

	if !p.NoDelete && p.leader != nil {
		if err := h.DeleteNode(p.leader); err != nil {
			slog.Warn(fmt.Sprintf("%s: delete node failed: %v", h, err))
		}
	}

	if p.leader != nil && p.leader.Metadata.K3sServiceRunning {
		p.removeEtcdMember(h)
	}

	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err == nil {
		if err := svc.Stop(ctx); err != nil {
			slog.Warn(fmt.Sprintf("%s: stop service failed: %v", h, err))
		}
	}

	if err := h.Sudo().Exec(fmt.Sprintf("rm -rf %s", shellQuote(h.K3sDataDir()))); err != nil {
		slog.Warn(fmt.Sprintf("%s: remove data dir failed: %v", h, err))
	}

	configPath := h.Configurer.K3sConfigPath()
	if h.Sudo().FS().FileExist(configPath) {
		if err := h.Sudo().FS().Remove(configPath); err != nil {
			slog.Warn(fmt.Sprintf("%s: remove config failed: %v", h, err))
		}
	}

	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", h.K3sServiceName())
	if h.Sudo().FS().FileExist(unitPath) {
		if err := h.Sudo().Exec(fmt.Sprintf("systemctl disable %s", h.K3sServiceName())); err != nil {
			slog.Warn(fmt.Sprintf("%s: disable service failed: %v", h, err))
		}
		if err := h.Sudo().FS().Remove(unitPath); err != nil {
			slog.Warn(fmt.Sprintf("%s: remove service unit failed: %v", h, err))
		}
	}

	binaryPath := h.Configurer.K3sBinaryPath()
	if h.Sudo().FS().FileExist(binaryPath) {
		if err := h.Sudo().FS().Remove(binaryPath); err != nil {
			slog.Warn(fmt.Sprintf("%s: remove binary failed: %v", h, err))
		}
	}

	slog.Info(fmt.Sprintf("%s: K3s server reset complete", h))
	return nil
}

// removeEtcdMember asks the leader to remove the host from the etcd cluster.
// K3s exposes etcd member management through its API; we use kubectl to
// identify the member by node name and then call `k3s etcd-snapshot` tooling
// to trigger member removal. If the member is already gone, this is a no-op.
func (p *ResetServers) removeEtcdMember(h *cluster.Host) {
	slog.Info(fmt.Sprintf("%s: removing etcd member via leader", h))

	peerAddr := h.PrivateAddress
	if peerAddr == "" {
		peerAddr = h.Address()
	}

	output, err := p.leader.Sudo().ExecOutput(
		fmt.Sprintf("%s etcd-snapshot list 2>&1 || true", shellQuote(p.leader.K3sBinaryDir())),
	)
	if err != nil || strings.Contains(output, "connection refused") {
		slog.Warn(fmt.Sprintf("%s: etcd not reachable on leader, skipping member removal: %v", h, err))
		return
	}

	memberListCmd := fmt.Sprintf(
		"ETCDCTL_API=3 %s etcdctl --endpoints https://127.0.0.1:2379 "+
			"--cert /var/lib/rancher/k3s/server/tls/etcd/server-client.crt "+
			"--key /var/lib/rancher/k3s/server/tls/etcd/server-client.key "+
			"--cacert /var/lib/rancher/k3s/server/tls/etcd/server-ca.crt "+
			"member list -w simple 2>/dev/null || true",
		shellQuote(p.leader.K3sBinaryDir()),
	)

	memberOutput, err := p.leader.Sudo().ExecOutput(memberListCmd)
	if err != nil {
		slog.Warn(fmt.Sprintf("%s: could not list etcd members: %v", h, err))
		return
	}

	memberID := findEtcdMemberID(memberOutput, peerAddr)
	if memberID == "" {
		slog.Info(fmt.Sprintf("%s: etcd member not found (already removed or different address)", h))
		return
	}

	removeCmd := fmt.Sprintf(
		"ETCDCTL_API=3 %s etcdctl --endpoints https://127.0.0.1:2379 "+
			"--cert /var/lib/rancher/k3s/server/tls/etcd/server-client.crt "+
			"--key /var/lib/rancher/k3s/server/tls/etcd/server-client.key "+
			"--cacert /var/lib/rancher/k3s/server/tls/etcd/server-ca.crt "+
			"member remove %s",
		shellQuote(p.leader.K3sBinaryDir()), memberID,
	)

	if err := p.leader.Sudo().Exec(removeCmd); err != nil {
		slog.Warn(fmt.Sprintf("%s: etcd member remove failed: %v", h, err))
		return
	}

	slog.Info(fmt.Sprintf("%s: etcd member %s removed", h, memberID))
}

// findEtcdMemberID parses etcdctl member list output to find the member ID
// whose peer URL contains the given address. The simple output format is:
// <id>, started, <name>, <peer-urls>, <client-urls>, <is-learner>
func findEtcdMemberID(output, peerAddr string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, peerAddr) {
			parts := strings.SplitN(line, ",", 2)
			if len(parts) >= 1 {
				return strings.TrimSpace(parts[0])
			}
		}
	}
	return ""
}
