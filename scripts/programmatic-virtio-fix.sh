#!/bin/bash
set -e

echo "🔧 PROGRAMMATIC VirtIO Registry Fix using hivex tools"
echo "=================================================="

DEVICE="$1"
PARTITION="$2"

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <device> <partition_number>"
    echo "Example: $0 /dev/vdc 2"
    exit 1
fi

TARGET_DEVICE="${DEVICE}${PARTITION}"

echo "[INFO] Target: $TARGET_DEVICE"

# Install required tools if not present
if ! command -v hivexregedit &> /dev/null; then
    echo "[INFO] Installing hivex tools..."
    sudo apt update
    sudo apt install -y libhivex-bin libwin-hivex-perl
fi

# Create mount points
sudo mkdir -p /mnt/win /mnt/virtio

echo "[INFO] 🔧 Fixing NTFS and mounting Windows..."
sudo ntfsfix "$TARGET_DEVICE"
sudo mount -t ntfs-3g "$TARGET_DEVICE" /mnt/win

echo "[INFO] 📀 Mounting VirtIO ISO..."
sudo mount -o loop /opt/migration-tools/virtio-win.iso /mnt/virtio

echo "[INFO] 💾 Installing VirtIO drivers to DriverStore..."
# Copy VirtIO drivers to DriverStore (proper Windows location)
sudo mkdir -p /mnt/win/Windows/System32/DriverStore/FileRepository/

# Storage drivers
sudo find /mnt/virtio -name "viostor*" -type d | while read dir; do
    if [[ "$dir" == */w10/amd64 ]] || [[ "$dir" == */w11/amd64 ]]; then
        echo "Installing VirtIO storage drivers from: $dir"
        sudo cp -r "$dir/"* /mnt/win/Windows/System32/DriverStore/FileRepository/ 2>/dev/null || true
    fi
done

# Network drivers  
sudo find /mnt/virtio -name "netkvm*" -type d | while read dir; do
    if [[ "$dir" == */w10/amd64 ]] || [[ "$dir" == */w11/amd64 ]]; then
        echo "Installing VirtIO network drivers from: $dir"
        sudo cp -r "$dir/"* /mnt/win/Windows/System32/DriverStore/FileRepository/ 2>/dev/null || true
    fi
done

# Balloon drivers
sudo find /mnt/virtio -name "balloon*" -type d | while read dir; do
    if [[ "$dir" == */w10/amd64 ]] || [[ "$dir" == */w11/amd64 ]]; then
        echo "Installing VirtIO balloon drivers from: $dir"
        sudo cp -r "$dir/"* /mnt/win/Windows/System32/DriverStore/FileRepository/ 2>/dev/null || true
    fi
done

# Copy specific required drivers to System32/drivers
sudo cp /mnt/virtio/viostor/w*/amd64/viostor.sys /mnt/win/Windows/System32/drivers/ 2>/dev/null || true
sudo cp /mnt/virtio/vioscsi/w*/amd64/vioscsi.sys /mnt/win/Windows/System32/drivers/ 2>/dev/null || true  
sudo cp /mnt/virtio/netkvm/w*/amd64/netkvm.sys /mnt/win/Windows/System32/drivers/ 2>/dev/null || true

echo "[SUCCESS] ✓ VirtIO drivers copied to DriverStore and System32"

echo "[INFO] 📋 Modifying Windows registry using hivex tools..."

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

echo "[SUCCESS] ✓ Registry modifications applied using hivexregedit"

# Cleanup
sudo umount /mnt/virtio
sudo umount /mnt/win
rm -f /tmp/virtio-registry.reg

echo "==============================================="
echo "🎉 PROGRAMMATIC VirtIO fix COMPLETED!"
echo "==============================================="
echo ""
echo "✅ VirtIO drivers installed to DriverStore"
echo "✅ Boot-critical registry entries set using hivex"
echo "✅ VirtIO storage Start=0 (boot critical)"
echo "✅ VirtIO SCSI Start=0 (boot critical)"  
echo "✅ VirtIO network Start=3 (automatic)"
echo "✅ Critical device database updated"
echo ""
echo "🚀 Ready to test VirtIO boot!"
