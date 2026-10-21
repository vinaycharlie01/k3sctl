package cluster

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadFileIsURL(t *testing.T) {
	tests := []struct {
		source   string
		expected bool
	}{
		{"https://example.com/file.tar.gz", true},
		{"http://example.com/file", true},
		{"/local/path/file.bin", false},
		{"relative/path", false},
		{"ftp://example.com/file", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			u := UploadFile{Source: tt.source}
			assert.Equal(t, tt.expected, u.IsURL())
		})
	}
}

func TestUploadFileDestination(t *testing.T) {
	tests := []struct {
		name     string
		file     UploadFile
		expected string
	}{
		{
			name:     "explicit dst",
			file:     UploadFile{Dst: "/opt/bin/k3s"},
			expected: "/opt/bin/k3s",
		},
		{
			name:     "dstDir with source basename",
			file:     UploadFile{Source: "/local/myfile.tar.gz", DstDir: "/opt/bin"},
			expected: filepath.Join("/opt/bin", "myfile.tar.gz"),
		},
		{
			name:     "dst takes precedence over dstDir",
			file:     UploadFile{Dst: "/explicit", DstDir: "/dir", Source: "/local/file"},
			expected: "/explicit",
		},
		{
			name:     "empty when neither set",
			file:     UploadFile{Source: "/local/file"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.file.Destination())
		})
	}
}

func TestUploadFileValidate(t *testing.T) {
	tests := []struct {
		name    string
		file    UploadFile
		wantErr bool
	}{
		{
			name:    "valid with dst",
			file:    UploadFile{Source: "/local/file", Dst: "/remote/file"},
			wantErr: false,
		},
		{
			name:    "valid with dstDir",
			file:    UploadFile{Source: "/local/file", DstDir: "/remote/dir"},
			wantErr: false,
		},
		{
			name:    "missing source",
			file:    UploadFile{Dst: "/remote/file"},
			wantErr: true,
		},
		{
			name:    "missing destination",
			file:    UploadFile{Source: "/local/file"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.file.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUploadFileVerifySHA256(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile")
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(testFile, content, 0644))

	h := sha256.Sum256(content)
	correctHash := hex.EncodeToString(h[:])

	t.Run("correct hash", func(t *testing.T) {
		u := UploadFile{SHA256: correctHash}
		assert.NoError(t, u.VerifySHA256(testFile))
	})

	t.Run("wrong hash", func(t *testing.T) {
		u := UploadFile{SHA256: "0000000000000000000000000000000000000000000000000000000000000000"}
		err := u.VerifySHA256(testFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sha256 mismatch")
	})

	t.Run("empty hash skips check", func(t *testing.T) {
		u := UploadFile{SHA256: ""}
		assert.NoError(t, u.VerifySHA256(testFile))
	})

	t.Run("nonexistent file", func(t *testing.T) {
		u := UploadFile{SHA256: correctHash}
		assert.Error(t, u.VerifySHA256("/nonexistent/file"))
	})
}

func TestUploadFileResolve(t *testing.T) {
	t.Run("URL passthrough", func(t *testing.T) {
		u := UploadFile{Source: "https://example.com/k3s", Dst: "/opt/k3s"}
		files, err := u.Resolve()
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "https://example.com/k3s", files[0].Source)
	})

	t.Run("plain file passthrough", func(t *testing.T) {
		u := UploadFile{Source: "/local/file.bin", Dst: "/remote/file.bin"}
		files, err := u.Resolve()
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("glob with matches", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644))

		u := UploadFile{Source: filepath.Join(tmpDir, "*.txt"), DstDir: "/remote/dir"}
		files, err := u.Resolve()
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("glob no matches", func(t *testing.T) {
		tmpDir := t.TempDir()
		u := UploadFile{Source: filepath.Join(tmpDir, "*.xyz"), DstDir: "/remote/dir"}
		_, err := u.Resolve()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no files matched")
	})
}

func TestUploadFileValidateRequiresSource(t *testing.T) {
	u := UploadFile{Dst: "/remote/file"}
	err := u.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source is required")
}

func TestUploadFileValidateRequiresDestination(t *testing.T) {
	u := UploadFile{Source: "/local/file"}
	err := u.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination")
}

func TestUploadFileDestination_DstTakesPrecedence(t *testing.T) {
	u := UploadFile{Dst: "/explicit/path", DstDir: "/dir", Source: "/local/file"}
	assert.Equal(t, "/explicit/path", u.Destination())
}

func TestUploadFileResolve_URLDoesNotGlob(t *testing.T) {
	u := UploadFile{Source: "https://example.com/app-*.tar.gz", Dst: "/opt/app.tar.gz"}
	files, err := u.Resolve()
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "https://example.com/app-*.tar.gz", files[0].Source)
}

func TestUploadFileResolve_GlobSetsDestFromDstDir(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "manifest.yaml"), []byte("data"), 0644))

	u := UploadFile{Source: filepath.Join(tmpDir, "*.yaml"), DstDir: "/opt/manifests"}
	files, err := u.Resolve()
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join("/opt/manifests", "manifest.yaml"), files[0].Dst)
}
