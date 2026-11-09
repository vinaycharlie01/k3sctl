package node

import (
	"context"
	"fmt"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/nirantarasi/k3sctl/pkg/k8s"
)

type retryFunc func(context.Context) error

// KubeNodeReadyFunc returns a retry function that reports whether the named
// node has the Ready condition set, using the Kubernetes API via client-go.
func KubeNodeReadyFunc(kc k8s.Client, nodeName string) retryFunc {
	return func(ctx context.Context) error {
		if kc == nil {
			return fmt.Errorf("no kubernetes client available")
		}
		ready, err := kc.IsNodeReady(ctx, nodeName)
		if err != nil {
			return fmt.Errorf("failed to get node status: %w", err)
		}
		if !ready {
			return fmt.Errorf("node %s is not ready", nodeName)
		}
		return nil
	}
}

func HTTPStatusFunc(h *cluster.Host, url string, expected ...int) retryFunc {
	return func(ctx context.Context) error {
		return h.CheckHTTPStatus(ctx, url, expected...)
	}
}

func ServiceRunningFunc(h *cluster.Host, service string) retryFunc {
	return func(ctx context.Context) error {
		svc, err := h.Sudo().Service(service)
		if err != nil {
			return fmt.Errorf("get service %s: %w", service, err)
		}
		if !svc.IsRunning(ctx) {
			return fmt.Errorf("service %s is not running", service)
		}
		return nil
	}
}

func ServiceStoppedFunc(h *cluster.Host, service string) retryFunc {
	return func(ctx context.Context) error {
		svc, err := h.Sudo().Service(service)
		if err != nil {
			return fmt.Errorf("get service %s: %w", service, err)
		}
		if svc.IsRunning(ctx) {
			return fmt.Errorf("service %s is still running", service)
		}
		return nil
	}
}
