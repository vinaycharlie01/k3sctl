package phase

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/nirantarasi/k3sctl/pkg/etcd"
)

type ValidateEtcdMembers struct {
	GenericPhase
	Force  bool
	leader *cluster.Host
}

func (p *ValidateEtcdMembers) Title() string {
	return "Validate etcd members"
}

func (p *ValidateEtcdMembers) Prepare() {
	p.leader = p.Config.Spec.Hosts.Leader()
}

func (p *ValidateEtcdMembers) ShouldRun() bool {
	if p.leader == nil {
		return false
	}
	servers := p.Config.Spec.Hosts.Servers()
	return len(servers) > 1
}

func (p *ValidateEtcdMembers) Run(ctx context.Context) error {
	slog.Info(fmt.Sprintf("%s: checking etcd member health", p.leader))

	output, err := p.leader.Sudo().ExecOutput(
		fmt.Sprintf("%s etcd-snapshot list 2>&1 || true", shellQuote(p.leader.K3sBinaryDir())),
	)
	if err != nil {
		slog.Warn(fmt.Sprintf("%s: could not query etcd: %v", p.leader, err))
		return nil
	}

	if strings.Contains(output, "etcdserver: not the leader") {
		slog.Warn(fmt.Sprintf("%s: etcd reports this node is not the leader, cluster may be in recovery", p.leader))
	}

	if strings.Contains(output, "context deadline exceeded") || strings.Contains(output, "connection refused") {
		return fmt.Errorf("%s: etcd is unhealthy: %s", p.leader, firstLine(output))
	}

	slog.Info(fmt.Sprintf("%s: etcd is healthy", p.leader))

	if err := p.validateMembers(ctx); err != nil {
		return err
	}

	servers := p.Config.Spec.Hosts.Servers()
	kc := p.Config.Metadata.KubeClient
	return p.parallelDo(ctx, servers, func(ctx context.Context, h *cluster.Host) error {
		if h == p.leader {
			return nil
		}

		if kc != nil {
			ready, err := kc.IsNodeReady(ctx, h.Metadata.Hostname)
			if err != nil {
				slog.Warn(fmt.Sprintf("%s: could not check node readiness: %v", h, err))
				return nil
			}
			if !ready {
				slog.Warn(fmt.Sprintf("%s: node is not ready", h))
			} else {
				slog.Info(fmt.Sprintf("%s: node is ready", h))
			}
		} else {
			slog.Warn(fmt.Sprintf("%s: no kubernetes client available, skipping node readiness check", h))
		}

		return nil
	})
}

// validateMembers checks the actual etcd member list against the configured
// server hosts using the native etcd v3 client. If a member's peer address
// does not match any known server, it is flagged as stale. With Force
// enabled, stale members are removed automatically; otherwise an error is
// returned.
func (p *ValidateEtcdMembers) validateMembers(ctx context.Context) error {
	ec, err := etcdClientOrBuild(ctx, p.Config.Metadata.EtcdClient, p.leader)
	if err != nil {
		slog.Warn(fmt.Sprintf("%s: could not connect to etcd, skipping stale member check: %v", p.leader, err))
		return nil
	}
	if p.Config.Metadata.EtcdClient == nil {
		p.Config.Metadata.EtcdClient = ec
	}

	members, err := ec.MemberList(ctx)
	if err != nil {
		slog.Warn(fmt.Sprintf("%s: could not list etcd members, skipping stale member check: %v", p.leader, err))
		return nil
	}

	knownAddrs := p.knownServerAddresses()
	staleMembers := etcd.FindStaleMembers(members, knownAddrs)
	if len(staleMembers) == 0 {
		slog.Info(fmt.Sprintf("%s: all etcd members match configured hosts", p.leader))
		return nil
	}

	slog.Warn(fmt.Sprintf("%s: found %d stale etcd member(s)", p.leader, len(staleMembers)))
	for _, sm := range staleMembers {
		peerURL := ""
		if len(sm.PeerURLs) > 0 {
			peerURL = sm.PeerURLs[0]
		}
		slog.Warn(fmt.Sprintf("  stale member: id=%x peer=%s", sm.ID, peerURL))
	}

	if !p.Force {
		return fmt.Errorf(
			"%s: %d stale etcd member(s) found; run with --force to remove them automatically",
			p.leader, len(staleMembers),
		)
	}

	for _, sm := range staleMembers {
		peerURL := ""
		if len(sm.PeerURLs) > 0 {
			peerURL = sm.PeerURLs[0]
		}
		slog.Info(fmt.Sprintf("%s: removing stale etcd member %x (%s)", p.leader, sm.ID, peerURL))
		if err := ec.MemberRemove(ctx, sm.ID); err != nil {
			return fmt.Errorf("%s: failed to remove stale etcd member %x: %w", p.leader, sm.ID, err)
		}
		slog.Info(fmt.Sprintf("%s: removed stale etcd member %x", p.leader, sm.ID))
	}

	return nil
}

func (p *ValidateEtcdMembers) knownServerAddresses() map[string]bool {
	addrs := make(map[string]bool)
	for _, h := range p.Config.Spec.Hosts.Servers() {
		if h.PrivateAddress != "" {
			addrs[h.PrivateAddress] = true
		}
		if addr := h.Address(); addr != "" {
			addrs[addr] = true
		}
	}
	return addrs
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
