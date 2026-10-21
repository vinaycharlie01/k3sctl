#!/bin/bash
# Helper script for k3sctl nested VM setup
# Used by the nested-vm-multinode.yml workflow

set -e

VM_NAME="${1:?VM name required}"
DISK_PATH="${2:?Disk path required}"
SEED_ISO="${3:?Seed ISO path required}"
TAP_INTERFACE="${4:?TAP interface required}"
SERIAL_LOG="${5:?Serial log path required}"

# Detect virtualization backend
detect_virt_backend() {
    if grep -q "kvm" /proc/cpuinfo && [ -e /dev/kvm ]; then
        echo "kvm"
    else
        echo "tcg"
    fi
}

# Start a single QEMU VM
start_vm() {
    local name="$VM_NAME"
    local disk="$DISK_PATH"
    local seed="$SEED_ISO"
    local tap="$TAP_INTERFACE"
    local serial="$SERIAL_LOG"
    local backend="$(detect_virt_backend)"

    echo "Starting VM: $name (backend: $backend)"

    local qemu_opts=(
        "-name" "$name"
        "-m" "768M"
        "-smp" "2"
        "-drive" "file=$disk,if=virtio,format=qcow2"
        "-drive" "file=$seed,if=virtio,media=cdrom"
        "-net" "nic,model=virtio"
        "-net" "tap,ifname=$tap,script=no,downscript=no"
        "-serial" "file:$serial"
        "-nographic"
        "-enable-kvm"
    )

    if [ "$backend" = "kvm" ]; then
        qemu_opts+=("-machine" "accel=kvm:tcg")
    else
        qemu_opts+=("-machine" "accel=tcg")
    fi

    qemu-system-x86_64 "${qemu_opts[@]}" &
    echo $! # Return PID
}

# Main execution
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    start_vm
fi
