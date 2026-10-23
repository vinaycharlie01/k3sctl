package phase

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type NodeRemove struct {
	GenericPhase
	Target  string
	NoDrain bool
	Force   bool
	host    *cluster.Host
	leader  *cluster.Host
}

func (p *NodeRemove) Title() string {
	return "Remove node from cluster"
}

func (p *NodeRemove) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
	for _, h := range p.Config.Spec.Hosts {
		if p.matchesTarget(h) {
			p.host = h
			break
		}
	}
}

func (p *NodeRemove) matchesTarget(h *cluster.Host) bool {
	target := strings.ToLower(p.Target)
	if strings.EqualFold(h.Hostname, p.Target) {
		return true
	}
	if strings.EqualFold(h.Address(), p.Target) {
		return true
	}
	if strings.EqualFold(h.Metadata.Hostname, p.Target) {
		return true
	}
	if strings.EqualFold(h.KubernetesNodeName(), target) {
		return true
	}
	return false
}

func (p *NodeRemove) ShouldRun() bool {
	return p.host != nil
}

func (p *NodeRemove) Run(ctx context.Context) error {
	if p.host == nil {
		return fmt.Errorf("host %q not found in cluster config", p.Target)
	}

	if p.host == p.leader && !p.Force {
		return fmt.Errorf("refusing to remove the leader node %s (use --force to override)", p.host)
	}

	log.Infof("%s: removing node from cluster", p.host)

	if !p.NoDrain && p.leader != nil {
		if err := p.host.CordonNode(p.leader); err != nil {
			log.Warnf("%s: cordon failed: %v", p.host, err)
		}
		if err := p.host.DrainNode(p.leader, p.Config.Spec.Options.GetDrainOptions()); err != nil {
			log.Warnf("%s: drain failed: %v", p.host, err)
		}
	}

	if p.leader != nil {
		if err := p.host.DeleteNode(p.leader); err != nil {
			log.Warnf("%s: delete node failed: %v", p.host, err)
		}
	}

	svc, err := p.host.Sudo().Service(p.host.K3sServiceName())
	if err == nil {
		if err := svc.Stop(ctx); err != nil {
			log.Warnf("%s: stop service failed: %v", p.host, err)
		}
	}

	if err := p.host.Sudo().Exec(fmt.Sprintf("rm -rf %s", shellQuote(p.host.K3sDataDir()))); err != nil {
		log.Warnf("%s: remove data dir failed: %v", p.host, err)
	}

	configPath := p.host.Configurer.K3sConfigPath()
	if p.host.Sudo().FS().FileExist(configPath) {
		if err := p.host.Sudo().FS().Remove(configPath); err != nil {
			log.Warnf("%s: remove config failed: %v", p.host, err)
		}
	}

	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", p.host.K3sServiceName())
	if p.host.Sudo().FS().FileExist(unitPath) {
		if err := p.host.Sudo().Exec(fmt.Sprintf("systemctl disable %s", p.host.K3sServiceName())); err != nil {
			log.Warnf("%s: disable service failed: %v", p.host, err)
		}
		if err := p.host.Sudo().FS().Remove(unitPath); err != nil {
			log.Warnf("%s: remove service unit failed: %v", p.host, err)
		}
	}

	binaryPath := p.host.Configurer.K3sBinaryPath()
	if p.host.Sudo().FS().FileExist(binaryPath) {
		if err := p.host.Sudo().FS().Remove(binaryPath); err != nil {
			log.Warnf("%s: remove binary failed: %v", p.host, err)
		}
	}

	log.Infof("%s: node removed from cluster", p.host)
	return nil
}
