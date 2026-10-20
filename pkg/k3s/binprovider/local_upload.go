package binprovider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	log "github.com/sirupsen/logrus"
)

type BinaryCacher interface {
	CacheBinary(ctx context.Context, version, arch string) (string, error)
	CachedBinaryPath(version, arch string) string
}

type localUploadProvider struct {
	stagedFile
	host    Host
	version string
	url     string
	cacher  BinaryCacher
}

func NewLocalUpload(host Host, version string, cacher BinaryCacher) *localUploadProvider {
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
	return &localUploadProvider{
		host:    host,
		version: version,
		url:     url,
		cacher:  cacher,
	}
}

func (p *localUploadProvider) NeedsUpgrade(_ context.Context) (bool, error) {
	return needsUpgrade(p.host, p.version), nil
}

func (p *localUploadProvider) IsUpload() bool {
	return true
}

func (p *localUploadProvider) Stage(ctx context.Context) error {
	arch, err := p.host.Arch()
	if err != nil {
		return fmt.Errorf("detect arch: %w", err)
	}

	var localPath string
	if p.cacher != nil {
		localPath = p.cacher.CachedBinaryPath(p.version, arch)
	}

	if localPath == "" || !fileExists(localPath) {
		localPath, err = downloadToFile(ctx, p.url, p.version, arch)
		if err != nil {
			return fmt.Errorf("download binary: %w", err)
		}
		if p.cacher != nil {
			if cached, err := p.cacher.CacheBinary(ctx, p.version, arch); err == nil {
				localPath = cached
			}
		}
	}

	remotePath := stageTempPath(p.host, p.version)
	log.Infof("%s: uploading k3s %s binary", p.host, p.version)
	if err := stageUpload(ctx, p.host, localPath, remotePath); err != nil {
		return fmt.Errorf("stage upload: %w", err)
	}
	p.stagedFile = stagedFile{host: p.host, path: remotePath}
	return nil
}

func downloadToFile(ctx context.Context, url, version, arch string) (string, error) {
	cacheDir := filepath.Join(xdg.CacheHome, "k3sctl", "binaries")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(url))
	hash := hex.EncodeToString(h.Sum(nil))[:12]
	localPath := filepath.Join(cacheDir, fmt.Sprintf("k3s-%s-%s-%s", version, arch, hash))

	if fileExists(localPath) {
		return localPath, nil
	}

	log.Infof("downloading k3s %s for %s from %s", version, arch, url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpPath := localPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("write binary: %w", err)
	}
	f.Close()

	if err := os.Rename(tmpPath, localPath); err != nil {
		return "", fmt.Errorf("rename temp file: %w", err)
	}

	return localPath, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
