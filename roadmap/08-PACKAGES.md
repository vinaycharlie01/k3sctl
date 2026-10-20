# Roadmap: Supporting Packages

## pkg/retry/retry.go
- Context-based retry helpers
- `Context(ctx, f)` — retry at `Interval` (5s) until context done
- `Timeout(ctx, timeout, f)` — retry with deadline
- `WithDefaultTimeout(ctx, f)` — retry with 2 minute default
- `Times(ctx, n, f)` — retry N times
- `ErrAbort` — sentinel to stop retrying
- `DefaultTimeout` = 2min, `Interval` = 5s
- **k3sctl**: Port as-is (generic utility, not k0s-specific)

### pkg/retry/retry_test.go
- **k3sctl**: Port directly

## pkg/download/download.go
- URL download tracker with caching/deduplication
- `Tracker` struct: manages downloads per host
- `Record` struct: tracks downloaded files (URLDigest, Destination, Size, ModTime, ETag, LastModified, Sha256)
- Key methods:
  - `NewTracker(files, user, label)` — create tracker per host
  - `Needed(ctx, url, dst, wantSum)` — check if download needed (returns Verdict)
  - `Remember(ctx, url, dst, sum)` — record an existing file
  - `Fetch(ctx, url, dst, wantSum)` — download with tracking
- Features: resume support, checksum verification, HTTP HEAD comparison, ETag/Last-Modified tracking
- Records stored in user cache dir (`~/.cache/k0sctl/downloads/`)
- **k3sctl**: Port, change cache dir to `~/.cache/k3sctl/downloads/`

### pkg/download/download_test.go
- **k3sctl**: Port

## pkg/k0s/provider.go → pkg/k3s/provider.go
- `BinaryProvider` interface:
  - `NeedsUpgrade() bool`
  - `IsUpload() bool`
  - `Stage(ctx) (string, error)`
  - `CleanUp(ctx)`
- `BinaryCacher` interface (optional):
  - `BinaryCacheKey() (string, error)`
  - `EnsureCached(ctx) error`
- **k3sctl**: Port as-is — generic binary provider interface

## pkg/k0s/binprovider/ → pkg/k3s/binprovider/

### binprovider/host.go
- `Host` interface for binary providers:
  - `FS()`, `Sudo()`, `IsWindows()`
  - `Arch()`, `InstalledK0sVersion()`, `RunningK0sVersion()`
  - `Dir()`, `OSKind()`, `DownloadURL()`, `Touch()`, `DeleteFile()`, `SetFileMode()`
- **k3sctl**: Port, rename to `InstalledK3sVersion()`, `RunningK3sVersion()`

### binprovider/helpers.go
- `stagedFile` struct: embedded by providers for temp file management
- `needsUpgrade()` — shared upgrade check logic
- `stageTempPath()` — generate temp file path (preserves .exe on Windows)
- `stageUpload()` — upload local file to temp path on remote host
- `stageDownload()` — download URL to temp path on remote host
- **k3sctl**: Port as-is (generic staging helpers)

### binprovider/download.go
- `downloadProvider` struct: downloads binary directly on remote host
- `NewGitHub(host, installPath, target)` — fetches from GitHub releases
  - URL: `v.DownloadURL(osKind, arch)` (k0s version library)
- `NewCustomURL(host, installPath, urlFor, target)` — downloads from custom URL
- **k3sctl**: Port with K3s download URLs:
  - `NewGitHub`: construct URL as `https://github.com/k3s-io/k3s/releases/download/{version}/k3s` (amd64)
  - ARM64: `k3s-arm64`, ARM: `k3s-armhf`, s390x: `k3s-s390x`

### binprovider/local_upload.go
- `localUpload` struct: downloads to local cache, then uploads to host
- Uses `http.Client` with configured timeouts
- Cache path: `~/.cache/k0sctl/k0s/{os}/{arch}/k0s-{version}`
- `downloadToFile()` — HTTP GET with temp file + atomic rename
- Implements `BinaryCacher` interface
- **k3sctl**: Port with K3s cache paths:
  - Cache: `~/.cache/k3sctl/k3s/{os}/{arch}/k3s-{version}`
  - Download URL: K3s GitHub releases

### binprovider/local_file.go
- `localFile` struct: uploads developer-supplied binary from local machine
- `NewLocalFile(host, localPath, installPath, changed)` — uses `changed()` callback for upgrade detection
- **k3sctl**: Port as-is

### binprovider/existing.go
- `existing` struct: uses already-present binary on host
- `NeedsUpgrade()`: compares installed vs running version
- `Stage()`: returns ("", nil) — no staging needed
- **k3sctl**: Port as-is

## pkg/manifest/reader.go
- Multi-document YAML reader using `k8s.io/apimachinery/pkg/util/yaml`
- `ResourceDefinition` struct: APIVersion, Kind, Metadata.Name, Origin, Raw
- `Reader` struct with methods:
  - `Parse(io.Reader)`, `ParseString()`, `ParseBytes()`
  - `Resources()` — all parsed resources
  - `FilterResources(filter)` — filtered resources
  - `GetResources(apiVersion, kind)` — resources by type
- **k3sctl**: Port as-is (generic manifest reading)

### pkg/manifest/reader_test.go
- **k3sctl**: Port directly

## pkg/node/statusfunc.go
- Status-checking functions used with retry:
  - `KubeNodeReadyFunc(host)` — checks node Ready condition via `kubectl get node -o json`
  - `K0sDynamicConfigReadyFunc(host)` — checks dynamic config reconciliation events
  - `ScheduledEventsAfterFunc(host, since)` — checks for Scheduled events after time
  - `HTTPStatusFunc(host, url, expected...)` — HTTP status check
  - `ServiceRunningFunc(host, service)` — systemd service running check
  - `ServiceStoppedFunc(host, service)` — systemd service stopped check
- **k3sctl**:
  - `KubeNodeReadyFunc` — port, use `k3s kubectl` instead
  - Remove `K0sDynamicConfigReadyFunc` (K3s has no dynamic config)
  - `ScheduledEventsAfterFunc` — port with `k3s kubectl`
  - `HTTPStatusFunc` — port as-is
  - `ServiceRunningFunc` / `ServiceStoppedFunc` — port as-is

### pkg/node/statusfunc_test.go
- **k3sctl**: Port

## integration/github/github.go
- GitHub API integration for version checks:
  - `Release` struct: URL, TagName, PreRelease, Assets
  - `LatestK0sBinaryURL(arch, osKind, preok)` — latest k0s binary URL
  - `LatestK0sVersion(preok)` — latest k0s version string
  - `LatestRelease(preok)` — latest k0sctl release
  - `unmarshalURLBody()` — generic HTTP GET + JSON unmarshal
- **k3sctl**:
  - `LatestK3sBinaryURL(arch, osKind, preok)` — from k3s-io/k3s releases
  - `LatestK3sVersion(preok)` — latest K3s version
  - `LatestRelease(preok)` — latest k3sctl release from vinaycharlie01/k3sctl
  - K3s release tags: `v1.28.2+k3s1` format
  - K3s binary naming: `k3s` (amd64), `k3s-arm64`, `k3s-armhf`

## Implementation Order
1. `pkg/retry/` (direct port)
2. `pkg/download/` (port with cache path change)
3. `pkg/manifest/` (direct port)
4. `pkg/k3s/provider.go` (interface)
5. `pkg/k3s/binprovider/host.go` (interface)
6. `pkg/k3s/binprovider/helpers.go`
7. `pkg/k3s/binprovider/download.go` (K3s URLs)
8. `pkg/k3s/binprovider/local_upload.go` (K3s cache)
9. `pkg/k3s/binprovider/local_file.go` (direct port)
10. `pkg/k3s/binprovider/existing.go` (direct port)
11. `pkg/node/statusfunc.go` (K3s adaptations)
12. `integration/github/github.go` (K3s GitHub API)
