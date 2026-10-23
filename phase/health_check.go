package phase

import (
	"context"
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type HealthInfo struct {
	Host           string
	Role           string
	K3sStatus      string
	K3sVersion     string
	APIServer      string
	Etcd           string
	Scheduler      string
	ControllerMgr  string
	NodeReady      string
	PodCount       string
}

type HealthCheck struct {
	GenericPhase
	Results []HealthInfo
	mu      sync.Mutex
}

func (p *HealthCheck) Title() string {
	return "Deep health check"
}

func (p *HealthCheck) ShouldRun() bool {
	return p.Config.Spec.Hosts.Count() > 0
}

func (p *HealthCheck) Run(ctx context.Context) error {
	leader := p.Config.Spec.K3sLeader()
	return p.parallelDo(ctx, p.Config.Spec.Hosts, func(ctx context.Context, h *cluster.Host) error {
		info := HealthInfo{
			Host:       h.String(),
			Role:       string(h.Role),
			K3sVersion: h.Metadata.K3sInstalledVersion,
		}

		if !h.Metadata.K3sServiceInstalled {
			info.K3sStatus = "not installed"
			info.APIServer = "-"
			info.Etcd = "-"
			info.Scheduler = "-"
			info.ControllerMgr = "-"
			info.NodeReady = "-"
			info.PodCount = "-"
			p.mu.Lock()
			p.Results = append(p.Results, info)
			p.mu.Unlock()
			return nil
		}

		if h.Metadata.K3sServiceRunning {
			info.K3sStatus = "running"
		} else {
			info.K3sStatus = "stopped"
		}

		if info.K3sVersion == "" {
			info.K3sVersion = "-"
		}

		if h.IsServer() && h.Metadata.K3sServiceRunning {
			info.APIServer = p.checkAPIServer(h)
			info.Etcd = p.checkEtcd(h)
			info.Scheduler = p.checkComponent(h, "scheduler")
			info.ControllerMgr = p.checkComponent(h, "controller-manager")
		} else {
			info.APIServer = "-"
			info.Etcd = "-"
			info.Scheduler = "-"
			info.ControllerMgr = "-"
		}

		if leader != nil && leader.Metadata.K3sServiceRunning {
			info.NodeReady = p.checkNodeReady(leader, h)
			if h.IsServer() && h.Metadata.K3sServiceRunning {
				info.PodCount = p.checkPodCount(h)
			} else {
				info.PodCount = "-"
			}
		} else {
			info.NodeReady = "-"
			info.PodCount = "-"
		}

		p.mu.Lock()
		p.Results = append(p.Results, info)
		p.mu.Unlock()
		return nil
	})
}

func (p *HealthCheck) checkAPIServer(h *cluster.Host) string {
	cmd := h.Configurer.K3sCmdf("kubectl get --raw /healthz 2>/dev/null")
	output, err := h.Sudo().ExecOutput(cmd)
	if err != nil {
		return "unhealthy"
	}
	if strings.TrimSpace(output) == "ok" {
		return "ok"
	}
	return "degraded"
}

func (p *HealthCheck) checkEtcd(h *cluster.Host) string {
	cmd := h.Configurer.K3sCmdf("kubectl get --raw /healthz/etcd 2>/dev/null")
	output, err := h.Sudo().ExecOutput(cmd)
	if err != nil {
		etcdDir := fmt.Sprintf("%s/server/db/etcd", h.K3sDataDir())
		checkCmd := fmt.Sprintf("test -d %s && echo exists", shellQuote(etcdDir))
		out, err2 := h.Sudo().ExecOutput(checkCmd)
		if err2 == nil && strings.TrimSpace(out) == "exists" {
			return "present"
		}
		return "n/a"
	}
	if strings.TrimSpace(output) == "ok" {
		return "ok"
	}
	return "degraded"
}

func (p *HealthCheck) checkComponent(h *cluster.Host, name string) string {
	cmd := h.Configurer.K3sCmdf("kubectl get --raw /healthz/%s 2>/dev/null", name)
	output, err := h.Sudo().ExecOutput(cmd)
	if err != nil {
		return "n/a"
	}
	if strings.TrimSpace(output) == "ok" {
		return "ok"
	}
	return "degraded"
}

func (p *HealthCheck) checkNodeReady(leader *cluster.Host, h *cluster.Host) string {
	nodeName := h.KubernetesNodeName()
	cmd := leader.Configurer.KubectlCmdf(leader, leader.K3sDataDir(),
		"get node %s -o jsonpath={.status.conditions[?(@.type==\"Ready\")].status} 2>/dev/null",
		shellQuote(nodeName))
	output, err := leader.Sudo().ExecOutput(cmd)
	if err != nil {
		log.Debugf("%s: node ready check failed: %v", h, err)
		return "unknown"
	}
	status := strings.TrimSpace(output)
	if status == "True" {
		return "ready"
	}
	if status == "False" {
		return "not ready"
	}
	return status
}

func (p *HealthCheck) checkPodCount(h *cluster.Host) string {
	cmd := h.Configurer.K3sCmdf("kubectl get pods --all-namespaces --field-selector spec.nodeName=%s --no-headers 2>/dev/null | wc -l", shellQuote(h.KubernetesNodeName()))
	output, err := h.Sudo().ExecOutput(cmd)
	if err != nil {
		return "-"
	}
	return strings.TrimSpace(output)
}
