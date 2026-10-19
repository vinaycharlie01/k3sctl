# k3sctl Roadmap — Implementation Plan

## Development Phases

### Phase 1: Foundation (Build Skeleton)

**Goal**: Project compiles, basic structure in place.

1. **go.mod** — Initialize module `github.com/vinaycharlie01/k3sctl`
2. **version/version.go** — Build-time version info
3. **main.go** — Entry point
4. **Makefile** — Build targets (k3sctl binary, cross-compile)
5. **.gitignore** — Standard Go ignores
6. **LICENSE** — MIT license

### Phase 2: Core Infrastructure (Generic Packages)

**Goal**: Reusable packages that don't depend on K3s specifics.

1. **pkg/retry/retry.go** — Retry helpers (direct port)
2. **pkg/retry/retry_test.go** — Tests
3. **pkg/manifest/reader.go** — YAML manifest reader (direct port)
4. **pkg/manifest/reader_test.go** — Tests
5. **pkg/download/download.go** — Download tracker (adapt cache dir)
6. **pkg/download/download_test.go** — Tests

### Phase 3: K3s Version Handling

**Goal**: K3s-specific version parsing, comparison, and resolution.

1. **pkg/k3sversion/version.go** — K3s version type with +k3s suffix handling
2. **pkg/k3sversion/version_test.go** — Tests
3. **pkg/k3sversion/channel.go** — K3s update channel API client
4. **pkg/k3sversion/channel_test.go** — Tests
5. **integration/github/github.go** — GitHub release lookups for K3s and k3sctl

### Phase 4: API Types

**Goal**: Complete configuration schema.

1. **pkg/apis/k3sctl.k3s.io/v1beta1/cluster.go** — Cluster top-level type
2. **pkg/apis/.../cluster/k3s.go** — K3s config type
3. **pkg/apis/.../cluster/spec.go** — Spec with hosts and K3s config
4. **pkg/apis/.../cluster/host.go** — Host type with K3s roles/paths
5. **pkg/apis/.../cluster/hosts.go** — Hosts collection
6. **pkg/apis/.../cluster/flags.go** — Flag builder (direct port)
7. **pkg/apis/.../cluster/quote.go** — Shell quoting (direct port)
8. **pkg/apis/.../cluster/options.go** — Drain/upgrade options
9. **pkg/apis/.../cluster/hook.go** — Lifecycle hooks
10. **pkg/apis/.../cluster/uploadfile.go** — File upload specs
11. **pkg/apis/.../cluster/binary_providers.go** — Binary provider selection
12. **Tests for all above**

### Phase 5: Configurer System

**Goal**: OS-specific behavior for all supported Linux distributions.

1. **configurer/interface.go** — Configurer interface with K3s methods
2. **configurer/registry.go** — OS configurer registry
3. **configurer/host.go** — Base host configurer
4. **configurer/linux.go** — Linux base with K3s paths/commands
5. **configurer/windows.go** — Windows stub (experimental)
6. **Linux distro configurers** (all in configurer/linux/):
   - alpine.go, archlinux.go, coreos.go, debian.go, ubuntu.go
   - flatcar.go, opensuse.go, sles.go
   - enterpriselinux/: base, almalinux, amazon, centos, fedora, oracle, rhel, rocky
7. **Tests for configurers**

### Phase 6: Binary Providers

**Goal**: Binary acquisition strategies for K3s.

1. **pkg/k3s/provider.go** — BinaryProvider interface
2. **pkg/k3s/binprovider/host.go** — Host interface for providers
3. **pkg/k3s/binprovider/helpers.go** — Staging helpers
4. **pkg/k3s/binprovider/download.go** — GitHub download provider
5. **pkg/k3s/binprovider/local_upload.go** — Local cache + upload
6. **pkg/k3s/binprovider/local_file.go** — User-supplied binary
7. **pkg/k3s/binprovider/existing.go** — Pre-installed binary
8. **Tests**

### Phase 7: Phase Manager & Generic Phases

**Goal**: Phase orchestration and non-K3s-specific phases.

1. **phase/manager.go** — Phase manager (direct port)
2. **phase/manager_test.go** — Tests
3. **phase/generic_phase.go** — Base phase type
4. **phase/connect.go** — SSH connection
5. **phase/disconnect.go** — SSH disconnect
6. **phase/detect_os.go** — OS detection
7. **phase/lock.go** — Host locking
8. **phase/unlock.go** — Host unlocking
9. **phase/daemon_reload.go** — Systemd daemon-reload
10. **phase/run_hooks.go** — Hook execution
11. **phase/uploadfiles.go** — File uploads
12. **phase/downloads.go** — Download orchestration
13. **Tests for all**

### Phase 8: K3s Lifecycle Phases

**Goal**: K3s-specific lifecycle management.

1. **phase/default_k3s_version.go** — Resolve default K3s version
2. **phase/prepare_hosts.go** — Host preparation (K3s dirs)
3. **phase/gather_facts.go** — System fact gathering
4. **phase/gather_k3s_facts.go** — K3s-specific facts
5. **phase/validate_hosts.go** — Host validation
6. **phase/validate_facts.go** — Fact validation
7. **phase/validate_etcd_members.go** — Etcd validation (HA mode only)
8. **phase/stage_binaries.go** — Binary staging
9. **phase/install_binaries.go** — Binary installation
10. **phase/configure_k3s.go** — K3s configuration generation/upload
11. **phase/initialize_k3s.go** — First server initialization
12. **phase/install_servers.go** — Additional server installation
13. **phase/install_agents.go** — Agent installation
14. **phase/upgrade_servers.go** — Server upgrade (rolling)
15. **phase/upgrade_agents.go** — Agent upgrade (rolling)
16. **phase/reset_servers.go** — Server reset/uninstall
17. **phase/reset_agents.go** — Agent reset/uninstall
18. **phase/reset_leader.go** — Leader reset
19. **phase/reinstall.go** — Service reinstall on flag change
20. **phase/get_kubeconfig.go** — Kubeconfig retrieval
21. **phase/backup.go** — Etcd snapshot backup
22. **phase/restore.go** — Snapshot restore
23. **phase/arm_prepare.go** — ARM setup
24. **phase/apply_manifests.go** — Manifest application
25. **pkg/node/statusfunc.go** — Node status polling
26. **Tests for all**

### Phase 9: Actions (Workflow Composition)

**Goal**: High-level workflows combining phases.

1. **action/apply.go** — Full cluster apply
2. **action/reset.go** — Cluster reset
3. **action/backup.go** — Cluster backup
4. **action/kubeconfig.go** — Kubeconfig extraction
5. **action/config_edit.go** — Remote config editing
6. **action/config_status.go** — Config/service status

### Phase 10: CLI Layer

**Goal**: Complete command-line interface.

1. **cmd/root.go** — App factory with signal handling
2. **cmd/flags.go** — Shared flags and helpers
3. **cmd/apply.go** — Apply command
4. **cmd/reset.go** — Reset command
5. **cmd/backup.go** — Backup command
6. **cmd/kubeconfig.go** — Kubeconfig command
7. **cmd/init.go** — Config template generation
8. **cmd/config_edit.go** — Config edit command
9. **cmd/config_status.go** — Config status command
10. **cmd/version.go** — Version command
11. **cmd/completion.go** — Shell completions
12. **cmd/logo.go** — ASCII art branding
13. **Tests for CLI**

### Phase 11: Build & Release

1. **Makefile** — Full build targets
2. **Dockerfile** — Container image
3. **.golangci.yml** — Linting config
4. **.github/workflows/** — CI/CD (if needed)

---

## Testing Strategy

### Unit Tests
- Every package gets `*_test.go` files
- Use `testify` for assertions
- Mock SSH connections for host-level tests
- Test K3s version parsing extensively
- Test config generation/parsing

### Integration Points to Test
- Config YAML → Cluster type → Phase execution
- Binary provider selection logic
- Configurer resolution for each OS
- Version comparison and upgrade detection
- Kubeconfig manipulation
- Flag building and quoting

### Build Verification
```bash
go build ./...        # Compiles
go test ./...         # All tests pass
go vet ./...          # No vet issues
gofmt -l .            # Formatting clean
golangci-lint run     # Linting clean
go test -race ./...   # No race conditions
```

---

## File Count Estimate

| Category | Files |
|----------|-------|
| Entry + version | 2 |
| cmd/ | ~14 (incl. tests) |
| action/ | 6 |
| phase/ | ~40 (incl. tests) |
| configurer/ | ~20 |
| pkg/apis/ | ~20 (incl. tests) |
| pkg/k3s/ | ~8 |
| pkg/k3sversion/ | ~4 |
| pkg/ (other) | ~10 |
| integration/ | 1 |
| Build files | ~5 |
| **Total** | **~130 files** |

---

## Critical Implementation Notes

### 1. No Shell Scripts
All K3s management logic must be pure Go:
- Systemd unit file generation: Go templates or string building
- Service management: via rig's service abstraction
- File operations: via rig's filesystem abstraction
- Process management: via rig's exec
- Network operations: Go net package
- Package installation: via configurer's package manager methods

### 2. K3s Binary Naming Convention
```
k3s           → linux/amd64
k3s-arm64     → linux/arm64
k3s-armhf     → linux/arm (armv7)
k3s-s390x     → linux/s390x
```
URL encoding: `+` in version tags becomes `%2B` in GitHub release URLs.

### 3. K3s Token Mechanism
- Server creates token at: `/var/lib/rancher/k3s/server/node-token`
- Token format: `K10<ca-hash>::server:<password>`
- Agents and additional servers use this token to join
- No `k3s token create` needed (unlike k0s)
- Read token after first server starts, distribute to joining nodes

### 4. K3s Config File Format
K3s reads config from `/etc/rancher/k3s/config.yaml`:
```yaml
# All k3s CLI flags work as YAML keys (remove -- prefix, use kebab-case)
write-kubeconfig-mode: "0644"
tls-san:
  - my-host.example.com
cluster-init: true           # for first server in HA etcd mode
token: my-secret-token       # for joining nodes
server: https://leader:6443  # for joining servers/agents
disable:
  - traefik
node-label:
  - "role=worker"
```

### 5. K3s Datastore Modes
1. **SQLite** (default, single server): No extra config needed
2. **Embedded etcd** (HA): `--cluster-init` on first server
3. **External datastore**: `--datastore-endpoint` pointing to external DB

k3sctl should detect/support all three modes.
