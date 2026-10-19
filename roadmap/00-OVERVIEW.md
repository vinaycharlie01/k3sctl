# k3sctl Roadmap — Project Overview

## What is k3sctl?

k3sctl is a cluster lifecycle management tool for K3s (lightweight Kubernetes by Rancher),
modeled after k0sctl (github.com/k0sproject/k0sctl). It manages the full lifecycle of K3s
clusters: install, upgrade, reset, backup, kubeconfig retrieval, and configuration.

## Roadmap Documents

| Document | Contents |
|----------|----------|
| [00-OVERVIEW.md](00-OVERVIEW.md) | This file — project overview, K3s vs K0s differences |
| [01-ENTRYPOINT-AND-CLI.md](01-ENTRYPOINT-AND-CLI.md) | main.go, version, cmd/ layer (15 files) |
| [02-ACTIONS.md](02-ACTIONS.md) | action/ layer — workflow composition (6 files) |
| [03-PHASE-MANAGER.md](03-PHASE-MANAGER.md) | phase/manager.go, generic_phase.go (3 files) |
| [04-PHASES-INFRASTRUCTURE.md](04-PHASES-INFRASTRUCTURE.md) | Reusable phases: connect, detect_os, facts, hooks (16 files) |
| [05-PHASES-K3S-SPECIFIC.md](05-PHASES-K3S-SPECIFIC.md) | K3s-specific phases: install, upgrade, reset (28 files) |
| [06-CONFIGURER.md](06-CONFIGURER.md) | OS-specific behavior: interface, registry, Linux distros (22 files) |
| [07-API-TYPES.md](07-API-TYPES.md) | Config data model: Cluster, Host, Spec, K3s, Options (22 files) |
| [08-PACKAGES.md](08-PACKAGES.md) | Supporting packages: retry, download, binprovider, node, github (18 files) |
| [09-FILE-MAPPING.md](09-FILE-MAPPING.md) | Complete file-by-file mapping table (k0sctl → k3sctl) |
| [10-IMPLEMENTATION-PLAN.md](10-IMPLEMENTATION-PLAN.md) | Phased implementation plan with dependency graph |

## Reference: k0sctl Architecture

k0sctl follows a three-layer architecture:

1. **cmd/** — CLI layer (urfave/cli/v2), defines commands and flags
2. **action/** — Action layer, composes phases into workflows
3. **phase/** — Phase layer, individual steps executed by phase.Manager

Supporting packages:
- **configurer/** — OS-specific behavior registry (Linux distros, Windows)
- **pkg/apis/** — API types (Cluster, Spec, Host, K0s config)
- **pkg/k0s/binprovider/** — Binary acquisition strategies
- **pkg/download/** — URL download tracking with cache
- **pkg/manifest/** — Multi-document YAML reader
- **pkg/node/** — Node status polling functions
- **pkg/retry/** — Retry helpers with context
- **integration/github/** — GitHub API for version lookups
- **version/** — Build-time version info

## Key Differences: K0s vs K3s

| Aspect | K0s | K3s |
|--------|-----|-----|
| Binary | `k0s` single binary | `k3s` single binary |
| Install dir | `/usr/local/bin/k0s` | `/usr/local/bin/k3s` |
| Data dir | `/var/lib/k0s` | `/var/lib/rancher/k3s` |
| Config dir | `/etc/k0s` | `/etc/rancher/k3s` |
| Config file | `/etc/k0s/k0s.yaml` | `/etc/rancher/k3s/config.yaml` |
| Service names | `k0scontroller`, `k0sworker` | `k3s` (server), `k3s-agent` (agent) |
| Roles | controller, worker, controller+worker | server, agent |
| Token mechanism | `k0s token create` command | Token file at `/var/lib/rancher/k3s/server/node-token` |
| Kubeconfig | `k0s kubeconfig admin` command | File at `/etc/rancher/k3s/k3s.yaml` |
| Backup | `k0s backup` command | `k3s etcd-snapshot save` command |
| Dynamic config | k0s ClusterConfig CRD | Not applicable (file-based) |
| Default CNI | kube-router | Flannel |
| Default datastore | etcd (embedded) | SQLite (embedded), etcd optional |
| Version API | k0sproject/version library | GitHub releases API (k3s-io/k3s) |
| Download URL | `github.com/k0sproject/k0s/releases` | `github.com/k3s-io/k3s/releases` |
| Binary naming | `k0s-v{ver}-{arch}` | `k3s` (amd64), `k3s-arm64`, `k3s-armhf` |
| Install mechanism | `k0s install controller/worker` | Systemd service unit + environment file |
| Uninstall | `k0s reset` | Stop service, remove dirs, remove binary |
| CPLB | Keepalived built-in | External LB or kube-vip |
| Windows support | Worker nodes via WinRM/SSH | Not supported |
| Version format | `v1.28.2+k0s.0` | `v1.28.2+k3s1` |

## K3s Installation Mechanics (Pure Go Implementation)

### Server (First Node)
1. Download k3s binary to `/usr/local/bin/k3s`
2. Write config to `/etc/rancher/k3s/config.yaml`
3. Create systemd service unit: `/etc/systemd/system/k3s.service`
4. Environment file: `/etc/systemd/system/k3s.service.env`
5. Start service: `systemctl enable --now k3s`
6. Wait for node-token generation at `/var/lib/rancher/k3s/server/node-token`
7. Wait for API readiness

### Server (Join Existing)
1. Download k3s binary
2. Write config with `server: https://<first-server>:6443` and `token: <node-token>`
3. Create systemd service unit
4. Start service
5. Wait for node ready

### Agent (Worker)
1. Download k3s binary
2. Write config with `server: https://<server>:6443` and `token: <node-token>`
3. Create systemd service unit for k3s-agent
4. Start service
5. Wait for node ready

### Systemd Service Unit (Generated in Go)
```ini
[Unit]
Description=Lightweight Kubernetes
Documentation=https://k3s.io
After=network-online.target

[Service]
Type=notify
EnvironmentFile=-/etc/systemd/system/k3s.service.env
ExecStart=/usr/local/bin/k3s <server|agent> --config /etc/rancher/k3s/config.yaml
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

## Module Path

`github.com/vinaycharlie01/k3sctl`

## Go Module Dependencies (adapted from k0sctl)

### Keep as-is:
- `github.com/k0sproject/rig/v2` — SSH/WinRM remote execution (core dependency)
- `github.com/k0sproject/dig` — Dynamic YAML mapping navigation
- `github.com/urfave/cli/v2` — CLI framework
- `github.com/sirupsen/logrus` — Structured logging
- `github.com/creasty/defaults` — Struct defaults
- `github.com/jellydator/validation` — Struct validation
- `github.com/carlmjohnson/versioninfo` — Build version info
- `github.com/a8m/envsubst` — Environment variable substitution
- `github.com/adrg/xdg` — XDG directory paths
- `github.com/bmatcuk/doublestar/v4` — Glob patterns
- `github.com/logrusorgru/aurora` — Terminal colors
- `github.com/shiena/ansicolor` — ANSI color support
- `github.com/stretchr/testify` — Testing assertions
- `github.com/AlecAivazis/survey/v2` — Interactive prompts
- `github.com/sergi/go-diff` — Diff output
- `gopkg.in/yaml.v2` — YAML parsing
- `k8s.io/client-go` — Kubeconfig manipulation
- `k8s.io/apimachinery` — YAML reader utility

### Replace/Adapt:
- `github.com/k0sproject/version` → Custom k3s version handling (K3s uses `v1.x.x+k3s1` format)

## Constraints

1. **Pure Go** — NO shell scripts for application functionality
2. All external executables via `os/exec` with explicit argument arrays
3. SSH via `k0sproject/rig` only
4. Preserve k0sctl architecture (three-layer: cmd → action → phase)
5. Clean, idiomatic Go with tests
6. Typed Go config structures (no raw map[string]interface{})
7. Go HTTP clients for all downloads
8. Context-aware operations throughout

## k0sctl Source File Count
| Directory | Go Files | Test Files | Total |
|-----------|----------|------------|-------|
| root (main.go) | 1 | 0 | 1 |
| version/ | 1 | 0 | 1 |
| cmd/ | 11 | 3 | 14 |
| action/ | 6 | 0 | 6 |
| phase/ | 32 | 14 | 46 |
| configurer/ | 5 | 2 | 7 |
| configurer/linux/ | 10 | 2 | 12 |
| configurer/linux/enterpriselinux/ | 7 | 0 | 7 |
| configurer/windows/ | 1 | 1 | 2 |
| pkg/apis/.../v1beta1/ | 1 | 1 | 2 |
| pkg/apis/.../cluster/ | 12 | 6 | 18 |
| pkg/retry/ | 1 | 1 | 2 |
| pkg/download/ | 1 | 1 | 2 |
| pkg/k0s/ | 1 | 0 | 1 |
| pkg/k0s/binprovider/ | 5 | 0 | 5 |
| pkg/manifest/ | 1 | 1 | 2 |
| pkg/node/ | 1 | 1 | 2 |
| integration/github/ | 1 | 0 | 1 |
| **Total** | **~99** | **~33** | **~132** |
