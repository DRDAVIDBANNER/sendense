#!/bin/bash
set -e

echo "🔧 COMPLETE VirtIO Registry Fix (matching working VM)"
echo "==================================================="

DEVICE="$1"
PARTITION="$2"

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <device> <partition_number>"
    echo "Example: $0 /dev/vdc 2"
    exit 1
fi

TARGET_DEVICE="${DEVICE}${PARTITION}"
echo "[INFO] Target: $TARGET_DEVICE"

# Create mount point
sudo mkdir -p /mnt/win

echo "[INFO] 🔧 Fixing NTFS and mounting Windows..."
sudo ntfsfix "$TARGET_DEVICE"
sudo mount -t ntfs-3g "$TARGET_DEVICE" /mnt/win

echo "[INFO] 📋 Applying COMPLETE VirtIO registry fix with OEM references..."

# Use the complete registry file we just created
hivexregedit --merge --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
    /mnt/win/Windows/System32/config/SYSTEM /home/pgrayson/migratekit-cloudstack/scripts/complete-virtio-registry-fix.reg

echo "[SUCCESS] ✓ COMPLETE Registry modifications applied!"

# Cleanup
sudo umount /mnt/win

echo "=========================================="
echo "🎉 COMPLETE VirtIO Fix APPLIED!"
echo "=========================================="
echo ""
echo "✅ Added missing entries matching working VM:"
echo "   • viostor: Owners=oem0.inf, Tag=34"  
echo "   • vioscsi: Owners=oem10.inf, Tag=64, DisplayName"
echo "   • ImagePath: REG_EXPAND_SZ format"
echo ""
echo "🚀 Ready to test VirtIO boot - should work now!"
