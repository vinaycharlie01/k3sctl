package binprovider

import (
	"context"
	"fmt"

	"log/slog"
)

type localFileProvider struct {
	stagedFile
	host      Host
	version   string
	localPath string
}

func NewLocalFile(host Host, version, localPath string) *localFileProvider {
	return &localFileProvider{
		host:      host,
		version:   version,
		localPath: localPath,
	}
}

func (p *localFileProvider) NeedsUpgrade(_ context.Context) (bool, error) {
	return needsUpgrade(p.host, p.version), nil
}

func (p *localFileProvider) IsUpload() bool {
	return true
}

func (p *localFileProvider) Stage(ctx context.Context) error {
	remotePath := stageTempPath(p.version)
	slog.Info(fmt.Sprintf("%s: uploading local k3s binary from %s", p.host, p.localPath))
	if err := stageUpload(ctx, p.host, p.localPath, remotePath); err != nil {
		return fmt.Errorf("stage local file: %w", err)
	}
	p.stagedFile = stagedFile{host: p.host, path: remotePath}
	return nil
}
