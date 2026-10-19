# 04: Retry Package and Utility Packages

## `pkg/retry/retry.go`
- **Source**: `k0sctl/pkg/retry/retry.go` (~122 lines)
- **Changes**: Import paths only — this is entirely generic
- **Exported**:
  - `DefaultTimeout = 2 * time.Minute`
  - `Interval = 5 * time.Second`
  - `ErrAbort` — sentinel for non-retryable errors
  - `Context(ctx, f)` — retry at Interval until success or context cancel
  - `Timeout(ctx, timeout, f)` — Context with timeout
  - `WithDefaultTimeout(ctx, f)` — Timeout with DefaultTimeout
  - `Times(ctx, times, f)` — retry up to N times

### `pkg/retry/retry_test.go`
- **Source**: `k0sctl/pkg/retry/retry_test.go`
- **Changes**: Import paths only

## `pkg/node/statusfunc.go`
- **Source**: `k0sctl/pkg/node/statusfunc.go`
- **Changes**: Adapt for K3s service names and API patterns
  - `ServiceRunningFunc(h, serviceName)` — checks if systemd service is running
  - `HTTPStatusFunc(h, url, expected...)` — checks HTTP status codes
  - `KubeNodeReadyFunc(h)` — checks if node is Ready via kubectl
- K3s differences:
  - Service names: `k3s` (server) and `k3s-agent` (agent) instead of `k0scontroller`/`k0sworker`
  - kubectl: uses `k3s kubectl` instead of `k0s kubectl`

### `pkg/node/statusfunc_test.go`
- Adapt tests for K3s service names

## `pkg/download/download.go`
- **Source**: `k0sctl/pkg/download/download.go`
- **Changes**: Generic download utility — import paths only
- Provides HTTP download with progress, hash verification

### `pkg/download/download_test.go`
- Import paths only

## `pkg/manifest/reader.go`
- **Source**: `k0sctl/pkg/manifest/reader.go`
- **Changes**: Import paths, change `k0sctl.k0sproject.io/v1beta1` to `k3sctl.k3sproject.io/v1beta1` (or our own API version)
- Reads multi-document YAML, validates apiVersion/kind

### `pkg/manifest/reader_test.go`
- Adapt for k3sctl API version

## `integration/github/github.go`
- **Source**: `k0sctl/integration/github/github.go`
- **Changes**: Point to K3s GitHub releases instead of K0s
  - K3s releases: `github.com/k3s-io/k3s/releases`
  - Version channel: K3s uses `update.k3s.io/v1-release/channels` instead of `docs.k0sproject.io`
