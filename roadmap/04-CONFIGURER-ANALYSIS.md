# k3sctl Roadmap — configurer/ Analysis

## Architecture

The configurer system provides OS-specific behavior through a registry pattern:
- `configurer/interface.go` — defines the Configurer interface
- `configurer/registry.go` — registry of OS configurers, resolution by OS info
- `configurer/host.go` — HostConfigurer base with common methods
- `configurer/linux.go` — Linux base configurer
- `configurer/windows.go` — Windows base configurer
- `configurer/linux/*.go` — Distribution-specific configurers

## configurer/interface.go — Configurer Interface

Defines methods that OS-specific configurers must implement:

**k0sctl interface methods** (to adapt for k3sctl):

```go
type Configurer interface {
    K0sCmdf(cmd string, args ...any) string           // → K3sCmdf
    K0sInstallCommand() string                          // → K3sInstallCommand
    K0sConfigPath() string                              // → K3sConfigPath
    K0sDataDir() string                                 // → K3sDataDir
    K0sBinaryPath() string                              // → K3sBinaryPath
    K0sJoinCommand(role, token string) string           // → K3sJoinCommand
    K0sStartCommand() string                            // → K3sStartCommand
    K0sStopCommand() string                             // → K3sStopCommand
    K0sServiceName(role string) string                  // → K3sServiceName
    KubectlCmdf(h, datadir, cmd string, ...) string    // → KubectlCmdf (via k3s kubectl)
    // OS-level operations (keep as-is)
    Hostname(h) (string, error)
    InstallPackage(h, pkg ...) error
    CheckPrivilege(h) error
    ServiceIsRunning(h, svc) bool
    StartService(h, svc) error
    StopService(h, svc) error
    RestartService(h, svc) error
    EnableService(h, svc) error
    DisableService(h, svc) error
    DaemonReload(h) error
    MachineID(h) (string, error)
    // ... plus cleanup, path helpers, etc.
}
```

**k3sctl adaptations**:
- All `K0s*` methods → `K3s*` equivalents
- Paths:
  - Binary: `/usr/local/bin/k0s` → `/usr/local/bin/k3s`
  - Config: `/etc/k0s/k0s.yaml` → `/etc/rancher/k3s/config.yaml`
  - Data: `/var/lib/k0s` → `/var/lib/rancher/k3s`
- Service names: `k0scontroller`/`k0sworker` → `k3s`/`k3s-agent`
- kubectl: `k0s kubectl` → `k3s kubectl`
- K3s-specific methods needed:
  - `K3sTokenPath() string` — `/var/lib/rancher/k3s/server/node-token`
  - `K3sKubeconfigPath() string` — `/etc/rancher/k3s/k3s.yaml`
  - `K3sManifestsDir() string` — `/var/lib/rancher/k3s/server/manifests/`
  - `K3sImagesDir() string` — `/var/lib/rancher/k3s/agent/images/`
  - `K3sUninstallDirs() []string` — directories to remove on reset

---

## configurer/host.go — Host Configurer Base

Provides base implementation for common operations used by all OS configurers.
Embeds rig connection information.

**k3sctl**: Adapt all K0s-specific default implementations.

---

## configurer/registry.go — Configurer Registry

Pattern: init() functions in each OS package register their configurer.
`ByHost(h)` resolves the right configurer by matching OS detection output.

**k3sctl**: Same registration pattern, different configurer implementations.

---

## configurer/linux.go — Linux Base

Provides base Linux implementation:
- systemd service management
- Package management (apt, yum, etc.)
- Path conventions
- K0s-specific paths and commands

**k3sctl MAJOR changes**:

```go
// k0sctl
func (l *Linux) K0sConfigPath() string { return "/etc/k0s/k0s.yaml" }
func (l *Linux) K0sDataDir() string { return "/var/lib/k0s" }
func (l *Linux) K0sBinaryPath() string { return "/usr/local/bin/k0s" }

// k3sctl
func (l *Linux) K3sConfigPath() string { return "/etc/rancher/k3s/config.yaml" }
func (l *Linux) K3sDataDir() string { return "/var/lib/rancher/k3s" }
func (l *Linux) K3sBinaryPath() string { return "/usr/local/bin/k3s" }
func (l *Linux) K3sTokenPath() string { return "/var/lib/rancher/k3s/server/node-token" }
func (l *Linux) K3sKubeconfigPath() string { return "/etc/rancher/k3s/k3s.yaml" }
```

K3s service management:
```go
func (l *Linux) K3sServiceName(role string) string {
    if role == "agent" { return "k3s-agent" }
    return "k3s"
}
```

K3s install command generation:
```go
func (l *Linux) K3sInstallCommand(role string, flags []string) string {
    // Generate systemd unit and install via Go, not shell script
}
```

K3s kubectl:
```go
func (l *Linux) KubectlCmdf(h *Host, dataDir, cmd string, args ...any) string {
    return fmt.Sprintf("k3s kubectl %s", fmt.Sprintf(cmd, args...))
}
```

---

## configurer/windows.go — Windows Base

Windows-specific configurer for K0s on Windows (worker-only).

**k3sctl**: K3s does NOT officially support Windows nodes in the same way as K0s.
K3s has experimental Windows agent support. Implement basic Windows agent
support if needed, or mark as unsupported initially.

**Decision**: Include Windows configurer stub, mark as experimental.

---

## Linux Distribution Configurers

### configurer/linux/alpine.go
- Alpine Linux specific: apk package manager, no systemd by default
- May use OpenRC instead of systemd
- **k3sctl**: Adapt for K3s paths, add OpenRC service management

### configurer/linux/archlinux.go
- Arch Linux: pacman package manager, systemd
- **k3sctl**: Adapt for K3s paths

### configurer/linux/coreos.go
- CoreOS/Fedora CoreOS: immutable OS, rpm-ostree
- Restricted package installation
- **k3sctl**: Adapt for K3s paths, handle immutable filesystem

### configurer/linux/debian.go
- Debian: apt package manager, systemd
- **k3sctl**: Adapt for K3s paths

### configurer/linux/ubuntu.go
- Ubuntu: apt package manager, systemd
- Extends Debian configurer
- **k3sctl**: Adapt for K3s paths

### configurer/linux/enterpriselinux/ (RHEL family)

All share EnterpriseLinux base:
- `enterpriselinux.go` — base with yum/dnf
- `almalinux.go` — AlmaLinux
- `amazon.go` — Amazon Linux
- `centos.go` — CentOS
- `fedora.go` — Fedora
- `oracle.go` — Oracle Linux
- `rhel.go` — RHEL
- `rocky.go` — Rocky Linux

**k3sctl**: All need K3s path adaptations. Enterprise Linux is the most
common K3s deployment target, so these are high priority.

### configurer/linux/flatcar.go
- Flatcar Container Linux: immutable OS
- **k3sctl**: Adapt for K3s paths, handle immutable filesystem

### configurer/linux/opensuse.go
- openSUSE: zypper package manager
- **k3sctl**: Adapt for K3s paths

### configurer/linux/sles.go
- SUSE Linux Enterprise Server
- **k3sctl**: Adapt for K3s paths (SUSE/Rancher = natural K3s target)

---

## Key K3s-Specific Configurer Methods Needed

### Systemd Unit Generation

K3s needs systemd unit files created natively in Go (no shell scripts):

**Server unit** (`/etc/systemd/system/k3s.service`):
```ini
[Unit]
Description=Lightweight Kubernetes
Documentation=https://k3s.io
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
EnvironmentFile=-/etc/default/k3s
EnvironmentFile=-/etc/sysconfig/k3s
ExecStart=/usr/local/bin/k3s server <flags>
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
```

**Agent unit** (`/etc/systemd/system/k3s-agent.service`):
```ini
[Unit]
Description=Lightweight Kubernetes Agent
Documentation=https://k3s.io
After=network-online.target
Wants=network-online.target

[Service]
Type=exec
EnvironmentFile=-/etc/default/k3s-agent
EnvironmentFile=-/etc/sysconfig/k3s-agent
ExecStart=/usr/local/bin/k3s agent <flags>
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
```

### K3s Uninstall Logic (Pure Go)

Instead of k3s-uninstall.sh:
1. Stop services: k3s, k3s-agent
2. Kill remaining K3s processes
3. Remove systemd units and reload daemon
4. Remove iptables rules (via rig commands)
5. Remove directories:
   - `/etc/rancher/k3s/`
   - `/var/lib/rancher/k3s/`
   - `/var/lib/kubelet/`
   - `/etc/cni/net.d/`
6. Remove binary: `/usr/local/bin/k3s`
7. Remove symlinks: `kubectl`, `crictl`, `ctr` if they point to k3s
