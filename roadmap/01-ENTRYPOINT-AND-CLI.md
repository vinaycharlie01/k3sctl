# Roadmap: Entry Point & CLI Layer

## Source Files (k0sctl)

### main.go
- Entry point: creates `cli.App` via `cmd.NewK0sctl()`, runs with `os.Args`
- Blank imports for `versioninfo` and `version` packages for side-effect registration
- **k3sctl**: Mirror as `main.go` with `cmd.NewK3sctl()`

### version/version.go
- Build-time injected vars: `Version`, `K3sVersion` (was `K0sVersion`), `HelperVersionOverride`
- LDFLAGS injection pattern
- **k3sctl**: Same pattern, inject K3s-specific version info

### cmd/root.go
- `NewK0sctl(in, out, errOut)` → returns `*cli.App`
- Commands: version, apply, kubeconfig, init, reset, backup, config (edit/status), completion
- Signal trapping: SIGINT/SIGTERM with graceful shutdown via context cancellation
- **k3sctl**: `NewK3sctl(in, out, errOut)`, same command set

### cmd/flags.go
- Defines CLI flags: debug, trace, redact, dryRun, force, config, concurrency, concurrentUploads, timeout, retryTimeout, retryInterval
- `initConfig`: reads YAML config, envsubst, glob expansion, manifest parsing
- `initManager`: creates `phase.Manager` from parsed config
- `initLogging`: logrus with screen+file hooks
- `readConfig`: unmarshals Cluster, merges k0s ClusterConfig resources, validates
- Config reader supports stdin (`-`), glob patterns, env variable substitution
- **k3sctl**: Port with k3s config format (no ClusterConfig merge needed — k3s uses file-based config)

### cmd/apply.go
- Apply command with flags: config, concurrency, no-wait, no-drain, restore-from, kubeconfig-out, kubeconfig-api-address, kubeconfig-user, kubeconfig-cluster, disable-downgrade-check, evict-taint, force
- Before chain: initLogging→initConfig→initManager→displayLogo→displayCopyright→warnOldCache→warnRigMigration
- Creates `action.NewApply(opts)` and runs it
- **k3sctl**: Same structure, create `action.NewApply(opts)` with K3s phase sequence

### cmd/init.go
- Generates config template YAML
- Flags: k0s (include skeleton k0s config), cluster-name, controller-count, user, key-path
- `buildHosts()` creates host entries from addresses
- `hostFromAddress()` parses `user@address:port`
- Contains `DefaultK0sYaml` with default k0s cluster config
- **k3sctl**: Generate k3sctl.yaml template with K3s defaults (server/agent roles, rancher paths)

### cmd/reset.go
- Reset command: uses action.NewReset
- Flags: config, concurrency, force
- **k3sctl**: Same pattern

### cmd/backup.go
- Backup command: uses action.NewBackup
- **k3sctl**: Port using `k3s etcd-snapshot save` instead of `k0s backup`

### cmd/kubeconfig.go
- Kubeconfig retrieval command
- **k3sctl**: Port — read `/etc/rancher/k3s/k3s.yaml` instead of `k0s kubeconfig admin`

### cmd/config_edit.go
- Dynamic config edit via k0s API
- **k3sctl**: K3s doesn't have dynamic config CRD. Implement file-based config editing on remote host.

### cmd/config_status.go
- Dynamic config status via k0s API
- **k3sctl**: Adapt to k3s config file status/validation

### cmd/version.go
- Displays version info, checks for updates via GitHub API
- **k3sctl**: Same pattern, check k3sctl releases

### cmd/completion.go
- Shell completion generation (bash/zsh/fish/powershell)
- **k3sctl**: Port as-is

### cmd/logo.go
- ASCII logo display
- **k3sctl**: New k3sctl logo

### cmd/flags_test.go, cmd/init_test.go, cmd/apply_test.go
- Unit tests for flag parsing, init generation, apply config
- **k3sctl**: Port with k3s-specific values

## Implementation Order
1. `main.go` + `version/version.go`
2. `cmd/root.go` (skeleton with all subcommands)
3. `cmd/flags.go` (config loading pipeline)
4. `cmd/init.go` (config template generation)
5. `cmd/apply.go`
6. `cmd/reset.go`
7. `cmd/backup.go`
8. `cmd/kubeconfig.go`
9. `cmd/config_edit.go` + `cmd/config_status.go`
10. `cmd/version.go` + `cmd/completion.go` + `cmd/logo.go`
