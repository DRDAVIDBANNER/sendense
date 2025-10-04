#!/bin/bash
set -e

echo "🔧 HIVEX Simple VirtIO Boot Fix"
echo "==============================="

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

echo "[INFO] 📋 Creating SIMPLE VirtIO boot registry fix..."

# Create the registry file - ONLY the essential boot services
cat > /tmp/virtio-boot.reg << 'REGEOF'
Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\SYSTEM\ControlSet001\Services\viostor]
"Start"=dword:00000000
"Type"=dword:00000001
"ErrorControl"=dword:00000001
"ImagePath"="\\SystemRoot\\System32\\drivers\\viostor.sys"
"Group"="SCSI miniport"

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

echo "[INFO] Using hivexregedit to merge SIMPLE registry changes..."

# Use hivexregedit to merge the registry file
hivexregedit --merge --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
    /mnt/win/Windows/System32/config/SYSTEM /tmp/virtio-boot.reg

echo "[SUCCESS] ✓ SIMPLE Registry modifications applied using hivexregedit!"

# Cleanup
sudo umount /mnt/win
rm -f /tmp/virtio-boot.reg

echo "======================================"
echo "🎉 SIMPLE VirtIO Boot Fix COMPLETED!"
echo "======================================"
echo ""
echo "✅ Boot-critical VirtIO services configured:"
echo "   • viostor: Start=0 (boot critical)"  
echo "   • vioscsi: Start=0 (boot critical)"
echo "   • netkvm: Start=3 (automatic)"
echo ""
echo "🚀 Ready to test VirtIO boot!"
