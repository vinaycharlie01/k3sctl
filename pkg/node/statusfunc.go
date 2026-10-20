package node

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/k0sproject/rig/v2/cmd"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

type retryFunc func(context.Context) error

type kubeNodeStatus struct {
	Status struct {
		Conditions []struct {
			Status string `json:"status"`
			Type   string `json:"type"`
		} `json:"conditions"`
	} `json:"status"`
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func KubeNodeReadyFunc(h *cluster.Host) retryFunc {
	return func(_ context.Context) error {
		nodeName := h.KubernetesNodeName()
		output, err := h.Sudo().ExecOutput(h.Configurer.KubectlCmdf(h, h.K3sDataDir(), "get node %s -o json", shellQuote(nodeName)), cmd.HideOutput())
		if err != nil {
			return fmt.Errorf("failed to get node status: %w", err)
		}
		status := &kubeNodeStatus{}
		if err := json.Unmarshal([]byte(output), status); err != nil {
			return fmt.Errorf("failed to decode kubectl get node status output: %w", err)
		}
		for _, c := range status.Status.Conditions {
			if c.Type == "Ready" {
				if c.Status == "True" {
					return nil
				}
				return fmt.Errorf("node %s is not ready", nodeName)
			}
		}
		return fmt.Errorf("node %s 'Ready' condition not found", nodeName)
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
