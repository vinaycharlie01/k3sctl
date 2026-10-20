# 10: Action Layer — Workflow Composition

## Overview

Actions compose phases into workflows. Each action creates an ordered list of phases and runs them through the Manager.

## `action/apply.go`
- **Source**: `k0sctl/action/apply.go` (~167 lines)
- **Changes**:
  - Phase names: K3s equivalents
  - `ApplyOptions` struct: same fields, rename K0s references
  - `NewApply()` phase list for k3sctl:

```go
Phases: phase.Phases{
    &phase.DefaultK3sVersion{},
    &phase.Connect{},
    &phase.DetectOS{},
    &phase.PrepareHosts{},
    &phase.GatherFacts{},
    &phase.ValidateHosts{},
    &phase.GatherK3sFacts{},
    &phase.ValidateFacts{SkipDowngradeCheck: opts.DisableDowngradeCheck},
    // ValidateEtcdMembers only if using embedded etcd
    &phase.StageBinaries{},
    &phase.UploadFiles{},
    &phase.InstallBinaries{},
    &phase.PrepareArm{},
    &phase.ConfigureK3s{},
    &phase.Restore{RestoreFrom: opts.RestoreFrom},
    &phase.RunHooks{Stage: "before", Action: "apply"},
    &phase.InitializeK3s{},
    &phase.InstallServers{},
    &phase.InstallAgents{},
    &phase.UpgradeServers{},
    &phase.UpgradeAgents{NoDrain: opts.NoDrain},
    &phase.Reinstall{},
    &phase.ResetAgents{NoDrain: opts.NoDrain},
    &phase.ResetServers{NoDrain: opts.NoDrain},
    &phase.RunHooks{Stage: "after", Action: "apply"},
    &phase.ApplyManifests{},
}
```

  - Success message: "k3s cluster version %s is now installed"
  - Kubeconfig tip: "k3sctl kubeconfig"
  - ConfigPaths default: "k3sctl.yaml"

## `action/reset.go`
- **Source**: `k0sctl/action/reset.go` (~78 lines)
- **Changes**:
  - Confirmation message: "Going to reset all of the hosts..."
  - Phase list:
    ```go
    &phase.DefaultK3sVersion{},
    &phase.Connect{},
    &phase.DetectOS{},
    lockPhase,
    &phase.PrepareHosts{},
    &phase.GatherFacts{SkipMachineIDs: true},
    &phase.GatherK3sFacts{},
    &phase.ResetAgents{NoDrain: true, NoDelete: true},
    &phase.ResetServers{NoDrain: true, NoDelete: true, NoLeave: true},
    &phase.ResetLeader{},
    &phase.DaemonReload{},
    &phase.Unlock{Cancel: lockPhase.Cancel},
    &phase.Disconnect{},
    ```

## `action/kubeconfig.go`
- **Source**: `k0sctl/action/kubeconfig.go` (~33 lines)
- **Changes**:
  - `K3sLeader()` instead of `K0sLeader()`
  - Phases: Connect → DetectOS → GetKubeconfig → Disconnect
  - Only connects to leader server node

## `action/config_edit.go`
- **Source**: `k0sctl/action/config_edit.go` (~103 lines)
- **Changes**:
  - K3s doesn't have `clusterconfig` CRD in kube-system
  - K3s config is a file on disk, not a Kubernetes resource
  - Approach for k3sctl: SSH to leader, edit `/etc/rancher/k3s/config.yaml` directly
  - Open in `$EDITOR`, diff, write back, restart K3s service
  - Alternatively: could read/write the config file remotely

## `action/config_status.go`
- **Source**: `k0sctl/action/config_status.go` (~41 lines)
- **Changes**:
  - K3s doesn't have config events in kube-system namespace
  - Could show: K3s version, cluster status, node status
  - Alternative: run `k3s kubectl get nodes -o wide` and show cluster info
  - Or: read `/etc/rancher/k3s/config.yaml` and display
