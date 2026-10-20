package cluster

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type UploadFile struct {
	Name    string `yaml:"name,omitempty"`
	Source  string `yaml:"src"`
	DstDir  string `yaml:"dstDir,omitempty"`
	Dst     string `yaml:"dst,omitempty"`
	Perm    string `yaml:"perm,omitempty" default:"0644"`
	DirPerm string `yaml:"dirPerm,omitempty" default:"0755"`
	User    string `yaml:"user,omitempty"`
	Group   string `yaml:"group,omitempty"`
	SHA256  string `yaml:"sha256,omitempty"`
}

func (u *UploadFile) IsURL() bool {
	_, err := url.ParseRequestURI(u.Source)
	return err == nil && (strings.HasPrefix(u.Source, "http://") || strings.HasPrefix(u.Source, "https://"))
}

func (u *UploadFile) Destination() string {
	if u.Dst != "" {
		return u.Dst
	}
	if u.DstDir != "" {
		return filepath.Join(u.DstDir, filepath.Base(u.Source))
	}
	return ""
}

func (u *UploadFile) Validate() error {
	if u.Source == "" {
		return fmt.Errorf("upload file source is required")
	}
	if u.Dst == "" && u.DstDir == "" {
		return fmt.Errorf("upload file destination (dst or dstDir) is required")
	}
	return nil
}

func (u *UploadFile) Resolve() ([]UploadFile, error) {
	if u.IsURL() {
		return []UploadFile{*u}, nil
	}

	if strings.Contains(u.Source, "*") || strings.Contains(u.Source, "?") {
		matches, err := doublestar.FilepathGlob(u.Source)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", u.Source, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no files matched pattern %q", u.Source)
		}
		var files []UploadFile
		for _, match := range matches {
			f := *u
			f.Source = match
			if u.DstDir != "" {
				f.Dst = filepath.Join(u.DstDir, filepath.Base(match))
			}
			files = append(files, f)
		}
		return files, nil
	}

	return []UploadFile{*u}, nil
}

func (u *UploadFile) VerifySHA256(path string) error {
	if u.SHA256 == "" {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file for checksum: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("compute checksum: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != u.SHA256 {
		return fmt.Errorf("sha256 mismatch for %s: expected %s, got %s", path, u.SHA256, actual)
	}
	return nil
}
