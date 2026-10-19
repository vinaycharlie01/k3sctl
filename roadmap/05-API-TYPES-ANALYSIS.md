# k3sctl Roadmap — pkg/apis/ and Supporting Packages Analysis

## API Types Overview

The API types define the configuration schema for k3sctl cluster definitions.

Module path: `pkg/apis/k3sctl.k3s.io/v1beta1/`

### cluster.go — Top-Level Cluster Type

**k0sctl**:
```yaml
apiVersion: k0sctl.k0sproject.io/v1beta1
kind: Cluster
metadata:
  name: my-cluster
spec:
  hosts: [...]
  k0s: {...}
```

**k3sctl**:
```yaml
apiVersion: k3sctl.k3s.io/v1beta1
kind: Cluster
metadata:
  name: my-cluster
spec:
  hosts: [...]
  k3s: {...}
```

Changes:
- apiVersion: `k0sctl.k0sproject.io/v1beta1` → `k3sctl.k3s.io/v1beta1`
- Spec.K0s → Spec.K3s
- Config merging: k0sctl merges separate `k0s.k0sproject.io/v1beta1/ClusterConfig` 
  resources found alongside the cluster config. K3s doesn't have a separate 
  ClusterConfig CRD, so this merge behavior simplifies to just the main config.

---

### cluster/spec.go — Spec Type

```go
type Spec struct {
    Hosts   Hosts   `yaml:"hosts,omitempty"`
    K0s     *K0s    `yaml:"k0s,omitempty"`     // → K3s *K3s
    Options Options `yaml:"options"`
    k0sLeader *Host                             // → k3sLeader
}
```

Key methods:
- `K0sLeader()` → `K3sLeader()` — finds first running server
- `KubeAPIURL()` — external API URL (same concept)
- `InternalKubeAPIURL()` — internal API URL
- `APIPort()` — default 6443 (same for K3s)
- `CPLBVIPs()` — **DROP** — K3s doesn't have built-in CPLB/Keepalived
- `cplbConfig()` — **DROP**
- `clusterExternalAddress()` — adapt for K3s config structure

**k3sctl changes**:
- Remove all CPLB (Control Plane Load Balancing) code
- K3s doesn't have `spec.api.externalAddress` in its config
  - Instead: `tls-san` entries in K3s config
- API port from K3s config: `https-listen-port` (default 6443)

---

### cluster/host.go — Host Type

The largest type — represents a single host in the cluster.

```go
type Host struct {
    Role             string          // controller, worker, controller+worker
    PrivateInterface string
    PrivateAddress   string
    Environment      map[string]string
    UploadBinary     bool
    K0sBinaryPath    string          // → K3sBinaryPath
    K0sDownloadURL   string          // → K3sDownloadURL
    InstallFlags     Flags
    Files            []UploadFile
    OS               string
    Hostname         string
    Hooks            Hooks
    NoTaints         bool
    
    // Metadata
    Metadata HostMetadata
    
    // Runtime
    Configurer  Configurer
    Connection  *rig.Client
    // ...
}
```

**k3sctl Role changes**:
- `controller` → `server`
- `worker` → `agent`
- `controller+worker` → `server` (K3s servers run kubelet by default unless `--disable-agent`)
- `single` → `server` (single-node K3s is just a server)

**k3sctl Host methods to adapt**:
- `K0sDataDir()` → `K3sDataDir()` → `/var/lib/rancher/k3s`
- `K0sConfigPath()` → `K3sConfigPath()` → `/etc/rancher/k3s/config.yaml`
- `K0sBinaryPath` → `K3sBinaryPath` → `/usr/local/bin/k3s`
- `K0sServiceName()` → `K3sServiceName()` → `k3s` or `k3s-agent`
- `IsController()` → `IsServer()`
- `IsWorker()` → `IsAgent()`

**HostMetadata**:
```go
type HostMetadata struct {
    Hostname            string
    LongHostname        string
    InternalAddress     string
    K0sBinaryVersion    *version.Version    // → K3sBinaryVersion
    K0sRunningVersion   *version.Version    // → K3sRunningVersion
    K0sInstalled        bool                // → K3sInstalled
    K0sExistingConfig   string              // → K3sExistingConfig
    K0sNewConfig        string              // → K3sNewConfig
    K0sJoinToken        string              // → K3sJoinToken (from node-token file)
    K0sJoinTokenID      string              // → K3sJoinTokenID
    Arch                string
    IsK0sLeader         bool                // → IsK3sLeader
    NeedsUpgrade        bool
    Ready               bool
    MachineID           string
    DrainingDisabled    bool
    Reset               bool
}
```

---

### cluster/hosts.go — Hosts Collection

```go
type Hosts []*Host

func (hosts Hosts) Controllers() Hosts  // → Servers()
func (hosts Hosts) Workers() Hosts      // → Agents()
func (hosts Hosts) First() *Host
func (hosts Hosts) Filter(fn) Hosts
func (hosts Hosts) Find(fn) *Host
func (hosts Hosts) Each(fn) error
func (hosts Hosts) ParallelEach(fn) error
func (hosts Hosts) BatchedParallelEach(concurrency, fn) error
```

---

### cluster/k0s.go → cluster/k3s.go — K3s Config Type

**k0sctl K0s type**:
```go
type K0s struct {
    Version        *version.Version
    VersionChannel string
    DynamicConfig  bool
    Config         dig.Mapping
    Metadata       K0sMetadata
}
```

**k3sctl K3s type**:
```go
type K3s struct {
    Version        string          // K3s version string "v1.31.0+k3s1"
    Channel        string          // "stable", "latest", "v1.31"
    Config         dig.Mapping     // K3s config.yaml contents
    Metadata       K3sMetadata
}

type K3sMetadata struct {
    ClusterID        string
    VersionDefaulted bool
}
```

Key differences:
- K3s version format: `v1.31.0+k3s1` (includes +k3s suffix)
- No DynamicConfig (K3s reads config at startup only)
- K3s version channels use update.k3s.io API
- No k0sproject/version dependency for K3s — custom version handling needed
- Token generation: read from file, not `k0s token create` command
- Kubeconfig: read file, not run command

**K3s Config Structure** (dig.Mapping representation):
```yaml
# Server config
write-kubeconfig-mode: "0644"
tls-san:
  - "my-cluster.example.com"
cluster-cidr: "10.42.0.0/16"
service-cidr: "10.43.0.0/16"
cluster-dns: "10.43.0.10"
flannel-backend: "vxlan"
disable:
  - traefik
  - servicelb
node-label:
  - "role=server"
```

---

### cluster/options.go — Options Type

```go
type Options struct {
    Drain DrainOptions
}

type DrainOptions struct {
    Enabled          bool
    DrainTimeout     time.Duration
    DeleteLocalData  bool
    GracePeriod      int
}
```

**k3sctl**: Direct port — drain options are Kubernetes-level, not K3s-specific.

---

### cluster/flags.go — Install Flags

Generic flag builder for command-line arguments. Uses quoting helpers.

**k3sctl**: Direct port — same flag building mechanism works for K3s commands.

---

### cluster/hook.go — Hooks

Before/after hooks for various lifecycle events.

**k3sctl**: Direct port — hooks are generic command execution.

---

### cluster/uploadfile.go — Upload Files

File upload specification for deploying files to hosts.

**k3sctl**: Direct port.

---

### cluster/binary_providers.go — Binary Provider Selection

Selects the right BinaryProvider for each host based on configuration:
- UploadBinary=true + K0sBinaryPath set → LocalFile
- UploadBinary=true → LocalUpload (download to local, upload to host)
- K0sDownloadURL set → CustomURL
- K0sBinaryPath set on host → Existing
- Default → GitHub download

**k3sctl changes**:
- Field names: K0sBinaryPath → K3sBinaryPath, K0sDownloadURL → K3sDownloadURL
- GitHub download URL pattern for K3s:
  ```
  https://github.com/k3s-io/k3s/releases/download/{version}/k3s         # amd64
  https://github.com/k3s-io/k3s/releases/download/{version}/k3s-arm64   # arm64
  https://github.com/k3s-io/k3s/releases/download/{version}/k3s-armhf   # armv7
  ```
- No k0sproject/version.DownloadURL() — implement K3s URL construction

---

## pkg/k3s/provider.go — Binary Provider Interface

```go
type BinaryProvider interface {
    NeedsUpgrade() bool
    IsUpload() bool
    Stage(ctx context.Context) (string, error)
    CleanUp(ctx context.Context)
}

type BinaryCacher interface {
    BinaryCacheKey() (string, error)
    EnsureCached(ctx context.Context) error
}
```

**k3sctl**: Direct port of interface. Implementations adapted for K3s URLs.

---

## pkg/k3s/binprovider/ — Provider Implementations

### download.go (GitHub direct download)
- `NewGitHub(h, installPath, target)` → K3s GitHub URL construction
- K3s URL: based on arch → `k3s`, `k3s-arm64`, `k3s-armhf`

### local_upload.go (local cache + upload)
- Cache path: `k0sctl/k0s/{os}/{arch}/` → `k3sctl/k3s/{os}/{arch}/`
- Download to local cache, then upload to host
- `downloadToFile()` — HTTP download with proper cleanup

### local_file.go (user-supplied binary)
- Direct upload of user-specified binary
- **k3sctl**: Direct port

### existing.go (pre-installed binary)
- Assumes binary is already on host
- **k3sctl**: Direct port

### helpers.go (staging helpers)
- `stageTempPath()` — temp file naming
- `stageUpload()` — upload + chmod
- `stageDownload()` — download + chmod
- `needsUpgrade()` — version comparison

**k3sctl**: Adapt for K3s install path (`/usr/local/bin/k3s`).

---

## pkg/download/download.go — Download Tracker

Smart download tracking with caching:
- Records what was downloaded (ETag, Last-Modified, SHA256)
- Skips re-download if server content unchanged
- Supports resume for interrupted downloads
- Cache directory: `k0sctl/downloads/` → `k3sctl/downloads/`

**k3sctl**: Change cache directory name. Everything else is generic.

---

## pkg/manifest/reader.go — Manifest Reader

Multi-document YAML reader using k8s.io/apimachinery.
- Parses `---` separated YAML documents
- Filters by apiVersion and kind
- Stores raw bytes for re-marshaling

**k3sctl**: Direct port — used for config file parsing.

---

## pkg/retry/retry.go — Retry Helpers

Context-aware retry with:
- `Context(ctx, f)` — retry until success or context done
- `Timeout(ctx, timeout, f)` — retry with timeout
- `Times(ctx, n, f)` — retry up to N times
- `ErrAbort` — stop retrying immediately

**k3sctl**: Direct port.

---

## pkg/node/statusfunc.go — Node Status Functions

Functions that return retry-compatible functions for waiting on node states:
- `KubeNodeReadyFunc(h)` — waits for node Ready condition
- `K0sDynamicConfigReadyFunc(h)` — **DROP** for K3s (no dynamic config)
- `ScheduledEventsAfterFunc(h, since)` — waits for scheduled events
- `HTTPStatusFunc(h, url, expected...)` — waits for HTTP status
- `ServiceRunningFunc(h, service)` — waits for service to start
- `ServiceStoppedFunc(h, service)` — waits for service to stop

**k3sctl changes**:
- `KubeNodeReadyFunc` — adapt kubectl command (`k3s kubectl` instead of `k0s kubectl`)
- Drop `K0sDynamicConfigReadyFunc`
- Keep others with K3s service names

---

## K3s Version Handling

k0sctl uses `k0sproject/version` library which provides:
- Semantic version parsing/comparison
- `LatestByPrerelease(preok)` — fetches from k0s update channels
- `DownloadURL(osKind, arch)` — constructs download URL
- `Collection` for sorting versions

**k3sctl needs custom version handling**:

```go
package k3sversion

// K3s versions are like "v1.31.0+k3s1"
// The "+k3s1" suffix is the K3s patch level

type Version struct {
    Major, Minor, Patch int
    K3sPatch            int  // the number after +k3s
    Original            string
}

func Parse(s string) (*Version, error)
func (v *Version) String() string
func (v *Version) LessThan(other *Version) bool
func (v *Version) Equal(other *Version) bool
func (v *Version) GreaterThan(other *Version) bool

// LatestStable fetches from K3s update channel API
func LatestStable() (*Version, error)
// LatestByChannel fetches from specific channel
func LatestByChannel(channel string) (*Version, error)

// DownloadURL constructs the binary download URL
func DownloadURL(version, arch string) string {
    // https://github.com/k3s-io/k3s/releases/download/v1.31.0%2Bk3s1/k3s
    // Note: + is URL-encoded as %2B in the release tag URL
}
```

K3s update channel API: `https://update.k3s.io/v1-release/channels`
Returns JSON with channel info including latest versions.
