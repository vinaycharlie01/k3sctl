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

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
