// Package localhost implements a single-node integration test that installs K3s
// on the current machine via `localhost: true` in the k3sctl config and verifies
// the resulting cluster. It mirrors integration/multinode but needs no SSH or
// container provisioning.
package localhost

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	v1beta1 "github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
)

const (
	defaultApplyTimeout  = 15 * time.Minute
	defaultVerifyTimeout = 2 * time.Minute
	defaultResetTimeout  = 5 * time.Minute
)

// Options configures the localhost integration test runner.
type Options struct {
	Binary      string
	WorkDir     string
	ArtifactDir string
	ClusterName string
	K3sVersion  string

	ApplyTimeout  time.Duration
	VerifyTimeout time.Duration
	ResetTimeout  time.Duration
}

// Runner orchestrates a single-node localhost K3s install, verification, and
// teardown using the real k3sctl binary.
type Runner struct {
	opts    Options
	applied bool
}

// NewRunner validates options and returns a runner.
func NewRunner(opts Options) (*Runner, error) {
	switch {
	case opts.Binary == "":
		return nil, errors.New("k3sctl binary path is required")
	case opts.WorkDir == "":
		return nil, errors.New("work directory is required")
	case opts.ArtifactDir == "":
		return nil, errors.New("artifact directory is required")
	case opts.K3sVersion == "":
		return nil, errors.New("k3s version is required")
	}

	if opts.ClusterName == "" {
		opts.ClusterName = "k3sctl-localhost"
	}
	if opts.ApplyTimeout == 0 {
		opts.ApplyTimeout = defaultApplyTimeout
	}
	if opts.VerifyTimeout == 0 {
		opts.VerifyTimeout = defaultVerifyTimeout
	}
	if opts.ResetTimeout == 0 {
		opts.ResetTimeout = defaultResetTimeout
	}
	return &Runner{opts: opts}, nil
}

// Run executes the full test lifecycle: preflight, apply, verify, reset, and
// verify teardown. Cleanup always runs, even when a step fails.
func (r *Runner) Run(ctx context.Context) (runErr error) {
	if err := os.MkdirAll(r.opts.WorkDir, 0700); err != nil {
		return fmt.Errorf("create work directory: %w", err)
	}
	if err := os.MkdirAll(r.opts.ArtifactDir, 0700); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}

	r.applied = false
	var configPath string

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.opts.ResetTimeout)
		defer cancel()

		if runErr != nil {
			r.collectDiagnostics(cleanupCtx)
		}
		if err := r.cleanup(cleanupCtx, configPath); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("cleanup: %w", err))
		}
		if r.applied && configPath != "" {
			if err := r.verifyTeardown(cleanupCtx, configPath); err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("verify teardown: %w", err))
			}
		}
	}()

	log.Info("running preflight checks")
	if err := r.preflight(ctx); err != nil {
		return fmt.Errorf("preflight: %w", err)
	}

	configPath, err := r.writeConfig()
	if err != nil {
		return err
	}

	log.Info("applying k3sctl config")
	if err := r.apply(ctx, configPath); err != nil {
		return err
	}

	log.Info("verifying cluster")
	if err := r.verify(ctx, configPath); err != nil {
		return err
	}

	log.Info("single-node localhost integration test passed")
	return nil
}

func (r *Runner) preflight(ctx context.Context) error {
	pid1, err := r.shell(ctx, "ps -p 1 -o comm=", r.opts.VerifyTimeout)
	if err != nil {
		return fmt.Errorf("inspect PID 1: %w", err)
	}
	if strings.TrimSpace(pid1) != "systemd" {
		return fmt.Errorf("PID 1 is %q, want systemd", strings.TrimSpace(pid1))
	}

	if _, err := r.shell(ctx, "sudo -n true", r.opts.VerifyTimeout); err != nil {
		return fmt.Errorf("passwordless sudo unavailable: %w", err)
	}

	// Clear a stale host lock from a previous cancelled run.
	_, _ = r.shell(ctx, "sudo -n rm -f /var/lock/k3sctl.lock", r.opts.VerifyTimeout)
	return nil
}

func (r *Runner) writeConfig() (string, error) {
	config := &v1beta1.Cluster{
		APIVersion: v1beta1.APIVersion,
		Kind:       "Cluster",
		Metadata:   v1beta1.ClusterMetadata{Name: r.opts.ClusterName},
		Spec: &cluster.Spec{
			Hosts: cluster.Hosts{
				&cluster.Host{
					ConnectionConfig: cluster.ConnectionConfig{Localhost: true},
					Role:             cluster.RoleServer,
				},
			},
			K3s: &cluster.K3s{Version: r.opts.K3sVersion},
		},
	}
	config.SetDefaults()

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal cluster config: %w", err)
	}

	path := filepath.Join(r.opts.ArtifactDir, "k3sctl.yaml")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("write cluster config: %w", err)
	}
	log.Infof("wrote cluster config to %s", path)
	r.writeArtifact("k3sctl.yaml.rendered", string(data))
	return path, nil
}

func (r *Runner) apply(ctx context.Context, configPath string) error {
	r.applied = true

	applyCtx, cancel := context.WithTimeout(ctx, r.opts.ApplyTimeout)
	defer cancel()

	stdout, stderr, exitCode, err := r.exec(applyCtx, r.opts.Binary, "--debug", "apply", "-c", configPath)
	r.writeArtifact("apply.log", fmt.Sprintf("exit code: %d\n--- stdout ---\n%s--- stderr ---\n%s", exitCode, stdout, stderr))
	if err != nil {
		return fmt.Errorf("k3sctl apply: %w", err)
	}
	return nil
}

func (r *Runner) verify(ctx context.Context, configPath string) error {
	if err := r.verifyService(ctx); err != nil {
		return err
	}
	if err := r.verifyVersion(ctx); err != nil {
		return err
	}
	if err := r.verifyNodes(ctx); err != nil {
		return err
	}
	if err := r.verifyKubeconfig(ctx, configPath); err != nil {
		return err
	}
	return nil
}

func (r *Runner) verifyService(ctx context.Context) error {
	if _, err := r.shell(ctx, "systemctl is-active --quiet k3s", r.opts.VerifyTimeout); err != nil {
		return fmt.Errorf("k3s.service is not active: %w", err)
	}
	return nil
}

func (r *Runner) verifyVersion(ctx context.Context) error {
	output, err := r.shell(ctx, "sudo -n /usr/local/bin/k3s --version", r.opts.VerifyTimeout)
	r.writeArtifact("k3s-version.log", output)
	if err != nil {
		return fmt.Errorf("read k3s version: %w", err)
	}
	if !strings.Contains(output, r.opts.K3sVersion) {
		return fmt.Errorf("installed version %q does not contain %q", strings.TrimSpace(output), r.opts.K3sVersion)
	}
	return nil
}

func (r *Runner) verifyNodes(ctx context.Context) error {
	output, err := r.shell(ctx, "sudo -n /usr/local/bin/k3s kubectl get nodes", r.opts.VerifyTimeout)
	r.writeArtifact("nodes.out", output)
	if err != nil {
		return fmt.Errorf("kubectl get nodes: %w", err)
	}
	if !strings.Contains(output, " Ready ") {
		return fmt.Errorf("no Ready node found:\n%s", output)
	}
	return nil
}

func (r *Runner) verifyKubeconfig(ctx context.Context, configPath string) error {
	stdout, stderr, _, err := r.exec(ctx, r.opts.Binary, "kubeconfig", "-c", configPath, "--api-address", "https://127.0.0.1:6443")
	r.writeArtifact("kubeconfig.yaml", stdout)
	r.writeArtifact("kubeconfig.log", stderr)
	if err != nil {
		return fmt.Errorf("k3sctl kubeconfig: %w", err)
	}
	if !strings.Contains(stdout, "server: https://127.0.0.1:6443") {
		return fmt.Errorf("kubeconfig does not point at expected API address:\n%s", stdout)
	}

	// Prove the generated kubeconfig actually works.
	kubeconfigPath := filepath.Join(r.opts.ArtifactDir, "kubeconfig.yaml")
	nodesOut, err := r.shell(ctx, fmt.Sprintf("sudo -n /usr/local/bin/k3s kubectl --kubeconfig=%s get nodes", kubeconfigPath), r.opts.VerifyTimeout)
	r.writeArtifact("kubeconfig-nodes.out", nodesOut)
	if err != nil {
		return fmt.Errorf("kubeconfig could not reach the cluster: %w", err)
	}
	if !strings.Contains(nodesOut, " Ready ") {
		return fmt.Errorf("kubeconfig kubectl get nodes has no Ready node:\n%s", nodesOut)
	}
	return nil
}

func (r *Runner) cleanup(ctx context.Context, configPath string) error {
	var errs []error

	if r.applied && configPath != "" {
		// Clear a stale host lock in case apply left one behind.
		_, _ = r.shell(ctx, "sudo -n rm -f /var/lock/k3sctl.lock", r.opts.VerifyTimeout)

		stdout, stderr, _, err := r.exec(ctx, r.opts.Binary, "reset", "-c", configPath, "--force")
		r.writeArtifact("reset.log", fmt.Sprintf("--- stdout ---\n%s--- stderr ---\n%s", stdout, stderr))
		if err != nil {
			errs = append(errs, fmt.Errorf("k3sctl reset: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (r *Runner) verifyTeardown(ctx context.Context, configPath string) error {
	checks := []struct {
		name string
		path string
	}{
		{"k3s binary", "/usr/local/bin/k3s"},
		{"k3s data dir", "/var/lib/rancher/k3s"},
		{"k3s config", "/etc/rancher/k3s/config.yaml"},
		{"k3s service unit", "/etc/systemd/system/k3s.service"},
	}
	for _, c := range checks {
		if _, err := os.Stat(c.path); err == nil {
			return fmt.Errorf("%s still present at %s", c.name, c.path)
		}
	}

	if output, err := r.shell(ctx, "systemctl is-active --quiet k3s 2>/dev/null; echo $?", r.opts.VerifyTimeout); err == nil {
		if strings.TrimSpace(output) == "0" {
			return errors.New("k3s.service is still active after reset")
		}
	}

	return nil
}

func (r *Runner) collectDiagnostics(ctx context.Context) {
	log.Warn("collecting failure diagnostics")
	var b strings.Builder

	commands := []struct {
		label string
		cmd   string
	}{
		{"systemctl status k3s", "systemctl status k3s --no-pager"},
		{"journalctl -u k3s", "sudo -n journalctl -u k3s --no-pager -n 200"},
		{"/etc/rancher/k3s", "sudo -n ls -la /etc/rancher/k3s"},
		{"/var/lib/rancher/k3s", "sudo -n ls -la /var/lib/rancher/k3s"},
	}
	for _, c := range commands {
		fmt.Fprintf(&b, "=== %s ===\n", c.label)
		output, _ := r.shell(ctx, c.cmd, r.opts.VerifyTimeout)
		b.WriteString(output)
		b.WriteString("\n")
	}
	r.writeArtifact("diagnostics.log", b.String())
}

func (r *Runner) shell(ctx context.Context, command string, timeout time.Duration) (string, error) {
	shellCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(shellCtx, "sh", "-c", command)
	cmd.Dir = r.opts.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("command %q: %w\nstderr: %s", command, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func (r *Runner) exec(ctx context.Context, name string, args ...string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = r.opts.WorkDir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		return stdout, stderr, exitCode, runErr
	}
	return stdout, stderr, 0, nil
}

func (r *Runner) writeArtifact(name, content string) {
	path := filepath.Join(r.opts.ArtifactDir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		log.Warnf("write artifact %s: %v", name, err)
	}
}
