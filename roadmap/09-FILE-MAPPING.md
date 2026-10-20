# Roadmap: Complete File-by-File Mapping (k0sctl → k3sctl)

## Legend
- **PORT**: Direct port, minimal changes (rename packages/imports)
- **ADAPT**: Requires K3s-specific changes in logic
- **NEW**: New file for K3s-specific functionality
- **OMIT**: Not needed for K3s

## Entry Point & Build
| k0sctl File | k3sctl File | Action |
|------------|------------|--------|
| main.go | main.go | PORT |
| version/version.go | version/version.go | PORT |
| go.mod | go.mod | NEW (k3sctl module path) |

## CLI Layer (cmd/)
| k0sctl File | k3sctl File | Action |
|------------|------------|--------|
| cmd/root.go | cmd/root.go | PORT |
| cmd/flags.go | cmd/flags.go | ADAPT (k3sctl config format) |
| cmd/apply.go | cmd/apply.go | PORT |
| cmd/init.go | cmd/init.go | ADAPT (k3s config template) |
| cmd/reset.go | cmd/reset.go | PORT |
| cmd/backup.go | cmd/backup.go | PORT |
| cmd/kubeconfig.go | cmd/kubeconfig.go | PORT |
| cmd/config_edit.go | cmd/config_edit.go | ADAPT (file-based config) |
| cmd/config_status.go | cmd/config_status.go | ADAPT (file-based config) |
| cmd/version.go | cmd/version.go | ADAPT (k3sctl releases) |
| cmd/completion.go | cmd/completion.go | PORT |
| cmd/logo.go | cmd/logo.go | NEW (k3sctl logo) |
| cmd/flags_test.go | cmd/flags_test.go | PORT |
| cmd/init_test.go | cmd/init_test.go | ADAPT |
| cmd/apply_test.go | cmd/apply_test.go | PORT |

## Action Layer (action/)
| k0sctl File | k3sctl File | Action |
|------------|------------|--------|
| action/apply.go | action/apply.go | ADAPT (K3s phase sequence) |
| action/reset.go | action/reset.go | ADAPT (K3s reset) |
| action/backup.go | action/backup.go | ADAPT (K3s backup) |
| action/kubeconfig.go | action/kubeconfig.go | PORT |
| action/config_edit.go | action/config_edit.go | ADAPT |
| action/config_status.go | action/config_status.go | ADAPT |

## Phase Layer (phase/)
| k0sctl File | k3sctl File | Action |
|------------|------------|--------|
| phase/manager.go | phase/manager.go | PORT |
| phase/manager_test.go | phase/manager_test.go | PORT |
| phase/generic_phase.go | phase/generic_phase.go | PORT |
| phase/connect.go | phase/connect.go | PORT |
| phase/connect_test.go | phase/connect_test.go | PORT |
| phase/disconnect.go | phase/disconnect.go | PORT |
| phase/detect_os.go | phase/detect_os.go | PORT |
| phase/lock.go | phase/lock.go | ADAPT (k3sctl lock path) |
| phase/unlock.go | phase/unlock.go | ADAPT |
| phase/prepare_hosts.go | phase/prepare_hosts.go | ADAPT |
| phase/gather_facts.go | phase/gather_facts.go | ADAPT |
| phase/gather_facts_test.go | phase/gather_facts_test.go | ADAPT |
| phase/validate_hosts.go | phase/validate_hosts.go | ADAPT |
| phase/validate_hosts_test.go | phase/validate_hosts_test.go | ADAPT |
| phase/validate_facts.go | phase/validate_facts.go | ADAPT |
| phase/validate_facts_test.go | phase/validate_facts_test.go | ADAPT |
| phase/run_hooks.go | phase/run_hooks.go | PORT |
| phase/daemon_reload.go | phase/daemon_reload.go | PORT |
| phase/downloads.go | phase/downloads.go | PORT |
| phase/uploadfiles.go | phase/uploadfiles.go | PORT |
| phase/uploadfiles_test.go | phase/uploadfiles_test.go | PORT |
| phase/default_k0s_version.go | phase/default_k3s_version.go | ADAPT (K3s version resolution) |
| phase/gather_k0s_facts.go | phase/gather_k3s_facts.go | ADAPT (K3s fact gathering) |
| phase/gather_k0s_facts_test.go | phase/gather_k3s_facts_test.go | ADAPT |
| phase/stage_binaries.go | phase/stage_binaries.go | PORT |
| phase/stage_binaries_test.go | phase/stage_binaries_test.go | PORT |
| phase/upload_k0s_test.go | phase/upload_k3s_test.go | ADAPT |
| phase/install_binaries.go | phase/install_binaries.go | ADAPT |
| phase/configure_k0s.go | phase/configure_k3s.go | ADAPT (K3s config format) |
| phase/configure_k0s_test.go | phase/configure_k3s_test.go | ADAPT |
| phase/initialize_k0s.go | phase/initialize_k3s.go | ADAPT (K3s server init) |
| phase/install_controllers.go | phase/install_servers.go | ADAPT (K3s server join) |
| phase/install_workers.go | phase/install_agents.go | ADAPT (K3s agent join) |
| phase/upgrade_controllers.go | phase/upgrade_servers.go | ADAPT |
| phase/upgrade_workers.go | phase/upgrade_agents.go | ADAPT |
| phase/reinstall.go | phase/reinstall.go | ADAPT |
| phase/reset_workers.go | phase/reset_agents.go | ADAPT |
| phase/reset_controllers.go | phase/reset_servers.go | ADAPT |
| phase/reset_leader.go | phase/reset_leader.go | ADAPT |
| phase/get_kubeconfig.go | phase/get_kubeconfig.go | ADAPT (read k3s.yaml) |
| phase/get_kubeconfig_test.go | phase/get_kubeconfig_test.go | ADAPT |
| phase/backup.go | phase/backup.go | ADAPT (K3s snapshot) |
| phase/restore.go | phase/restore.go | ADAPT (K3s restore) |
| phase/apply_manifests.go | phase/apply_manifests.go | ADAPT (k3s kubectl) |
| phase/validate_etcd_members.go | phase/validate_etcd_members.go | ADAPT (conditional) |
| phase/arm_prepare.go | phase/arm_prepare.go | ADAPT (cgroup check) |
| phase/ensure_join_token_workaround.go | — | OMIT |
| phase/ensure_join_token_workaround_test.go | — | OMIT |

## Configurer Layer (configurer/)
| k0sctl File | k3sctl File | Action |
|------------|------------|--------|
| configurer/interface.go | configurer/interface.go | ADAPT (K3s methods) |
| configurer/host.go | configurer/host.go | PORT |
| configurer/registry.go | configurer/registry.go | PORT |
| configurer/linux.go | configurer/linux.go | ADAPT (K3s paths) |
| configurer/windows.go | — | OMIT |
| configurer/windows/windows.go | — | OMIT |
| configurer/windows/windows_test.go | — | OMIT |
| configurer/windows_test.go | — | OMIT |
| configurer/linux/alpine.go | configurer/linux/alpine.go | ADAPT |
| configurer/linux/alpine_test.go | configurer/linux/alpine_test.go | ADAPT |
| configurer/linux/debian.go | configurer/linux/debian.go | ADAPT |
| configurer/linux/ubuntu.go | configurer/linux/ubuntu.go | ADAPT |
| configurer/linux/enterpriselinux.go | configurer/linux/enterpriselinux.go | ADAPT |
| configurer/linux/enterpriselinux/rhel.go | configurer/linux/enterpriselinux/rhel.go | ADAPT |
| configurer/linux/enterpriselinux/centos.go | configurer/linux/enterpriselinux/centos.go | ADAPT |
| configurer/linux/enterpriselinux/fedora.go | configurer/linux/enterpriselinux/fedora.go | ADAPT |
| configurer/linux/enterpriselinux/rocky.go | configurer/linux/enterpriselinux/rocky.go | ADAPT |
| configurer/linux/enterpriselinux/almalinux.go | configurer/linux/enterpriselinux/almalinux.go | ADAPT |
| configurer/linux/enterpriselinux/oracle.go | configurer/linux/enterpriselinux/oracle.go | ADAPT |
| configurer/linux/enterpriselinux/amazon.go | configurer/linux/enterpriselinux/amazon.go | ADAPT |
| configurer/linux/archlinux.go | configurer/linux/archlinux.go | ADAPT |
| configurer/linux/opensuse.go | configurer/linux/opensuse.go | ADAPT |
| configurer/linux/sles.go | configurer/linux/sles.go | ADAPT |
| configurer/linux/flatcar.go | configurer/linux/flatcar.go | ADAPT |
| configurer/linux/coreos.go | configurer/linux/coreos.go | ADAPT |
| configurer/linux/linux_test.go | configurer/linux/linux_test.go | ADAPT |

## API Types (pkg/apis/)
| k0sctl File | k3sctl File | Action |
|------------|------------|--------|
| pkg/apis/.../cluster.go | pkg/apis/.../cluster.go | ADAPT (K3s API version) |
| pkg/apis/.../cluster_test.go | pkg/apis/.../cluster_test.go | ADAPT |
| .../cluster/spec.go | .../cluster/spec.go | ADAPT (K3s spec) |
| .../cluster/spec_test.go | .../cluster/spec_test.go | ADAPT |
| .../cluster/host.go | .../cluster/host.go | ADAPT (K3s host methods) |
| .../cluster/host_test.go | .../cluster/host_test.go | ADAPT |
| .../cluster/host_posix_test.go | .../cluster/host_posix_test.go | ADAPT |
| .../cluster/hosts.go | .../cluster/hosts.go | ADAPT (Servers/Agents) |
| .../cluster/hosts_test.go | .../cluster/hosts_test.go | ADAPT |
| .../cluster/k0s.go | .../cluster/k3s.go | ADAPT (K3s config) |
| .../cluster/k0s_test.go | .../cluster/k3s_test.go | ADAPT |
| .../cluster/flags.go | .../cluster/flags.go | PORT |
| .../cluster/flags_test.go | .../cluster/flags_test.go | PORT |
| .../cluster/flags_quoting_test.go | .../cluster/flags_quoting_test.go | PORT |
| .../cluster/options.go | .../cluster/options.go | PORT |
| .../cluster/hook.go | .../cluster/hook.go | PORT |
| .../cluster/uploadfile.go | .../cluster/uploadfile.go | PORT |
| .../cluster/uploadfile_test.go | .../cluster/uploadfile_test.go | PORT |
| .../cluster/binary_providers.go | .../cluster/binary_providers.go | ADAPT |
| .../cluster/quote.go | .../cluster/quote.go | PORT |

## Supporting Packages (pkg/)
| k0sctl File | k3sctl File | Action |
|------------|------------|--------|
| pkg/retry/retry.go | pkg/retry/retry.go | PORT |
| pkg/retry/retry_test.go | pkg/retry/retry_test.go | PORT |
| pkg/download/download.go | pkg/download/download.go | ADAPT (cache path) |
| pkg/download/download_test.go | pkg/download/download_test.go | ADAPT |
| pkg/k0s/provider.go | pkg/k3s/provider.go | PORT |
| pkg/k0s/binprovider/host.go | pkg/k3s/binprovider/host.go | ADAPT |
| pkg/k0s/binprovider/helpers.go | pkg/k3s/binprovider/helpers.go | PORT |
| pkg/k0s/binprovider/download.go | pkg/k3s/binprovider/download.go | ADAPT (K3s URLs) |
| pkg/k0s/binprovider/local_upload.go | pkg/k3s/binprovider/local_upload.go | ADAPT (K3s cache) |
| pkg/k0s/binprovider/local_file.go | pkg/k3s/binprovider/local_file.go | PORT |
| pkg/k0s/binprovider/existing.go | pkg/k3s/binprovider/existing.go | PORT |
| pkg/manifest/reader.go | pkg/manifest/reader.go | PORT |
| pkg/manifest/reader_test.go | pkg/manifest/reader_test.go | PORT |
| pkg/node/statusfunc.go | pkg/node/statusfunc.go | ADAPT |
| pkg/node/statusfunc_test.go | pkg/node/statusfunc_test.go | ADAPT |

## Integration
| k0sctl File | k3sctl File | Action |
|------------|------------|--------|
| integration/github/github.go | integration/github/github.go | ADAPT (K3s releases) |

## Summary Count
- **PORT** (direct or minimal): ~40 files
- **ADAPT** (K3s-specific changes): ~60 files
- **NEW** (brand new): ~3 files
- **OMIT** (not needed): ~6 files (Windows support, join token workaround)
- **Total**: ~103 files to create
