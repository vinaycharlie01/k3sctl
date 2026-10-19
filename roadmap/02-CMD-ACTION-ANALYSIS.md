# k3sctl Roadmap — cmd/ and action/ Analysis

## main.go

**Source**: `main.go` (7 lines)
**k3sctl changes**: Rename import from `k0sproject/k0sctl/cmd` to `vinaycharlie01/k3sctl/cmd`.

```go
// k0sctl original
k0sctl := cmd.NewK0sctl(os.Stdin, os.Stdout, os.Stderr)
// k3sctl adaptation
k3sctl := cmd.NewK3sctl(os.Stdin, os.Stdout, os.Stderr)
```

---

## cmd/root.go — CLI App Factory

**Function**: `NewK0sctl(in, out, errOut) → *cli.App`

Creates urfave/cli App with:
- Name: "k0sctl" → **"k3sctl"**
- Usage: "k0s cluster management tool" → **"k3s cluster management tool"**
- Commands: version, apply, kubeconfig, init, reset, backup, config (edit/status), completion
- Global flags: --debug, --trace, --redact
- Before hook: context creation, signal trapping
- After hook: cancel timeout

**k3sctl changes**: Rename function to `NewK3sctl`, update app name/usage.
Signal handling (trapSignals) and context creation are identical.

---

## cmd/flags.go — Shared Flags & Helpers

**Key functions**:
1. `actions(...func(*cli.Context) error)` — chains Before/After hooks
2. `initConfig(c)` — reads config files (glob expansion, envsubst, manifest parsing)
3. `readConfig(c) (*v1beta1.Cluster, error)` — unmarshals cluster config
4. `initManager(c)` — creates phase.Manager from config
5. `initLogging / initSilentLogging` — logrus setup with file hooks
6. `LogFile()` — creates log at XDG cache `k0sctl/k0sctl.log`
7. `configReader` — reads from file or stdin
8. `displayCopyright`, `displayLogo`, `warnOldCache`, `warnRigMigration`

**k3sctl changes**:
- Default config filename: `k0sctl.yaml` → `k3sctl.yaml`
- Log file path: `k0sctl/k0sctl.log` → `k3sctl/k3sctl.log`
- Copyright text update
- Logo replacement
- `readConfig` must handle k3sctl apiVersion: `k3sctl.k3s.io/v1beta1` (or similar)
- Config merging: k0sctl merges `k0s.k0sproject.io/v1beta1/ClusterConfig` resources;
  k3sctl may not need this (K3s config is simpler)
- `warnOldCache` path changes

**Flags kept as-is**: debug, trace, redact, force, concurrency, concurrent-uploads,
timeout, retry-timeout, retry-interval, dry-run, config

---

## cmd/apply.go — Apply Command

**Flags**: config, concurrency, concurrent-uploads, dry-run, no-wait, no-drain,
restore-from, kubeconfig-out, kubeconfig-api-address, kubeconfig-user,
kubeconfig-cluster, disable-downgrade-check, evict-taint, force, debug, trace,
redact, retry/timeout

**Flow**: initLogging → initConfig → initManager → displayLogo → displayCopyright →
warnOldCache → warnRigMigration → action.NewApply(opts).Run()

**k3sctl changes**:
- Default cluster name: "k0s-cluster" → "k3s-cluster"
- Evict taint: `node.k0sproject.io/unschedulable` → K3s equivalent
  (K3s uses standard Kubernetes taints, no custom taint like k0s)
- `getNoDrainFlagOrConfig` — same logic, different default source

---

## cmd/backup.go — Backup Command

**Flow**: Creates action.Backup with output writer, runs it.
Default output: `k0s_backup_<timestamp>.tar.gz`

**k3sctl changes**:
- Default filename: `k3s_backup_<timestamp>.tar.gz`
- K3s backup is fundamentally different: etcd snapshot via `k3s etcd-snapshot save`
  rather than k0s's built-in backup command
- For SQLite mode: backup is copying `/var/lib/rancher/k3s/server/db/`
- Output format may differ

---

## cmd/completion.go — Shell Completion

**k3sctl changes**: Replace all "k0sctl" with "k3sctl" in embedded completion templates.
`prog()` fallback: "k0sctl" → "k3sctl"

---

## cmd/config_edit.go — Config Edit Command

**k0sctl behavior**: Connects to leader, fetches k0s ClusterConfig CRD via kubectl,
opens in $VISUAL/$EDITOR/vi, applies changes via kubectl apply.

**k3sctl adaptation**: K3s does NOT have a ClusterConfig CRD. Options:
1. SSH to server, read `/etc/rancher/k3s/config.yaml`, edit locally, upload back
2. Provide a different approach entirely
3. Keep the concept but implement as remote file edit

**Decision**: Implement as remote config file editor — read K3s config from server,
open in editor, upload modified version back. Restart K3s service to apply changes.

---

## cmd/config_status.go — Config Status Command

**k0sctl behavior**: Shows k0s dynamic config reconciliation events.

**k3sctl adaptation**: K3s has no dynamic config reconciliation. Options:
1. Show K3s service status (`systemctl status k3s`)
2. Show current K3s config
3. Show K3s node status

**Decision**: Show K3s server/agent service status and current configuration summary.

---

## cmd/init.go — Generate Template Config

**k0sctl behavior**: Generates template YAML with `--k0s` flag for k0s config skeleton,
default cluster name, controller count, SSH user/key config.

**k3sctl changes**:
- `--k0s` flag → `--k3s` flag for K3s config skeleton
- Default cluster name: "k0s-cluster" → "k3s-cluster"
- Default K3s YAML structure (completely different from k0s):
  ```yaml
  apiVersion: k3sctl.k3s.io/v1beta1
  kind: Cluster
  spec:
    hosts:
      - role: server
        ssh: ...
      - role: agent
        ssh: ...
    k3s:
      version: "v1.31.0+k3s1"
      config:
        # K3s server config options
  ```
- Host roles: controller/worker → server/agent
- `buildHosts()` adapted for server/agent terminology

---

## cmd/kubeconfig.go — Kubeconfig Command

**k0sctl behavior**: Connects to leader, runs GetKubeconfig phase, prints to stdout.

**k3sctl changes**: K3s stores kubeconfig at `/etc/rancher/k3s/k3s.yaml`.
The phase reads this file directly rather than running a command.

---

## cmd/logo.go — ASCII Art

**k3sctl changes**: New k3sctl-branded ASCII art.

---

## cmd/reset.go — Reset Command

**Flow**: Requires --force or interactive confirmation, sets all hosts Reset=true,
runs action.Reset.

**k3sctl changes**: Minimal at cmd layer. Major changes in phase layer for
K3s-specific uninstall procedures.

---

## cmd/version.go — Version Command

**Flags**: `--k0s` (latest k0s version), `--k0sctl` (latest k0sctl version), `--pre`

**k3sctl changes**:
- `--k0s` → `--k3s` (latest K3s version from GitHub releases)
- `--k0sctl` → `--k3sctl` (latest k3sctl version)
- GitHub API calls change to k3s-io/k3s and vinaycharlie01/k3sctl

---

## action/apply.go — Apply Action

**ApplyOptions**: Manager, DisableDowngradeCheck, NoWait, NoDrain, RestoreFrom,
KubeconfigOut, KubeconfigAPIAddress, KubeconfigUser, KubeconfigCluster, ConfigPaths

**Phase list** (k0sctl):
```
DefaultK0sVersion, Connect, DetectOS, PrepareHosts, GatherFacts, ValidateHosts,
GatherK0sFacts, ValidateFacts, ValidateEtcdMembers, EnsureJoinTokenWorkaround,
StageBinaries, UploadFiles, InstallBinaries, PrepareArm, ConfigureK0s, Restore,
RunHooks(before), InitializeK0s, InstallControllers, InstallWorkers,
UpgradeControllers, UpgradeWorkers, Reinstall, ResetWorkers, ResetControllers,
RunHooks(after), ApplyManifests, [GetKubeconfig], Disconnect
```

**k3sctl phase list**:
```
DefaultK3sVersion, Connect, DetectOS, PrepareHosts, GatherFacts, ValidateHosts,
GatherK3sFacts, ValidateFacts, [ValidateEtcdMembers],
StageBinaries, UploadFiles, InstallBinaries, PrepareArm, ConfigureK3s, Restore,
RunHooks(before), InitializeK3s, InstallServers, InstallAgents,
UpgradeServers, UpgradeAgents, Reinstall, ResetAgents, ResetServers,
RunHooks(after), ApplyManifests, [GetKubeconfig], Disconnect
```

Changes:
- Drop EnsureJoinTokenWorkaround (K3s token mechanism is different)
- ValidateEtcdMembers only when using etcd datastore (not SQLite)
- Rename controller→server, worker→agent throughout

---

## action/backup.go — Backup Action

**k0sctl phases**: DefaultK0sVersion, Connect, DetectOS, Lock, PrepareHosts,
GatherFacts, GatherK0sFacts, Backup, Unlock, Disconnect

**k3sctl changes**: Backup phase completely different:
- K3s etcd: `k3s etcd-snapshot save --dir <path>`
- K3s SQLite: copy database files from `/var/lib/rancher/k3s/server/db/`
- Download snapshot from server to local machine

---

## action/kubeconfig.go — Kubeconfig Action

**k0sctl behavior**: Reduces to leader host only, runs Connect → DetectOS →
GetKubeconfig → Disconnect.

**k3sctl changes**: Same flow, different GetKubeconfig implementation.

---

## action/reset.go — Reset Action

**k0sctl phases**: DefaultK0sVersion, Connect, DetectOS, Lock, PrepareHosts,
GatherFacts, GatherK0sFacts, ResetWorkers, ResetControllers, ResetLeader,
DaemonReload, Unlock, Disconnect

**k3sctl phases**:
```
DefaultK3sVersion, Connect, DetectOS, Lock, PrepareHosts, GatherFacts,
GatherK3sFacts, ResetAgents, ResetServers, ResetLeader, DaemonReload,
Unlock, Disconnect
```

---

## action/config_edit.go — Config Edit Action

**k0sctl behavior**: Connects to leader, gets ClusterConfig via kubectl,
opens in editor, applies via kubectl. Uses shellEditor() for $VISUAL/$EDITOR.

**k3sctl adaptation**: Connect to leader, read `/etc/rancher/k3s/config.yaml`,
open in local editor, upload modified file, restart K3s service.
Editor resolution (shellEditor) stays identical.

---

## action/config_status.go — Config Status Action

**k0sctl behavior**: Gets reconciliation events via kubectl.

**k3sctl adaptation**: Show K3s service status, config file contents,
and node readiness. Support --format json/yaml output.

---

## integration/github/github.go — GitHub Integration

**Types**: Asset, Release (with IsNewer comparison)

**Functions**:
- `LatestK0sBinaryURL(arch, osKind, preok)` → `LatestK3sBinaryURL`
- `LatestK0sVersion(preok)` → `LatestK3sVersion`
- `LatestRelease(preok)` — k0sctl releases → k3sctl releases
- `fetchLatestRelease()` — URL: `k0sproject/k0sctl` → `vinaycharlie01/k3sctl`

**k3sctl changes**:
- K3s versions: Use GitHub API at `k3s-io/k3s/releases`
- K3s binary URL pattern: `https://github.com/k3s-io/k3s/releases/download/{tag}/k3s`
  (varies by arch: `k3s` for amd64, `k3s-arm64`, `k3s-armhf`, `k3s-s390x`)
- Cannot use `k0sproject/version` library directly for K3s — need custom version
  resolution from K3s GitHub releases
- k3sctl tool releases: `vinaycharlie01/k3sctl/releases`
