# k3sctl Roadmap — phase/ Analysis

## Phase Manager (phase/manager.go)

The Manager orchestrates an ordered list of phases. Core behavior:
- `Run()` iterates through phases, calling `ShouldRun()` then `Run()`
- `DryMsg()` prints what would happen without executing
- Supports `Wet()` flag for dry-run mode
- `CleanUp()` runs cleanup on all phases in reverse order
- Concurrency controlled by Manager.Concurrency field
- `parallelDo()` / `BatchedParallelEach()` for concurrent host operations

**k3sctl**: Direct port — no K3s-specific changes needed.

---

## phase/generic_phase.go — Base Phase

Provides GenericPhase base struct with host filtering helpers:
- `GetHostsFilteredByRole(hosts, role)` — filter by role
- `IsWet()` — checks dry-run state

**k3sctl**: Direct port. Role constants change (controller→server, worker→agent).

---

## phase/connect.go — SSH Connection

Establishes SSH connections to all hosts via rig.
- Opens connections concurrently
- Sets up client configuration
- Tests connectivity

**k3sctl**: Direct port.

---

## phase/disconnect.go — SSH Disconnect

Closes all SSH connections.

**k3sctl**: Direct port.

---

## phase/detect_os.go — OS Detection

Resolves each host's configurer by detecting the OS:
- Calls `configurer.ByHost(h)` to find matching OS configurer
- Sets `h.Configurer` field

**k3sctl**: Direct port — configurer resolution is the same pattern.

---

## phase/prepare_hosts.go — Host Preparation

Prepares hosts for k0s operations:
- Creates data directories
- Sets up required permissions
- Validates prerequisites

**k3sctl changes**:
- Data directory: `/var/lib/k0s` → `/var/lib/rancher/k3s`
- Config directory: `/etc/k0s` → `/etc/rancher/k3s`
- May need to create `/etc/rancher/k3s/` directory

---

## phase/gather_facts.go — Fact Gathering

Gathers system facts from hosts:
- System info (hostname, OS, arch)
- Network interfaces and addresses
- Private/public IP resolution
- Machine ID

**k3sctl**: Mostly direct port. Same system facts needed.

---

## phase/gather_k0s_facts.go → gather_k3s_facts.go

Gathers K0s-specific facts:
- Installed k0s version (`k0s version`)
- Running k0s version (from service status)
- k0s service status
- k0s configuration status
- etcd member status

**k3sctl MAJOR rewrite**:
- Installed K3s version: `k3s --version` → parse output
- Running K3s version: `k3s --version` or check process
- K3s service status: `systemctl status k3s` / `systemctl status k3s-agent`
- K3s config: read from `/etc/rancher/k3s/config.yaml`
- Token: read from `/var/lib/rancher/k3s/server/node-token`
- Cluster ID: via kubeconfig + kubectl
- No dynamic config status (K3s doesn't have reconciliation events)

---

## phase/validate_hosts.go — Host Validation

Validates host configurations:
- Checks for duplicate addresses
- Validates roles
- Checks SSH connectivity
- Validates OS compatibility

**k3sctl changes**:
- Role validation: server/agent instead of controller/worker
- K3s-specific OS compatibility checks
- Validate K3s prerequisites (container runtime, iptables, etc.)

---

## phase/validate_facts.go — Fact Validation

Validates gathered facts:
- K0s version compatibility
- Downgrade detection
- Configuration consistency

**k3sctl changes**:
- K3s minimum version check
- K3s version format validation (v1.x.y+k3s1)
- Downgrade detection for K3s

---

## phase/validate_etcd_members.go — Etcd Validation

Validates etcd cluster membership consistency.

**k3sctl**: Only applicable when K3s uses etcd (--cluster-init / --server).
Skip for SQLite mode. When etcd is used, K3s has different etcd
management commands.

---

## phase/default_k0s_version.go → default_k3s_version.go

Resolves default K0s version when not specified in config:
- Fetches latest from k0s release channel
- Sets `config.Spec.K0s.Version`

**k3sctl MAJOR rewrite**:
- Fetch latest K3s version from GitHub releases API (k3s-io/k3s)
- Parse K3s version format: `v1.31.0+k3s1`
- Use "stable" channel: `https://update.k3s.io/v1-release/channels/stable`
- Set `config.Spec.K3s.Version`

---

## phase/stage_binaries.go — Binary Staging

Stages K0s binaries on hosts:
- Determines binary provider per host (GitHub, custom URL, local file, upload)
- Downloads/uploads binary to temp location
- Handles concurrent uploads with separate concurrency limit

**k3sctl changes**:
- K3s binary URL pattern is different from K0s
- K3s binary names: `k3s` (amd64), `k3s-arm64`, `k3s-armhf`
- Binary providers adapted for K3s download URLs
- Install path: `/usr/local/bin/k3s`

---

## phase/install_binaries.go — Binary Installation

Moves staged binary to final install location:
- Renames temp file to install path
- Sets permissions
- Handles service restarts if needed

**k3sctl changes**:
- Install path: `/usr/local/bin/k0s` → `/usr/local/bin/k3s`
- K3s may need symlinks: `k3s kubectl`, `k3s crictl`, `k3s ctr`

---

## phase/configure_k0s.go → configure_k3s.go

Generates and uploads K0s configuration:
- Creates k0s.yaml from cluster config
- Uploads to host at appropriate path
- Handles dynamic config updates

**k3sctl MAJOR rewrite**:
- K3s config format: YAML at `/etc/rancher/k3s/config.yaml`
- K3s config structure is flat key-value (not nested like k0s):
  ```yaml
  token: <token>
  tls-san:
    - <addr>
  cluster-cidr: 10.42.0.0/16
  service-cidr: 10.43.0.0/16
  ```
- Server vs agent configs differ
- No dynamic config (file is read at startup)

---

## phase/initialize_k0s.go → initialize_k3s.go

Initializes the first K0s controller:
- Runs `k0s install controller`
- Starts k0s service
- Waits for readiness

**k3sctl MAJOR rewrite**:
- First server: install as systemd service, start with `--cluster-init` for HA etcd
- Create systemd unit file for K3s
- Start service: `systemctl start k3s`
- Wait for K3s to be ready (kubeconfig available, API responding)
- Retrieve node token for joining additional nodes

---

## phase/install_controllers.go → install_servers.go

Installs additional K0s controllers:
- Generates join token
- Runs `k0s install controller --token-file`
- Starts k0s service

**k3sctl MAJOR rewrite**:
- Additional servers join with: K3S_URL=https://<leader>:6443 K3S_TOKEN=<token>
- Or via config.yaml: `server: https://<leader>:6443` and `token: <token>`
- Create systemd unit, start service
- Wait for node ready

---

## phase/install_workers.go → install_agents.go

Installs K0s workers:
- Generates worker token
- Runs `k0s install worker --token-file`
- Starts k0s service

**k3sctl MAJOR rewrite**:
- K3s agents: K3S_URL=https://<server>:6443 K3S_TOKEN=<token>
- Config: `server:` and `token:` in `/etc/rancher/k3s/config.yaml`
- Create systemd unit for k3s-agent, start service
- Wait for node to register

---

## phase/upgrade_controllers.go → upgrade_servers.go

Upgrades K0s controllers one at a time:
- Stops k0s service
- Replaces binary
- Updates config
- Restarts and waits for readiness
- Drains/uncordons if needed

**k3sctl adaptation**:
- Stop k3s service: `systemctl stop k3s`
- Replace binary at `/usr/local/bin/k3s`
- Update config if needed
- Start k3s: `systemctl start k3s`
- Wait for readiness
- Rolling upgrade: one server at a time

---

## phase/upgrade_workers.go → upgrade_agents.go

Upgrades K0s workers:
- Drains node
- Stops k0s service
- Replaces binary
- Restarts
- Uncordons node

**k3sctl adaptation**:
- Drain node via kubectl
- Stop k3s-agent: `systemctl stop k3s-agent`
- Replace binary
- Start k3s-agent: `systemctl start k3s-agent`
- Uncordon node

---

## phase/reset_controllers.go → reset_servers.go

Resets K0s controllers:
- Stops k0s service
- Runs `k0s reset`
- Cleans up data/config directories

**k3sctl MAJOR rewrite**:
- Stop k3s service
- Remove: `/etc/rancher/k3s/`, `/var/lib/rancher/k3s/`
- Remove K3s binary and symlinks
- Remove systemd unit files
- Clean iptables rules
- Remove CNI config (`/etc/cni/net.d/`)

---

## phase/reset_workers.go → reset_agents.go

Resets K0s workers:
- Stops k0s worker service
- Cleans up worker data

**k3sctl MAJOR rewrite**:
- Stop k3s-agent service
- Remove: `/etc/rancher/k3s/`, `/var/lib/rancher/k3s/`
- Remove K3s binary
- Remove systemd unit files
- Clean iptables/CNI

---

## phase/reset_leader.go — Leader Reset

Resets the last controller (leader). Runs after other controllers reset.

**k3sctl**: Same pattern, adapted for K3s paths and commands.

---

## phase/reinstall.go — Service Reinstall

Reinstalls K0s service when install flags change:
- Compares current install flags with desired
- Stops, reinstalls, starts

**k3sctl adaptation**: Update systemd unit file content when K3s flags change.

---

## phase/get_kubeconfig.go — Kubeconfig Retrieval

Gets kubeconfig from the cluster:
- k0sctl: runs `k0s kubeconfig admin`
- Replaces server address with external address
- Merges with existing kubeconfig if requested

**k3sctl MAJOR rewrite**:
- Read `/etc/rancher/k3s/k3s.yaml` from server node
- Replace `127.0.0.1` with actual server address
- Use k8s.io/client-go for kubeconfig manipulation (same as k0sctl)

---

## phase/backup.go — Backup Phase

Runs k0s backup on the leader.

**k3sctl MAJOR rewrite**:
- For etcd: `k3s etcd-snapshot save --name <name>`
- Download snapshot file from server
- For SQLite: tar the database directory
- Package as tar.gz for user

---

## phase/restore.go — Restore Phase

Restores from a k0s backup.

**k3sctl MAJOR rewrite**:
- Upload snapshot to server
- For etcd: `k3s server --cluster-reset --cluster-reset-restore-path=<snapshot>`
- For SQLite: restore database files
- Restart K3s service

---

## phase/run_hooks.go — Hook Execution

Runs user-defined hooks (before/after commands on hosts).

**k3sctl**: Direct port — hook mechanism is generic.

---

## phase/uploadfiles.go — File Upload

Uploads user-specified files to hosts.

**k3sctl**: Direct port — generic file upload mechanism.

---

## phase/downloads.go — Download Orchestration

Manages downloads of additional files to hosts (images, etc.).

**k3sctl changes**:
- K3s images directory: `/var/lib/rancher/k3s/agent/images/`
- K3s uses images tarball import mechanism

---

## phase/lock.go / unlock.go — Host Locking

Prevents concurrent k0sctl operations on the same host.

**k3sctl**: Direct port — change lock file path from k0sctl to k3sctl.

---

## phase/daemon_reload.go — Systemd Reload

Runs `systemctl daemon-reload` after service file changes.

**k3sctl**: Direct port.

---

## phase/arm_prepare.go — ARM Preparation

Handles ARM-specific setup (cgroup, kernel modules).

**k3sctl**: K3s has better ARM support natively. May need minimal changes
for K3s-specific ARM requirements.

---

## phase/apply_manifests.go — Manifest Application

Applies extra Kubernetes manifests from config:
- Uses kubectl from the k0s binary
- Applies to cluster via kubeconfig

**k3sctl changes**:
- K3s has built-in manifest directory: `/var/lib/rancher/k3s/server/manifests/`
- Can either use `k3s kubectl apply` or drop files in manifests dir
- Prefer kubectl approach for consistency

---

## phase/ensure_join_token_workaround.go — Token Workaround

K0s-specific workaround for token creation issues in certain versions.

**k3sctl**: DROP — not applicable to K3s. K3s token mechanism is different
(file-based at `/var/lib/rancher/k3s/server/node-token`).
