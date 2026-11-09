package phase

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/nirantarasi/k3sctl/pkg/etcd"
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

	kc := p.Config.Metadata.KubeClient
	if !p.NoDrain && kc != nil {
		if err := h.CordonNode(ctx, kc); err != nil {
			slog.Warn(fmt.Sprintf("%s: cordon failed: %v", h, err))
		}
		if err := h.DrainNode(ctx, kc, p.Config.Spec.Options.GetDrainOptions()); err != nil {
			slog.Warn(fmt.Sprintf("%s: drain failed: %v", h, err))
		}
	}

	if !p.NoDelete && kc != nil {
		if err := h.DeleteNode(ctx, kc); err != nil {
			slog.Warn(fmt.Sprintf("%s: delete node failed: %v", h, err))
		}
	}

	if p.leader != nil && p.leader.Metadata.K3sServiceRunning {
		p.removeEtcdMember(ctx, h)
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

// removeEtcdMember asks the leader to remove the host from the etcd cluster
// using the native etcd v3 client. If the member is already gone, this is a
// no-op.
func (p *ResetServers) removeEtcdMember(ctx context.Context, h *cluster.Host) {
	slog.Info(fmt.Sprintf("%s: removing etcd member via leader", h))

	peerAddr := h.PrivateAddress
	if peerAddr == "" {
		peerAddr = h.Address()
	}

	ec, err := etcdClientOrBuild(ctx, p.Config.Metadata.EtcdClient, p.leader)
	if err != nil {
		slog.Warn(fmt.Sprintf("%s: could not connect to etcd on leader, skipping member removal: %v", h, err))
		return
	}
	if p.Config.Metadata.EtcdClient == nil {
		p.Config.Metadata.EtcdClient = ec
	}

	members, err := ec.MemberList(ctx)
	if err != nil {
		slog.Warn(fmt.Sprintf("%s: could not list etcd members: %v", h, err))
		return
	}

	m := etcd.FindMemberByPeerAddr(members, peerAddr)
	if m == nil {
		slog.Info(fmt.Sprintf("%s: etcd member not found (already removed or different address)", h))
		return
	}

	if err := ec.MemberRemove(ctx, m.ID); err != nil {
		slog.Warn(fmt.Sprintf("%s: etcd member remove failed: %v", h, err))
		return
	}

	slog.Info(fmt.Sprintf("%s: etcd member %x removed", h, m.ID))
}
