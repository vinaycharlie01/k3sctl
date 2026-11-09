package phase

import (
	"context"
	"fmt"
	"io"

	"log/slog"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/nirantarasi/k3sctl/pkg/k8s"
	"k8s.io/client-go/tools/clientcmd"
)

type GetKubeconfig struct {
	GenericPhase
	APIAddress string
	User       string
	Cluster    string
	Out        io.Writer
	leader     *cluster.Host
}

func (p *GetKubeconfig) Title() string {
	return "Get admin kubeconfig"
}

func (p *GetKubeconfig) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *GetKubeconfig) ShouldRun() bool {
	return p.leader != nil
}

func (p *GetKubeconfig) Run(_ context.Context) error {
	h := p.leader
	kubeconfigPath := "/etc/rancher/k3s/k3s.yaml"
	if kc := p.Config.Spec.K3s.Kubeconfig; kc != nil && kc.Path != "" {
		kubeconfigPath = kc.Path
	}

	slog.Info(fmt.Sprintf("%s: reading kubeconfig from %s", h, kubeconfigPath))
	output, err := h.Sudo().ExecOutput(fmt.Sprintf("cat %s", shellQuote(kubeconfigPath)))
	if err != nil {
		return fmt.Errorf("%s: read kubeconfig: %w", h, err)
	}

	apiAddress := p.APIAddress
	if apiAddress == "" {
		apiAddress = fmt.Sprintf("https://%s:6443", h.Address())
	}

	rewritten, err := k8s.RewriteKubeconfig([]byte(output), apiAddress, p.User, p.Cluster)
	if err != nil {
		return fmt.Errorf("rewrite kubeconfig: %w", err)
	}

	kubeconfig := string(rewritten)
	if p.Out != nil {
		if _, err := io.WriteString(p.Out, kubeconfig); err != nil {
			return fmt.Errorf("write kubeconfig: %w", err)
		}
	}

	p.Config.Metadata.Kubeconfig = kubeconfig

	kc, err := k8s.NewFromKubeconfig(rewritten)
	if err != nil {
		slog.Warn(fmt.Sprintf("could not build kubernetes client from kubeconfig: %v", err))
	} else {
		p.Config.Metadata.KubeClient = kc
	}

	slog.Info("kubeconfig retrieved successfully")
	return nil
}

// WriteKubeconfigToFile writes the kubeconfig to a file using clientcmd.WriteToFile.
func WriteKubeconfigToFile(kubeconfigYAML string, path string) error {
	config, err := clientcmd.Load([]byte(kubeconfigYAML))
	if err != nil {
		return fmt.Errorf("parse kubeconfig: %w", err)
	}
	return clientcmd.WriteToFile(*config, path)
}
