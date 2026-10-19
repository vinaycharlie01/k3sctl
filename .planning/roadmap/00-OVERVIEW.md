# k3sctl Roadmap: Complete Project Overview

## Mission

Build `k3sctl` — a pure Go cluster management tool for K3s — modeled after `k0sctl` (the K0s cluster management tool). The tool will share the same architecture, phase-based execution model, and user experience patterns, but adapted entirely for K3s.

## Source Reference

- **Reference repo**: `github.com/k0sproject/k0sctl` (analyzed file-by-file)
- **Target repo**: `github.com/vinaycharlie01/k3sctl`
- **Go module**: `github.com/vinaycharlie01/k3sctl`

## Key Differences: K0s vs K3s

| Aspect | K0s (k0sctl) | K3s (k3sctl) |
|--------|-------------|-------------|
| Binary path | `/usr/local/bin/k0s` | `/usr/local/bin/k3s` |
| Config path | `/etc/k0s/k0s.yaml` | `/etc/rancher/k3s/config.yaml` |
| Data dir | `/var/lib/k0s` | `/var/lib/rancher/k3s` |
| Token path | `/etc/k0s/k0stoken` | `/etc/rancher/k3s/token` (server token file) |
| Kubeconfig | `k0s kubeconfig admin` | `/etc/rancher/k3s/k3s.yaml` (file on server node) |
| Service names | `k0scontroller`, `k0sworker` | `k3s` (server), `k3s-agent` (agent) |
| Roles | controller, worker, controller+worker, single | server, agent |
| Join mechanism | Token-based (`k0s token create`) | Token-based (server token at `/var/lib/rancher/k3s/server/node-token`) |
| Default storage | etcd | Embedded SQLite (or external etcd/MySQL/PostgreSQL) |
| Windows support | Yes (workers only) | No |
| Install command | `k0s install controller/worker` | `k3s server` / `k3s agent` (systemd units) |
| kubectl | `k0s kubectl` | `k3s kubectl` |
| Version format | `v1.30.0+k0s.0` | `v1.30.0+k3s1` |
| API port default | 6443 | 6443 |
| Backup | `k0s backup` | No native backup command |
| Dynamic config | `k0s` supports it | K3s uses config file or CLI flags |
| Kubelet extra args | Via `--kubelet-extra-args` flag | Via `--kubelet-arg` flag |
| CNI | kube-router (default) | flannel (default) |

## Architecture (Preserved from k0sctl)

```
┌─────────────────────────────────────────────┐
│                   CLI Layer                  │
│               (cmd/ - urfave/cli)            │
│  apply | reset | init | kubeconfig | config  │
└───────────────────────┬─────────────────────┘
                        │
┌───────────────────────▼─────────────────────┐
│                Action Layer                  │
│              (action/ - workflows)           │
│   Compose phases in correct order            │
└───────────────────────┬─────────────────────┘
                        │
┌───────────────────────▼─────────────────────┐
│          Phase Manager + Phases              │
│              (phase/ - execution)            │
│   Prepare → ShouldRun → Before → Run →      │
│   After → CleanUp (lifecycle)                │
└───────────────────────┬─────────────────────┘
                        │
┌───────────────────────▼─────────────────────┐
│         Configurer Layer                     │
│     (configurer/ - OS abstraction)           │
│   Linux base + distro-specific modules       │
└───────────────────────┬─────────────────────┘
                        │
┌───────────────────────▼─────────────────────┐
│            Remote Execution                  │
│        (rig v2 - SSH/localhost)              │
│   Connect, exec, file transfer, sudo         │
└─────────────────────────────────────────────┘
```

## Implementation Phases (Development Order)

1. **Foundation** (Documents 01-03): Go module, core types, configurer interface
2. **Execution Engine** (Documents 04-05): Phase manager, generic phase, retry
3. **API Types** (Document 06): Cluster, Spec, Host, K3s config, Options
4. **Phases** (Documents 07-09): All ~35 phases adapted for K3s
5. **Actions** (Document 10): Apply, Reset, Kubeconfig, Init workflows
6. **CLI** (Document 11): Commands, flags, config loading
7. **Supporting Packages** (Document 12): Binary providers, download, node status
8. **Testing** (Document 13): Unit tests for all packages
9. **Build & CI** (Document 14): Makefile, goreleaser, GitHub Actions

## Files Dropped (Not Needed for k3sctl)

- `configurer/windows.go` — K3s has no Windows support
- `configurer/windows/windows.go` — K3s has no Windows support
- `configurer/windows/windows_test.go` — K3s has no Windows support
- `configurer/windows_test.go` — K3s has no Windows support
- `phase/backup.go` — K3s has no native backup command (defer or implement differently)
- `phase/restore.go` — K3s has no native restore command
- `action/backup.go` — K3s has no native backup
- `cmd/backup.go` — K3s has no native backup
- All Windows-specific code paths in `host.go`, `install_workers.go`, etc.

## Files Modified Significantly

- Every file referencing `k0s` paths, commands, versions → adapted for K3s
- `k0s.go` → `k3s.go` — completely different token mechanism, version format
- `host.go` — roles change (server/agent), service names differ, no Windows
- All install/upgrade/reset phases — different commands and service management
- `get_kubeconfig.go` — K3s writes kubeconfig to a file, not via CLI command
- `configure_k0s.go` → `configure_k3s.go` — different config format/location
- `init.go` — different default config template
