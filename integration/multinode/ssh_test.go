package multinode

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestGenerateSSHKeyWritesUsableKeyPair(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")

	keyPair, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	if want := filepath.Join(dir, "id_ed25519"); keyPair.PrivateKeyPath != want {
		t.Fatalf("PrivateKeyPath = %q, want %q", keyPair.PrivateKeyPath, want)
	}

	info, err := os.Stat(keyPair.PrivateKeyPath)
	if err != nil {
		t.Fatalf("stat private key: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Fatalf("private key mode = %o, want 600", mode)
	}

	pemData, err := os.ReadFile(keyPair.PrivateKeyPath)
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}
	if _, err := ssh.ParsePrivateKey(pemData); err != nil {
		t.Fatalf("parse generated private key: %v", err)
	}

	if !strings.HasPrefix(keyPair.AuthorizedKey, "ssh-ed25519 AAAA") {
		t.Fatalf("AuthorizedKey = %q, want ssh-ed25519 prefix", keyPair.AuthorizedKey)
	}
	if strings.Contains(keyPair.AuthorizedKey, "PRIVATE KEY") {
		t.Fatal("AuthorizedKey contains private key material")
	}
}

func TestGenerateSSHKeyReplacesExistingKey(t *testing.T) {
	dir := t.TempDir()

	first, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("first GenerateSSHKey() error = %v", err)
	}
	second, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("second GenerateSSHKey() error = %v", err)
	}

	if first.AuthorizedKey == second.AuthorizedKey {
		t.Fatal("expected a fresh key pair on the second call")
	}
}

func TestSSHRunnerRunMissingKeyFailsBeforeDialing(t *testing.T) {
	runner := &SSHRunner{
		User:    "ubuntu",
		Port:    22,
		KeyPath: filepath.Join(t.TempDir(), "absent"),
		Timeout: time.Second,
	}

	result, err := runner.Run(context.Background(), "127.0.0.1", "true")
	if err == nil {
		t.Fatal("expected an error for a missing private key")
	}
	if !strings.Contains(err.Error(), "ssh connect to 127.0.0.1") {
		t.Fatalf("error = %v, want ssh connect failure", err)
	}
	if !strings.Contains(err.Error(), "read private key") {
		t.Fatalf("error = %v, want read private key cause", err)
	}
	if result.ExitCode != -1 {
		t.Fatalf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestSSHRunnerRunConnectionRefused(t *testing.T) {
	dir := t.TempDir()
	keyPair, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	runner := &SSHRunner{
		User:    "ubuntu",
		Port:    port,
		KeyPath: keyPair.PrivateKeyPath,
		Timeout: 2 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := runner.Run(ctx, "127.0.0.1", "true")
	if err == nil {
		t.Fatal("expected an error for a refused connection")
	}
	if !strings.Contains(err.Error(), "ssh connect to 127.0.0.1") {
		t.Fatalf("error = %v, want ssh connect failure", err)
	}
	if result.ExitCode != -1 {
		t.Fatalf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestSSHRunnerRunHonoursCancelledContext(t *testing.T) {
	dir := t.TempDir()
	keyPair, err := GenerateSSHKey(dir)
	if err != nil {
		t.Fatalf("GenerateSSHKey() error = %v", err)
	}

	runner := &SSHRunner{
		User:    "ubuntu",
		Port:    22,
		KeyPath: keyPair.PrivateKeyPath,
		Timeout: 30 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err = runner.Run(ctx, "127.0.0.1", "true")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("Run took %s, want a prompt return for a cancelled context", elapsed)
	}
}
