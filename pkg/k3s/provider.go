package k3s

import (
	"context"

	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/vinaycharlie01/k3sctl/pkg/k3s/binprovider"
)

type BinaryProvider interface {
	NeedsUpgrade(ctx context.Context) (bool, error)
	IsUpload() bool
	Stage(ctx context.Context) error
	CleanUp(ctx context.Context) error
	StagedPath() string
}

func NewBinaryProvider(h *cluster.Host, k3s *cluster.K3s) BinaryProvider {
	if h.K3sBinaryPath != "" {
		return binprovider.NewLocalFile(h, k3s.Version, h.K3sBinaryPath)
	}

	if h.UploadBinary {
		return binprovider.NewLocalUpload(h, k3s.Version, nil)
	}

	return binprovider.NewGitHub(h, k3s.Version)
}
