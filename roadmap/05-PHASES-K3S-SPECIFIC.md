# Roadmap: K3s-Specific Phases

These phases require significant adaptation from k0sctl to handle K3s differences.

## phase/default_k0s_version.go → default_k3s_version.go
- Original: resolves default k0s version via k0s channel server or GitHub
- Uses `k0sproject/version.LatestByPrerelease()`
- Sets `Spec.K0s.Version` if not explicitly set
- **k3sctl**: Resolve K3s version via GitHub releases API (`k3s-io/k3s`)
- Need custom K3s version resolution (no k0s channel server equivalent)
- K3s release naming: `v1.28.2+k3s1` format

## phase/gather_k0s_facts.go → gather_k3s_facts.go
- Original: on each host, runs `k0s version` to get installed binary version
- Detects running version via `k0s status` output parsing
- Reads existing k0s config from host
- Checks if k0s service is installed
- **k3sctl**:
  - Run `k3s --version` to get installed binary version (parses "k3s version v1.x.x+k3s1")
  - Check if k3s/k3s-agent service is running via systemd
  - Read existing config from `/etc/rancher/k3s/config.yaml`
  - Check for existing node-token at `/var/lib/rancher/k3s/server/node-token`

### phase/gather_k0s_facts_test.go
- **k3sctl**: Port with K3s version parsing tests

## phase/stage_binaries.go
- Original: stages k0s binary using BinaryProvider pattern
- Downloads/uploads binary to temp path on host
- Handles BinaryCacher interface for local upload dedup
- **k3sctl**: Port with K3s binary providers
- K3s binary URL format: `https://github.com/k3s-io/k3s/releases/download/{version}/k3s` (amd64)
- ARM64: `k3s-arm64`, ARMhf: `k3s-armhf`

### phase/stage_binaries_test.go
- **k3sctl**: Port

## phase/install_binaries.go
- Original: moves staged binary to install location, sets permissions
- **k3sctl**: Same pattern, move to `/usr/local/bin/k3s`

## phase/configure_k0s.go → configure_k3s.go
- Original: generates k0s.yaml config, writes to host, handles dynamic config
- Creates node-specific config from cluster config
- **k3sctl**:
  - Generate `/etc/rancher/k3s/config.yaml` (YAML format)
  - Server config: cluster-init, tls-san, node-ip, flannel settings, etc.
  - Agent config: server URL, token
  - No dynamic config CRD (K3s is file-based)

### phase/configure_k0s_test.go
- **k3sctl**: Port with K3s config structure

## phase/initialize_k0s.go → initialize_k3s.go
- Original: runs `k0s install controller` + `k0s start` on first controller
- Sets up initial cluster with `--config` flag
- **k3sctl**:
  - First server: `k3s server --cluster-init` (for HA) or plain `k3s server`
  - Install as systemd service
  - Wait for node-token to be generated
  - Wait for K3s API to be ready

## phase/install_controllers.go → install_servers.go
- Original: generates join tokens from leader, installs k0s controller service, waits for readiness
- Quorum-aware batch installation for multiple controllers
- CleanUp resets failed installs
- After hook invalidates tokens
- **k3sctl**:
  - Read node-token from first server
  - Install k3s server with `--server https://<first-server>:6443 --token <token>`
  - Install as systemd service
  - Wait for API readiness
  - No explicit token generation command (token is a file)

## phase/install_workers.go → install_agents.go
- Original: generates worker join tokens, writes token files, installs k0s worker service
- Validates API connectivity via token kubeconfig
- **k3sctl**:
  - Read node-token from server
  - Install k3s agent with `K3S_URL=https://<server>:6443 K3S_TOKEN=<token>`
  - Or use config.yaml with server/token fields
  - Install as systemd service
  - Wait for node ready status

## phase/upgrade_controllers.go → upgrade_servers.go
- Original: stops service, replaces binary, reinstalls, starts, waits for ready
- Does rolling upgrade (one at a time for controllers)
- **k3sctl**: Same rolling upgrade pattern for K3s servers
- Stop k3s service, replace binary, restart

## phase/upgrade_workers.go → upgrade_agents.go
- Original: drain, stop, replace binary, reinstall, start, uncordon
- Batched parallel upgrade respecting WorkerDisruptionPercent
- **k3sctl**: Same pattern with k3s-agent service

## phase/reinstall.go
- Original: reinstalls k0s service when install flags have changed
- Detects flag changes by comparing current vs desired
- **k3sctl**: Port — detect changed config, restart k3s with new flags

## phase/reset_workers.go → reset_agents.go
- Original: stops worker, runs `k0s reset`, removes data
- **k3sctl**: Stop k3s-agent, remove data dirs, remove binary, remove systemd unit

## phase/reset_controllers.go → reset_servers.go
- Original: stops controller, runs `k0s reset`, removes data
- **k3sctl**: Stop k3s, remove data dirs (`/var/lib/rancher/k3s`), remove config, remove binary

## phase/reset_leader.go → reset_leader.go
- Original: handles leader reset specially (last controller to reset)
- **k3sctl**: Same pattern — reset initial server last

## phase/get_kubeconfig.go
- Original: runs `k0s kubeconfig admin` on controller, rewrites server address
- Uses k8s client-go clientcmd for kubeconfig manipulation
- **k3sctl**: Read `/etc/rancher/k3s/k3s.yaml`, rewrite server address/user/cluster

### phase/get_kubeconfig_test.go
- **k3sctl**: Port

## phase/backup.go
- Original: runs `k0s backup --save-path`
- Downloads backup archive from host
- **k3sctl**: Run `k3s etcd-snapshot save` or copy SQLite database
- For etcd mode: `k3s etcd-snapshot save --dir /path`
- For SQLite mode: copy `/var/lib/rancher/k3s/server/db/`

## phase/restore.go
- Original: uploads backup, runs `k0s restore`
- **k3sctl**: Upload snapshot, run `k3s server --cluster-reset --cluster-reset-restore-path=<snapshot>`

## phase/apply_manifests.go
- Original: applies extra Kubernetes manifests to cluster
- Writes manifests to leader, runs kubectl apply
- **k3sctl**: Port — use k3s kubectl or direct kubectl with kubeconfig

## phase/validate_etcd_members.go
- Original: validates etcd member list matches expected controllers
- **k3sctl**: Only applicable when K3s uses etcd (--datastore-endpoint or --cluster-init)
- For SQLite mode: skip entirely

## phase/ensure_join_token_workaround.go
- Original: workaround for k0s token creation bug in certain versions
- **k3sctl**: Not needed — K3s uses file-based tokens

### phase/ensure_join_token_workaround_test.go
- **k3sctl**: Not needed

## phase/arm_prepare.go
- Original: ARM-specific preparation (kernel modules, settings)
- **k3sctl**: K3s supports ARM natively, but may still need cgroup setup
- Consider: check cgroup v2 configuration on ARM hosts

## Implementation Order
1. default_k3s_version.go
2. gather_k3s_facts.go
3. stage_binaries.go + install_binaries.go
4. configure_k3s.go
5. initialize_k3s.go
6. install_servers.go
7. install_agents.go
8. get_kubeconfig.go
9. upgrade_servers.go + upgrade_agents.go
10. reinstall.go
11. reset_agents.go + reset_servers.go + reset_leader.go
12. backup.go + restore.go
13. apply_manifests.go
14. validate_etcd_members.go (conditional)
