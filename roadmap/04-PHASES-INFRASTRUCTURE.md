# Roadmap: Infrastructure Phases (Reusable)

These phases are mostly infrastructure and can be ported with minimal changes.

## phase/connect.go
- Connects to all hosts in parallel with retry (10min timeout)
- Handles auth rejection limit (12 attempts before abort)
- Uses `rig.NewClient` with retry disabled (phases handle retries)
- Configures reconnect, keep-alive
- **k3sctl**: Port as-is. Rig connection logic is not k0s-specific.

### phase/connect_test.go
- Tests for connection setup
- **k3sctl**: Port directly

## phase/detect_os.go
- Resolves OS configurer per host via `configurer.ResolveOSModule`
- Falls back to `ID_LIKE` entries from `/etc/os-release`
- Sets `K0sBinaryPath` via configurer
- **k3sctl**: Port, set `K3sBinaryPath` instead

## phase/disconnect.go
- Disconnects from all hosts
- **k3sctl**: Port as-is

## phase/lock.go
- Creates lock file on each host to prevent concurrent k0sctl operations
- Uses `/var/lock/k0sctl.lock` or configurer-provided path
- **k3sctl**: Port with `/var/lock/k3sctl.lock`

## phase/unlock.go
- Removes lock file from each host
- **k3sctl**: Port with k3sctl lock path

## phase/prepare_hosts.go
- Validates host connectivity, sets up host metadata
- Prepares download tracker per host
- **k3sctl**: Port as-is (infrastructure)

## phase/gather_facts.go
- Detects per-host: architecture, machine ID, hostname, private interface/address
- Handles hostname override, CPLB VIP filtering
- Checks for existing k0s binary (`useExistingK0s`)
- Checks `kubeletRootDir` version compatibility
- **k3sctl**: Port, adapt:
  - No CPLB VIP filtering (K3s doesn't have built-in keepalived)
  - Check for existing k3s binary
  - No kubeletRootDir version check (K3s manages this internally)

### phase/gather_facts_test.go
- Tests for fact gathering logic
- **k3sctl**: Port with adaptations

## phase/validate_hosts.go
- Validates gathered host facts (arch support, connectivity)
- **k3sctl**: Port with K3s validation rules

### phase/validate_hosts_test.go
- **k3sctl**: Port

## phase/validate_facts.go
- Validates cluster-level facts after gathering
- Checks for version compatibility, downgrade prevention
- **k3sctl**: Port, adapt validation rules for K3s

### phase/validate_facts_test.go
- **k3sctl**: Port

## phase/run_hooks.go
- Executes user-defined hooks (before/after actions)
- Hooks defined in host config as command strings
- **k3sctl**: Port as-is. NOTE: hooks run shell commands on remote hosts via rig exec,
  which is acceptable (user-defined, not application logic)

## phase/daemon_reload.go
- Runs `systemctl daemon-reload` on each host
- **k3sctl**: Port as-is

## phase/downloads.go
- Downloads extra files defined in host upload config (URL sources)
- Uses download.Tracker for caching/deduplication
- **k3sctl**: Port as-is (generic file download)

## phase/uploadfiles.go
- Uploads local files to remote hosts
- Handles glob patterns, permissions, ownership
- **k3sctl**: Port as-is

### phase/uploadfiles_test.go
- **k3sctl**: Port

## Implementation Order
1. connect.go + disconnect.go (core connectivity)
2. detect_os.go (OS detection)
3. lock.go + unlock.go
4. prepare_hosts.go
5. gather_facts.go
6. validate_hosts.go + validate_facts.go
7. run_hooks.go + daemon_reload.go
8. downloads.go + uploadfiles.go
