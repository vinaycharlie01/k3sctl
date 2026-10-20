package phase

import (
	"context"
	"fmt"
	"io"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
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

	log.Infof("%s: reading kubeconfig from %s", h, kubeconfigPath)
	output, err := h.Sudo().ExecOutput(fmt.Sprintf("cat %s", shellQuote(kubeconfigPath)))
	if err != nil {
		return fmt.Errorf("%s: read kubeconfig: %w", h, err)
	}

	kubeconfig := output

	if p.APIAddress != "" {
		kubeconfig = strings.ReplaceAll(kubeconfig, "https://127.0.0.1:6443", p.APIAddress)
	} else {
		addr := h.Address()
		kubeconfig = strings.ReplaceAll(kubeconfig, "https://127.0.0.1:6443", fmt.Sprintf("https://%s:6443", addr))
	}

	if p.Cluster != "" {
		kubeconfig = strings.ReplaceAll(kubeconfig, "cluster: default", fmt.Sprintf("cluster: %s", p.Cluster))
		kubeconfig = strings.ReplaceAll(kubeconfig, "name: default", fmt.Sprintf("name: %s", p.Cluster))
	}

	if p.User != "" {
		kubeconfig = strings.ReplaceAll(kubeconfig, "user: default", fmt.Sprintf("user: %s", p.User))
	}

	if p.Out != nil {
		if _, err := io.WriteString(p.Out, kubeconfig); err != nil {
			return fmt.Errorf("write kubeconfig: %w", err)
		}
	}

	p.Config.Metadata.Kubeconfig = kubeconfig
	log.Infof("kubeconfig retrieved successfully")

	return nil
}
