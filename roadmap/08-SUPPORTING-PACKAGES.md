# Roadmap: Supporting Packages

---

## pkg/retry/retry.go — Retry Helpers

### k0sctl behavior
- `Context(ctx, f)` — retry at constant 5s interval until success or context cancel
- `Timeout(ctx, timeout, f)` — retry with deadline
- `WithDefaultTimeout(ctx, f)` — 2-minute timeout
- `Times(ctx, n, f)` — retry N times
- `ErrAbort` sentinel to stop retrying
- All functions accept `func(context.Context) error`

### k3sctl adaptation
- **Direct port** — generic retry infrastructure, not k0s-specific
- Same API, same behavior

---

## pkg/manifest/reader.go — Multi-document YAML Reader

### k0sctl behavior
- Reads multi-document YAML (separated by `---`)
- Returns individual documents as byte slices
- Used by config loading to support multiple resources in one file

### k3sctl adaptation
- **Direct port** — generic YAML utility

---

## pkg/node/statusfunc.go — Node Status Functions

### k0sctl behavior
- Returns retry-compatible functions for status checking:
  - `KubeNodeReadyFunc(h)` — kubectl get node, check Ready condition
  - `K0sDynamicConfigReadyFunc(h)` — kubectl get events for reconciliation
  - `ScheduledEventsAfterFunc(h, since)` — check for Scheduled events after time
  - `HTTPStatusFunc(h, url, expected)` — HTTP status check
  - `ServiceRunningFunc(h, service)` — systemd service running
  - `ServiceStoppedFunc(h, service)` — systemd service stopped

### k3sctl adaptation
- `KubeNodeReadyFunc(h)` → adapt: use `k3s kubectl get node` on a server
  - For agents, the check runs on a server host (agents don't have kubectl)
  - The host parameter must be a server or we need a separate server reference
- `K0sDynamicConfigReadyFunc` → **Drop** — K3s has no dynamic config
- `ScheduledEventsAfterFunc(h, since)` → port as-is (generic kube events check)
- `HTTPStatusFunc(h, url, expected)` → port as-is (generic HTTP check)
- `ServiceRunningFunc(h, service)` → port as-is (generic systemd check)
- `ServiceStoppedFunc(h, service)` → port as-is (generic systemd check)
- Add `K3sNodeReadyFunc(serverHost, nodeName)` for checking agent readiness via server

---

## pkg/k3s/provider.go (was pkg/k0s/provider.go) — BinaryProvider Interface

### k0sctl interface
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

### k3sctl adaptation
- **Direct rename** — same interface, different package path
- `pkg/k3s/provider.go` with identical interface definitions
- Implementations in `pkg/k3s/binprovider/`

---

## pkg/k3s/binprovider/ — Binary Provider Implementations

### download.go (GitHub download)

#### k0sctl behavior
- `NewGitHub(h, installPath, target)` — downloads from k0s GitHub releases
- Uses `version.DownloadURL(osKind, arch)` to resolve URL
- `NewCustomURL(h, installPath, urlFor, target)` — user-provided URL

#### k3sctl adaptation
- `NewGitHub(h, installPath, target)` — download from K3s releases
- K3s download URL pattern:
  ```
  https://github.com/k3s-io/k3s/releases/download/{version}/k3s         # amd64
  https://github.com/k3s-io/k3s/releases/download/{version}/k3s-arm64   # arm64
  https://github.com/k3s-io/k3s/releases/download/{version}/k3s-armhf   # armhf
  ```
- Checksum URL pattern:
  ```
  https://github.com/k3s-io/k3s/releases/download/{version}/sha256sum-amd64.txt
  https://github.com/k3s-io/k3s/releases/download/{version}/sha256sum-arm64.txt
  https://github.com/k3s-io/k3s/releases/download/{version}/sha256sum-arm.txt
  ```
- Implement `k3sDownloadURL(version, arch)` function
- `NewCustomURL` — port as-is (generic URL)

### existing.go
- Provider that reports no upgrade needed (binary already exists)
- **Direct port**

### local_file.go
- Provider using a local file path on the host
- **Direct port**

### local_upload.go
- Provider that uploads from controller machine to remote
- Downloads binary locally first, then uploads via rig
- Needs K3s URL pattern for local download
- **Adapt** — use K3s download URLs

### host.go — Host interface for providers
- Subset of host behavior needed by providers
- **Direct port** with rename from K0s to K3s version accessors

### helpers.go — Provider helper functions
- `needsUpgrade(h, target, installed, running)` — version comparison logic
- `stageDownload(h, url, installPath, target)` — download to temp, set mode 0700
- **Adapt** — change paths reference but logic is the same

---

## pkg/download/download.go — Download Tracking

### k0sctl behavior
- `Record` struct: URLDigest, Destination, Size, ModTime, ContentLength, ETag, LastModified, Sha256
- `Tracker` struct: manages download state per host
  - `Needed(ctx, url, dest, checksum)` — check if download needed (HEAD request, compare ETag/size)
  - `Fetch(ctx, url, dest)` — download with resume support (3 attempts)
  - `Remember(record)` — cache download record
- Uses rig FS for remote file operations
- Cache validation via HTTP HEAD (ETag, Content-Length, Last-Modified)
- Resume support via Range headers

### k3sctl adaptation
- **Direct port** with minor adaptations:
  - Change cache directory name from `k0sctl` to `k3sctl` in XDG cache path
  - All download/caching logic is generic, not k0s-specific
  - The Tracker works with any URL, not just k0s URLs

---

## integration/github/github.go — GitHub API Client

### k0sctl behavior
- Queries GitHub releases for k0s versions
- Used by version resolution and update checking
- Rate-limited API access

### k3sctl adaptation
- Change to K3s repository: `k3s-io/k3s`
- K3s version format: `v1.XX.Y+k3s1` (note the `+k3s1` suffix)
- Also support K3s update channels API: `https://update.k3s.io/v1-release/channels`
- Parse release assets for binary URLs
- Handle K3s release naming conventions

---

## version/version.go — Build Version Info

### k0sctl behavior
- Build-time injected: Version, K0sVersion, HelperVersionOverride
- `versioninfo` integration for Go build info

### k3sctl adaptation
- Rename to K3sVersion
- No HelperVersionOverride needed (K3s doesn't have separate helper images)

---

## New: pkg/k3s/version.go — K3s Version Utilities

### Not in k0sctl (new for k3sctl)
- K3s version parsing: `v1.31.2+k3s1` → major, minor, patch, k3s revision
- Version comparison with K3s-specific format awareness
- The `+k3s1` suffix is significant and part of the version
- Consider wrapping or adapting `k0sproject/version` for K3s format
- OR implement simple semver+suffix comparison

---

## New: pkg/k3s/systemd.go — Systemd Unit Generation

### Not in k0sctl (new for k3sctl)
K0s has built-in `k0s install` that creates systemd units. K3s relies on install script.
Since we're pure Go, we need to generate systemd units:

```go
func GenerateServerUnit(opts ServerUnitOptions) string
func GenerateAgentUnit(opts AgentUnitOptions) string
func GenerateEnvFile(envVars map[string]string) string
```

Server unit template (generated as Go string, not a shell script):
```ini
[Unit]
Description=Lightweight Kubernetes
Documentation=https://k3s.io
Wants=network-online.target
After=network-online.target

[Install]
WantedBy=multi-user.target

[Service]
Type=notify
EnvironmentFile=-/etc/systemd/system/k3s.service.env
KillMode=process
Delegate=yes
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
Restart=always
RestartSec=5s
ExecStartPre=/bin/sh -c '! /usr/bin/systemctl is-enabled --quiet nm-cloud-setup.service 2>/dev/null'
ExecStart=/usr/local/bin/k3s server
```

This is a data template, not a shell script. It is a systemd unit file that systemd
interprets as configuration, similar to writing a YAML config file.

---

## Implementation Order

1. `pkg/retry/retry.go` (direct port)
2. `pkg/manifest/reader.go` (direct port)
3. `pkg/k3s/provider.go` (interface rename)
4. `pkg/k3s/version.go` (new — K3s version parsing)
5. `pkg/k3s/systemd.go` (new — unit generation)
6. `pkg/k3s/binprovider/` (all providers)
7. `pkg/download/download.go` (port with cache dir rename)
8. `pkg/node/statusfunc.go` (adapt for K3s commands)
9. `integration/github/github.go` (K3s repo + channels)
10. `version/version.go` (build info)
