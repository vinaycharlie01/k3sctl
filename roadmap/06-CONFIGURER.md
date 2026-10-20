# Roadmap: Configurer Layer (OS-Specific Behavior)

## Architecture

The configurer layer provides OS-specific behavior through a registry pattern:
1. `configurer/interface.go` defines the `Configurer` interface
2. `configurer/registry.go` provides `RegisterOSModule` / `ResolveOSModule`
3. OS modules register via `init()` functions
4. Each module matches by OS release info and provides path/command implementations

## Source Files (k0sctl)

### configurer/interface.go
- `Configurer` interface methods:
  - `Kind() string` — OS identifier
  - `OSKind() string` — "linux" or "windows"
  - `K0sCmdf(format, args...) string` — format k0s command string
  - `K0sBinaryPath() string` — path to k0s binary
  - `K0sConfigPath() string` — path to k0s config file
  - `DataDirDefaultPath() string` — default data directory
  - `K0sJoinTokenPath() string` — path to join token file
  - `UpdateEnvironment(map[string]string) string` — update env vars
  - `KubectlCmdf(host, dataDir, format, args...) string` — format kubectl command
  - `KubeconfigPath(host, dataDir) string` — kubeconfig file path
  - `FixContainer() string` — container fix command
  - `PrivateInterface() string` — command to detect private interface
  - `PrivateAddress() string` — command to detect private address
  - `K0sctlLockFilePath() string` — lock file path
  - `SetPath(key, value)` — set configurer path
- `HostValidator` interface:
  - `ValidateHost(host, dataDir) error` — validate host for compatibility
- **k3sctl**: Rename to K3s equivalents:
  - `K3sCmdf` → format k3s command
  - `K3sBinaryPath` → `/usr/local/bin/k3s`
  - `K3sConfigPath` → `/etc/rancher/k3s/config.yaml`
  - `DataDirDefaultPath` → `/var/lib/rancher/k3s`
  - `K3sJoinTokenPath` → `/var/lib/rancher/k3s/server/node-token`
  - `K3sctlLockFilePath` → `/var/lock/k3sctl.lock`
  - `KubeconfigPath` → `/etc/rancher/k3s/k3s.yaml`

### configurer/host.go
- `Host` interface: requires `cmd.SimpleRunner`, `Sudo() *rig.Client`, `FS() remotefs.FS`
- **k3sctl**: Port as-is (rig interface, not k0s-specific)

### configurer/registry.go
- `RegisterOSModule(matcher, builder)` — register OS module with matcher function
- `ResolveOSModule(osRelease) (Configurer, error)` — resolve OS module from os-release info
- Matcher: `func(os *rig.OSRelease) bool`
- Builder: `func() Configurer`
- **k3sctl**: Port as-is

### configurer/linux.go
- `Linux` struct with paths map
- Default paths:
  - `K0sBinaryPath` = `/usr/local/bin/k0s`
  - `K0sConfigPath` = `/etc/k0s/k0s.yaml`
  - `K0sJoinTokenPath` = `/etc/k0s/k0stoken`
  - `DataDirDefaultPath` = `/var/lib/k0s`
- Methods:
  - `K0sCmdf`: formats command with binary path
  - `KubectlCmdf`: uses KUBECONFIG env var with kubectl subcommand
  - `PrivateInterface`: uses `ip route` to detect
  - `PrivateAddress`: uses `ip -o addr show` to detect
  - `UpdateEnvironment`: patches `/etc/environment`
  - `FixContainer`: runs `mount --make-rshared /`
- **k3sctl**: Port with K3s paths:
  - `K3sBinaryPath` = `/usr/local/bin/k3s`
  - `K3sConfigPath` = `/etc/rancher/k3s/config.yaml`
  - `K3sJoinTokenPath` = `/var/lib/rancher/k3s/server/node-token`
  - `DataDirDefaultPath` = `/var/lib/rancher/k3s`
  - `K3sCmdf`: format with k3s binary path
  - `KubectlCmdf`: `k3s kubectl` subcommand
  - `KubeconfigPath`: `/etc/rancher/k3s/k3s.yaml`

### configurer/windows.go
- Windows configurer base
- **k3sctl**: K3s does not support Windows nodes, so this can be omitted initially

### configurer/windows/windows.go
- Windows-specific paths and commands
- **k3sctl**: Omit (K3s is Linux-only for server/agent)

## Linux Distribution Modules

### configurer/linux/alpine.go
- Alpine Linux configurer
- Matcher: ID="alpine"
- Handles: apk package manager, OpenRC service manager
- **k3sctl**: Port with K3s paths

### configurer/linux/debian.go
- Debian configurer
- Matcher: ID="debian"
- **k3sctl**: Port with K3s paths

### configurer/linux/ubuntu.go
- Ubuntu configurer (extends Debian)
- Matcher: ID="ubuntu"
- **k3sctl**: Port with K3s paths

### configurer/linux/enterpriselinux.go
- Base for RHEL-family distros
- **k3sctl**: Port with K3s paths

### configurer/linux/enterpriselinux/rhel.go
- RHEL configurer
- **k3sctl**: Port

### configurer/linux/enterpriselinux/centos.go
- CentOS configurer
- **k3sctl**: Port

### configurer/linux/enterpriselinux/fedora.go
- Fedora configurer
- **k3sctl**: Port

### configurer/linux/enterpriselinux/rocky.go
- Rocky Linux configurer
- **k3sctl**: Port

### configurer/linux/enterpriselinux/almalinux.go
- AlmaLinux configurer
- **k3sctl**: Port

### configurer/linux/enterpriselinux/oracle.go
- Oracle Linux configurer
- **k3sctl**: Port

### configurer/linux/enterpriselinux/amazon.go
- Amazon Linux configurer
- **k3sctl**: Port

### configurer/linux/archlinux.go
- Arch Linux configurer
- **k3sctl**: Port

### configurer/linux/opensuse.go
- openSUSE configurer
- **k3sctl**: Port

### configurer/linux/sles.go
- SUSE Linux Enterprise Server configurer
- **k3sctl**: Port

### configurer/linux/flatcar.go
- Flatcar Container Linux configurer
- **k3sctl**: Port

### configurer/linux/coreos.go
- CoreOS configurer
- **k3sctl**: Port

## Test Files
- configurer/linux/alpine_test.go
- configurer/linux/linux_test.go
- configurer/windows/windows_test.go
- configurer/windows_test.go
- **k3sctl**: Port Linux tests, omit Windows tests

## Implementation Order
1. `configurer/interface.go` (K3s interface)
2. `configurer/host.go` (direct port)
3. `configurer/registry.go` (direct port)
4. `configurer/linux.go` (K3s paths)
5. Linux distro modules (one by one):
   - debian.go, ubuntu.go (most common)
   - enterpriselinux.go + rhel.go, centos.go, rocky.go, almalinux.go
   - fedora.go, amazon.go, oracle.go
   - alpine.go, archlinux.go
   - opensuse.go, sles.go
   - flatcar.go, coreos.go
6. Tests
