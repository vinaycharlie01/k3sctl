package binprovider

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"log/slog"

	"github.com/k0sproject/rig/v2/remotefs"
)

type stagedFile struct {
	host Host
	path string
}

// StagedPath returns the path of the staged binary on the host, or an empty
// string when nothing has been staged yet.
func (s *stagedFile) StagedPath() string {
	return s.path
}

func (s *stagedFile) CleanUp(ctx context.Context) error {
	if s.path == "" {
		return nil
	}
	slog.Debug(fmt.Sprintf("%s: cleaning up staged binary at %s", s.host, s.path))
	return s.host.DeleteFile(s.path)
}

func needsUpgrade(host Host, targetVersion string) bool {
	installed := host.InstalledK3sVersion()
	if installed == "" {
		return true
	}
	return installed != targetVersion
}

func stageTempPath(version string) string {
	return filepath.Join("/tmp", fmt.Sprintf("k3s-%s.tmp", version))
}

func stageUpload(ctx context.Context, host Host, localPath, remotePath string) error {
	slog.Info(fmt.Sprintf("%s: uploading k3s binary from %s", host, localPath))
	if err := host.Sudo().FS().MkdirAll(filepath.Dir(remotePath), fs.FileMode(0755)); err != nil {
		return fmt.Errorf("create directory for binary: %w", err)
	}

	return remotefs.Upload(host.Sudo().FS(), localPath, remotePath)
}

func stageDownload(ctx context.Context, host Host, url, remotePath string) error {
	slog.Info(fmt.Sprintf("%s: downloading k3s binary from %s", host, url))
	return host.DownloadURL(url, remotePath)
}
