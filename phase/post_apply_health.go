package phase

import (
	"context"
	"fmt"
	"time"

	"log/slog"

	"github.com/nirantarasi/k3sctl/pkg/k8s"
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

	kc := p.Config.Metadata.KubeClient
	if kc == nil {
		leader := p.Config.Spec.K3sLeader()
		if leader == nil || !leader.Metadata.K3sServiceRunning {
			slog.Warn("no running leader found, skipping health verification")
			return nil
		}
		var err error
		kc, err = kubeClientOrBuild(nil, leader)
		if err != nil {
			return fmt.Errorf("build kube client for health check: %w", err)
		}
	}

	slog.Info(fmt.Sprintf("verifying cluster health (timeout: %s)", timeout))

	if err := p.waitForAPIServer(ctx, kc); err != nil {
		return fmt.Errorf("API server health check failed: %w", err)
	}

	if err := p.waitForNodes(ctx, kc); err != nil {
		return fmt.Errorf("node readiness check failed: %w", err)
	}

	slog.Info("cluster health verification passed")
	return nil
}

func (p *PostApplyHealthCheck) waitForAPIServer(ctx context.Context, kc k8s.Client) error {
	slog.Info("waiting for API server")
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for API server")
		default:
		}

		if err := kc.CheckHealth(ctx); err == nil {
			slog.Info("API server healthy")
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

func (p *PostApplyHealthCheck) waitForNodes(ctx context.Context, kc k8s.Client) error {
	expected := p.Config.Spec.Hosts.Count()
	slog.Info(fmt.Sprintf("waiting for %d node(s) to be ready", expected))

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for all %d nodes to be ready", expected)
		default:
		}

		count, err := kc.CountReadyNodes(ctx)
		if err == nil {
			slog.Debug(fmt.Sprintf("ready nodes: %d/%d", count, expected))
			if count >= expected {
				slog.Info(fmt.Sprintf("all %d nodes are ready", expected))
				return nil
			}
		}

		time.Sleep(10 * time.Second)
	}
}
