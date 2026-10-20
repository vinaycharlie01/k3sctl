# Roadmap: API Types (Config Data Model)

## Package Path
- k0sctl: `pkg/apis/k0sctl.k0sproject.io/v1beta1/`
- k3sctl: `pkg/apis/k3sctl.k3sproject.io/v1beta1/` (or simpler: `pkg/apis/v1beta1/`)

## Source Files (k0sctl)

### cluster.go (top-level)
- `APIVersion` constant = `"k0sctl.k0sproject.io/v1beta1"`
- `ClusterMetadata` struct: Name, User, Kubeconfig, EtcdMembers, Manifests
- `Cluster` struct: APIVersion, Kind, Metadata, Spec, Origin
- Methods: UnmarshalYAML, String, SetDefaults, Validate, StorageType, Resolve
- **k3sctl**:
  - `APIVersion` = `"k3sctl.k3sproject.io/v1beta1"` (or `"k3sctl.io/v1beta1"`)
  - `ClusterMetadata`: Name, User, Kubeconfig (no EtcdMembers — K3s manages internally)
  - `StorageType()`: check for etcd vs sqlite in k3s config
  - Keep Manifests for extra manifest support

### cluster_test.go
- Validation tests, StorageType tests
- **k3sctl**: Port with K3s-specific storage type detection

### cluster/spec.go
- `Spec` struct: Hosts, K0s(*K0s), Options
- Methods:
  - `K0sLeader()` — picks first running controller or falls back
  - `KubeAPIURL()`, `InternalKubeAPIURL()` — construct API URLs
  - `APIPort()` — reads from k0s config or defaults to 6443
  - `CPLBVIPs()` — control plane load balancer VIPs
  - `cplbConfig()` — parse CPLB keepalived config
  - `Resolve()` — resolve upload file paths
- **k3sctl**:
  - `Spec` struct: Hosts, K3s(*K3s), Options
  - `K3sLeader()` — picks first running server
  - Same API URL methods (K3s uses 6443 by default too)
  - No CPLB/keepalived (remove cplbConfig, CPLBVIPs)
  - Add `NodeInternalKubeAPIURL` for per-node API access

### cluster/spec_test.go
- Tests for API URL construction, leader selection
- **k3sctl**: Port

### cluster/host.go (LARGE FILE — key type)
- `Host` struct embeds `rig.CompositeConfig` + `*rig.Client`
- Key fields:
  - Role: "controller", "controller+worker", "single", "worker"
  - Reset, PrivateInterface, PrivateAddress
  - DataDir, KubeletRootDir
  - Environment map
  - UploadBinary, UseExistingK0s, K0sBinaryPath
  - K0sInstallPath, K0sDownloadURLOverride
  - InstallFlags (Flags type)
  - Files ([]*UploadFile)
  - OSIDOverride, HostnameOverride, NoTaints
  - Hooks (map of action/stage to commands)
  - Metadata (HostMetadata)
  - Configurer
  - OSRelease
- `HostMetadata`:
  - K0sBinaryVersion, K0sRunningVersion
  - K0sInstalled, K0sExistingConfig, K0sNewConfig
  - K0sTokenData, K0sStatusArgs
  - Arch, IsK0sLeader, Hostname, Ready
  - NeedsUpgrade, MachineID, DryRunFakeLeader
- Key methods:
  - `Connect(ctx)` — establish rig connection
  - `K0sInstallCommand()` — build install command string
  - `K0sResetCommand()` — build reset command
  - `K0sBackupCommand()` — build backup command
  - `K0sRestoreCommand()` — build restore command
  - `IsController()` — checks controller/controller+worker/single roles
  - `K0sServiceName()` — returns "k0scontroller" or "k0sworker"
  - `InstallK0sBinary()` — install binary at path
  - `K0sBinaryVersion()` — get version from binary
  - `UpdateK0sBinary()` — replace binary
  - `K0sDataDir()` — resolve data directory
  - `DrainNode()`, `CordonNode()`, `UncordonNode()`, `DeleteNode()`
  - `Taints()`, `AddTaint()`, `RemoveTaint()`
  - `CheckHTTPStatus()` — HTTP health check
  - `ExpandTokens()` — expand version/arch tokens in strings
  - `FlagsChanged()` — detect install flag changes
  - `RunHooks()` — execute hooks
- **k3sctl**:
  - Role: "server", "agent" (K3s terminology)
  - Replace all K0s prefixes with K3s
  - `K3sServiceName()`: "k3s" (server), "k3s-agent" (agent)
  - `K3sInstallCommand()`: Build systemd service file or use k3s install script equivalent in Go
  - `K3sResetCommand()`: Stop service, remove dirs
  - `K3sBackupCommand()`: `k3s etcd-snapshot save`
  - `K3sDataDir()`: `/var/lib/rancher/k3s` or custom
  - Token handling: read `/var/lib/rancher/k3s/server/node-token` file instead of `token create`
  - `HostMetadata`: K3sBinaryVersion, K3sRunningVersion, K3sInstalled, etc.

### cluster/host_test.go, cluster/host_posix_test.go
- Tests for service names, install commands, flag changes
- **k3sctl**: Port with K3s values

### cluster/hosts.go
- `Hosts` type: `[]*Host`
- Methods: Validate, First, Last, Find, Filter, WithRole, Controllers, Workers
- Each, ParallelEach, BatchedParallelEach
- **k3sctl**:
  - `Controllers()` → `Servers()` (filter `h.IsServer()`)
  - `Workers()` → `Agents()` (filter `h.Role == "agent"`)
  - Validation: at least one server required

### cluster/hosts_test.go
- **k3sctl**: Port

### cluster/k0s.go → k3s.go
- `K0s` struct: Version, VersionChannel, DynamicConfig, Config(dig.Mapping), Metadata
- `K0sMetadata`: ClusterID, VersionDefaulted
- Key methods:
  - `GenerateToken()` — runs `k0s token create`
  - `GetClusterID()` — via kubectl
  - `ParseToken()` — decode base64+gzip token
  - `NodeConfig()` — extract node-specific config subset
- **k3sctl**:
  - `K3s` struct: Version, VersionChannel, Config(dig.Mapping), Metadata
  - No DynamicConfig (K3s is file-based)
  - No `GenerateToken()` — tokens are files
  - `ReadNodeToken()` — read from `/var/lib/rancher/k3s/server/node-token`
  - No `ParseToken()` — K3s tokens are plain strings
  - `ServerConfig()` / `AgentConfig()` — generate role-specific config YAML

### cluster/k0s_test.go
- **k3sctl**: Port with K3s version parsing

### cluster/flags.go
- `Flags` type: string slice with flag manipulation
- Methods: Add, AddUnlessExist, AddOrReplace, Include, Index, Get, GetValue, GetBoolean, Delete, Merge, MergeOverwrite, Join, Each, Map, Equals, NewFlags
- Uses `shellescape.Unquote/Split` for safe flag parsing
- **k3sctl**: Port as-is (generic flag handling, not k0s-specific)

### cluster/flags_test.go, cluster/flags_quoting_test.go
- **k3sctl**: Port directly

### cluster/options.go
- `Options` struct: Wait, Drain, Concurrency, EvictTaint
- `WaitOption`: Enabled bool pointer with default true
- `DrainOption`: Enabled, GracePeriod, Timeout, Force, IgnoreDaemonSets, DeleteEmptyDirData, PodSelector, SkipWaitForDeleteTimeout
- `ConcurrencyOption`: Limit(30), WorkerDisruptionPercent(10), Uploads(5)
- `EvictTaintOption`: Enabled, Taint, Effect, ControllerWorkers
- `DrainOption.ToKubectlArgs()` — build kubectl drain flags
- **k3sctl**: Port as-is (generic cluster options)

### cluster/hook.go
- `Hooks` type: `map[string]map[string][]string`
- `ForActionAndStage(action, stage) []string`
- **k3sctl**: Port as-is

### cluster/uploadfile.go
- `UploadFile` struct: Name, Source, Data, Sha256, DestinationDir, DestinationFile, PermMode, DirPermMode, User, Group
- Supports glob patterns, URLs, inline data
- `ResolveRelativeTo(baseDir)` — resolve local paths
- **k3sctl**: Port as-is (generic file upload, not k0s-specific)

### cluster/uploadfile_test.go
- **k3sctl**: Port directly

### cluster/binary_providers.go
- `Host.defaultBinaryProvider()` — selects provider based on host config:
  - UseExistingK0s → binprovider.NewExisting
  - K0sBinaryPath → binprovider.NewLocalFile
  - UploadBinary → binprovider.NewLocalUpload
  - K0sDownloadURLOverride → binprovider.NewCustomURL
  - Default → binprovider.NewGitHub
- **k3sctl**: Same pattern with K3s providers:
  - UseExistingK3s → binprovider.NewExisting
  - K3sBinaryPath → binprovider.NewLocalFile
  - UploadBinary → binprovider.NewLocalUpload
  - K3sDownloadURLOverride → binprovider.NewCustomURL
  - Default → binprovider.NewGitHub (k3s-io/k3s releases)

### cluster/quote.go
- `quoter` interface and `quote()` helper for shell-safe quoting
- **k3sctl**: Port as-is

## Implementation Order
1. `cluster/quote.go` (utility)
2. `cluster/flags.go` (utility)
3. `cluster/hook.go` (utility)
4. `cluster/uploadfile.go` (utility)
5. `cluster/options.go` (config options)
6. `cluster/k3s.go` (K3s config type)
7. `cluster/host.go` (core host type)
8. `cluster/hosts.go` (host collection)
9. `cluster/spec.go` (cluster spec)
10. `cluster.go` (top-level cluster type)
11. `cluster/binary_providers.go` (binary provider selection)
12. All test files
