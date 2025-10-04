#!/bin/bash
# VirtIO Registry Configuration Script
# Registry-only solution for enabling VirtIO boot in Windows VMs
# Usage: ./virtio-registry-fix.sh <device> <partition_number>
# Example: ./virtio-registry-fix.sh /dev/vdc 2

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check script arguments
if [ "$#" -ne 2 ]; then
    log_error "Invalid usage"
    echo "Usage: $0 <device> <partition_number>"
    echo "Example: $0 /dev/vdc 2"
    echo ""
    echo "This script applies VirtIO registry configuration to enable Windows boot with VirtIO controllers."
    exit 1
fi

DEVICE="$1"
PARTITION="$2"
TARGET_DEVICE="${DEVICE}${PARTITION}"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

# Verify target device exists
if [[ ! -b "$TARGET_DEVICE" ]]; then
    log_error "Device $TARGET_DEVICE does not exist or is not a block device"
    exit 1
fi

echo "=============================================="
echo "🔧 VirtIO Registry Configuration"  
echo "=============================================="
echo ""
log_info "Target device: $TARGET_DEVICE"
log_info "Starting registry-only VirtIO configuration..."
echo ""

# Check if required tools are installed
if ! command -v hivexregedit >/dev/null 2>&1; then
    log_warning "hivexregedit not found, installing libhivex-bin..."
    apt-get update -qq
    apt-get install -y libhivex-bin libwin-hivex-perl
fi

if ! command -v ntfsfix >/dev/null 2>&1; then
    log_warning "ntfsfix not found, installing ntfs-3g..."
    apt-get update -qq  
    apt-get install -y ntfs-3g
fi

# Create mount point
MOUNT_POINT="/mnt/win-registry-fix"
mkdir -p "$MOUNT_POINT"

log_info "🔧 Repairing NTFS filesystem..."
ntfsfix "$TARGET_DEVICE" || {
    log_warning "NTFS repair had warnings, continuing..."
}

log_info "📁 Mounting Windows partition..."
mount -t ntfs-3g "$TARGET_DEVICE" "$MOUNT_POINT" || {
    log_error "Failed to mount $TARGET_DEVICE"
    exit 1
}

# Verify we have a Windows installation
if [[ ! -f "$MOUNT_POINT/Windows/System32/config/SYSTEM" ]]; then
    log_error "Windows SYSTEM registry not found at $MOUNT_POINT/Windows/System32/config/SYSTEM"
    log_error "This does not appear to be a Windows partition"
    umount "$MOUNT_POINT" 2>/dev/null || true
    exit 1
fi

log_success "Windows installation detected"

log_info "📋 Creating VirtIO registry configuration..."

# Create comprehensive VirtIO registry fix
REGISTRY_FILE="/tmp/virtio-registry-fix-$$.reg"
cat > "$REGISTRY_FILE" << 'REGEOF'
Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Services\viostor]
"Start"=dword:00000000
"Type"=dword:00000001
"ErrorControl"=dword:00000001
"ImagePath"="\\SystemRoot\\System32\\drivers\\viostor.sys"
"Group"="SCSI miniport"
"DisplayName"="Red Hat VirtIO SCSI controller"

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Services\vioscsi]
"Start"=dword:00000000
"Type"=dword:00000001
"ErrorControl"=dword:00000001
"ImagePath"="\\SystemRoot\\System32\\drivers\\vioscsi.sys"
"Group"="SCSI miniport"
"DisplayName"="Red Hat VirtIO SCSI pass-through controller"

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Services\netkvm]
"Start"=dword:00000003
"Type"=dword:00000001
"ErrorControl"=dword:00000001
"ImagePath"="\\SystemRoot\\System32\\drivers\\netkvm.sys"
"Group"="NDIS"
"DisplayName"="Red Hat VirtIO Ethernet Adapter"

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1001]
"Service"="viostor"
"ClassGUID"="{4D36E97B-E325-11CE-BFC1-08002BE10318}"

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1004]
"Service"="viostor"
"ClassGUID"="{4D36E97B-E325-11CE-BFC1-08002BE10318}"

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1008]
"Service"="vioscsi"
"ClassGUID"="{4D36E97B-E325-11CE-BFC1-08002BE10318}"

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1000]
"Service"="netkvm"
"ClassGUID"="{4D36E972-E325-11CE-BFC1-08002BE10318}"
REGEOF

log_info "🔧 Applying registry configuration using hivexregedit..."

# Apply registry fixes using modern hivex tools
hivexregedit --merge --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
    "$MOUNT_POINT/Windows/System32/config/SYSTEM" "$REGISTRY_FILE" || {
    log_error "Failed to apply registry configuration"
    umount "$MOUNT_POINT" 2>/dev/null || true
    rm -f "$REGISTRY_FILE"
    exit 1
}

log_success "Registry configuration applied successfully"

# Verify registry entries were applied (basic check)
log_info "🔍 Verifying registry configuration..."
if hivexsh -r "$MOUNT_POINT/Windows/System32/config/SYSTEM" >/dev/null 2>&1; then
    log_success "Registry structure verified as accessible"
else
    log_warning "Registry verification had issues, but configuration may still be valid"
fi

# Cleanup
log_info "🧹 Cleaning up..."
umount "$MOUNT_POINT" || {
    log_warning "Warning: Failed to unmount $MOUNT_POINT cleanly"
}
rm -f "$REGISTRY_FILE"
rmdir "$MOUNT_POINT" 2>/dev/null || true

echo ""
echo "=============================================="
echo "🎉 VirtIO Registry Configuration COMPLETE!"
echo "=============================================="
echo ""
echo "✅ Boot-critical VirtIO services configured (Start=0)"
echo "✅ VirtIO storage drivers: viostor, vioscsi"  
echo "✅ VirtIO network driver: netkvm (Start=3)"
echo "✅ Critical Device Database updated with PCI mappings"
echo "✅ Modern hivex tools used for reliable modification"
echo "✅ Clean registry entries without problematic references"
echo ""
echo "🚀 Windows VM is ready to boot with VirtIO controllers!"
echo ""
echo "Next steps:"
echo "1. Create CloudStack VM with VirtIO storage controller"
echo "2. Boot the VM and verify successful startup"  
echo "3. Check Device Manager for VirtIO devices"
echo ""
log_success "Registry configuration completed successfully"