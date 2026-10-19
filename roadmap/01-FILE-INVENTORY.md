# k3sctl Roadmap — Complete File Inventory

Every Go source file from k0sctl mapped to its k3sctl equivalent.

## Legend
- **Direct** = Straightforward rename/rebrand
- **Adapt** = Needs K3s-specific logic changes
- **Major** = Significant rewrite for K3s
- **New** = No k0sctl equivalent, new for k3sctl
- **Drop** = Not applicable to K3s

---

## Entry Point

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `main.go` | `main.go` | Direct |

## version/

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `version/version.go` | `version/version.go` | Direct |

## cmd/ (CLI Layer)

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `cmd/root.go` | `cmd/root.go` | Direct — rename app to "k3sctl" |
| `cmd/flags.go` | `cmd/flags.go` | Adapt — change paths, defaults |
| `cmd/flags_test.go` | `cmd/flags_test.go` | Adapt — update apiVersion in fixtures |
| `cmd/apply.go` | `cmd/apply.go` | Adapt — K3s-specific flag names |
| `cmd/apply_test.go` | `cmd/apply_test.go` | Direct |
| `cmd/backup.go` | `cmd/backup.go` | Adapt — different backup mechanism |
| `cmd/completion.go` | `cmd/completion.go` | Direct — rename references |
| `cmd/config_edit.go` | `cmd/config_edit.go` | Major — K3s uses file-based config |
| `cmd/config_status.go` | `cmd/config_status.go` | Major — No dynamic config in K3s |
| `cmd/init.go` | `cmd/init.go` | Major — K3s config structure |
| `cmd/init_test.go` | `cmd/init_test.go` | Adapt |
| `cmd/kubeconfig.go` | `cmd/kubeconfig.go` | Adapt — different kubeconfig path |
| `cmd/logo.go` | `cmd/logo.go` | Direct — new ASCII art |
| `cmd/reset.go` | `cmd/reset.go` | Direct |
| `cmd/version.go` | `cmd/version.go` | Adapt — K3s version sources |

## action/ (Action Layer)

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `action/apply.go` | `action/apply.go` | Adapt — different phase list |
| `action/backup.go` | `action/backup.go` | Major — etcd snapshot based |
| `action/config_edit.go` | `action/config_edit.go` | Major — file-based config |
| `action/config_status.go` | `action/config_status.go` | Major — service status based |
| `action/kubeconfig.go` | `action/kubeconfig.go` | Adapt |
| `action/reset.go` | `action/reset.go` | Adapt — different reset steps |

## phase/ (Phase Layer)

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `phase/manager.go` | `phase/manager.go` | Direct |
| `phase/manager_test.go` | `phase/manager_test.go` | Direct |
| `phase/generic_phase.go` | `phase/generic_phase.go` | Direct |
| `phase/connect.go` | `phase/connect.go` | Direct |
| `phase/connect_test.go` | `phase/connect_test.go` | Direct |
| `phase/disconnect.go` | `phase/disconnect.go` | Direct |
| `phase/detect_os.go` | `phase/detect_os.go` | Direct |
| `phase/prepare_hosts.go` | `phase/prepare_hosts.go` | Adapt — K3s data dirs |
| `phase/gather_facts.go` | `phase/gather_facts.go` | Adapt |
| `phase/gather_facts_test.go` | `phase/gather_facts_test.go` | Adapt |
| `phase/gather_k0s_facts.go` | `phase/gather_k3s_facts.go` | Major — K3s version detection |
| `phase/gather_k0s_facts_test.go` | `phase/gather_k3s_facts_test.go` | Major |
| `phase/validate_hosts.go` | `phase/validate_hosts.go` | Adapt |
| `phase/validate_hosts_test.go` | `phase/validate_hosts_test.go` | Adapt |
| `phase/validate_facts.go` | `phase/validate_facts.go` | Adapt |
| `phase/validate_facts_test.go` | `phase/validate_facts_test.go` | Adapt |
| `phase/validate_etcd_members.go` | `phase/validate_etcd_members.go` | Adapt — only for HA etcd mode |
| `phase/default_k0s_version.go` | `phase/default_k3s_version.go` | Major — K3s GitHub releases |
| `phase/stage_binaries.go` | `phase/stage_binaries.go` | Adapt — K3s binary naming |
| `phase/stage_binaries_test.go` | `phase/stage_binaries_test.go` | Adapt |
| `phase/install_binaries.go` | `phase/install_binaries.go` | Adapt — K3s install path |
| `phase/configure_k0s.go` | `phase/configure_k3s.go` | Major — K3s config format |
| `phase/configure_k0s_test.go` | `phase/configure_k3s_test.go` | Major |
| `phase/initialize_k0s.go` | `phase/initialize_k3s.go` | Major — K3s server init |
| `phase/install_controllers.go` | `phase/install_servers.go` | Major — K3s server install |
| `phase/install_workers.go` | `phase/install_agents.go` | Major — K3s agent install |
| `phase/upgrade_controllers.go` | `phase/upgrade_servers.go` | Major — K3s upgrade |
| `phase/upgrade_workers.go` | `phase/upgrade_agents.go` | Major — K3s upgrade |
| `phase/reset_controllers.go` | `phase/reset_servers.go` | Major — K3s uninstall |
| `phase/reset_workers.go` | `phase/reset_agents.go` | Major — K3s uninstall |
| `phase/reset_leader.go` | `phase/reset_leader.go` | Major — K3s leader reset |
| `phase/reinstall.go` | `phase/reinstall.go` | Major — K3s reinstall |
| `phase/get_kubeconfig.go` | `phase/get_kubeconfig.go` | Major — /etc/rancher/k3s/k3s.yaml |
| `phase/get_kubeconfig_test.go` | `phase/get_kubeconfig_test.go` | Major |
| `phase/backup.go` | `phase/backup.go` | Major — etcd snapshot |
| `phase/restore.go` | `phase/restore.go` | Major — etcd snapshot restore |
| `phase/run_hooks.go` | `phase/run_hooks.go` | Direct |
| `phase/uploadfiles.go` | `phase/uploadfiles.go` | Direct |
| `phase/uploadfiles_test.go` | `phase/uploadfiles_test.go` | Direct |
| `phase/upload_k0s_test.go` | `phase/upload_k3s_test.go` | Adapt |
| `phase/downloads.go` | `phase/downloads.go` | Adapt — K3s URLs |
| `phase/lock.go` | `phase/lock.go` | Direct |
| `phase/unlock.go` | `phase/unlock.go` | Direct |
| `phase/daemon_reload.go` | `phase/daemon_reload.go` | Direct |
| `phase/arm_prepare.go` | `phase/arm_prepare.go` | Adapt — K3s ARM handling |
| `phase/apply_manifests.go` | `phase/apply_manifests.go` | Adapt — kubectl path |
| `phase/ensure_join_token_workaround.go` | Drop | K3s uses different token mechanism |
| `phase/ensure_join_token_workaround_test.go` | Drop | |

## configurer/ (OS Configurers)

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `configurer/interface.go` | `configurer/interface.go` | Major — K3s methods |
| `configurer/host.go` | `configurer/host.go` | Adapt |
| `configurer/registry.go` | `configurer/registry.go` | Direct |
| `configurer/linux.go` | `configurer/linux.go` | Major — K3s paths/commands |
| `configurer/windows.go` | `configurer/windows.go` | Major — K3s Windows |
| `configurer/windows_test.go` | `configurer/windows_test.go` | Adapt |
| `configurer/linux/alpine.go` | `configurer/linux/alpine.go` | Adapt |
| `configurer/linux/alpine_test.go` | `configurer/linux/alpine_test.go` | Adapt |
| `configurer/linux/archlinux.go` | `configurer/linux/archlinux.go` | Adapt |
| `configurer/linux/coreos.go` | `configurer/linux/coreos.go` | Adapt |
| `configurer/linux/debian.go` | `configurer/linux/debian.go` | Adapt |
| `configurer/linux/enterpriselinux/*.go` | `configurer/linux/enterpriselinux/*.go` | Adapt |
| `configurer/linux/flatcar.go` | `configurer/linux/flatcar.go` | Adapt |
| `configurer/linux/linux_test.go` | `configurer/linux/linux_test.go` | Adapt |
| `configurer/linux/opensuse.go` | `configurer/linux/opensuse.go` | Adapt |
| `configurer/linux/sles.go` | `configurer/linux/sles.go` | Adapt |
| `configurer/linux/ubuntu.go` | `configurer/linux/ubuntu.go` | Adapt |
| `configurer/windows/windows.go` | `configurer/windows/windows.go` | Major |
| `configurer/windows/windows_test.go` | `configurer/windows/windows_test.go` | Major |

## pkg/apis/ (API Types)

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `pkg/apis/.../cluster.go` | `pkg/apis/.../cluster.go` | Adapt — apiVersion |
| `pkg/apis/.../cluster_test.go` | `pkg/apis/.../cluster_test.go` | Adapt |
| `pkg/apis/.../cluster/spec.go` | `pkg/apis/.../cluster/spec.go` | Major — No CPLB |
| `pkg/apis/.../cluster/spec_test.go` | `pkg/apis/.../cluster/spec_test.go` | Adapt |
| `pkg/apis/.../cluster/host.go` | `pkg/apis/.../cluster/host.go` | Major — K3s paths/roles |
| `pkg/apis/.../cluster/host_test.go` | `pkg/apis/.../cluster/host_test.go` | Major |
| `pkg/apis/.../cluster/host_posix_test.go` | `pkg/apis/.../cluster/host_posix_test.go` | Adapt |
| `pkg/apis/.../cluster/hosts.go` | `pkg/apis/.../cluster/hosts.go` | Adapt — role names |
| `pkg/apis/.../cluster/hosts_test.go` | `pkg/apis/.../cluster/hosts_test.go` | Adapt |
| `pkg/apis/.../cluster/k0s.go` | `pkg/apis/.../cluster/k3s.go` | Major — K3s config |
| `pkg/apis/.../cluster/k0s_test.go` | `pkg/apis/.../cluster/k3s_test.go` | Major |
| `pkg/apis/.../cluster/flags.go` | `pkg/apis/.../cluster/flags.go` | Direct |
| `pkg/apis/.../cluster/flags_test.go` | `pkg/apis/.../cluster/flags_test.go` | Direct |
| `pkg/apis/.../cluster/flags_quoting_test.go` | `pkg/apis/.../cluster/flags_quoting_test.go` | Direct |
| `pkg/apis/.../cluster/options.go` | `pkg/apis/.../cluster/options.go` | Direct |
| `pkg/apis/.../cluster/hook.go` | `pkg/apis/.../cluster/hook.go` | Direct |
| `pkg/apis/.../cluster/quote.go` | `pkg/apis/.../cluster/quote.go` | Direct |
| `pkg/apis/.../cluster/uploadfile.go` | `pkg/apis/.../cluster/uploadfile.go` | Direct |
| `pkg/apis/.../cluster/uploadfile_test.go` | `pkg/apis/.../cluster/uploadfile_test.go` | Direct |
| `pkg/apis/.../cluster/binary_providers.go` | `pkg/apis/.../cluster/binary_providers.go` | Major — K3s URLs |

## pkg/ (Supporting Packages)

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `pkg/retry/retry.go` | `pkg/retry/retry.go` | Direct |
| `pkg/retry/retry_test.go` | `pkg/retry/retry_test.go` | Direct |
| `pkg/manifest/reader.go` | `pkg/manifest/reader.go` | Direct |
| `pkg/manifest/reader_test.go` | `pkg/manifest/reader_test.go` | Direct |
| `pkg/node/statusfunc.go` | `pkg/node/statusfunc.go` | Adapt — K3s commands |
| `pkg/node/statusfunc_test.go` | `pkg/node/statusfunc_test.go` | Adapt |
| `pkg/download/download.go` | `pkg/download/download.go` | Adapt — cache dir name |
| `pkg/download/download_test.go` | `pkg/download/download_test.go` | Adapt |
| `pkg/k0s/provider.go` | `pkg/k3s/provider.go` | Direct — rename |
| `pkg/k0s/binprovider/download.go` | `pkg/k3s/binprovider/download.go` | Major — K3s URLs |
| `pkg/k0s/binprovider/existing.go` | `pkg/k3s/binprovider/existing.go` | Direct |
| `pkg/k0s/binprovider/helpers.go` | `pkg/k3s/binprovider/helpers.go` | Adapt |
| `pkg/k0s/binprovider/host.go` | `pkg/k3s/binprovider/host.go` | Direct |
| `pkg/k0s/binprovider/local_file.go` | `pkg/k3s/binprovider/local_file.go` | Direct |
| `pkg/k0s/binprovider/local_upload.go` | `pkg/k3s/binprovider/local_upload.go` | Major — K3s URL pattern |

## integration/ (External Integrations)

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `integration/github/github.go` | `integration/github/github.go` | Major — K3s repo URLs |

## Build & Config Files

| k0sctl File | k3sctl File | Effort |
|-------------|-------------|--------|
| `Makefile` | `Makefile` | Adapt — rename targets |
| `Dockerfile` | `Dockerfile` | Adapt |
| `go.mod` | `go.mod` | Adapt — new module path |
| `.golangci.yml` | `.golangci.yml` | Direct |
| `.gitignore` | `.gitignore` | Direct |
| `LICENSE` | `LICENSE` | Adapt — new copyright |

## Total File Count

- **Go source files**: ~90
- **Direct port**: ~25
- **Adapt (moderate changes)**: ~30
- **Major rewrite**: ~30
- **Drop (not applicable)**: ~2-3
- **New (K3s-specific)**: ~2-3
