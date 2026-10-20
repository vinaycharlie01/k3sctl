# 12: Binary Providers — K3s Binary Acquisition

## Overview

Binary providers implement the strategy pattern for acquiring K3s binaries on target hosts. The interface and five implementations are adapted from k0sctl's `pkg/k0s/` package.

## `pkg/k3s/provider.go`
- **Source**: `k0sctl/pkg/k0s/provider.go` (44 lines)
- **Changes**:
  - Package: `k3s` instead of `k0s`
  - `BinaryProvider` interface: identical methods
    - `NeedsUpgrade() bool`
    - `IsUpload() bool`
    - `Stage(ctx context.Context) (string, error)`
    - `CleanUp(ctx context.Context)`
  - `BinaryCacher` interface: identical
    - `BinaryCacheKey() (string, error)`
    - `EnsureCached(ctx context.Context) error`
  - Module path: `github.com/vinaycharlie01/k3sctl/pkg/k3s`

## `pkg/k3s/binprovider/host.go`
- **Source**: `k0sctl/pkg/k0s/binprovider/host.go` (42 lines)
- **Changes**:
  - `Host` interface: same methods
    - `FS()`, `Sudo()`, `Arch()`, `Dir()`, `OSKind()`, `DownloadURL()`, `Touch()`, `DeleteFile()`, `SetFileMode()`
  - **Remove**: `IsWindows() bool` — K3s is Linux-only
  - Remove: `InstalledK0sVersion()` → `InstalledK3sVersion() *version.Version`
  - Remove: `RunningK0sVersion()` → `RunningK3sVersion() *version.Version`
  - Note: version type may use `github.com/k0sproject/version` or a k3s-specific version type (K3s uses `v1.30.0+k3s1` format)

## `pkg/k3s/binprovider/helpers.go`
- **Source**: `k0sctl/pkg/k0s/binprovider/helpers.go` (148 lines)
- **Changes**:
  - `stagedFile` struct: same, holds host + tmpPath
  - `CleanUp()`: same, removes temp binary
  - `needsUpgrade()`: same logic, compare target vs binary vs running
  - `stageTempPath()`: **remove Windows .exe handling**, Linux-only
  - `stageUpload()`: same logic (dir creation, upload, chmod 0755)
    - Remove `!h.IsWindows()` guard on chmod — always set
  - `stageDownload()`: same logic (dir creation, download, chmod)
    - Remove Windows guard
  - Log messages: "k3s" instead of "k0s"

## `pkg/k3s/binprovider/download.go`
- **Source**: `k0sctl/pkg/k0s/binprovider/download.go` (72 lines)
- **Changes**:
  - `downloadProvider` struct: same fields
  - `NewGitHub()` — **Significant change**: K3s download URL pattern differs from K0s
    - K0s: `v.DownloadURL(osKind, arch)` on `github.com/k0sproject/version`
    - K3s: `https://github.com/k3s-io/k3s/releases/download/{version}/k3s` for amd64
    - K3s: `https://github.com/k3s-io/k3s/releases/download/{version}/k3s-arm64` for arm64
    - K3s: `https://github.com/k3s-io/k3s/releases/download/{version}/k3s-armhf` for arm
    - Version format: `v1.30.0+k3s1` (note the `+` must be URL-encoded as `%2B`)
    - No Windows binary
    - No `s390x` binary (K3s doesn't support it)
    - May need custom version type or URL builder
  - `NewCustomURL()`: same pattern, user-provided URL resolver
  - `urlFor` function: implement K3s-specific arch-to-binary-suffix mapping

### K3s Download URL Builder
```go
func k3sDownloadURL(version, arch string) string {
    // K3s uses + in version which must be URL-encoded
    encodedVersion := strings.ReplaceAll(version, "+", "%2B")
    base := fmt.Sprintf("https://github.com/k3s-io/k3s/releases/download/%s/k3s", encodedVersion)
    switch arch {
    case "arm64", "aarch64":
        return base + "-arm64"
    case "arm", "armv7l":
        return base + "-armhf"
    default: // amd64
        return base
    }
}
```

## `pkg/k3s/binprovider/local_upload.go`
- **Source**: `k0sctl/pkg/k0s/binprovider/local_upload.go` (202 lines)
- **Changes**:
  - `localUpload` struct: same
  - `downloadHTTPClient`: same HTTP client config
  - `cacheFilePath()`: path changes:
    - k0sctl: `k0sctl/k0s/{os}/{arch}/k0s-{version}{ext}`
    - k3sctl: `k3sctl/k3s/linux/{arch}/k3s-{version}` — no `.exe`, no Windows
  - `BinaryCacheKey()`: same logic, remove `OSKind` (always linux)
  - `EnsureCached()`: download URL uses K3s URL builder instead of `v.DownloadURL()`
  - `downloadToFile()`: identical (HTTP download with atomic rename)
  - `Stage()`: same (upload from local cache to remote)
  - `NewLocalUpload()`: same constructor

## `pkg/k3s/binprovider/local_file.go`
- **Source**: `k0sctl/pkg/k0s/binprovider/local_file.go` (40 lines)
- **Changes**: Import paths only — `k3s` package reference
  - `localFile` struct: same
  - `NewLocalFile()`: same

## `pkg/k3s/binprovider/existing.go`
- **Source**: `k0sctl/pkg/k0s/binprovider/existing.go` (38 lines)
- **Changes**:
  - `InstalledK0sVersion()` → `InstalledK3sVersion()`
  - `RunningK0sVersion()` → `RunningK3sVersion()`
  - Log messages: "k3s" instead of "k0s"

## K3s Version Handling

K3s versions use `+k3s1` suffix (e.g., `v1.30.0+k3s1`). Options:
1. **Use `github.com/k0sproject/version`**: may work if it handles `+` suffix generically
2. **Custom version type**: parse K3s version strings specifically
3. **String-based**: simpler, just use string comparison for versions

Recommendation: Start with string-based version comparison, potentially wrap in a thin type for validation. The `+` in the version string requires URL encoding for downloads.

## Test Files

### `pkg/k3s/binprovider/download_test.go`
- Test K3s URL generation for amd64, arm64, armhf
- Test URL encoding of `+` in version strings

### `pkg/k3s/binprovider/helpers_test.go`
- Test `needsUpgrade()` logic
- Test `stageTempPath()` without Windows paths
