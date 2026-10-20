# 05: API Types — Cluster Config Model

## Overview

The API types define the YAML configuration format. This is the most significantly changed layer because K3s has different concepts, paths, and behaviors.

## API Version

- **k0sctl**: `k0sctl.k0sproject.io/v1beta1`
- **k3sctl**: `k3sctl.k3sproject.io/v1beta1` (our own namespace)

## Files to Create

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster.go`
- **Source**: `k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster.go` (~122 lines)
- **Changes**:
  - `APIVersion = "k3sctl.k3sproject.io/v1beta1"`
  - `ClusterMetadata` — same fields: Name, User, Kubeconfig, EtcdMembers, Manifests
  - `Cluster` struct — same: APIVersion, Kind, Metadata, Spec, Origin
  - `StorageType()` — K3s defaults to "sqlite" not "etcd"; check k3s config for `datastore-endpoint`
  - `Resolve(baseDir)` — same delegation to Spec

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/spec.go`
- **Source**: `k0sctl/...cluster/spec.go` (~273 lines)
- **Changes**:
  - `Spec` struct: `Hosts`, `K3s` (was `K0s`), `Options`
  - `K3sLeader()` → returns first server node (was K0sLeader)
  - `KubeAPIURL()` — same pattern
  - `InternalKubeAPIURL()` — same pattern
  - `NodeInternalKubeAPIURL(h)` — adapt for K3s
  - `APIPort()` — check k3s config for `https-listen-port` instead of `spec.api.port`
  - `CPLBVIPs()` — K3s has no native CPLB/keepalived; may drop or adapt for external LB
  - `clusterExternalAddress()` — check k3s config for `tls-san` instead of `spec.api.externalAddress`
  - `cplbConfig()` — likely not applicable for K3s; could be removed or adapted

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/host.go`
- **Source**: `k0sctl/...cluster/host.go` (~957 lines) — largest file
- **Significant changes**:

| Field/Method | k0sctl | k3sctl |
|-------------|--------|--------|
| `Role` | controller, worker, controller+worker, single | server, agent |
| `K0sBinaryPath` | field name | `K3sBinaryPath` |
| `K0sInstallPath` | field name | `K3sInstallPath` |
| `K0sDownloadURLOverride` | field name | `K3sDownloadURLOverride` |
| `K0sServiceName()` | k0scontroller/k0sworker | k3s/k3s-agent |
| `K0sRole()` | maps roles | `K3sRole()` — server/agent |
| `K0sInstallCommand()` | `k0s install controller ...` | Different: K3s uses systemd unit installation |
| `K0sResetCommand()` | `k0s reset` | `k3s-uninstall.sh` or `k3s-agent-uninstall.sh` |
| `K0sBackupCommand()` | `k0s backup` | Not applicable |
| `K0sRestoreCommand()` | `k0s restore` | Not applicable |
| `K0sBinaryVersion()` | `k0s version` | `k3s --version` |
| `K0sDataDir()` | dataDir or default | Same pattern, different default |
| `K0sInstallFlags()` | Builds install flags | Build K3s server/agent flags |
| `IsController()` | checks role | `IsServer()` — checks server role |
| `DrainNode()` | `k0s kubectl drain` | `k3s kubectl drain` |
| `Connect()` | same | same (rig v2) |
| `ResolveConfigurer()` | same | same (no Windows) |
| `IsWindows()` | checks WinRM | Always false / remove |

- **K3s Install Command differences**:
  - K3s server: `curl -sfL https://get.k3s.io | sh -` or pre-placed binary
  - K3s agent: `curl -sfL https://get.k3s.io | K3S_URL=... K3S_TOKEN=... sh -`
  - For k3sctl: we'll use pre-placed binary approach like k0sctl does
  - Install: place binary, create systemd unit, start service
  - Flags: `--data-dir`, `--token`, `--server` (for agents), `--config`

- **K3s Join Token differences**:
  - K3s stores server token at `/var/lib/rancher/k3s/server/node-token`
  - Agents use `K3S_TOKEN` env var or `--token` flag
  - No `k3s token create` command — read token from server node

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/hosts.go`
- **Source**: `k0sctl/...cluster/hosts.go`
- **Changes**:
  - `Controllers()` → `Servers()` — filter by server role
  - `Workers()` → `Agents()` — filter by agent role
  - `ParallelEach`, `BatchedParallelEach` — same
  - `Each` — same
  - `Filter` — same
  - `First` — same

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/k0s.go` → `k3s.go`
- **Source**: `k0sctl/...cluster/k0s.go` (~255 lines)
- **Completely rewritten for K3s**:

```go
type K3s struct {
    Version        *version.Version `yaml:"version,omitempty"`
    VersionChannel string           `yaml:"versionChannel,omitempty"`
    Config         dig.Mapping      `yaml:"config,omitempty"`  // K3s config.yaml content
    Metadata       K3sMetadata      `yaml:"-"`
}

type K3sMetadata struct {
    ClusterID        string
    VersionDefaulted bool
}
```

- `GenerateToken()` — K3s: read `/var/lib/rancher/k3s/server/node-token` from leader
- `ParseToken()` — K3s tokens are plain strings (not base64+gzip like k0s)
- `GetClusterID()` — same kubectl approach
- `NodeConfig()` — adapt for K3s config structure
- `K3sMinVersion` — set to earliest supported K3s version
- No `DynamicConfig` — K3s doesn't have this concept
- Version validation: K3s version format is `v1.30.0+k3s1`

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/options.go`
- **Source**: `k0sctl/...cluster/options.go` (~204 lines)
- **Changes**: Minimal — same Options, WaitOption, DrainOption, ConcurrencyOption, EvictTaintOption
- `DrainOption.ToKubectlArgs()` — same (kubectl drain is standard)
- All the UnmarshalYAML/Validate methods — same

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/flags.go`
- **Source**: `k0sctl/...cluster/flags.go`
- **Changes**: Import paths only — generic flag manipulation

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/hook.go`
- **Source**: `k0sctl/...cluster/hook.go`
- **Changes**: Import paths only — generic hook definitions

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/uploadfile.go`
- **Source**: `k0sctl/...cluster/uploadfile.go`
- **Changes**: Import paths only — generic file upload

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/quote.go`
- **Source**: `k0sctl/...cluster/quote.go`
- **Changes**: Import paths only — shell quoting utility

### `pkg/apis/k3sctl.k3sproject.io/v1beta1/cluster/binary_providers.go`
- **Source**: `k0sctl/...cluster/binary_providers.go`
- **Changes**: Adapt for K3s binary download URLs and version format

## Test Files

All test files from k0sctl adapted with K3s-specific assertions:
- `cluster_test.go`
- `host_test.go`, `host_posix_test.go`
- `hosts_test.go`
- `k3s_test.go` (was `k0s_test.go`)
- `flags_test.go`, `flags_quoting_test.go`
- `spec_test.go`
- `uploadfile_test.go`
- `options_test.go` (if exists)
