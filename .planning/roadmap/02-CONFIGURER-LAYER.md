# 02: Configurer Layer — OS Abstraction

## Overview

The configurer layer provides OS-specific command builders and path resolvers. It uses a registry pattern where each OS module registers itself via `init()` with a matcher function.

## Files to Create

### `configurer/interface.go`
- **Source**: `k0sctl/configurer/interface.go`
- **Changes**: Rename all `K0s*` methods to `K3s*`:
  - `K0sCmdf` → `K3sCmdf` — builds `k3s <cmd>` commands
  - `K0sBinaryPath` → `K3sBinaryPath` — returns `/usr/local/bin/k3s`
  - `K0sConfigPath` → `K3sConfigPath` — returns `/etc/rancher/k3s/config.yaml`
  - `K0sJoinTokenPath` → `K3sJoinTokenPath` — returns `/etc/rancher/k3s/token`
  - `DataDirDefaultPath` — returns `/var/lib/rancher/k3s`
  - `K0sctlLockFilePath` → `K3sctlLockFilePath` — returns `/run/lock/k3sctl`
  - `KubectlCmdf` — uses `k3s kubectl` instead of `k0s kubectl`
  - `KubeconfigPath` — returns `/etc/rancher/k3s/k3s.yaml`
  - Keep: `Kind()`, `OSKind()`, `UpdateEnvironment()`, `FixContainer()`, `PrivateInterface()`, `PrivateAddress()`, `SetPath()`
- Also includes `HostValidator` interface with `ValidateHost(Host) error`

### `configurer/host.go`
- **Source**: `k0sctl/configurer/host.go`
- **Changes**: None — same interface wrapping rig's SimpleRunner, Sudo, FS

### `configurer/registry.go`
- **Source**: `k0sctl/configurer/registry.go`
- **Changes**: None — same `RegisterOSModule`/`ResolveOSModule` pattern with mutex-protected registry

### `configurer/linux.go`
- **Source**: `k0sctl/configurer/linux.go` (~200 lines)
- **This is the most heavily modified file**. All K0s paths become K3s paths:

| Method | k0sctl value | k3sctl value |
|--------|-------------|-------------|
| `Kind()` | `"linux"` | `"linux"` (same) |
| `OSKind()` | `"linux"` | `"linux"` (same) |
| `K3sBinaryPath()` | `/usr/local/bin/k0s` | `/usr/local/bin/k3s` |
| `K3sConfigPath()` | `/etc/k0s/k0s.yaml` | `/etc/rancher/k3s/config.yaml` |
| `DataDirDefaultPath()` | `/var/lib/k0s` | `/var/lib/rancher/k3s` |
| `K3sJoinTokenPath()` | `/etc/k0s/k0stoken` | `/etc/rancher/k3s/token` |
| `K3sctlLockFilePath()` | `/run/lock/k0sctl` | `/run/lock/k3sctl` |
| `K3sCmdf()` | `/usr/local/bin/k0s <cmd>` | `/usr/local/bin/k3s <cmd>` |
| `KubectlCmdf()` | `k0s kubectl --data-dir=...` | `k3s kubectl --data-dir=...` |
| `KubeconfigPath()` | checks pki/admin.conf | `/etc/rancher/k3s/k3s.yaml` |

- `PrivateInterface()` — unchanged (uses `ip route list`)
- `PrivateAddress()` — unchanged (uses `ip -o addr show`)
- `UpdateEnvironment()` — unchanged (patches `/etc/environment`)
- `FixContainer()` — unchanged (`mount --make-rshared /`)
- `SetPath()` — unchanged (dynamic path override)

### `configurer/windows.go` — **NOT CREATED** (K3s has no Windows support)

### Linux Distro Modules (all follow same pattern)

Each file: registers via `init()`, embeds `configurer.Linux` (directly or via a base type), optionally overrides `Prepare()` for package installation.

| File | Source | Changes |
|------|--------|---------|
| `configurer/linux/alpine.go` | Same | Rename references only |
| `configurer/linux/archlinux.go` | Same | Rename references only |
| `configurer/linux/coreos.go` | Same | Rename references only |
| `configurer/linux/debian.go` | Same | Rename references only |
| `configurer/linux/enterpriselinux.go` | Same | Base type, rename refs |
| `configurer/linux/enterpriselinux/almalinux.go` | Same | Rename references only |
| `configurer/linux/enterpriselinux/amazon.go` | Same | Rename references only |
| `configurer/linux/enterpriselinux/centos.go` | Same | Rename references only |
| `configurer/linux/enterpriselinux/fedora.go` | Same | Rename references only |
| `configurer/linux/enterpriselinux/oracle.go` | Same | Rename references only |
| `configurer/linux/enterpriselinux/rhel.go` | Same | Rename references only |
| `configurer/linux/enterpriselinux/rocky.go` | Same | Rename references only |
| `configurer/linux/flatcar.go` | Same | Override `K3sBinaryPath` → `/opt/bin/k3s` |
| `configurer/linux/opensuse.go` | Same | Rename references only |
| `configurer/linux/sles.go` | Same | Rename references only |
| `configurer/linux/ubuntu.go` | Same | Rename references only |

### Test Files

| File | Source | Changes |
|------|--------|---------|
| `configurer/linux/linux_test.go` | Same | Test K3s paths instead of K0s |
| `configurer/linux/alpine_test.go` | Same | Adapt for K3s |

## Key Implementation Notes

1. The `SetPath` mechanism allows runtime override of `K3sBinaryPath` — essential for custom install locations
2. `KubeconfigPath` for K3s is simpler: always `/etc/rancher/k3s/k3s.yaml` on the server node (no fallback logic needed)
3. `K3sCmdf` builds commands like `/usr/local/bin/k3s server`, `/usr/local/bin/k3s agent`, etc.
4. No Windows modules needed — K3s is Linux-only
