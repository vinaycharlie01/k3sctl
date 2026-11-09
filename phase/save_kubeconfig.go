package phase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"log/slog"

	"k8s.io/client-go/tools/clientcmd"
)

type SaveKubeconfig struct {
	GenericPhase
	Path string
}

func (p *SaveKubeconfig) Title() string {
	return "Save admin kubeconfig"
}

func (p *SaveKubeconfig) ShouldRun() bool {
	return p.Config.Metadata.Kubeconfig != ""
}

func (p *SaveKubeconfig) Run(_ context.Context) error {
	path := p.Path
	if path == "" {
		path = clientcmd.RecommendedHomeFile
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create kubeconfig directory %s: %w", dir, err)
	}

	if err := WriteKubeconfigToFile(p.Config.Metadata.Kubeconfig, path); err != nil {
		return fmt.Errorf("save kubeconfig to %s: %w", path, err)
	}

	slog.Info(fmt.Sprintf("kubeconfig saved to %s", path))
	return nil
}
