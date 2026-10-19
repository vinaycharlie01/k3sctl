# Roadmap: Configurer Layer

The configurer layer provides OS-specific behavior. Each OS variant registers itself
and the system resolves the correct configurer based on detected OS.

---

## Architecture

```
configurer/
├── interface.go          # Configurer interface definition
├── host.go               # ConfigurerHost wrapper type
├── registry.go           # RegisterOSModule / ResolveOSModule
├── linux.go              # Default Linux configurer (base implementation)
├── windows.go            # Windows configurer
├── windows_test.go
├── linux/                # Linux distro-specific overrides
│   ├── alpine.go
│   ├── alpine_test.go
│   ├── archlinux.go
│   ├── coreos.go
│   ├── debian.go
│   ├── enterpriselinux/
│   │   ├── el.go         # Base RHEL-like (Amazon, CentOS, Fedora, RHEL, Rocky, Oracle)
│   │   ├── amazon.go
│   │   ├── centos.go
│   │   ├── fedora.go
│   │   ├── oracle.go
│   │   ├── rhel.go
│   │   ├── rocky.go
│   │   └── el_test.go
│   ├── flatcar.go
│   ├── opensuse.go
│   ├── sles.go
│   ├── ubuntu.go
│   └── linux_test.go
└── windows/
    ├── windows.go
    └── windows_test.go
```

---

## configurer/interface.go

### k0sctl interface methods
```go
type Configurer interface {
    Kind() string                                    // "linux", "windows"
    OSKind() string                                  // "linux", "windows"
    K0sCmdf(cmd string, args ...any) string          // Format k0s CLI command
    K0sBinaryPath() string                           // /usr/local/bin/k0s
    K0sConfigPath() string                           // /etc/k0s/k0s.yaml
    K0sDataDir() string                              // /var/lib/k0s
    K0sJoinTokenPath() string                        // temp token file path
    K0sStatusSocketPath() string                     // status socket
    KubectlCmdf(h KubectlNode, dataDir, cmd string, args ...any) string
    ServiceName(role string) string                  // k0scontroller, k0sworker
    ServiceScriptPath(role string) string            // systemd/init.d path
    PrivateInterface(h Host) (string, error)         // Detect private network
    PrivateAddress(h Host, iface string) (string, error)
    LockPath() string                                // /var/lock/k0sctl.lock
    TempDir(h Host) (string, error)                  // temp directory
    Arch(h Host) (string, error)                     // architecture detection
    DownloadURL(h Host, url, dest string) error      // download via curl/wget
    InstallPackage(h Host, packages ...string) error // apt/yum/apk
    FileContains(h Host, path, text string) (bool, error)
    UpdateEnvironment(h Host, env map[string]string) error
    CleanupStaleFiles(h Host, dir, prefix string) error
    TouchFile(h Host, path string, modTime time.Time) error
    DeleteFile(h Host, path string) error
    SetFileMode(h Host, path string, mode fs.FileMode) error
}
```

### k3sctl interface — adapted methods
```go
type Configurer interface {
    Kind() string
    OSKind() string
    K3sCmdf(cmd string, args ...any) string          // Format k3s CLI command
    K3sBinaryPath() string                           // /usr/local/bin/k3s
    K3sConfigPath() string                           // /etc/rancher/k3s/config.yaml
    K3sDataDir() string                              // /var/lib/rancher/k3s
    K3sTokenPath() string                            // /var/lib/rancher/k3s/server/node-token
    K3sKubeconfigPath() string                       // /etc/rancher/k3s/k3s.yaml
    K3sEnvFilePath(role string) string               // /etc/systemd/system/k3s.service.env
    K3sServiceFilePath(role string) string            // /etc/systemd/system/k3s.service
    KubectlCmdf(h KubectlNode, dataDir, cmd string, args ...any) string
    ServiceName(role string) string                  // k3s, k3s-agent
    PrivateInterface(h Host) (string, error)
    PrivateAddress(h Host, iface string) (string, error)
    LockPath() string                                // /var/lock/k3sctl.lock
    TempDir(h Host) (string, error)
    Arch(h Host) (string, error)
    DownloadURL(h Host, url, dest string) error
    InstallPackage(h Host, packages ...string) error
    FileContains(h Host, path, text string) (bool, error)
    UpdateEnvironment(h Host, env map[string]string) error
    CleanupStaleFiles(h Host, dir, prefix string) error
    TouchFile(h Host, path string, modTime time.Time) error
    DeleteFile(h Host, path string) error
    SetFileMode(h Host, path string, mode fs.FileMode) error
}
```

### Key path differences
| Method | k0sctl | k3sctl |
|--------|--------|--------|
| BinaryPath | `/usr/local/bin/k0s` | `/usr/local/bin/k3s` |
| ConfigPath | `/etc/k0s/k0s.yaml` | `/etc/rancher/k3s/config.yaml` |
| DataDir | `/var/lib/k0s` | `/var/lib/rancher/k3s` |
| TokenPath | N/A (generated) | `/var/lib/rancher/k3s/server/node-token` |
| KubeconfigPath | N/A (via command) | `/etc/rancher/k3s/k3s.yaml` |
| LockPath | `/var/lock/k0sctl.lock` | `/var/lock/k3sctl.lock` |
| ServiceName(server) | `k0scontroller` | `k3s` |
| ServiceName(agent) | `k0sworker` | `k3s-agent` |

---

## configurer/registry.go

### k0sctl behavior
- `RegisterOSModule(factory func() Configurer)` — registers configurer factories
- `ResolveOSModule(h Host) (Configurer, error)` — matches OS ID from /etc/os-release
- Falls back to ID_LIKE entries

### k3sctl adaptation
- Port as-is. Generic registration/resolution pattern.
- Each distro module calls `RegisterOSModule()` in its `init()` function

---

## configurer/linux.go

### k0sctl behavior
- Default implementation for all `Configurer` methods on Linux
- Uses curl for downloads, systemd for services
- Implements PrivateInterface via `ip route` parsing
- DownloadURL uses `curl -sSLf -o <dest> <url>`

### k3sctl adaptation
- Change all K0s paths to K3s paths
- `K3sCmdf`: format `k3s <subcommand>` strings
- `KubectlCmdf`: use `k3s kubectl` instead of `k0s kubectl`
- `ServiceName`: "k3s" for server, "k3s-agent" for agent
- `K3sEnvFilePath`: return env file path per role
- `K3sServiceFilePath`: return systemd unit path per role
- Keep PrivateInterface, PrivateAddress, DownloadURL as-is

### DownloadURL implementation note
The `DownloadURL` method executes `curl` on the remote host. This is acceptable because:
- curl is an external system utility, not application logic
- It's the standard way to download files on Linux systems
- The configurer invokes it as an external executable via rig exec

---

## configurer/linux/ distro modules

Each module overrides specific methods for its distribution:

### alpine.go
- Package install: `apk add --update`
- No systemd (uses OpenRC)
- **k3sctl**: K3s on Alpine uses OpenRC service scripts instead of systemd. Need to generate
  OpenRC init scripts instead of systemd units.

### debian.go / ubuntu.go
- Package install: `apt-get install -y`
- Uses systemd
- **k3sctl**: Port with K3s paths

### enterpriselinux/ (RHEL, CentOS, Rocky, Fedora, Amazon, Oracle)
- Package install: `yum install -y` or `dnf install -y`
- Uses systemd
- Firewall: may need firewalld rules for K3s ports
- SELinux: K3s has SELinux support via `k3s-selinux` package
- **k3sctl**: Port with K3s paths, consider SELinux policy

### archlinux.go
- Package install: `pacman -Sy --noconfirm`
- **k3sctl**: Port with K3s paths

### coreos.go / flatcar.go
- Immutable OS, limited package management
- **k3sctl**: K3s works on these via their container-optimized approach

### opensuse.go / sles.go
- Package install: `zypper install -y`
- **k3sctl**: Port with K3s paths

---

## configurer/windows.go

### k0sctl behavior
- Windows-specific paths and commands
- Uses PowerShell for operations
- Windows service management

### k3sctl adaptation
- K3s has experimental Windows agent support
- Windows paths: `C:\var\lib\rancher\k3s\`
- Service: Windows service via `sc.exe` or NSSM
- Binary: `k3s.exe`
- **Note**: Windows support is lower priority and can be deferred

---

## Implementation Order

1. `configurer/interface.go` — Define K3s configurer interface
2. `configurer/registry.go` — Port registry (direct)
3. `configurer/linux.go` — Default Linux with K3s paths
4. `configurer/linux/debian.go` + `ubuntu.go` — Most common targets
5. `configurer/linux/enterpriselinux/` — RHEL family
6. `configurer/linux/alpine.go` — OpenRC support
7. `configurer/linux/archlinux.go` + `opensuse.go` + `sles.go`
8. `configurer/linux/coreos.go` + `flatcar.go`
9. `configurer/windows.go` — Deferred (experimental in K3s)
