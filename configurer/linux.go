package configurer

import (
	"fmt"
	"strings"
	"sync"
)

type Linux struct {
	paths    map[string]string
	pathMu   sync.RWMutex
	pathOnce sync.Once
}

func (l *Linux) initPaths() {
	l.pathOnce.Do(func() {
		l.paths = map[string]string{
			"K3sBinaryPath":      "/usr/local/bin/k3s",
			"K3sConfigPath":      "/etc/rancher/k3s/config.yaml",
			"K3sTokenPath":       "/var/lib/rancher/k3s/server/node-token",
			"DataDirDefaultPath": "/var/lib/rancher/k3s",
		}
	})
}

func (l *Linux) path(key string) string {
	l.initPaths()
	l.pathMu.RLock()
	defer l.pathMu.RUnlock()
	return l.paths[key]
}

func (l *Linux) Kind() string {
	return "linux"
}

func (l *Linux) OSKind() string {
	return "linux"
}

func (l *Linux) K3sBinaryPath() string {
	return l.path("K3sBinaryPath")
}

func (l *Linux) K3sConfigPath() string {
	return l.path("K3sConfigPath")
}

func (l *Linux) K3sTokenPath() string {
	return l.path("K3sTokenPath")
}

func (l *Linux) DataDirDefaultPath() string {
	return l.path("DataDirDefaultPath")
}

func (l *Linux) K3sDataDirDefaultPath() string {
	return l.path("DataDirDefaultPath")
}

func (l *Linux) K3sConfigDirPath() string {
	return "/etc/rancher/k3s"
}

func (l *Linux) SetPath(key, value string) {
	l.initPaths()
	l.pathMu.Lock()
	defer l.pathMu.Unlock()
	l.paths[key] = value
}

func (l *Linux) K3sCmdf(template string, args ...interface{}) string {
	return fmt.Sprintf("%s %s", l.K3sBinaryPath(), fmt.Sprintf(template, args...))
}

func (l *Linux) KubeconfigPath(_ Host, dataDir string) string {
	if dataDir == "" {
		dataDir = l.DataDirDefaultPath()
	}
	return "/etc/rancher/k3s/k3s.yaml"
}

func (l *Linux) KubectlCmdf(h Host, dataDir, s string, args ...interface{}) string {
	kubecfg := l.KubeconfigPath(h, dataDir)
	formatted := fmt.Sprintf(s, args...)
	if !strings.Contains(formatted, "--kubeconfig") {
		formatted = formatted + " --kubeconfig=" + kubecfg
	}
	return l.K3sCmdf("kubectl %s", formatted)
}

func (l *Linux) PrivateInterface(h Host) (string, error) {
	output, err := h.ExecOutput("ip -o route show scope link | head -1 | sed -e 's/.* dev //' -e 's/ .*//'")
	if err != nil {
		return "", fmt.Errorf("failed to detect private interface: %w", err)
	}
	iface := strings.TrimSpace(output)
	if iface == "" {
		return "", fmt.Errorf("no private interface found, define host privateInterface manually")
	}
	return iface, nil
}

func (l *Linux) PrivateAddress(h Host, iface, publicIP string) (string, error) {
	output, err := h.ExecOutput(fmt.Sprintf("ip -o addr show dev %s scope global | awk '{print $4}' | cut -d/ -f1", iface))
	if err != nil {
		return "", fmt.Errorf("failed to get IP for interface %s: %w", iface, err)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		ip := strings.TrimSpace(line)
		if ip != "" && ip != publicIP {
			return ip, nil
		}
	}
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		return strings.TrimSpace(lines[0]), nil
	}
	return "", fmt.Errorf("no private address found for interface %s", iface)
}

func (l *Linux) UpdateEnvironment(h Host, env map[string]string) error {
	for k, v := range env {
		cmd := fmt.Sprintf("grep -qxF '%s=%s' /etc/environment 2>/dev/null || echo '%s=%s' >> /etc/environment", k, v, k, v)
		if err := h.Exec(cmd); err != nil {
			return fmt.Errorf("failed to update environment %s: %w", k, err)
		}
	}
	return nil
}

func (l *Linux) FixContainer(h Host) error {
	return h.Exec("mount --make-rshared / 2>/dev/null || true")
}

func (l *Linux) K3sctlLockFilePath() string {
	return "/var/lock/k3sctl.lock"
}
