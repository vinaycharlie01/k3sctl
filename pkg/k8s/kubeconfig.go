package k8s

import (
	"fmt"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// RewriteKubeconfig rewrites a raw kubeconfig YAML from a K3s server,
// replacing the server address and optionally the user/cluster names.
// Uses clientcmd for proper parsing and serialization.
func RewriteKubeconfig(raw []byte, apiAddress, userName, clusterName string) ([]byte, error) {
	config, err := clientcmd.Load(raw)
	if err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}

	for name, c := range config.Clusters {
		if strings.Contains(c.Server, "127.0.0.1") || strings.Contains(c.Server, "localhost") {
			c.Server = apiAddress
		}

		if clusterName != "" && name == "default" {
			config.Clusters[clusterName] = c
			delete(config.Clusters, name)

			for ctxName, ctx := range config.Contexts {
				if ctx.Cluster == name {
					ctx.Cluster = clusterName
				}
				if userName != "" && ctx.AuthInfo == "default" {
					ctx.AuthInfo = userName
				}
				config.Contexts[ctxName] = ctx
			}
		}
	}

	if userName != "" {
		if info, ok := config.AuthInfos["default"]; ok {
			config.AuthInfos[userName] = info
			delete(config.AuthInfos, "default")
		}
	}

	return clientcmd.Write(*config)
}

// WriteKubeconfigToFile writes a clientcmd API config to a file using
// clientcmd.WriteToFile.
func WriteKubeconfigToFile(config *clientcmdapi.Config, path string) error {
	return clientcmd.WriteToFile(*config, path)
}

// ParseKubeconfig parses raw kubeconfig bytes into a clientcmd API config.
func ParseKubeconfig(raw []byte) (*clientcmdapi.Config, error) {
	return clientcmd.Load(raw)
}
