package binprovider

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type downloadProvider struct {
	stagedFile
	host    Host
	version string
	url     string
}

func NewGitHub(host Host, version string) *downloadProvider {
	arch, err := host.Arch()
	if err != nil {
		arch = "amd64"
	}

	binaryName := "k3s"
	switch arch {
	case "arm64":
		binaryName = "k3s-arm64"
	case "arm":
		binaryName = "k3s-armhf"
	}

	url := fmt.Sprintf("https://github.com/k3s-io/k3s/releases/download/%s/%s", version, binaryName)
	return &downloadProvider{
		host:    host,
		version: version,
		url:     url,
	}
}

func NewCustomURL(host Host, version, url string) *downloadProvider {
	return &downloadProvider{
		host:    host,
		version: version,
		url:     url,
	}
}

func (p *downloadProvider) NeedsUpgrade(_ context.Context) (bool, error) {
	return needsUpgrade(p.host, p.version), nil
}

func (p *downloadProvider) IsUpload() bool {
	return false
}

func (p *downloadProvider) Stage(ctx context.Context) error {
	remotePath := stageTempPath(p.host, p.version)
	log.Infof("%s: staging k3s %s from %s", p.host, p.version, p.url)
	if err := stageDownload(ctx, p.host, p.url, remotePath); err != nil {
		return fmt.Errorf("stage download: %w", err)
	}
	p.stagedFile = stagedFile{host: p.host, path: remotePath}
	return nil
}
