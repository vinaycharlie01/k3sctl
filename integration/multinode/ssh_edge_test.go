package multinode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSSHKeyCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "ssh")
	kp, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", dir, err)
	}
	if !info.IsDir() {
		t.Errorf("%s is not a directory", dir)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("dir permissions = %o, want 0700", info.Mode().Perm())
	}

	if kp.PrivateKeyPath == "" {
		t.Error("PrivateKeyPath is empty")
	}
	if kp.AuthorizedKey == "" {
		t.Error("AuthorizedKey is empty")
	}
}

func TestGenerateSSHKeyPrivateKeyPermissions(t *testing.T) {
	dir := t.TempDir()
	kp, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	info, err := os.Stat(kp.PrivateKeyPath)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", kp.PrivateKeyPath, err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("key permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestGenerateSSHKeyPrivateKeyIsPEM(t *testing.T) {
	dir := t.TempDir()
	kp, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	data, err := os.ReadFile(kp.PrivateKeyPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "-----BEGIN PRIVATE KEY-----") {
		t.Error("private key does not contain PEM header")
	}
	if !strings.Contains(string(data), "-----END PRIVATE KEY-----") {
		t.Error("private key does not contain PEM footer")
	}
}

func TestGenerateSSHKeyAuthorizedKeyFormat(t *testing.T) {
	dir := t.TempDir()
	kp, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	if !strings.HasPrefix(kp.AuthorizedKey, "ssh-ed25519 ") {
		t.Errorf("AuthorizedKey prefix = %q, want 'ssh-ed25519 '", kp.AuthorizedKey[:20])
	}
	if strings.Contains(kp.AuthorizedKey, "\n") {
		t.Error("AuthorizedKey contains a newline")
	}
}

func TestGenerateSSHKeyUniqueness(t *testing.T) {
	dir1 := filepath.Join(t.TempDir(), "key1")
	dir2 := filepath.Join(t.TempDir(), "key2")

	kp1, err := GenerateSSHKey(dir1)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}
	kp2, err := GenerateSSHKey(dir2)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	if kp1.AuthorizedKey == kp2.AuthorizedKey {
		t.Error("two generated keys have the same authorized key — not unique")
	}
}

func TestGenerateSSHKeyFileName(t *testing.T) {
	dir := t.TempDir()
	kp, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	expected := filepath.Join(dir, "id_ed25519")
	if kp.PrivateKeyPath != expected {
		t.Errorf("PrivateKeyPath = %q, want %q", kp.PrivateKeyPath, expected)
	}
}

func TestGenerateSSHKeyInExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	// Create a file in the directory first to prove it doesn't fail on existing dir
	if err := os.WriteFile(filepath.Join(dir, "marker"), []byte("exists"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	kp, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	if _, err := os.Stat(kp.PrivateKeyPath); err != nil {
		t.Fatalf("key file missing: %v", err)
	}
	// Marker file should still exist
	if _, err := os.Stat(filepath.Join(dir, "marker")); err != nil {
		t.Fatalf("marker file missing after key generation: %v", err)
	}
}

func TestSSHRunnerDefaultTimeout(t *testing.T) {
	runner := &SSHRunner{
		User:    "ubuntu",
		Port:    22,
		KeyPath: "/nonexistent/key",
	}
	if runner.Timeout != 0 {
		t.Errorf("default Timeout = %v, want 0 (triggers 30s default)", runner.Timeout)
	}
}

func TestSSHRunnerMissingKeyReturnsError(t *testing.T) {
	runner := &SSHRunner{
		User:    "ubuntu",
		Port:    22,
		KeyPath: "/nonexistent/key",
	}

	_, err := runner.Run(context.Background(), "10.0.0.1", "uname")
	if err == nil {
		t.Fatal("Run() error = nil, want an error for missing key")
	}
	if !strings.Contains(err.Error(), "ssh connect") {
		t.Errorf("error = %v, want ssh connect error", err)
	}
}

func TestKeyPairStruct(t *testing.T) {
	kp := KeyPair{
		PrivateKeyPath: "/tmp/id_ed25519",
		AuthorizedKey:  "ssh-ed25519 AAAA... test@host",
	}
	if kp.PrivateKeyPath != "/tmp/id_ed25519" {
		t.Errorf("PrivateKeyPath = %q", kp.PrivateKeyPath)
	}
	if !strings.HasPrefix(kp.AuthorizedKey, "ssh-ed25519") {
		t.Errorf("AuthorizedKey = %q", kp.AuthorizedKey)
	}
}
