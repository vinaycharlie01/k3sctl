# k3sctl

`k3sctl` is a declarative lifecycle management tool for [K3s](https://k3s.io)
clusters, created and inspired by [k0sctl](https://github.com/k0sproject/k0sctl). Point it
at a YAML file describing your hosts and it will install, upgrade, reconfigure,
back up and reset the cluster over SSH.

## Features

- Declarative cluster configuration (servers and agents in one file)
- Installs and upgrades K3s binaries from GitHub releases, a URL, a local file,
  or an existing installation on the host
- Generates server/agent configuration and systemd units
- Supports labels, taints, extra install flags, environment variables and
  arbitrary file uploads
- Applies additional Kubernetes manifests bundled in the configuration file
- etcd snapshots (`backup`), etcd member validation and restore (`restore`)
- Fetches a ready-to-use admin kubeconfig
- Idempotent `apply`: safe to re-run, only changes what drifted
- Rolling upgrades with drain/cordon for both servers and agents
- Configuration validation (`config validate`)
- Configuration diff (`config diff`) — preview what `apply` would change
- Air-gap install — upload pre-downloaded image bundles for offline clusters
- Dry-run mode (`--dry-run`)

Linux hosts only.

## Installation

From a release, download the binary for your platform and place it on your
`PATH`:

```sh
curl -sSfL -o k3sctl \
  https://github.com/vinaycharlie01/k3sctl/releases/latest/download/k3sctl_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m)
chmod +x k3sctl
sudo install -m 755 k3sctl /usr/local/bin/k3sctl
```

With Go:

```sh
go install github.com/vinaycharlie01/k3sctl@latest
```

Or build it yourself with [Task](https://taskfile.dev):

```sh
go install github.com/go-task/task/v3/cmd/task@latest

task build      # ./k3sctl
task install    # go install
task build-all  # cross-compiled binaries in ./dist
```

Run `task --list` for every available task.

## Quick start

```sh
k3sctl init --name demo --address 10.0.0.1 --agent --agent-address 10.0.0.2 > k3sctl.yaml
$EDITOR k3sctl.yaml
k3sctl apply --config k3sctl.yaml
k3sctl kubeconfig > kubeconfig.yaml
```

## Configuration

```yaml
apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: demo
spec:
  hosts:
    - role: server
      ssh:
        address: 10.0.0.1
        user: root
        port: 22
        keyPath: ~/.ssh/id_ed25519
      privateAddress: 10.0.0.1        # advertises this IP to the cluster
      installFlags:
        - --disable=traefik
      labels:
        node-role.example.com/edge: "true"
      taints:
        - dedicated=edge:NoSchedule
      environment:
        ETCD_UNSUPPORTED_ARCH: arm     # rendered into the systemd unit
      files:
        - src: ./configs/registries.yaml
          dst: /etc/rancher/k3s/registries.yaml
          perm: "0600"
      hooks:
        apply:
          before:
            - echo preparing
          after:
            - echo done

    - role: agent
      ssh:
        address: 10.0.0.2
        user: root
        port: 22
        keyPath: ~/.ssh/id_ed25519

  k3s:
    version: v1.30.0+k3s1
    config:                            # merged into config.yaml on every host
      write-kubeconfig-mode: "0644"
      disable:
        - servicelb
    extraArgs:
      - --kubelet-arg=eviction-hard=memory.available<100Mi

    etcdSnapshots:                     # automated etcd snapshot scheduling
      scheduled: true
      scheduleCron: "0 */12 * * *"
      retention: 5
      dir: /var/lib/rancher/k3s/server/db/snapshots
      compress: true
      s3:
        enabled: true
        endpoint: s3.amazonaws.com
        bucket: my-k3s-backups
        region: us-east-1
        folder: snapshots
        accessKey: AKIAIOSFODNN7EXAMPLE
        secretKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

    secretsEncrypt:                    # enable secrets encryption at rest
      enabled: true

    certRotate:                        # rotate K3s TLS certificates on apply
      enabled: true

    token:                             # explicit cluster tokens
      serverToken: my-server-secret
      agentToken: my-agent-secret

    preflight:                         # pre-flight checks before apply
      enabled: true
      checkCgroups: true
      checkModules: true
      checkPorts: true
      checkDisk: true
      minDiskGB: 10

    airgap:                            # air-gap (offline) installation
      enabled: true
      imageBundle: /path/to/k3s-airgap-images-amd64.tar.zst

  options:
    drain:
      enabled: true
      timeout: 5m
      gracePeriod: -1
      ignoreDaemonSets: true
      deleteEmptyDirData: true
```

### Host options

| Key | Description |
|---|---|
| `role` | `server` (default) or `agent` |
| `ssh` / `openSSH` / `localhost` | How to reach the host |
| `privateAddress` | Address used for node IP and API advertises |
| `installFlags` | Extra `k3s server`/`k3s agent` command line flags |
| `labels`, `taints` | Convenience shortcuts for `--node-label` / `--node-taint` |
| `environment` | Environment variables written to the host and the systemd unit |
| `files` | Files to upload (local path or URL) |
| `hooks` | Commands to run before/after `apply` |
| `dataDir` | Overrides the K3s data directory |
| `uploadBinary` | Upload a locally built binary instead of downloading |
| `k3sBinaryPath` | Use an existing binary on the host |
| `noFlannel` | Disable the flannel CNI |
| `os` | Override OS detection |

### K3s options

| Key | Description |
|---|---|
| `version` | K3s version to install (e.g. `v1.30.0+k3s1`) |
| `config` | Key-value map merged into each host's K3s `config.yaml` |
| `extraArgs` | Extra flags appended to the `k3s server`/`k3s agent` command |
| `etcdSnapshots` | Automated etcd snapshot configuration (see below) |
| `secretsEncrypt` | Enable K3s secrets encryption at rest |
| `certRotate` | Rotate TLS certificates on every `apply` |
| `token` | Set explicit server and agent join tokens |
| `preflight` | Pre-flight checks run before apply |
| `airgap` | Air-gap (offline) installation settings |

#### `etcdSnapshots`

| Key | Description |
|---|---|
| `scheduled` | Enable scheduled snapshots |
| `scheduleCron` | Cron expression for snapshot schedule |
| `retention` | Number of snapshots to retain |
| `dir` | Local directory for snapshots |
| `compress` | Compress snapshots |
| `s3.enabled` | Store snapshots in S3 |
| `s3.endpoint` | S3 endpoint |
| `s3.endpointCa` | Path to CA cert for S3 endpoint |
| `s3.skipSslVerify` | Skip TLS verification for S3 |
| `s3.accessKey` | S3 access key |
| `s3.secretKey` | S3 secret key |
| `s3.bucket` | S3 bucket name (required when S3 enabled) |
| `s3.region` | S3 region |
| `s3.folder` | Folder within the bucket |
| `s3.insecure` | Use HTTP instead of HTTPS |

#### `secretsEncrypt`

| Key | Description |
|---|---|
| `enabled` | Enable K3s secrets encryption at rest (`--secrets-encryption`) |

#### `certRotate`

| Key | Description |
|---|---|
| `enabled` | Run `k3s certificate rotate --force` on each server during `apply` and restart K3s |

#### `token`

| Key | Description |
|---|---|
| `serverToken` | Token used for server-to-server communication (`--token`) |
| `agentToken` | Token used for agent-to-server communication (`--agent-token`) |

#### `preflight`

| Key | Description |
|---|---|
| `enabled` | Enable pre-flight checks |
| `checkCgroups` | Verify cgroup v1/v2 controllers are available |
| `checkModules` | Verify required kernel modules are loaded (`br_netfilter`, `overlay`, `ip_tables`, `nf_conntrack`) |
| `checkPorts` | Verify required ports are not in use (6443, 10250, 2379, 2380 for servers; 10250 for agents) |
| `checkDisk` | Verify minimum disk space is available |
| `minDiskGB` | Minimum disk space in GB (default: 5) |

#### `airgap`

| Key | Description |
|---|---|
| `enabled` | Enable air-gap mode (skip downloading K3s images from the internet) |
| `imageBundle` | Local path to a K3s air-gap image bundle (e.g. `k3s-airgap-images-amd64.tar.zst`); uploaded to `<dataDir>/agent/images/` on every host |

### Additional manifests

Any other YAML documents in the configuration file are applied to the cluster
after installation:

```yaml
# ... Cluster document ...
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: example
data:
  key: value
```

Each document must be a valid Kubernetes manifest. Documents are passed to
`kubectl apply` on the server node.

## Commands

| Command | Description |
|---|---|
| `k3sctl apply` | Install or update the cluster |
| `k3sctl reset` | Remove K3s from the hosts |
| `k3sctl kubeconfig` | Print the admin kubeconfig |
| `k3sctl backup` | Create an etcd snapshot |
| `k3sctl restore` | Restore cluster from an etcd snapshot |
| `k3sctl config edit` | Edit the cluster configuration |
| `k3sctl config diff` | Show differences between desired config and actual cluster state |
| `k3sctl config validate` | Validate a cluster configuration file |
| `k3sctl config status` | Show the current cluster status |
| `k3sctl config status --deep` | Deep health check (API server, etcd, scheduler, node readiness) |
| `k3sctl etcd-snapshot list` | List etcd snapshots on servers |
| `k3sctl etcd-snapshot delete` | Delete an etcd snapshot by name |
| `k3sctl certificate check` | Check TLS certificate expiry dates |
| `k3sctl token list` | List cluster join tokens |
| `k3sctl node remove` | Remove a node from the cluster |
| `k3sctl init` | Print a configuration template |
| `k3sctl version` | Print version information |
| `k3sctl completion` | Print shell completion script |

Global flags: `--debug`, `--trace`, `--no-redact`.
`apply` additionally accepts `--dry-run`, `--no-wait`, `--no-drain`,
`--restore-from`, `--kubeconfig-out`, `--concurrency`,
`--concurrent-uploads` and `--disable-downgrade-check`.
`node remove` accepts `--no-drain` and `--force` (to remove the leader).
`restore` accepts `--from` (required snapshot path) and `--force` (skip confirmation).

## Roadmap

### Done

- Declarative cluster apply (install, upgrade, idempotent re-run)
- Rolling upgrades with drain/cordon for both servers and agents
- etcd backup (`backup`) and restore (`restore`) with confirmation prompt
- Configuration validation (`config validate`)
- Configuration diff (`config diff`)
- Deep cluster health checks (`config status --deep`)
- etcd snapshot management (`etcd-snapshot list`, `etcd-snapshot delete`)
- Certificate expiry checking (`certificate check`)
- Token management (`token list`)
- Node removal with drain (`node remove`)
- Dry-run mode, kubeconfig export, shell completions
- Air-gap install — upload image bundles for offline clusters
- Multi-server HA — quorum-safe batched server joining, etcd member removal on reset, stale member detection

### Next

- **Cluster import**: adopt an existing K3s cluster into k3sctl management
- **Registry mirror config**: first-class support for private registry mirrors
- **Plugin system**: hooks for custom pre/post-apply logic

## Development

```sh
task check      # gofmt + go vet + go test
task lint       # golangci-lint (see .golangci.yml)
task build-all  # linux/darwin, amd64/arm64/arm
docker build -t k3sctl .
```

Mocks are generated with [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter)
via `go generate` (`task generate`); do not hand-write fakes.

## License

Apache License 2.0. See [LICENSE](LICENSE).
