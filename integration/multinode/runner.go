package multinode

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vinaycharlie01/k3sctl/integration/environment"
)

const (
	defaultClusterName      = "k3sctl-integration"
	defaultApplyTimeout     = 15 * time.Minute
	defaultProvisionTimeout = 5 * time.Minute
	defaultVerifyTimeout    = 2 * time.Minute
	defaultResetTimeout     = 5 * time.Minute
	defaultCleanupTimeout   = 5 * time.Minute
	k3sAPIAddressFormat     = "https://%s:6443"
	sshWaitTimeout          = 3 * time.Minute
)

// Options configures the multi-node integration test runner.
type Options struct {
	// Binary is the compiled k3sctl binary under test.
	Binary string
	// WorkDir holds the generated SSH key and is removed during cleanup.
	WorkDir string
	// KeyPath is the private SSH key. It must live in WorkDir.
	KeyPath string
	// ArtifactDir receives logs and the generated manifest.
	ArtifactDir string
	// ClusterName is written to the generated manifest.
	ClusterName string
	// K3sVersion is installed by k3sctl.
	K3sVersion string
	// Environment provisions the nodes.
	Environment environment.Environment
	// Remote runs commands on provisioned nodes.
	Remote RemoteExecutor
	// Commands runs the k3sctl binary.
	Commands CommandRunner
	// APICheck probes the Kubernetes API. It defaults to CheckAPIServer and is
	// overridable in tests so the runner does not need a reachable cluster.
	APICheck func(ctx context.Context, address string) error

	ApplyTimeout     time.Duration
	ProvisionTimeout time.Duration
	VerifyTimeout    time.Duration
	ResetTimeout     time.Duration
	CleanupTimeout   time.Duration
}

// Runner orchestrates provisioning, installation and verification. It is not
// safe for concurrent use.
type Runner struct {
	options Options
	applied bool
}

// NewRunner validates options and returns a runner.
func NewRunner(options Options) (*Runner, error) {
	switch {
	case options.Binary == "":
		return nil, errors.New("k3sctl binary path is required")
	case options.WorkDir == "":
		return nil, errors.New("work directory is required")
	case options.KeyPath == "":
		return nil, errors.New("SSH key path is required")
	case options.ArtifactDir == "":
		return nil, errors.New("artifact directory is required")
	case options.Environment == nil:
		return nil, errors.New("environment is required")
	case options.Remote == nil:
		return nil, errors.New("remote executor is required")
	}

	if options.Commands == nil {
		options.Commands = ExecRunner{}
	}
	if options.APICheck == nil {
		options.APICheck = CheckAPIServer
	}
	if options.ClusterName == "" {
		options.ClusterName = defaultClusterName
	}
	if options.ApplyTimeout == 0 {
		options.ApplyTimeout = defaultApplyTimeout
	}
	if options.ProvisionTimeout == 0 {
		options.ProvisionTimeout = defaultProvisionTimeout
	}
	if options.VerifyTimeout == 0 {
		options.VerifyTimeout = defaultVerifyTimeout
	}
	if options.ResetTimeout == 0 {
		options.ResetTimeout = defaultResetTimeout
	}
	if options.CleanupTimeout == 0 {
		options.CleanupTimeout = defaultCleanupTimeout
	}

	return &Runner{options: options}, nil
}

// Run provisions the environment, installs K3s with the real CLI and verifies
// the resulting cluster. Cleanup always runs, even when a step fails.
func (r *Runner) Run(ctx context.Context) (runErr error) {
	if err := os.MkdirAll(r.options.WorkDir, 0700); err != nil {
		return fmt.Errorf("create work directory: %w", err)
	}
	if err := os.MkdirAll(r.options.ArtifactDir, 0700); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}

	r.applied = false
	var nodes []environment.Node
	var configPath string

	defer func() {
		// Detach from the test context so cleanup still runs after a failure
		// or timeout.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.options.CleanupTimeout)
		defer cancel()

		if runErr != nil {
			r.collectDiagnostics(cleanupCtx, nodes)
		}
		if err := r.cleanup(cleanupCtx, configPath); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("cleanup: %w", err))
		}
	}()

	log.Info("creating nodes")
	var err error
	nodes, err = r.options.Environment.Create(ctx)
	if err != nil {
		return fmt.Errorf("create environment: %w", err)
	}
	log.Infof("created %d nodes", len(nodes))

	log.Info("bootstrapping nodes")
	if err := r.options.Environment.Bootstrap(ctx, nodes); err != nil {
		return fmt.Errorf("bootstrap environment: %w", err)
	}

	if err := r.prepareNodes(ctx, nodes); err != nil {
		return err
	}
	if err := r.verifyConnectivity(ctx, nodes); err != nil {
		return err
	}

	configPath, err = r.writeConfig(nodes)
	if err != nil {
		return err
	}

	if err := r.apply(ctx, configPath); err != nil {
		return err
	}
	if err := r.verify(ctx, nodes, configPath); err != nil {
		return err
	}

	log.Info("multi-node K3s integration test passed")
	return nil
}

func (r *Runner) writeConfig(nodes []environment.Node) (string, error) {
	data, err := ClusterConfig{
		Name:       r.options.ClusterName,
		K3sVersion: r.options.K3sVersion,
		KeyPath:    r.options.KeyPath,
		Nodes:      nodes,
	}.Generate()
	if err != nil {
		return "", fmt.Errorf("generate cluster config: %w", err)
	}

	path := filepath.Join(r.options.ArtifactDir, "k3sctl.yaml")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("write cluster config: %w", err)
	}
	log.Infof("wrote cluster config to %s", path)
	return path, nil
}

func (r *Runner) prepareNodes(ctx context.Context, nodes []environment.Node) error {
	for _, node := range nodes {
		if err := r.waitForSSH(ctx, node); err != nil {
			return err
		}
		if err := r.provisionNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) waitForSSH(ctx context.Context, node environment.Node) error {
	waitCtx, cancel := context.WithTimeout(ctx, sshWaitTimeout)
	defer cancel()

	for {
		probeCtx, probeCancel := context.WithTimeout(waitCtx, 10*time.Second)
		_, err := r.options.Remote.Run(probeCtx, node.Address, "true")
		probeCancel()
		if err == nil {
			return nil
		}

		select {
		case <-waitCtx.Done():
			return fmt.Errorf("wait for SSH on %s: %w", node.Name, waitCtx.Err())
		case <-time.After(time.Second):
		}
	}
}

func (r *Runner) provisionNode(ctx context.Context, node environment.Node) error {
	if _, err := r.remote(ctx, node.Address,
		"sudo apt-get update -qq && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl iputils-ping >/dev/null 2>&1",
		r.options.ProvisionTimeout); err != nil {
		return fmt.Errorf("install prerequisites on %s: %w", node.Name, err)
	}

	if _, err := r.remote(ctx, node.Address, "sudo -n true", r.options.VerifyTimeout); err != nil {
		return fmt.Errorf("passwordless sudo unavailable on %s: %w", node.Name, err)
	}

	result, err := r.remote(ctx, node.Address, "ps -p 1 -o comm=", r.options.VerifyTimeout)
	if err != nil {
		return fmt.Errorf("inspect PID 1 on %s: %w", node.Name, err)
	}
	if pid1 := strings.TrimSpace(result.Stdout); pid1 != "systemd" {
		return fmt.Errorf("PID 1 on %s is %q, want systemd", node.Name, pid1)
	}

	if _, err := r.remote(ctx, node.Address, "curl -sf --max-time 10 https://get.k3s.io >/dev/null", r.options.VerifyTimeout); err != nil {
		return fmt.Errorf("no internet access on %s: %w", node.Name, err)
	}

	return nil
}

func (r *Runner) verifyConnectivity(ctx context.Context, nodes []environment.Node) error {
	server := findRole(nodes, environment.RoleServer)
	if server == nil {
		return errors.New("no server node found")
	}
	if net.ParseIP(server.Address) == nil {
		return fmt.Errorf("server address %q is not an IP address", server.Address)
	}

	for _, node := range nodes {
		if node.Role != environment.RoleAgent {
			continue
		}
		command := fmt.Sprintf("ping -c 1 -W 3 %s >/dev/null", server.Address)
		if _, err := r.remote(ctx, node.Address, command, r.options.VerifyTimeout); err != nil {
			return fmt.Errorf("agent %s cannot reach server %s: %w", node.Name, server.Address, err)
		}
	}
	return nil
}

func (r *Runner) apply(ctx context.Context, configPath string) error {
	// Mark the run as applied before invoking k3sctl so a partially installed
	// cluster is still reset during cleanup.
	r.applied = true

	applyCtx, cancel := context.WithTimeout(ctx, r.options.ApplyTimeout)
	defer cancel()

	result, err := r.options.Commands.Run(applyCtx, r.options.WorkDir, r.options.Binary, "--debug", "apply", "-c", configPath)
	r.writeArtifact("apply.log", formatResult(result, err))
	if err != nil {
		return fmt.Errorf("k3sctl apply: %w", err)
	}
	return nil
}

func (r *Runner) verify(ctx context.Context, nodes []environment.Node, configPath string) error {
	server := findRole(nodes, environment.RoleServer)
	if server == nil {
		return errors.New("no server node found")
	}

	if err := r.verifyStatus(ctx, nodes, configPath); err != nil {
		return err
	}
	if err := r.verifyServices(ctx, nodes); err != nil {
		return err
	}
	if err := r.verifyVersion(ctx, server); err != nil {
		return err
	}
	if err := r.verifyAPIServer(ctx, server); err != nil {
		return err
	}
	if err := r.verifyReadyNodes(ctx, nodes, server); err != nil {
		return err
	}

	servers := findAllRole(nodes, environment.RoleServer)
	if len(servers) > 1 {
		if err := r.verifyEtcdCluster(ctx, server, len(servers)); err != nil {
			return err
		}
		if err := r.verifyAllServerAPIs(ctx, servers); err != nil {
			return err
		}
	}

	return r.verifyKubeconfig(ctx, configPath, server)
}

func (r *Runner) verifyStatus(ctx context.Context, nodes []environment.Node, configPath string) error {
	verifyCtx, cancel := context.WithTimeout(ctx, r.options.VerifyTimeout)
	defer cancel()

	result, err := r.options.Commands.Run(verifyCtx, r.options.WorkDir, r.options.Binary, "config", "status", "-c", configPath)
	r.writeArtifact("status.log", formatResult(result, err))
	if err != nil {
		return fmt.Errorf("k3sctl config status: %w", err)
	}
	return VerifyStatus(result.Stdout, nodes)
}

func (r *Runner) verifyServices(ctx context.Context, nodes []environment.Node) error {
	for _, node := range nodes {
		service := "k3s-agent"
		if node.Role == environment.RoleServer {
			service = "k3s"
		}
		if _, err := r.remote(ctx, node.Address, "systemctl is-active --quiet "+service, r.options.VerifyTimeout); err != nil {
			return fmt.Errorf("%s is not active on %s: %w", service, node.Name, err)
		}
	}
	return nil
}

func (r *Runner) verifyVersion(ctx context.Context, server *environment.Node) error {
	result, err := r.remote(ctx, server.Address, "sudo /usr/local/bin/k3s --version", r.options.VerifyTimeout)
	r.writeArtifact("k3s-version.log", result.Stdout+result.Stderr)
	if err != nil {
		return fmt.Errorf("read K3s version on %s: %w", server.Name, err)
	}
	return VerifyVersion(result.Stdout, r.options.K3sVersion)
}

func (r *Runner) verifyAPIServer(ctx context.Context, server *environment.Node) error {
	apiCtx, cancel := context.WithTimeout(ctx, r.options.VerifyTimeout)
	defer cancel()

	if err := r.options.APICheck(apiCtx, server.Address); err != nil {
		return fmt.Errorf("kubernetes API on %s: %w", server.Name, err)
	}
	return nil
}

func (r *Runner) verifyReadyNodes(ctx context.Context, nodes []environment.Node, server *environment.Node) error {
	result, err := r.remote(ctx, server.Address, "sudo /usr/local/bin/k3s kubectl get nodes -o json", r.options.VerifyTimeout)
	r.writeArtifact("nodes.json", result.Stdout)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}

	expected := make([]string, 0, len(nodes))
	for _, node := range nodes {
		expected = append(expected, node.Name)
	}
	return VerifyNodes(result.Stdout, expected)
}

func (r *Runner) verifyKubeconfig(ctx context.Context, configPath string, server *environment.Node) error {
	apiAddress := fmt.Sprintf(k3sAPIAddressFormat, server.Address)

	verifyCtx, cancel := context.WithTimeout(ctx, r.options.VerifyTimeout)
	defer cancel()

	result, err := r.options.Commands.Run(verifyCtx, r.options.WorkDir, r.options.Binary, "kubeconfig", "-c", configPath, "--api-address", apiAddress)
	r.writeArtifact("kubeconfig.yaml", result.Stdout)
	r.writeArtifact("kubeconfig.log", result.Stderr)
	if err != nil {
		return fmt.Errorf("k3sctl kubeconfig: %w", err)
	}
	return VerifyKubeconfig(result.Stdout, apiAddress)
}

func (r *Runner) verifyEtcdCluster(ctx context.Context, leader *environment.Node, expectedMembers int) error {
	log.Infof("verifying etcd cluster health (%d expected members)", expectedMembers)

	result, err := r.remote(ctx, leader.Address,
		"sudo sh -c 'ETCDCTL_API=3 /var/lib/rancher/k3s/data/*/bin/etcdctl "+
			"--endpoints https://127.0.0.1:2379 "+
			"--cacert /var/lib/rancher/k3s/server/tls/etcd/server-ca.crt "+
			"--cert /var/lib/rancher/k3s/server/tls/etcd/server-client.crt "+
			"--key /var/lib/rancher/k3s/server/tls/etcd/server-client.key "+
			"member list'",
		r.options.VerifyTimeout)
	r.writeArtifact("etcd-members.log", result.Stdout+"\n"+result.Stderr)
	if err != nil {
		return fmt.Errorf("etcd member list on %s: %w", leader.Name, err)
	}

	return VerifyEtcdMembers(result.Stdout, expectedMembers)
}

func (r *Runner) verifyAllServerAPIs(ctx context.Context, servers []*environment.Node) error {
	log.Infof("verifying API server on all %d servers", len(servers))

	for _, server := range servers {
		if err := r.verifyAPIServer(ctx, server); err != nil {
			return fmt.Errorf("API server on %s: %w", server.Name, err)
		}
	}
	return nil
}

func (r *Runner) cleanup(ctx context.Context, configPath string) error {
	var errs []error

	if r.applied && configPath != "" {
		resetCtx, cancel := context.WithTimeout(ctx, r.options.ResetTimeout)
		result, err := r.options.Commands.Run(resetCtx, r.options.WorkDir, r.options.Binary, "reset", "-c", configPath, "--force")
		cancel()
		r.writeArtifact("reset.log", formatResult(result, err))
		if err != nil {
			errs = append(errs, fmt.Errorf("k3sctl reset: %w", err))
		}
	}

	if err := r.options.Environment.Destroy(ctx); err != nil {
		errs = append(errs, fmt.Errorf("destroy environment: %w", err))
	}

	if err := os.RemoveAll(r.options.WorkDir); err != nil {
		errs = append(errs, fmt.Errorf("remove work directory: %w", err))
	}

	return errors.Join(errs...)
}

func (r *Runner) collectDiagnostics(ctx context.Context, nodes []environment.Node) {
	log.Warn("collecting failure diagnostics")

	targets := make([]DiagnosticTarget, 0, len(nodes))
	for _, node := range nodes {
		if node.Address == "" {
			continue
		}
		service := "k3s-agent"
		if node.Role == environment.RoleServer {
			service = "k3s"
		}
		targets = append(targets, DiagnosticTarget{Name: node.Name, Address: node.Address, Service: service})
	}

	var platform func(context.Context) (string, error)
	if provider, ok := r.options.Environment.(environment.DiagnosticProvider); ok {
		platform = func(ctx context.Context) (string, error) {
			return provider.Diagnostics(ctx, nodes)
		}
	}

	r.writeArtifact("diagnostics.log", CollectDiagnostics(ctx, r.options.Remote, targets, platform))
}

func (r *Runner) remote(ctx context.Context, address, command string, timeout time.Duration) (Result, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return r.options.Remote.Run(commandCtx, address, command)
}

func (r *Runner) writeArtifact(name, content string) {
	path := filepath.Join(r.options.ArtifactDir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		log.Warnf("write artifact %s: %v", name, err)
	}
}

func formatResult(result Result, err error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "args: %s\n", strings.Join(result.Args, " "))
	fmt.Fprintf(&b, "exit code: %d\n", result.ExitCode)
	fmt.Fprintf(&b, "duration: %s\n", result.Duration)
	b.WriteString("--- stdout ---\n")
	b.WriteString(ensureTrailingNewline(result.Stdout))
	b.WriteString("--- stderr ---\n")
	b.WriteString(ensureTrailingNewline(result.Stderr))
	if err != nil {
		fmt.Fprintf(&b, "--- error ---\n%v\n", err)
	}
	return b.String()
}

func findRole(nodes []environment.Node, role environment.Role) *environment.Node {
	for i := range nodes {
		if nodes[i].Role == role {
			return &nodes[i]
		}
	}
	return nil
}

func findAllRole(nodes []environment.Node, role environment.Role) []*environment.Node {
	var result []*environment.Node
	for i := range nodes {
		if nodes[i].Role == role {
			result = append(result, &nodes[i])
		}
	}
	return result
}
