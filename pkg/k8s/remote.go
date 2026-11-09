package k8s

import (
	"fmt"
)

// NewFromRemoteHost fetches the kubeconfig from a remote K3s server and
// builds a Client. The execOutput function should execute a command on the
// remote host and return stdout (e.g. leader.Sudo().ExecOutput).
func NewFromRemoteHost(execOutput func(string) (string, error), kubeconfigPath, apiAddress string) (Client, error) {
	output, err := execOutput(fmt.Sprintf("cat '%s'", kubeconfigPath))
	if err != nil {
		return nil, fmt.Errorf("read kubeconfig from remote: %w", err)
	}

	rewritten, err := RewriteKubeconfig([]byte(output), apiAddress, "", "")
	if err != nil {
		return nil, fmt.Errorf("rewrite kubeconfig: %w", err)
	}

	return NewFromKubeconfig(rewritten)
}
