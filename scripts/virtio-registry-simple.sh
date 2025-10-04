#!/bin/bash
# Simplified VirtIO Registry Configuration Script
set -e

DEVICE="$1"
PARTITION="$2"
TARGET_DEVICE="${DEVICE}${PARTITION}"

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <device> <partition_number>"
    echo "Example: $0 /dev/vde 2"
    exit 1
fi

echo "🔧 VirtIO Registry Configuration (Essential Services)"
echo "Target: $TARGET_DEVICE"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo "Error: Must run as root"
    exit 1
fi

# Create mount point
MOUNT_POINT="/mnt/win-registry"
mkdir -p "$MOUNT_POINT"

echo "[INFO] Repairing NTFS..."
ntfsfix "$TARGET_DEVICE"

echo "[INFO] Mounting Windows partition..."
mount -t ntfs-3g "$TARGET_DEVICE" "$MOUNT_POINT"

# Verify Windows installation
if [[ ! -f "$MOUNT_POINT/Windows/System32/config/SYSTEM" ]]; then
    echo "Error: Windows SYSTEM registry not found"
    umount "$MOUNT_POINT"
    exit 1
fi

echo "[INFO] Creating essential VirtIO registry entries..."

# Create simplified registry file
REGISTRY_FILE="/tmp/virtio-simple-$$.reg"
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

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Services\netkvm]
"Start"=dword:00000003
"Type"=dword:00000001
"ErrorControl"=dword:00000001
"ImagePath"="\\SystemRoot\\System32\\drivers\\netkvm.sys"
"Group"="NDIS"
REGEOF

echo "[INFO] Applying VirtIO services registry..."
hivexregedit --merge --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
    "$MOUNT_POINT/Windows/System32/config/SYSTEM" "$REGISTRY_FILE"

echo "[SUCCESS] Essential VirtIO services configured"

# Cleanup
umount "$MOUNT_POINT"
rm -f "$REGISTRY_FILE"
rmdir "$MOUNT_POINT"

echo ""
echo "🎉 VirtIO Registry Configuration COMPLETE!"
echo "✅ viostor: Start=0 (boot critical)"
echo "✅ vioscsi: Start=0 (boot critical)"  
echo "✅ netkvm: Start=3 (automatic)"
echo "🚀 Ready for VirtIO boot testing!"
