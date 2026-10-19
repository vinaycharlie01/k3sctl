# Roadmap: Action Layer

## Source Files (k0sctl)

### action/apply.go
- `ApplyOptions` struct: Manager, DisableDowngradeCheck, NoDrain, NoWait, RestoreFrom, KubeconfigOut, KubeconfigAPIAddress, KubeconfigUser, KubeconfigCluster, EvictTaint
- `NewApply(opts)` creates phase sequence:
  1. DefaultK0sVersion
  2. Connect
  3. DetectOS
  4. PrepareHosts
  5. GatherFacts
  6. ValidateHosts
  7. GatherK0sFacts
  8. ValidateFacts
  9. ValidateEtcdMembers
  10. EnsureJoinTokenWorkaround
  11. StageBinaries
  12. UploadFiles
  13. InstallBinaries
  14. PrepareArm
  15. ConfigureK0s
  16. Restore
  17. RunHooks (before, apply)
  18. InitializeK0s
  19. InstallControllers
  20. InstallWorkers
  21. UpgradeControllers
  22. UpgradeWorkers
  23. Reinstall
  24. ResetWorkers
  25. ResetControllers
  26. RunHooks (after, apply)
  27. ApplyManifests
  28. [GetKubeconfig] (optional)
  29. Disconnect
- `Run()`: executes phases, prints duration, suggests kubeconfig command
- **k3sctl adaptation**:
  - Replace `DefaultK0sVersion` → `DefaultK3sVersion`
  - Replace `GatherK0sFacts` → `GatherK3sFacts`
  - Remove `ValidateEtcdMembers` (not applicable for default SQLite; add for etcd mode)
  - Remove `EnsureJoinTokenWorkaround` (K3s has different token mechanism)
  - Replace `ConfigureK0s` → `ConfigureK3s`
  - Replace `InitializeK0s` → `InitializeK3s`
  - Replace `InstallControllers` → `InstallServers`
  - Replace `InstallWorkers` → `InstallAgents`
  - Replace `UpgradeControllers` → `UpgradeServers`
  - Replace `UpgradeWorkers` → `UpgradeAgents`
  - Remove `PrepareArm` (K3s handles ARM natively)
  - Adapt `Restore` for `k3s etcd-snapshot restore`

### action/reset.go
- Confirms with user (survey prompt or --force)
- Marks all hosts Reset=true
- Phase sequence: DefaultK0sVersion→Connect→DetectOS→Lock→PrepareHosts→GatherFacts→GatherK0sFacts→ResetWorkers→ResetControllers→ResetLeader→DaemonReload→Unlock→Disconnect
- **k3sctl**: Replace with K3s reset (stop service, remove data dirs, remove binary)

### action/backup.go
- Phase sequence: DefaultK0sVersion→Connect→DetectOS→Lock→PrepareHosts→GatherFacts→GatherK0sFacts→Backup→Unlock→Disconnect
- **k3sctl**: Use `k3s etcd-snapshot save` on server node

### action/kubeconfig.go
- Filters hosts to only leader controller
- Phase sequence: Connect→DetectOS→GetKubeconfig→Disconnect
- **k3sctl**: Read `/etc/rancher/k3s/k3s.yaml` from first server

### action/config_edit.go
- Dynamic config editing via k0s API
- **k3sctl**: Edit `/etc/rancher/k3s/config.yaml` on remote host

### action/config_status.go
- Dynamic config status retrieval
- **k3sctl**: Show current config from remote host

## Implementation Order
1. `action/apply.go` (core workflow)
2. `action/reset.go`
3. `action/kubeconfig.go`
4. `action/backup.go`
5. `action/config_edit.go` + `action/config_status.go`
