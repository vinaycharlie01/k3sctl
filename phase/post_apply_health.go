package phase

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type PostApplyHealthCheck struct {
	GenericPhase
}

func (p *PostApplyHealthCheck) Title() string {
	return "Post-apply health verification"
}

func (p *PostApplyHealthCheck) ShouldRun() bool {
	cfg := p.Config.Spec.K3s
	return cfg != nil && cfg.HealthCheck != nil && cfg.HealthCheck.Enabled
}

func (p *PostApplyHealthCheck) Run(ctx context.Context) error {
	timeout := 2 * time.Minute
	if p.Config.Spec.K3s.HealthCheck.Timeout != "" {
		d, err := time.ParseDuration(p.Config.Spec.K3s.HealthCheck.Timeout)
		if err == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	leader := p.Config.Spec.K3sLeader()
	if leader == nil || !leader.Metadata.K3sServiceRunning {
		log.Warnf("no running leader found, skipping health verification")
		return nil
	}

	log.Infof("verifying cluster health (timeout: %s)", timeout)

	if err := p.waitForAPIServer(ctx, leader); err != nil {
		return fmt.Errorf("API server health check failed: %w", err)
	}

	if err := p.waitForNodes(ctx, leader); err != nil {
		return fmt.Errorf("node readiness check failed: %w", err)
	}

	log.Infof("cluster health verification passed")
	return nil
}

func (p *PostApplyHealthCheck) waitForAPIServer(ctx context.Context, leader *cluster.Host) error {
	log.Infof("%s: waiting for API server", leader)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for API server on %s", leader)
		default:
		}

		cmd := leader.Configurer.KubectlCmdf(leader, leader.K3sDataDir(), "get --raw /healthz 2>/dev/null")
		output, err := leader.Sudo().ExecOutput(cmd)
		if err == nil && strings.TrimSpace(output) == "ok" {
			log.Infof("%s: API server healthy", leader)
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

func (p *PostApplyHealthCheck) waitForNodes(ctx context.Context, leader *cluster.Host) error {
	expected := p.Config.Spec.Hosts.Count()
	log.Infof("waiting for %d node(s) to be ready", expected)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for all %d nodes to be ready", expected)
		default:
		}

		cmd := leader.Configurer.KubectlCmdf(leader, leader.K3sDataDir(),
			`get nodes --no-headers 2>/dev/null | grep -c " Ready"`)
		output, err := leader.Sudo().ExecOutput(cmd)
		if err == nil {
			count := strings.TrimSpace(output)
			log.Debugf("ready nodes: %s/%d", count, expected)
			if count == fmt.Sprintf("%d", expected) {
				log.Infof("all %d nodes are ready", expected)
				return nil
			}
		}

		time.Sleep(10 * time.Second)
	}
}
