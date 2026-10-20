# 14: Build System, CI, and Release

## Makefile
- **Source**: `k0sctl/Makefile` (128 lines)
- **Changes**:
  - Binary name: `k3sctl` instead of `k0sctl`
  - LD_FLAGS: `github.com/vinaycharlie01/k3sctl/version.Environment`
  - Build targets: same cross-platform approach
  - **Remove**: `k3sctl-win-amd64.exe` — no Windows support for K3s
  - Keep: linux-amd64, linux-arm64, linux-arm, linux-riscv64, darwin-amd64, darwin-arm64
  - Docker image: `ghcr.io/vinaycharlie01/k3sctl` (if applicable)
  - Smoke test targets: adapt for k3sctl smoke tests
  - Lint: same golangci-lint integration
  - Test: `go test -v ./...`

### k3sctl Makefile Targets
```makefile
k3sctl:              # Build for local platform
build-all:           # Cross-compile all platforms
test:                # Run unit tests
lint:                # Run golangci-lint
clean:               # Remove build artifacts
install:             # Install to /usr/local/bin
```

## Build Flags
```makefile
LD_FLAGS = -s -w \
  -X github.com/vinaycharlie01/k3sctl/version.Environment=$(ENVIRONMENT) \
  -X github.com/carlmjohnson/versioninfo.Revision=$(GIT_COMMIT) \
  -X github.com/carlmjohnson/versioninfo.Version=$(TAG_NAME)

BUILD_FLAGS = -trimpath -a \
  -tags "netgo,osusergo,static_build" \
  -installsuffix netgo \
  -ldflags "$(LD_FLAGS) -extldflags '-static'"
```

## GitHub Actions Workflows

### `.github/workflows/go.yml` — Unit Tests
- **Source**: `k0sctl/.github/workflows/go.yml`
- **Changes**:
  - Trigger: PRs touching `**.go`, `go.mod`, `go.sum`, `Taskfile.yml`
  - Matrix: ubuntu-latest, macos-latest (remove windows-latest)
  - Steps: checkout → setup-go → `go test -v ./...`
  - Build job: `task build-all`

### `.github/workflows/golangci-lint.yml` — Linting
- **Source**: `k0sctl/.github/workflows/golangci-lint.yml`
- **Changes**: Import paths in config, same golangci-lint version

### `.github/workflows/release.yml` — Release
- **Source**: `k0sctl/.github/workflows/release.yml`
- **Changes**:
  - Trigger: push tags `v*`
  - Build: `task build-all`
  - Upload artifacts: `bin/k3sctl-*`
  - Docker image build: adapt for k3sctl
  - GitHub release: draft with checksums

### `.github/workflows/smoke.yml` — Smoke Tests (Future)
- **Source**: `k0sctl/.github/workflows/smoke.yml`
- **Changes**: Adapt for k3sctl smoke tests when implemented
- Matrix of OS images for testing

### `.github/workflows/codeql-analysis.yml` — Security
- **Source**: `k0sctl/.github/workflows/codeql-analysis.yml`
- **Changes**: Same CodeQL analysis, adapt language config

## Dockerfile
- **Source**: `k0sctl/Dockerfile`
- **Changes**:
  - Base: same multi-stage build
  - Binary: `k3sctl` instead of `k0sctl`
  - Build args: same (ENVIRONMENT, GIT_COMMIT, TAG_NAME)
  - Entrypoint: `["/k3sctl"]`

## `.golangci.yml`
- **Source**: `k0sctl/.golangci.yml`
- **Changes**: Import paths in any linter exceptions, same rules

## Implementation Priority

### Phase 1 (MVP Build)
1. `Makefile` with `k3sctl`, `test`, `lint`, `build-all` targets
2. `.golangci.yml` linter config
3. `go.mod` with correct module path and dependencies

### Phase 2 (CI)
1. `.github/workflows/go.yml` — unit tests
2. `.github/workflows/golangci-lint.yml` — linting
3. `.github/workflows/release.yml` — release pipeline

### Phase 3 (Smoke Tests)
1. `smoke-test/` infrastructure
2. `.github/workflows/smoke.yml`

## Cross-Platform Build Matrix

| Target | GOOS | GOARCH | Binary Name |
|---|---|---|---|
| Linux amd64 | linux | amd64 | k3sctl-linux-amd64 |
| Linux arm64 | linux | arm64 | k3sctl-linux-arm64 |
| Linux arm | linux | arm | k3sctl-linux-arm |
| Linux riscv64 | linux | riscv64 | k3sctl-linux-riscv64 |
| macOS amd64 | darwin | amd64 | k3sctl-darwin-amd64 |
| macOS arm64 | darwin | arm64 | k3sctl-darwin-arm64 |

Note: No Windows build — K3s is Linux-only for the server/agent, and the CLI tool targets Linux/macOS users.
