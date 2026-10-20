# Roadmap: Implementation Plan

## Phase 1: Project Skeleton & Core Infrastructure
**Goal**: Compilable Go project with CLI skeleton

### 1.1 Project Setup
- [ ] `go.mod` with module path `github.com/vinaycharlie01/k3sctl`
- [ ] `main.go` entry point
- [ ] `version/version.go` with LDFLAGS
- [ ] `Makefile` for build/test/lint
- [ ] `.gitignore` for Go projects

### 1.2 Retry & Utility Packages
- [ ] `pkg/retry/retry.go` — direct port
- [ ] `pkg/retry/retry_test.go`
- [ ] `pkg/manifest/reader.go` — direct port
- [ ] `pkg/manifest/reader_test.go`

### 1.3 CLI Framework
- [ ] `cmd/root.go` — CLI app skeleton with all subcommands
- [ ] `cmd/flags.go` — flag definitions and config loading
- [ ] `cmd/logo.go` — k3sctl ASCII art
- [ ] `cmd/completion.go` — shell completion
- [ ] `cmd/version.go` — version display

## Phase 2: API Types & Configuration
**Goal**: Full config data model with validation

### 2.1 Core Types
- [ ] `pkg/apis/.../cluster/quote.go` — direct port
- [ ] `pkg/apis/.../cluster/flags.go` — direct port
- [ ] `pkg/apis/.../cluster/flags_test.go`
- [ ] `pkg/apis/.../cluster/flags_quoting_test.go`
- [ ] `pkg/apis/.../cluster/hook.go` — direct port
- [ ] `pkg/apis/.../cluster/uploadfile.go` — direct port
- [ ] `pkg/apis/.../cluster/uploadfile_test.go`
- [ ] `pkg/apis/.../cluster/options.go` — direct port
- [ ] `pkg/apis/.../cluster/k3s.go` — K3s config type (adapted from k0s.go)
- [ ] `pkg/apis/.../cluster/k3s_test.go`

### 2.2 Host & Cluster Types
- [ ] `pkg/apis/.../cluster/host.go` — K3s host type (major adaptation)
- [ ] `pkg/apis/.../cluster/host_test.go`
- [ ] `pkg/apis/.../cluster/host_posix_test.go`
- [ ] `pkg/apis/.../cluster/hosts.go` — with Servers()/Agents()
- [ ] `pkg/apis/.../cluster/hosts_test.go`
- [ ] `pkg/apis/.../cluster/spec.go` — K3s spec
- [ ] `pkg/apis/.../cluster/spec_test.go`
- [ ] `pkg/apis/.../cluster.go` — top-level Cluster type
- [ ] `pkg/apis/.../cluster_test.go`

### 2.3 Init Command
- [ ] `cmd/init.go` — generate k3sctl.yaml template
- [ ] `cmd/init_test.go`

## Phase 3: Configurer Layer
**Goal**: OS-specific behavior for all supported distros

### 3.1 Core Configurer
- [ ] `configurer/interface.go` — K3s configurer interface
- [ ] `configurer/host.go` — direct port
- [ ] `configurer/registry.go` — direct port
- [ ] `configurer/linux.go` — K3s default paths

### 3.2 Linux Distro Modules
- [ ] `configurer/linux/debian.go`
- [ ] `configurer/linux/ubuntu.go`
- [ ] `configurer/linux/enterpriselinux.go`
- [ ] `configurer/linux/enterpriselinux/rhel.go`
- [ ] `configurer/linux/enterpriselinux/centos.go`
- [ ] `configurer/linux/enterpriselinux/rocky.go`
- [ ] `configurer/linux/enterpriselinux/almalinux.go`
- [ ] `configurer/linux/enterpriselinux/fedora.go`
- [ ] `configurer/linux/enterpriselinux/amazon.go`
- [ ] `configurer/linux/enterpriselinux/oracle.go`
- [ ] `configurer/linux/alpine.go`
- [ ] `configurer/linux/archlinux.go`
- [ ] `configurer/linux/opensuse.go`
- [ ] `configurer/linux/sles.go`
- [ ] `configurer/linux/flatcar.go`
- [ ] `configurer/linux/coreos.go`
- [ ] `configurer/linux/linux_test.go`
- [ ] `configurer/linux/alpine_test.go`

## Phase 4: Phase Manager & Infrastructure Phases
**Goal**: Phase execution framework and connectivity

### 4.1 Phase Manager
- [ ] `phase/manager.go` — direct port
- [ ] `phase/manager_test.go`
- [ ] `phase/generic_phase.go` — direct port

### 4.2 Connectivity Phases
- [ ] `phase/connect.go` — direct port
- [ ] `phase/connect_test.go`
- [ ] `phase/disconnect.go` — direct port
- [ ] `phase/detect_os.go`
- [ ] `phase/lock.go` + `phase/unlock.go`
- [ ] `phase/prepare_hosts.go`
- [ ] `phase/run_hooks.go`
- [ ] `phase/daemon_reload.go`

### 4.3 Fact Gathering
- [ ] `phase/gather_facts.go`
- [ ] `phase/gather_facts_test.go`
- [ ] `phase/validate_hosts.go`
- [ ] `phase/validate_hosts_test.go`

## Phase 5: Binary Management & Downloads
**Goal**: K3s binary acquisition and staging

### 5.1 Binary Provider
- [ ] `pkg/k3s/provider.go` — interface
- [ ] `pkg/k3s/binprovider/host.go` — interface
- [ ] `pkg/k3s/binprovider/helpers.go`
- [ ] `pkg/k3s/binprovider/download.go` — K3s GitHub download
- [ ] `pkg/k3s/binprovider/local_upload.go` — K3s local cache
- [ ] `pkg/k3s/binprovider/local_file.go`
- [ ] `pkg/k3s/binprovider/existing.go`
- [ ] `pkg/apis/.../cluster/binary_providers.go`

### 5.2 Download Tracker
- [ ] `pkg/download/download.go`
- [ ] `pkg/download/download_test.go`

### 5.3 GitHub Integration
- [ ] `integration/github/github.go` — K3s releases

### 5.4 Binary Phases
- [ ] `phase/default_k3s_version.go`
- [ ] `phase/stage_binaries.go`
- [ ] `phase/stage_binaries_test.go`
- [ ] `phase/install_binaries.go`
- [ ] `phase/downloads.go`
- [ ] `phase/uploadfiles.go`
- [ ] `phase/uploadfiles_test.go`

## Phase 6: K3s Cluster Operations
**Goal**: Core cluster lifecycle (apply flow)

### 6.1 K3s Facts & Validation
- [ ] `phase/gather_k3s_facts.go`
- [ ] `phase/gather_k3s_facts_test.go`
- [ ] `phase/validate_facts.go`
- [ ] `phase/validate_facts_test.go`
- [ ] `phase/validate_etcd_members.go` (conditional)

### 6.2 K3s Configuration & Init
- [ ] `phase/configure_k3s.go`
- [ ] `phase/configure_k3s_test.go`
- [ ] `phase/initialize_k3s.go`

### 6.3 Node Installation
- [ ] `phase/install_servers.go`
- [ ] `phase/install_agents.go`
- [ ] `pkg/node/statusfunc.go`
- [ ] `pkg/node/statusfunc_test.go`

### 6.4 Upgrade Phases
- [ ] `phase/upgrade_servers.go`
- [ ] `phase/upgrade_agents.go`
- [ ] `phase/reinstall.go`

### 6.5 Apply Action
- [ ] `action/apply.go` — complete apply workflow
- [ ] `cmd/apply.go` — apply command wiring
- [ ] `cmd/apply_test.go`

## Phase 7: Reset, Backup, Kubeconfig
**Goal**: Remaining cluster operations

### 7.1 Reset
- [ ] `phase/reset_agents.go`
- [ ] `phase/reset_servers.go`
- [ ] `phase/reset_leader.go`
- [ ] `action/reset.go`
- [ ] `cmd/reset.go`

### 7.2 Backup & Restore
- [ ] `phase/backup.go`
- [ ] `phase/restore.go`
- [ ] `action/backup.go`
- [ ] `cmd/backup.go`

### 7.3 Kubeconfig
- [ ] `phase/get_kubeconfig.go`
- [ ] `phase/get_kubeconfig_test.go`
- [ ] `action/kubeconfig.go`
- [ ] `cmd/kubeconfig.go`

### 7.4 Manifests & Config
- [ ] `phase/apply_manifests.go`
- [ ] `action/config_edit.go`
- [ ] `action/config_status.go`
- [ ] `cmd/config_edit.go`
- [ ] `cmd/config_status.go`

### 7.5 ARM Support
- [ ] `phase/arm_prepare.go`

## Phase 8: Testing & Quality
**Goal**: Complete test coverage and quality gates

### 8.1 Quality Gates
- [ ] `go test ./...` passes
- [ ] `gofmt` clean
- [ ] `go vet ./...` clean
- [ ] `golangci-lint` clean
- [ ] Race detection: `go test -race ./...`

### 8.2 Final Audit
- [ ] No .sh files in repository
- [ ] No embedded shell in Go strings (no `sh -c`, `bash -c`)
- [ ] All exec uses explicit argument arrays
- [ ] SSH via rig only
- [ ] All types are strongly typed Go structs

## Dependency Graph

```
Phase 1 (skeleton)
  └─> Phase 2 (types) ─> Phase 3 (configurer)
        └─> Phase 4 (manager + infra phases)
              └─> Phase 5 (binary management)
                    └─> Phase 6 (cluster operations)
                          └─> Phase 7 (reset/backup/kubeconfig)
                                └─> Phase 8 (testing & audit)
```

## Estimated File Count by Phase
| Phase | New Files | Tests | Total |
|-------|----------|-------|-------|
| 1. Skeleton | 7 | 2 | 9 |
| 2. Types | 12 | 9 | 21 |
| 3. Configurer | 18 | 3 | 21 |
| 4. Manager | 10 | 4 | 14 |
| 5. Binary | 12 | 3 | 15 |
| 6. Operations | 11 | 5 | 16 |
| 7. Rest | 12 | 1 | 13 |
| 8. Quality | 0 | 0 | 0 |
| **Total** | **82** | **27** | **~109** |
