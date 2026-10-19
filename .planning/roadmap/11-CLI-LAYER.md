# 11: CLI Layer — Commands and Flags

## `cmd/root.go`
- **Source**: `k0sctl/cmd/root.go` (~83 lines)
- **Changes**:
  - `NewK3sctl(in, out, errOut) *cli.App`
  - App name: `k3sctl`
  - Usage: `k3s cluster management tool`
  - Commands: version, apply, kubeconfig, init, reset, config(edit/status), completion
  - Same global flags: debug, trace, no-redact
  - Same signal trapping: SIGINT/SIGTERM graceful shutdown
  - Same context management with cancel

## `cmd/flags.go`
- **Source**: `k0sctl/cmd/flags.go` (~600+ lines)
- **Changes**:
  - `configFlag` default: `k3sctl.yaml` instead of `k0sctl.yaml`
  - All context keys: same pattern
  - `initConfig()` — same multi-doc YAML reading with glob support
  - `initManager()` — creates phase.Manager with config
  - `initLogging()` — same logrus setup with file logging
  - `displayLogo()` — k3sctl logo instead of k0sctl
  - `displayCopyright()` — updated copyright
  - `warnOldCache()` — adapt XDG cache path for k3sctl
  - `warnRigMigration()` — same rig v2 migration warning
  - All flag definitions: same (concurrency, timeout, dry-run, etc.)
  - Config loading: parse `k3sctl.k3sproject.io/v1beta1` API version

## `cmd/apply.go`
- **Source**: `k0sctl/cmd/apply.go` (~139 lines)
- **Changes**:
  - Before: actions(initLogging, initConfig, initManager, displayLogo, displayCopyright, warnOldCache, warnRigMigration)
  - Same flags: config, concurrency, dry-run, no-wait, no-drain, restore-from, kubeconfig-out, etc.
  - `evict-taint` flag: same support
  - Creates `action.NewApply(opts)` and runs it

## `cmd/reset.go`
- **Source**: `k0sctl/cmd/reset.go`
- **Changes**: Import paths, action.Reset{}

## `cmd/kubeconfig.go`
- **Source**: `k0sctl/cmd/kubeconfig.go`
- **Changes**: Import paths, action.Kubeconfig{}

## `cmd/init.go`
- **Source**: `k0sctl/cmd/init.go` (~232 lines)
- **Changes**:
  - Default K3s YAML config (not k0s):
    ```yaml
    write-kubeconfig-mode: "0644"
    tls-san: []
    ```
  - API version: `k3sctl.k3sproject.io/v1beta1`
  - Default roles: server, agent (not controller, worker)
  - Cluster name default: `k3s-cluster`
  - Same address parsing, host generation

### `cmd/init_test.go`
- Adapt for k3sctl output format

## `cmd/config_edit.go`
- **Source**: `k0sctl/cmd/config_edit.go`
- **Changes**: Import paths, action.ConfigEdit{}

## `cmd/config_status.go`
- **Source**: `k0sctl/cmd/config_status.go`
- **Changes**: Import paths, action.ConfigStatus{}

## `cmd/version.go`
- **Source**: `k0sctl/cmd/version.go`
- **Changes**: Display k3sctl version info

## `cmd/completion.go`
- **Source**: `k0sctl/cmd/completion.go`
- **Changes**: Shell completion for k3sctl

## `cmd/logo.go`
- **Source**: `k0sctl/cmd/logo.go`
- **Changes**: k3sctl ASCII art logo

## Test Files

### `cmd/apply_test.go`
- Adapt for k3sctl

### `cmd/flags_test.go`
- Adapt config file loading for k3sctl.yaml
