# 07: Install and Configuration Phases

## Phase: DefaultK3sVersion (`phase/default_k3s_version.go`)
- **Source**: `k0sctl/phase/default_k0s_version.go`
- **Changes**:
  - Fetch latest K3s version from K3s channels: `https://update.k3s.io/v1-release/channels`
  - K3s version channel: "stable", "latest", "v1.30", etc.
  - Set `Config.Spec.K3s.Version` if not specified
  - K3s channel API returns JSON with `data[].name` and `data[].latest`

## Phase: StageBinaries (`phase/stage_binaries.go`)
- **Source**: `k0sctl/phase/stage_binaries.go`
- **Changes**:
  - Uses `h.K3sBinaryProvider(target)` instead of K0s
  - Binary caching pattern: `BinaryCacher` interface — same
  - Downloads K3s binary from GitHub releases: `github.com/k3s-io/k3s/releases/download/v1.30.0+k3s1/k3s` (or `k3s-arm64`, `k3s-armhf`)
  - No `.exe` variant needed (no Windows)
  - Upload vs download distinction: same pattern

### `phase/stage_binaries_test.go`
- Adapt for K3s binary URLs

## Phase: UploadFiles (`phase/uploadfiles.go`)
- **Source**: `k0sctl/phase/uploadfiles.go`
- **Changes**: Import paths only — generic file upload to hosts
- Uses `parallelDoUpload` for concurrency control

### `phase/uploadfiles_test.go`
- Import paths only

## Phase: InstallBinaries (`phase/install_binaries.go`)
- **Source**: `k0sctl/phase/install_binaries.go`
- **Changes**:
  - Calls `h.InstallK3sBinary(path)` instead of K0s
  - Verifies installed version: `k3s --version`
  - Sets correct permissions (0750)
  - Creates binary directory if needed

## Phase: PrepareArm (`phase/arm_prepare.go`)
- **Source**: `k0sctl/phase/arm_prepare.go`
- **Changes**: Minimal — ARM preparation is generic
- Sets up any ARM-specific requirements

## Phase: ConfigureK3s (`phase/configure_k3s.go`)
- **Source**: `k0sctl/phase/configure_k0s.go`
- **Significant changes**:
  - K3s config is a YAML file at `/etc/rancher/k3s/config.yaml`
  - K3s config format is flat key-value, not nested like k0s:
    ```yaml
    write-kubeconfig-mode: "0644"
    tls-san:
      - "my-lb.example.com"
    cluster-init: true
    ```
  - Write config file to each server node
  - Agent nodes may also need config (for `--server` URL, token, etc.)
  - No dynamic config support
  - Config diff detection for upgrades

### `phase/configure_k3s_test.go`
- Test K3s config generation and diff

## Phase: EnsureJoinTokenWorkaround (`phase/ensure_join_token_workaround.go`)
- **Source**: `k0sctl/phase/ensure_join_token_workaround.go`
- **Changes**: May not be needed for K3s — the workaround is k0s-version-specific
  - Evaluate whether K3s has similar token bugs
  - If not, create empty/no-op version or skip

### `phase/ensure_join_token_workaround_test.go`
- Adapt or remove

## Phase: InitializeK3s (`phase/initialize_k3s.go`)
- **Source**: `k0sctl/phase/initialize_k0s.go`
- **Changes**:
  - Start the first K3s server node (the leader)
  - K3s server start: create systemd service, start it
  - For HA: use `--cluster-init` flag on first server (enables embedded etcd)
  - Wait for K3s to be ready: check `/readyz` endpoint
  - Read and store the join token from `/var/lib/rancher/k3s/server/node-token`
  - No `k0s install` equivalent — K3s uses systemd units directly or install script

## Phase: InstallControllers → InstallServers (`phase/install_servers.go`)
- **Source**: `k0sctl/phase/install_controllers.go` (~297 lines)
- **Significant changes**:
  - Role: "server" instead of "controller"
  - Join token: read from leader's `/var/lib/rancher/k3s/server/node-token`
  - Install command: K3s uses `INSTALL_K3S_SKIP_DOWNLOAD=true k3s-install.sh` or direct binary+systemd
  - For k3sctl approach: place binary, create systemd unit, configure, start
  - Server flags: `--server https://<leader>:6443 --token <token> --data-dir <dir>`
  - Quorum-aware parallel batching: same algorithm (odd number batches)
  - Service name: `k3s` (not `k0scontroller`)
  - Readiness check: `k3s kubectl get --raw='/readyz'`
  - Token invalidation: K3s doesn't have `token invalidate` — skip or use different approach
  - Token overwrite: not needed the same way

## Phase: InstallWorkers → InstallAgents (`phase/install_agents.go`)
- **Source**: `k0sctl/phase/install_workers.go` (~283 lines)
- **Significant changes**:
  - Role: "agent" instead of "worker"
  - Join token: same token from leader, but for agent use
  - Install: place binary, create systemd unit with `k3s agent` command
  - Agent flags: `--server https://<server>:6443 --token <token> --data-dir <dir>`
  - Service name: `k3s-agent` (not `k0sworker`)
  - API validation: check connectivity to server API
  - Node ready wait: `k3s kubectl get node` from leader
  - No Windows path (remove `cmd.AllowWinStderr()`)
  - Token invalidation: not applicable for K3s
  - Kubeconfig for validation: use leader's kubeconfig, not token-derived

## Phase: RunHooks (`phase/run_hooks.go`)
- **Source**: `k0sctl/phase/run_hooks.go`
- **Changes**: Import paths only — generic hook execution
- Runs before/after hooks for apply, install, upgrade, reset actions

## Phase: ApplyManifests (`phase/apply_manifests.go`)
- **Source**: `k0sctl/phase/apply_manifests.go`
- **Changes**: Uses `k3s kubectl apply` instead of `k0s kubectl apply`
- Applies additional manifests after cluster is ready

## Phase: DaemonReload (`phase/daemon_reload.go`)
- **Source**: `k0sctl/phase/daemon_reload.go`
- **Changes**: Import paths only — runs `systemctl daemon-reload`

## Phase: Downloads (`phase/downloads.go`)
- **Source**: `k0sctl/phase/downloads.go`
- **Changes**: Import paths and K3s download URL pattern
