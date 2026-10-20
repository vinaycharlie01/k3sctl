# 13: Testing Strategy

## Overview

k3sctl follows k0sctl's testing approach: hermetic unit tests with mocks/fakes (no real hosts), behavior-named tests, and smoke tests for integration validation.

## Unit Test File Mapping

### Phase Tests

| k0sctl Test File | k3sctl Test File | Changes |
|---|---|---|
| `phase/manager_test.go` | `phase/manager_test.go` | Import paths only |
| `phase/connect_test.go` | `phase/connect_test.go` | Import paths only |
| `phase/validate_hosts_test.go` | `phase/validate_hosts_test.go` | Adapt role names (server/agent) |
| `phase/validate_facts_test.go` | `phase/validate_facts_test.go` | Adapt K3s version format |
| `phase/gather_facts_test.go` | `phase/gather_facts_test.go` | Adapt host fact fields |
| `phase/gather_k0s_facts_test.go` | `phase/gather_k3s_facts_test.go` | K3s version parsing, service detection |
| `phase/configure_k0s_test.go` | `phase/configure_k3s_test.go` | K3s YAML config format |
| `phase/get_kubeconfig_test.go` | `phase/get_kubeconfig_test.go` | File read vs CLI command |
| `phase/stage_binaries_test.go` | `phase/stage_binaries_test.go` | K3s binary URLs |
| `phase/uploadfiles_test.go` | `phase/uploadfiles_test.go` | Import paths only |
| `phase/upload_k0s_test.go` | — | Removed (merged into stage_binaries) |
| `phase/ensure_join_token_workaround_test.go` | — | Removed (K0s-specific) |

### API Type Tests

| k0sctl Test File | k3sctl Test File | Changes |
|---|---|---|
| `pkg/apis/.../cluster_test.go` | `pkg/apis/.../cluster_test.go` | API version, storage type |
| `pkg/apis/.../cluster/k0s_test.go` | `pkg/apis/.../cluster/k3s_test.go` | Token format, version parsing |
| `pkg/apis/.../cluster/host_test.go` | `pkg/apis/.../cluster/host_test.go` | Role names, service names, K3s commands |
| `pkg/apis/.../cluster/host_posix_test.go` | `pkg/apis/.../cluster/host_test.go` | Merge (no Windows split needed) |
| `pkg/apis/.../cluster/hosts_test.go` | `pkg/apis/.../cluster/hosts_test.go` | Adapt role-based methods |
| `pkg/apis/.../cluster/spec_test.go` | `pkg/apis/.../cluster/spec_test.go` | K3sLeader, K3s fields |
| `pkg/apis/.../cluster/flags_test.go` | `pkg/apis/.../cluster/flags_test.go` | Import paths only |
| `pkg/apis/.../cluster/flags_quoting_test.go` | `pkg/apis/.../cluster/flags_quoting_test.go` | Import paths only |
| `pkg/apis/.../cluster/uploadfile_test.go` | `pkg/apis/.../cluster/uploadfile_test.go` | Import paths only |

### Configurer Tests

| k0sctl Test File | k3sctl Test File | Changes |
|---|---|---|
| `configurer/linux/linux_test.go` | `configurer/linux/linux_test.go` | K3s paths, service names |
| `configurer/linux/alpine_test.go` | `configurer/linux/alpine_test.go` | Import paths, K3s specifics |
| `configurer/windows/windows_test.go` | — | Removed (no Windows) |
| `configurer/windows_test.go` | — | Removed (no Windows) |

### CLI Tests

| k0sctl Test File | k3sctl Test File | Changes |
|---|---|---|
| `cmd/apply_test.go` | `cmd/apply_test.go` | k3sctl output format |
| `cmd/flags_test.go` | `cmd/flags_test.go` | k3sctl.yaml config file |
| `cmd/init_test.go` | `cmd/init_test.go` | K3s default config format |

### Package Tests

| k0sctl Test File | k3sctl Test File | Changes |
|---|---|---|
| `pkg/retry/retry_test.go` | `pkg/retry/retry_test.go` | Import paths only |
| `pkg/download/download_test.go` | `pkg/download/download_test.go` | Import paths only |
| `pkg/node/statusfunc_test.go` | `pkg/node/statusfunc_test.go` | Import paths only |
| `pkg/manifest/reader_test.go` | `pkg/manifest/reader_test.go` | Import paths only |

## New Tests (No k0sctl Equivalent)

### K3s-Specific Tests
- `pkg/k3s/version_test.go` — K3s version parsing (`v1.30.0+k3s1` format)
- `pkg/k3s/binprovider/download_test.go` — K3s GitHub URL generation
- `phase/default_k3s_version_test.go` — K3s channel API parsing
- `configurer/linux/k3s_paths_test.go` — K3s-specific paths across distros

## Testing Conventions

1. **Naming**: `TestMethodName_scenario` (e.g., `TestValidateHosts_DuplicateAddresses`)
2. **Assertions**: `github.com/stretchr/testify` (assert and require)
3. **Deep equality**: `github.com/go-test/deep` where needed
4. **Mocking**: Interface-based mocks, no external mock frameworks required
5. **Hermetic**: No real SSH connections, no network calls, no host-specific config
6. **Build tags**: No `_test` build tags needed (tests in same package)

## Smoke Tests

k0sctl has extensive smoke tests in `smoke-test/` using bootloose (container-based fake VMs). For k3sctl v1:

### Phase 1: Unit Tests Only
- Focus on comprehensive unit test coverage
- Mock SSH/rig interactions via interfaces

### Phase 2: Smoke Tests (Future)
- Adapt k0sctl's bootloose-based smoke test infrastructure
- Key smoke tests to port:
  - `smoke-basic` — basic cluster create/destroy
  - `smoke-upgrade` — version upgrade flow
  - `smoke-reset` — cluster reset
  - `smoke-dryrun` — dry-run mode verification
  - `smoke-files` — file upload functionality
  - `smoke-init` — init command output

### Smoke Test Infrastructure
- Uses bootloose (containerized SSH-accessible "VMs")
- Each smoke test: create cluster → verify → cleanup
- Matrix: multiple Linux distros (Alpine, Ubuntu, CentOS, etc.)
- Same Makefile-driven approach as k0sctl
