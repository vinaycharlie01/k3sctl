# 06: Core Phases — Connection, Detection, Facts

## Phase: Connect (`phase/connect.go`)
- **Source**: `k0sctl/phase/connect.go` (~59 lines)
- **Changes**: Import paths only
- Connects to all hosts in parallel via rig v2
- Auth rejection limit (12 attempts ~60 seconds)
- Retries with 10-minute timeout
- Handles `rig.ErrNonRetryable`, `host key mismatch`, `rig.ErrAuthFailed`
- Identical for k3sctl — SSH connection is tool-agnostic

## Phase: DetectOS (`phase/detect_os.go`)
- **Source**: `k0sctl/phase/detect_os.go` (~72 lines)
- **Changes**:
  - Import paths: `k3sctl/configurer` and subpackages
  - Blank imports for OS modules:
    ```go
    _ "github.com/vinaycharlie01/k3sctl/configurer"
    _ "github.com/vinaycharlie01/k3sctl/configurer/linux"
    _ "github.com/vinaycharlie01/k3sctl/configurer/linux/enterpriselinux"
    ```
  - Remove Windows import: ~~`_ "k0sctl/configurer/windows"`~~
  - `SetPath("K0sBinaryPath", ...)` → `SetPath("K3sBinaryPath", ...)`
  - `h.K0sInstallLocation()` → `h.K3sInstallLocation()`
  - Log messages: "k0s install path" → "k3s install path"
- `After()` hook: runs "connect: after" hooks — unchanged

## Phase: GatherFacts (`phase/gather_facts.go`)
- **Source**: `k0sctl/phase/gather_facts.go` (~131 lines)
- **Changes**:
  - Remove `uniqueMachineIDSince` version check (K3s doesn't have this concern)
  - Remove `kubeletRootDirSince` version check (K3s handles this differently)
  - `K0sLeader()` → `K3sLeader()`
  - `K0s.Version` → `K3s.Version`
  - `useExistingK0s` → `useExistingK3s`
  - `K0sBinaryPath` → `K3sBinaryPath`
  - `K0sInstallPath` → `K3sInstallPath`
  - `K0sBinaryPath` LookPath: look for `k3s` instead of `k0s`
  - CPLB VIPs: K3s doesn't have keepalived CPLB — simplify or remove
  - Architecture detection, hostname resolution, private interface/address — unchanged
  - `KubeletRootDir` validation — adapt for K3s version constraints

### `phase/gather_facts_test.go`
- Adapt for K3s field names

## Phase: GatherK0sFacts → GatherK3sFacts (`phase/gather_k0s_facts.go` → `gather_k3s_facts.go`)
- **Source**: `k0sctl/phase/gather_k0s_facts.go`
- **Significant changes**:
  - Detect installed K3s binary version: `k3s --version` (not `k0s version`)
  - Detect running K3s version: check systemd service status
  - Parse K3s version from output (format: `k3s version v1.30.0+k3s1 (hash)`)
  - Read existing K3s config from `/etc/rancher/k3s/config.yaml`
  - Detect if K3s is leader (first server node)
  - Get cluster ID via `k3s kubectl`
  - K3s status args: parse from systemd service file
  - No `DynamicConfig` detection — K3s doesn't have this

### `phase/gather_k3s_facts_test.go`
- Adapt tests for K3s version parsing

## Phase: ValidateFacts (`phase/validate_facts.go`)
- **Source**: `k0sctl/phase/validate_facts.go`
- **Changes**:
  - Downgrade check: compare K3s versions
  - Version defaulting: if no version specified, use running version or latest
  - Machine ID uniqueness: may not apply to K3s
  - `K0s` → `K3s` references throughout
  - Version format: `v1.30.0+k3s1` instead of `v1.30.0+k0s.0`

### `phase/validate_facts_test.go`
- Adapt for K3s

## Phase: ValidateHosts (`phase/validate_hosts.go`)
- **Source**: `k0sctl/phase/validate_hosts.go`
- **Changes**: Minimal — calls host.Validate() and configurer.ValidateHost()
- Remove Windows validation

### `phase/validate_hosts_test.go`
- Remove Windows test cases

## Phase: ValidateEtcdMembers (`phase/validate_etcd_members.go`)
- **Source**: `k0sctl/phase/validate_etcd_members.go`
- **Changes**: K3s uses embedded SQLite by default, not etcd
  - If external etcd configured, validate members
  - If SQLite (default), skip entirely
  - If embedded etcd (K3s supports `--cluster-init`), validate
  - May rename to `ValidateDatastore`

## Phase: PrepareHosts (`phase/prepare_hosts.go`)
- **Source**: `k0sctl/phase/prepare_hosts.go`
- **Changes**: Minimal — installs required packages via configurer.Prepare()
- Handles: curl, iptables (for workers), inetutils (for hostname)
- K3s may need additional packages (e.g., container runtime deps)

## Phase: Lock/Unlock (`phase/lock.go`, `phase/unlock.go`)
- **Source**: `k0sctl/phase/lock.go`, `unlock.go`
- **Changes**: Lock file path uses `K3sctlLockFilePath` instead of `K0sctlLockFilePath`
- Same advisory locking mechanism

## Phase: Disconnect (`phase/disconnect.go`)
- **Source**: `k0sctl/phase/disconnect.go`
- **Changes**: Import paths only — generic disconnect all hosts
