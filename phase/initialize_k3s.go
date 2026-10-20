package phase

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster"
	"github.com/vinaycharlie01/k3sctl/pkg/node"
	"github.com/vinaycharlie01/k3sctl/pkg/retry"
)

type InitializeK3s struct {
	GenericPhase
	leader *cluster.Host
}

func (p *InitializeK3s) Title() string {
	return "Initialize K3s cluster"
}

func (p *InitializeK3s) Prepare() {
	p.leader = p.Config.Spec.K3sLeader()
}

func (p *InitializeK3s) ShouldRun() bool {
	if p.leader == nil {
		return false
	}
	return !p.leader.Metadata.K3sServiceRunning
}

func (p *InitializeK3s) Run(ctx context.Context) error {
	h := p.leader

	log.Infof("%s: installing K3s service (server, cluster-init)", h)
	if err := p.installK3sService(h); err != nil {
		return fmt.Errorf("%s: install K3s service: %w", h, err)
	}

	log.Infof("%s: starting K3s service", h)
	svc, err := h.Sudo().Service(h.K3sServiceName())
	if err != nil {
		return fmt.Errorf("%s: get service: %w", h, err)
	}
	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("%s: start service: %w", h, err)
	}

	log.Infof("%s: waiting for K3s to be ready", h)
	if err := p.waitForReady(ctx, h); err != nil {
		return fmt.Errorf("%s: wait for ready: %w", h, err)
	}

	tokenPath := h.Configurer.K3sTokenPath()
	output, err := h.Sudo().ExecOutput(fmt.Sprintf("cat %s", shellQuote(tokenPath)))
	if err != nil {
		return fmt.Errorf("%s: read node token: %w", h, err)
	}
	h.Metadata.NodeToken = strings.TrimSpace(output)
	h.Metadata.NeedsUpgrade = false
	log.Infof("%s: K3s cluster initialized", h)

	return nil
}

func (p *InitializeK3s) installK3sService(h *cluster.Host) error {
	installFlags := h.K3sInstallFlags()

	unitContent := k3sServerUnit(h.Configurer.K3sBinaryPath(), installFlags, h.Environment)
	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", h.K3sServiceName())

	if err := h.Sudo().FS().WriteFile(unitPath, []byte(unitContent), fs.FileMode(0644)); err != nil {
		return fmt.Errorf("write service unit: %w", err)
	}

	if err := h.Sudo().Exec("systemctl daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	if err := h.Sudo().Exec(fmt.Sprintf("systemctl enable %s", h.K3sServiceName())); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}

	return nil
}

func (p *InitializeK3s) waitForReady(ctx context.Context, h *cluster.Host) error {
	return retry.Context(ctx, node.KubeNodeReadyFunc(h, h.KubernetesNodeName()),
		retry.Timeout(5*time.Minute), retry.Interval(10*time.Second))
}

func k3sServerUnit(binaryPath string, flags []string, env map[string]string) string {
	execStart := binaryPath + " server"
	for _, f := range flags {
		execStart += " " + f
	}
	return fmt.Sprintf(`[Unit]
Description=K3s - Lightweight Kubernetes
Documentation=https://k3s.io
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
%sExecStartPre=-/sbin/modprobe br_netfilter
ExecStartPre=-/sbin/modprobe overlay
ExecStart=%s
KillMode=process
Delegate=yes
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
`, systemdEnvironment(env), execStart)
}

func k3sAgentUnit(binaryPath string, flags []string, env map[string]string) string {
	execStart := binaryPath + " agent"
	for _, f := range flags {
		execStart += " " + f
	}
	return fmt.Sprintf(`[Unit]
Description=K3s Agent - Lightweight Kubernetes
Documentation=https://k3s.io
After=network-online.target
Wants=network-online.target

[Service]
Type=exec
%sExecStart=%s
KillMode=process
Delegate=yes
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
`, systemdEnvironment(env), execStart)
}

// systemdEnvironment renders the cluster's environment variables as
// Environment= directives. Keys are sorted so the generated unit is stable.
func systemdEnvironment(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "Environment=%q\n", key+"="+env[key])
	}
	return b.String()
}
