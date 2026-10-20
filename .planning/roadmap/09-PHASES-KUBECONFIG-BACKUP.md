# 09: Kubeconfig, Backup, and Remaining Phases

## Phase: GetKubeconfig (`phase/get_kubeconfig.go`)
- **Source**: `k0sctl/phase/get_kubeconfig.go` (~130 lines)
- **Significant changes**:
  - K0s: runs `k0s kubeconfig admin --data-dir=...` to generate kubeconfig
  - K3s: reads file directly from `/etc/rancher/k3s/k3s.yaml` on server node
  - The kubeconfig rewriting logic (cluster name, user, API address) stays the same
  - Uses `k8s.io/client-go/tools/clientcmd` — same
  - `readKubeconfig` function:
    ```go
    var readKubeconfig = func(h *cluster.Host) (string, error) {
        output, err := h.Sudo().FS().ReadFile("/etc/rancher/k3s/k3s.yaml")
        // or use configurer.KubeconfigPath()
        return string(output), err
    }
    ```
  - DryRun handling: same pattern

### `phase/get_kubeconfig_test.go`
- Test kubeconfig reading from file instead of command
- Test rewriting logic (same)

## Phase: Backup (`phase/backup.go`)
- **Source**: `k0sctl/phase/backup.go`
- **K3s consideration**:
  - K3s does NOT have a native `backup` command like `k0s backup`
  - Options for k3sctl:
    1. **Skip entirely**: don't implement backup/restore initially
    2. **SQLite backup**: if using default SQLite, backup the DB file
    3. **Etcd snapshot**: if using embedded etcd (`--cluster-init`), use `k3s etcd-snapshot save`
    4. **Custom implementation**: tar up the data directory
  - Recommendation: Implement etcd snapshot support and data-dir backup
  - `k3s etcd-snapshot save --name <name> --data-dir <dir>`
  - Download snapshot file from server

## Phase: Restore (`phase/restore.go`)
- **Source**: `k0sctl/phase/restore.go`
- **K3s consideration**:
  - K3s can restore from etcd snapshot: `k3s server --cluster-reset --cluster-reset-restore-path=<snapshot>`
  - Only works with embedded etcd clusters
  - Recommendation: Implement basic restore from etcd snapshot
  - No SQLite restore (would need to stop all servers, replace DB)

## Phase: Backup/Restore Actions
- **Source**: `k0sctl/action/backup.go`
- **k3sctl**: If implementing backup, adapt action to use K3s etcd-snapshot
- Phase sequence: Connect → DetectOS → GatherFacts → GatherK3sFacts → Backup → Disconnect

## Remaining Minor Phases

### `phase/apply_manifests.go`
- **Source**: `k0sctl/phase/apply_manifests.go`
- **Changes**: Use `k3s kubectl apply` to apply additional manifests after cluster ready

### `phase/daemon_reload.go`
- **Source**: `k0sctl/phase/daemon_reload.go`
- **Changes**: Import paths only — runs `systemctl daemon-reload`

### `phase/downloads.go`
- **Source**: `k0sctl/phase/downloads.go`
- **Changes**: Adapt K3s download URLs

### `phase/lock.go` / `phase/unlock.go`
- **Source**: `k0sctl/phase/lock.go`, `unlock.go`
- **Changes**: Lock file path from configurer (`K3sctlLockFilePath`)
