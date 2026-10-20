# k3sctl

`k3sctl` is a declarative lifecycle management tool for [K3s](https://k3s.io)
clusters, modeled on [k0sctl](https://github.com/k0sproject/k0sctl). Point it
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
- etcd snapshots (`backup`), etcd member validation and restore
- Fetches a ready-to-use admin kubeconfig
- Idempotent `apply`: safe to re-run, only changes what drifted
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
| `k3sctl config edit` | Edit the cluster configuration |
| `k3sctl config status` | Show the current cluster status |
| `k3sctl init` | Print a configuration template |
| `k3sctl version` | Print version information |
| `k3sctl completion` | Print shell completion script |

Global flags: `--debug`, `--trace`, `--no-redact`.
`apply` additionally accepts `--dry-run`, `--no-wait`, `--no-drain`,
`--restore-from`, `--kubeconfig-out`, `--concurrency`,
`--concurrent-uploads` and `--disable-downgrade-check`.

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
