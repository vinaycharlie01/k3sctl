package phase

import (
	"fmt"

	"github.com/nirantarasi/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/nirantarasi/k3sctl/pkg/k8s"
)

// kubeClientOrBuild returns the provided kube client if non-nil, otherwise
// builds one on the fly by fetching the kubeconfig from the leader via SSH.
func kubeClientOrBuild(kc k8s.Client, leader *cluster.Host) (k8s.Client, error) {
	if kc != nil {
		return kc, nil
	}
	if leader == nil {
		return nil, fmt.Errorf("no leader available to fetch kubeconfig")
	}
	apiAddress := fmt.Sprintf("https://%s:6443", leader.Address())
	execFn := func(cmd string) (string, error) {
		return leader.Sudo().ExecOutput(cmd)
	}
	return k8s.NewFromRemoteHost(execFn, "/etc/rancher/k3s/k3s.yaml", apiAddress)
}
