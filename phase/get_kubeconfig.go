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

	apiAddress := p.APIAddress
	if apiAddress == "" {
		apiAddress = fmt.Sprintf("https://%s:6443", h.Address())
	}
	kubeconfig = rewriteKubeconfig(kubeconfig, apiAddress, p.User, p.Cluster)

	if p.Out != nil {
		if _, err := io.WriteString(p.Out, kubeconfig); err != nil {
			return fmt.Errorf("write kubeconfig: %w", err)
		}
	}

	p.Config.Metadata.Kubeconfig = kubeconfig
	log.Infof("kubeconfig retrieved successfully")

	return nil
}

// rewriteKubeconfig points a k3s-generated kubeconfig at the given API
// address and optionally renames the default user and cluster.
func rewriteKubeconfig(raw, apiAddress, user, cluster string) string {
	out := strings.ReplaceAll(raw, "https://127.0.0.1:6443", apiAddress)
	if cluster != "" {
		out = strings.ReplaceAll(out, "cluster: default", "cluster: "+cluster)
		out = strings.ReplaceAll(out, "name: default", "name: "+cluster)
	}
	if user != "" {
		out = strings.ReplaceAll(out, "user: default", "user: "+user)
	}
	return out
}
