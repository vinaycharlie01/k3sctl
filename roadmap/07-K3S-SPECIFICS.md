# k3sctl Roadmap — K3s-Specific Technical Details

## K3s Update Channel API

Endpoint: `https://update.k3s.io/v1-release/channels`

Returns JSON with channel definitions:
```json
{
  "data": [
    {
      "id": "stable",
      "type": "channel",
      "name": "stable",
      "latest": "v1.31.0+k3s1"
    },
    {
      "id": "latest",
      "type": "channel",
      "name": "latest",
      "latest": "v1.31.1-rc1+k3s1"
    },
    {
      "id": "v1.31",
      "type": "channel",
      "name": "v1.31",
      "latest": "v1.31.0+k3s1"
    }
  ]
}
```

Usage:
- Default version → fetch "stable" channel
- User specifies channel → fetch that channel
- User specifies exact version → use as-is

## K3s Binary Download URLs

Pattern: `https://github.com/k3s-io/k3s/releases/download/{version}/{binary}`

Where `{version}` is URL-encoded (+ → %2B):
- `v1.31.0+k3s1` → `v1.31.0%2Bk3s1`

Binary names by architecture:
| Arch | Binary | SHA256 |
|------|--------|--------|
| amd64 | `k3s` | `sha256sum-amd64.txt` |
| arm64 | `k3s-arm64` | `sha256sum-arm64.txt` |
| armhf | `k3s-armhf` | `sha256sum-arm.txt` |
| s390x | `k3s-s390x` | `sha256sum-s390x.txt` |

## K3s Installation (Pure Go Implementation)

### Server Installation Steps

1. **Create directories**:
   ```
   /etc/rancher/k3s/
   /var/lib/rancher/k3s/
   ```

2. **Place binary**: Upload/download k3s to `/usr/local/bin/k3s`

3. **Write config** `/etc/rancher/k3s/config.yaml`:
   ```yaml
   # First server (HA etcd)
   cluster-init: true
   tls-san:
     - <external-address>
   write-kubeconfig-mode: "0644"
   
   # Additional servers
   server: https://<leader>:6443
   token: <node-token>
   tls-san:
     - <external-address>
   ```

4. **Write systemd unit** `/etc/systemd/system/k3s.service`:
   - See configurer analysis for full unit content
   - ExecStart includes all install flags

5. **Enable and start service**:
   ```
   systemctl daemon-reload
   systemctl enable k3s
   systemctl start k3s
   ```

6. **Wait for ready**: Poll kubeconfig availability + API health

7. **Retrieve token**: Read `/var/lib/rancher/k3s/server/node-token`

### Agent Installation Steps

1. **Create directories**:
   ```
   /etc/rancher/k3s/
   ```

2. **Place binary**: Same as server

3. **Write config** `/etc/rancher/k3s/config.yaml`:
   ```yaml
   server: https://<server>:6443
   token: <node-token>
   ```

4. **Write systemd unit** `/etc/systemd/system/k3s-agent.service`

5. **Enable and start**

6. **Wait for node registration** via kubectl on server

## K3s Upgrade Strategy

### Server Rolling Upgrade

For each server (one at a time, leader last):
1. If drain enabled: `k3s kubectl drain <node>`
2. Stop service: `systemctl stop k3s`
3. Replace binary: `/usr/local/bin/k3s`
4. Update config if needed
5. Start service: `systemctl start k3s`
6. Wait for node Ready
7. If drained: `k3s kubectl uncordon <node>`

### Agent Rolling Upgrade

For each agent (concurrent up to concurrency limit):
1. If drain enabled: drain via leader's kubectl
2. Stop service: `systemctl stop k3s-agent`
3. Replace binary
4. Start service: `systemctl start k3s-agent`
5. Wait for node Ready
6. If drained: uncordon

## K3s Reset/Uninstall (Pure Go)

Must implement equivalent of `k3s-uninstall.sh` in Go:

1. **Stop services**:
   - `systemctl stop k3s` or `systemctl stop k3s-agent`

2. **Kill remaining processes**:
   - Kill any `k3s` processes still running
   - Kill containerd-shim processes

3. **Remove service files**:
   - `/etc/systemd/system/k3s.service`
   - `/etc/systemd/system/k3s-agent.service`
   - `/etc/systemd/system/k3s.service.env` (if exists)
   - `systemctl daemon-reload`

4. **Clean up networking**:
   - Remove CNI config: `/etc/cni/net.d/`
   - Flush iptables rules for flannel (careful approach)
   - Remove flannel network interface

5. **Remove data**:
   - `/var/lib/rancher/k3s/`
   - `/etc/rancher/k3s/`
   - `/var/lib/kubelet/` (K3s creates this)

6. **Remove binary and symlinks**:
   - `/usr/local/bin/k3s`
   - Check and remove if symlinks: `kubectl`, `crictl`, `ctr`

## K3s Backup/Restore

### Backup (etcd mode)

```
k3s etcd-snapshot save --name k3sctl-backup-<timestamp> \
    --dir /tmp/k3sctl-snapshots/
```

Then download the snapshot file from the server.

### Backup (SQLite mode)

Stop k3s service, tar `/var/lib/rancher/k3s/server/db/`, start service.
Or use k3s's built-in SQLite snapshot mechanism.

### Restore (etcd mode)

1. Upload snapshot to server
2. Stop k3s on all nodes
3. On first server:
   ```
   k3s server --cluster-reset \
       --cluster-reset-restore-path=/path/to/snapshot
   ```
4. Start k3s on first server
5. Join other servers (they will sync from restored state)
6. Join agents

### Restore (SQLite mode)

1. Stop k3s
2. Replace database files
3. Start k3s

## K3s Kubeconfig

Location: `/etc/rancher/k3s/k3s.yaml`

Format (standard kubeconfig):
```yaml
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: <base64>
    server: https://127.0.0.1:6443
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
users:
- name: default
  user:
    client-certificate-data: <base64>
    client-key-data: <base64>
```

k3sctl must:
1. Read this file from the server
2. Replace `127.0.0.1` with the actual server address
3. Rename cluster/context/user from "default" to user-specified names
4. Merge with existing kubeconfig if requested

## K3s Version Detection

On a host with K3s installed:
```
k3s --version
# Output: "k3s version v1.31.0+k3s1 (abc1234)"
```

Parse: extract version string from output.

K3s running check:
- `systemctl is-active k3s` or `systemctl is-active k3s-agent`
- Process check: pgrep for k3s process

## K3s Environment Files

K3s supports environment files read by systemd:
- `/etc/default/k3s` (Debian-style)
- `/etc/sysconfig/k3s` (RHEL-style)

Content: `K3S_TOKEN=xxx`, `K3S_URL=xxx`, etc.

k3sctl can optionally write these, or put everything in config.yaml.
Prefer config.yaml for consistency.
