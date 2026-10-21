# Nested VM Multi-Node Integration Test Workflow

## Overview

This document describes the `nested-vm-multinode.yml` workflow, which performs end-to-end integration testing of k3sctl on a three-node K3s cluster running inside nested QEMU virtual machines.

**Purpose**: Verify that k3sctl's `apply`, `reset`, and `config` commands work correctly in a real multi-host, multi-role environment with SSH-based remote execution.

**Topology**:
- 1 K3s server node (control plane)
- 2 K3s agent nodes (workers)
- All running as independent Ubuntu 24.04 VMs
- Private virtual network with DHCP
- SSH-based access with temporary key pair

## Design Principles

1. **Isolation**: Test runs in a fully isolated environment with dedicated VMs, networks, and firewall rules.
2. **Reproducibility**: Uses cloud-init and cloud images for deterministic guest setup.
3. **Security**: Generates temporary SSH keys, validates host identities, never logs credentials.
4. **Correctness over Speed**: Uses bounded polling instead of fixed sleeps; TCG fallback ensures tests run even without KVM.
5. **Observability**: Comprehensive diagnostics collection on failure (serial logs, systemd journal, cloud-init logs).
6. **Guaranteed Cleanup**: Uses `if: always()` with explicit PID tracking to ensure VMs are reaped and networks cleaned up.

## Virtualization Backend

The workflow automatically detects and uses the best available virtualization backend:

### KVM (Hardware Accelerated)
- **Detected**: When CPU has KVM flag and `/dev/kvm` is available.
- **Performance**: ~5-10x faster than TCG.
- **Availability**: GitHub-hosted `ubuntu-24.04` runners may have nested KVM enabled (not guaranteed).

### TCG (Software Emulation)
- **Fallback**: Used when KVM is unavailable.
- **Performance**: Slower, but reliable on all runners.
- **Deployment**: Automatically selected if KVM setup fails.

**Detection Logic** (in "Detect virtualization capability" step):
```bash
if grep -q "kvm" /proc/cpuinfo && [ -e /dev/kvm ]; then
    VIRT_BACKEND=kvm
else
    VIRT_BACKEND=tcg
fi
```

**QEMU Machine Type**:
```bash
-machine accel=kvm:tcg    # Try KVM, fall back to TCG
-machine accel=tcg        # TCG only
```

## Network Setup

The workflow creates an isolated virtual network topology:

### Components
1. **Virtual Bridge** (`br-k3stest`): Managed by the workflow, created and destroyed dynamically.
2. **TAP Interfaces** (3x): `tap-k3s-0`, `tap-k3s-1`, `tap-k3s-2` — one per VM.
3. **DHCP Server** (`dnsmasq`): Assigns IPs in range `192.168.100.10-100`.
4. **Bridge IP**: `192.168.100.1` with `/24` subnet.

### IP Address Assignment
```
Bridge:          192.168.100.1/24
Server VM:       192.168.100.10-100 (via DHCP, typically .10)
Agent-1 VM:      192.168.100.10-100 (via DHCP, typically .11)
Agent-2 VM:      192.168.100.10-100 (via DHCP, typically .12)
```

### Configuration Discovery
The workflow uses ARP to discover actual VM IPs:
```bash
arp -a | grep "k3s-server" | awk '{print $1}'
```

This is more reliable than assuming fixed DHCP addresses.

### NAT and Routing
- Guest-to-host communication: Direct via bridge.
- Internet access: Works via host's default route (if host has internet access).
- VM-to-VM communication: Direct via virtual bridge.

## Workflow Steps

### 1. Initialization
- Creates temporary directories for VMs, SSH keys, logs, diagnostics.
- Validates Go toolchain and Go version from `go.mod`.

### 2. Virtualization Detection
- Checks for KVM support in CPU and `/dev/kvm` availability.
- Measures system resources (memory, CPUs).
- Fails if insufficient memory (needs 2GB+ for 3 VMs @ 768MB each).

### 3. Image Download
- Downloads official Ubuntu 24.04 cloud image from `cloud-images.ubuntu.com`.
- Stores in `$WORKFLOW_DIR/vms/`.

### 4. Cloud-Init Configuration
- Generates user-data files for each VM with:
  - SSH public key injection.
  - Passwordless sudo for `ubuntu` user.
  - Hostname and FQDN setup.
  - DNS resolver configuration.
  - Auto-reboot after cloud-init.

### 5. VM Disk Creation
- Creates COW disks from base image: `k3s-${VM}-root.qcow2`.
- Generates cloud-init seed ISOs: `k3s-${VM}-seed.iso`.

### 6. Virtual Network Setup
- Creates bridge and TAP interfaces.
- Starts `dnsmasq` for DHCP and DNS.
- Stores TAP interface names for cleanup.

### 7. VM Launch
- Starts 3 QEMU processes with:
  - 768MB memory per VM.
  - 2 CPUs per VM.
  - TAP-based networking.
  - Serial console to file.
- Waits 30s for cloud-init to initialize.

### 8. IP Discovery & SSH Setup
- Polls ARP table until all 3 VMs appear.
- Waits for SSH to respond on all VMs.
- Validates SSH host keys (not used in this workflow, but recommended for production).

### 9. Host Validation
- On each VM:
  - Verify systemd is PID 1.
  - Verify passwordless sudo works.
  - Verify Ubuntu 24.04 is installed.
  - Verify DNS resolution works.
  - Report disk and memory capacity.

### 10. Build k3sctl
- Runs `go build` with version and git commit injected via LDFLAGS.
- Produces binary at `$WORKFLOW_DIR/k3sctl`.

### 11. Configuration Generation
- Generates k3sctl cluster YAML with:
  - 1 server + 2 agents.
  - SSH config with temporary key.
  - Private IP addresses for API connectivity.
  - K3s version from workflow input (defaults to `v1.28.0+k3s1`).

**Example generated config**:
```yaml
apiVersion: k3sctl.k3sproject.io/v1beta1
kind: Cluster
metadata:
  name: k3s-nested-test
spec:
  hosts:
    - role: server
      hostname: k3s-server
      privateAddress: 192.168.100.10
      ssh:
        address: 192.168.100.10
        user: ubuntu
        keyPath: /tmp/k3sctl-workflow/ssh/id_rsa
    - role: agent
      hostname: k3s-agent-1
      privateAddress: 192.168.100.11
      ssh:
        address: 192.168.100.11
        user: ubuntu
        keyPath: /tmp/k3sctl-workflow/ssh/id_rsa
    - role: agent
      hostname: k3s-agent-2
      privateAddress: 192.168.100.12
      ssh:
        address: 192.168.100.12
        user: ubuntu
        keyPath: /tmp/k3sctl-workflow/ssh/id_rsa
  k3s:
    version: v1.28.0+k3s1
```

### 12. Run k3sctl Apply
- Executes `./k3sctl apply --config config/k3sctl.yaml --force`.
- Writes kubeconfig to `config/kubeconfig.yaml`.
- Captures logs in `logs/apply.log`.
- Fails immediately on non-zero exit.

### 13. Wait for Cluster Ready
- Waits up to 300s for all 3 nodes to reach Kubernetes "Ready" status.
- Polls via `k3s kubectl get nodes` on the server.
- Verifies cluster API is reachable.

### 14. Verify Cluster Health
- Checks K3s service status on each node.
- Verifies K3s version consistency.
- Lists nodes and their status.
- Confirms API connectivity.

### 15. Diagnostics Collection (on failure)
- Collects:
  - QEMU process list.
  - VM serial console logs.
  - dnsmasq logs.
  - Cloud-init logs from all VMs.
  - Systemd journal (k3s unit).
  - Network info (ip addr, ip route).
  - k3sctl apply logs.
- Stores in `$WORKFLOW_DIR/diagnostics/`.

### 16. Cleanup (always)
- Attempts `k3sctl reset --force` (best-effort).
- Kills all QEMU processes by PID.
- Removes TAP interfaces and bridge.
- Stops dnsmasq.
- Deletes VM disks and seed ISOs.
- Deletes temporary SSH keys.
- **Preserves** non-sensitive logs and diagnostics.

### 17. Upload Artifacts
- Uploads logs and diagnostics to GitHub for 7 days.
- Excludes kubeconfig and SSH keys.

## Known Limitations

### 1. Nested Virtualization on GitHub-Hosted Runners
- **Issue**: GitHub-hosted `ubuntu-24.04` runners may not have nested KVM enabled.
- **Mitigation**: Workflow falls back to TCG automatically.
- **Consequence**: Test takes 10-15 minutes instead of 2-3 minutes.
- **Workaround**: None available for GitHub-hosted runners; use self-hosted for faster tests.

### 2. Cloud Image Download
- **Issue**: Depends on `cloud-images.ubuntu.com` availability.
- **Mitigation**: Image is cached by the runner if possible (but fresh download on each run is safest).
- **Fallback**: Could mirror image to GitHub releases or a CDN.

### 3. SSH Host Key Verification
- **Current**: Workflow uses `StrictHostKeyChecking=no` for simplicity.
- **Security**: Not suitable for production; uses temporary keys anyway.
- **Improvement**: Could collect and verify guest public keys from serial console.

### 4. Cleanup on Cancellation
- **Issue**: If runner is forcibly terminated, QEMU VMs may remain running.
- **Mitigation**: GitHub runner reclamation happens after ~5 minutes; VMs are cleaned up.
- **Guarantee**: Not perfect, but good enough for CI runners.

### 5. Multi-architecture Support
- **Current**: Only x86_64 (KVM and TCG).
- **Limitation**: ARM64, ppc64le would need separate image sources and machine types.

### 6. Resource Constraints
- **Memory**: 768MB per VM is tight; some workloads may fail.
- **CPU**: 2 CPUs per VM; K3s is single-threaded for many operations.
- **Disk**: 10GB per root disk; sufficient for K3s + test workloads.

### 7. Network Isolation
- **Issue**: VMs can reach the internet if the host can.
- **Consequence**: Not fully isolated from external dependencies.
- **Note**: For CI, this is acceptable (tests can fail due to network issues, but that's real-world testing).

## Configuration Options

### Workflow Inputs (dispatched via `workflow_dispatch`)

| Input | Default | Description |
|-------|---------|-------------|
| `k3s_version` | `v1.28.0+k3s1` | K3s version to install on all nodes |
| `timeout_minutes` | `60` | Job timeout in minutes (allows TCG tests to complete) |

### Environment Variables (hardcoded)
```bash
E2E_TIMEOUT=600                    # Timeout for individual steps (10 min)
WORKFLOW_DIR=/tmp/k3sctl-workflow  # Base directory for test files
```

## Validation Checklist

Before considering this workflow "complete", the following tests should pass:

### Static Validation (No VMs)
- [ ] YAML syntax is valid (GitHub Actions linter).
- [ ] All referenced environment variables are defined.
- [ ] Go module version from `go.mod` is correctly detected.
- [ ] Cluster YAML generation produces a schema-valid configuration.

### Unit Tests
- [ ] Existing unit tests pass: `go test ./...`.
- [ ] Config parser validates the generated YAML.
- [ ] Host YAML unmarshaling works with SSH config format.

### Integration Test (Full Workflow)
- [ ] Workflow runs to completion without timeout (KVM: ~3 min, TCG: ~15 min).
- [ ] All 3 VMs boot and cloud-init completes.
- [ ] SSH access works from host to all 3 VMs.
- [ ] k3sctl apply succeeds.
- [ ] All 3 nodes reach Kubernetes "Ready" status.
- [ ] Cluster API is reachable via kubeconfig.
- [ ] k3sctl reset succeeds.
- [ ] Cleanup removes all VMs, networks, and temporary files.

### Failure Scenarios
- [ ] Diagnostics collected if apply fails.
- [ ] Diagnostics collected if cluster never becomes ready.
- [ ] Cleanup still runs even if apply fails.
- [ ] No VMs, bridges, or TAP interfaces left after cleanup (even on partial failure).

### Security Validation
- [ ] SSH private key is removed in cleanup.
- [ ] kubeconfig is removed in cleanup.
- [ ] No secrets or kubeconfig appear in logs.
- [ ] Generated config is not uploaded as artifact (security risk).

## Performance Expectations

| Backend | Time | Notes |
|---------|------|-------|
| KVM | 3-5 min | Fast VM boot, rapid cloud-init. If available. |
| TCG | 12-15 min | Slower emulation, but reliable. Most GitHub runners. |

Breakdown (approximate, TCG):
- Image download: 1-2 min
- VM disk creation: 30s
- Network setup: 20s
- VM boot and cloud-init: 5-7 min
- SSH readiness: 2-3 min
- k3sctl apply: 2-3 min
- Cluster readiness: 2-3 min
- Cleanup: 1-2 min

## Debugging the Workflow

### Check KVM Availability
```bash
grep kvm /proc/cpuinfo
ls -la /dev/kvm
```

### Review Logs
- GitHub Actions UI: "Artifacts" section → `k3sctl-nested-vm-logs`.
- Files:
  - `logs/apply.log` — k3sctl apply output.
  - `logs/serial-*.log` — VM boot and cloud-init.
  - `diagnostics/cloud-init.log` — Per-VM cloud-init details.
  - `diagnostics/systemd-journal.log` — K3s service logs.

### Local Testing
To test the workflow locally (on a Linux machine with KVM):
```bash
# Install dependencies
sudo apt-get install qemu-system-x86 qemu-utils cloud-image-utils dnsmasq

# Manually run a subset of workflow steps
WORKFLOW_DIR=/tmp/test-k3sctl
mkdir -p $WORKFLOW_DIR/{vms,ssh,logs,diagnostics,config}

# Generate SSH key, download image, create cloud-init, etc.
# Refer to workflow steps for exact commands.
```

## Future Improvements

1. **Multi-version Testing**: Parameterize K3s version, test multiple versions in matrix.
2. **E2E Workload Testing**: Deploy a sample workload (e.g., nginx) and verify it runs.
3. **Upgrade Testing**: Start with v1.27, upgrade to v1.28, verify no data loss.
4. **HA Server Setup**: Test 3-server + 2-agent topology.
5. **Network Chaos**: Introduce temporary network delays or packet loss to test resilience.
6. **Custom Manifests**: Verify k3sctl's manifest deployment works.
7. **Hook Testing**: Verify k3sctl's before/after hooks execute correctly.
8. **Self-Hosted Runner**: Use self-hosted runners with KVM for faster tests.
9. **Image Caching**: Cache downloaded cloud image across runs.
10. **Root Cause Analysis**: Auto-parse failure logs and generate summary report.

## References

- **k3sctl GitHub**: https://github.com/vinaycharlie01/k3sctl
- **k3s Documentation**: https://docs.k3s.io/
- **QEMU Docs**: https://www.qemu.org/docs/
- **Cloud-init Docs**: https://cloud-init.io/
- **GitHub Actions Docs**: https://docs.github.com/en/actions

## Support

For issues or questions about this workflow:
1. Check GitHub Actions logs for error messages.
2. Review diagnostics in artifacts (serial logs, cloud-init logs).
3. Verify k3sctl configuration is valid: `k3sctl apply --dry-run`.
4. Test SSH connectivity manually to the VMs (if still running).
5. File an issue with workflow logs and diagnostics attached.
