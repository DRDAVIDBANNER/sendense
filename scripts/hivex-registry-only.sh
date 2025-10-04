#!/bin/bash
set -e

echo "🔧 HIVEX Registry-Only Fix for VirtIO Boot"
echo "=========================================="

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

echo "[INFO] 📋 Creating VirtIO registry fix..."

# Create the registry file
cat > /tmp/virtio-registry.reg << 'REGEOF'
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

echo "[INFO] Using hivexregedit to merge registry changes..."

# Use hivexregedit to merge the registry file
hivexregedit --merge --prefix 'HKEY_LOCAL_MACHINE\SYSTEM' \
    /mnt/win/Windows/System32/config/SYSTEM /tmp/virtio-registry.reg

echo "[SUCCESS] ✓ Registry modifications applied using hivexregedit!"

# Cleanup
sudo umount /mnt/win
rm -f /tmp/virtio-registry.reg

echo "==========================================="
echo "🎉 HIVEX VirtIO Registry Fix COMPLETED!"
echo "==========================================="
echo ""
echo "✅ Boot-critical registry entries set using modern hivex tools"
echo "✅ VirtIO storage Start=0 (boot critical)"  
echo "✅ VirtIO SCSI Start=0 (boot critical)"
echo "✅ VirtIO network Start=3 (automatic)"
echo "✅ Critical device database updated"
echo ""
echo "🚀 Ready to test VirtIO boot!"
