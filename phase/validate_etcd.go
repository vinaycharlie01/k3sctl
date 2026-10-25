package phase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
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
	log.Infof("%s: checking etcd member health", p.leader)

	output, err := p.leader.Sudo().ExecOutput(
		fmt.Sprintf("%s etcd-snapshot list 2>&1 || true", shellQuote(p.leader.K3sBinaryDir())),
	)
	if err != nil {
		log.Warnf("%s: could not query etcd: %v", p.leader, err)
		return nil
	}

	if strings.Contains(output, "etcdserver: not the leader") {
		log.Warnf("%s: etcd reports this node is not the leader, cluster may be in recovery", p.leader)
	}

	if strings.Contains(output, "context deadline exceeded") || strings.Contains(output, "connection refused") {
		return fmt.Errorf("%s: etcd is unhealthy: %s", p.leader, firstLine(output))
	}

	log.Infof("%s: etcd is healthy", p.leader)

	if err := p.validateMembers(ctx); err != nil {
		return err
	}

	servers := p.Config.Spec.Hosts.Servers()
	return p.parallelDo(ctx, servers, func(ctx context.Context, h *cluster.Host) error {
		if h == p.leader {
			return nil
		}

		output, err := h.Sudo().ExecOutput(
			fmt.Sprintf("%s kubectl get node %s -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}' 2>/dev/null || echo 'unknown'",
				shellQuote(h.K3sBinaryDir()), shellQuote(h.Metadata.Hostname)),
		)
		if err != nil {
			log.Warnf("%s: could not check node readiness: %v", h, err)
			return nil
		}

		output = strings.Trim(output, "'")
		if output != "True" {
			log.Warnf("%s: node is not ready (status: %s)", h, output)
		} else {
			log.Infof("%s: node is ready", h)
		}

		return nil
	})
}

// validateMembers checks the actual etcd member list against the configured
// server hosts. If a member's peer address does not match any known server,
// it is flagged as stale. With Force enabled, stale members are removed
// automatically; otherwise an error is returned.
func (p *ValidateEtcdMembers) validateMembers(ctx context.Context) error {
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
		log.Warnf("%s: could not list etcd members, skipping stale member check: %v", p.leader, err)
		return nil
	}

	if strings.TrimSpace(memberOutput) == "" {
		log.Debugf("%s: empty etcd member list, skipping stale check", p.leader)
		return nil
	}

	knownAddrs := p.knownServerAddresses()
	staleMembers := findStaleMembers(memberOutput, knownAddrs)
	if len(staleMembers) == 0 {
		log.Infof("%s: all etcd members match configured hosts", p.leader)
		return nil
	}

	log.Warnf("%s: found %d stale etcd member(s)", p.leader, len(staleMembers))
	for _, sm := range staleMembers {
		log.Warnf("  stale member: id=%s peer=%s", sm.id, sm.peerURL)
	}

	if !p.Force {
		return fmt.Errorf(
			"%s: %d stale etcd member(s) found; run with --force to remove them automatically",
			p.leader, len(staleMembers),
		)
	}

	for _, sm := range staleMembers {
		log.Infof("%s: removing stale etcd member %s (%s)", p.leader, sm.id, sm.peerURL)
		removeCmd := fmt.Sprintf(
			"ETCDCTL_API=3 %s etcdctl --endpoints https://127.0.0.1:2379 "+
				"--cert /var/lib/rancher/k3s/server/tls/etcd/server-client.crt "+
				"--key /var/lib/rancher/k3s/server/tls/etcd/server-client.key "+
				"--cacert /var/lib/rancher/k3s/server/tls/etcd/server-ca.crt "+
				"member remove %s",
			shellQuote(p.leader.K3sBinaryDir()), sm.id,
		)
		if err := p.leader.Sudo().Exec(removeCmd); err != nil {
			return fmt.Errorf("%s: failed to remove stale etcd member %s: %w", p.leader, sm.id, err)
		}
		log.Infof("%s: removed stale etcd member %s", p.leader, sm.id)
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

type staleMember struct {
	id      string
	peerURL string
}

// findStaleMembers parses etcdctl member list output and returns members
// whose peer URL does not contain any of the known addresses.
// Format: <id>, started, <name>, <peer-urls>, <client-urls>, <is-learner>
func findStaleMembers(output string, knownAddrs map[string]bool) []staleMember {
	var stale []staleMember
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		id := strings.TrimSpace(parts[0])
		peerURL := strings.TrimSpace(parts[3])

		found := false
		for addr := range knownAddrs {
			if strings.Contains(peerURL, addr) {
				found = true
				break
			}
		}
		if !found {
			stale = append(stale, staleMember{id: id, peerURL: peerURL})
		}
	}
	return stale
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
