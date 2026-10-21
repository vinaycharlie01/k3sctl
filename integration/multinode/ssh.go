package multinode

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// KeyPair is an ephemeral SSH key generated for a test run.
type KeyPair struct {
	PrivateKeyPath string
	AuthorizedKey  string
}

// GenerateSSHKey creates an ed25519 key pair in dir. The private key is written
// with 0600 permissions and is never logged.
func GenerateSSHKey(dir string) (KeyPair, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return KeyPair{}, fmt.Errorf("create key directory: %w", err)
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, fmt.Errorf("generate key: %w", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return KeyPair{}, fmt.Errorf("marshal private key: %w", err)
	}

	path := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600); err != nil {
		return KeyPair{}, fmt.Errorf("write private key: %w", err)
	}

	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return KeyPair{}, fmt.Errorf("marshal public key: %w", err)
	}

	return KeyPair{
		PrivateKeyPath: path,
		AuthorizedKey:  strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey))),
	}, nil
}

// RemoteExecutor runs a shell command on a remote node.
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.14.0 --fake-name FakeRemoteExecutor -o ./mocks/fake_remote_executor.go . RemoteExecutor
type RemoteExecutor interface {
	Run(ctx context.Context, address, command string) (Result, error)
}

// SSHRunner executes commands on provisioned nodes.
type SSHRunner struct {
	User    string
	Port    int
	KeyPath string
	Timeout time.Duration
}

func (s *SSHRunner) Run(ctx context.Context, address, command string) (Result, error) {
	result := Result{Args: []string{"ssh", address, command}}

	client, err := s.connect(ctx, address)
	if err != nil {
		result.ExitCode = -1
		return result, fmt.Errorf("ssh connect to %s: %w", address, err)
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		result.ExitCode = -1
		return result, fmt.Errorf("ssh session on %s: %w", address, err)
	}
	defer func() { _ = session.Close() }()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	start := time.Now()
	if err := session.Start(command); err != nil {
		result.ExitCode = -1
		return result, fmt.Errorf("ssh start on %s: %w", address, err)
	}

	done := make(chan error, 1)
	go func() { done <- session.Wait() }()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		_ = session.Close()
		<-done

		result.Duration = time.Since(start)
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.ExitCode = -1
		return result, fmt.Errorf("ssh command on %s: %w", address, ctx.Err())
	case waitErr := <-done:
		result.Duration = time.Since(start)
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()

		if waitErr != nil {
			var exitErr *ssh.ExitError
			if errors.As(waitErr, &exitErr) {
				result.ExitCode = exitErr.ExitStatus()
				return result, fmt.Errorf("ssh command on %s failed with exit code %d: %s", address, result.ExitCode, tail(result.Stderr, 20))
			}
			result.ExitCode = -1
			return result, fmt.Errorf("ssh command on %s: %w", address, waitErr)
		}
		return result, nil
	}
}

func (s *SSHRunner) connect(ctx context.Context, address string) (*ssh.Client, error) {
	keyData, err := os.ReadFile(s.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	timeout := s.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	config := &ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		// The containers are ephemeral and generate host keys at boot, so
		// there is nothing stable to pin.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	target := net.JoinHostPort(address, strconv.Itoa(s.Port))
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return nil, err
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, target, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return ssh.NewClient(clientConn, chans, reqs), nil
}
