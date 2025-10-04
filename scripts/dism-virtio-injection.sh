#!/bin/bash
# dism-virtio-injection.sh - Proper Windows VirtIO driver injection using DISM
# This uses Microsoft's official DISM tool for proper driver installation

set -e

DEVICE="$1"
PARTITION="$2"

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <device> <partition_number>"
    echo "Example: $0 /dev/vdc 2"
    echo ""
    echo "This script uses Microsoft DISM for PROPER Windows driver installation"
    echo "NOT simple file copying - this integrates with Windows driver database"
    exit 1
fi

TARGET_DEVICE="${DEVICE}${PARTITION}"
echo "[INFO] Target Windows partition: $TARGET_DEVICE"

# Verify prerequisites
echo "[INFO] 🔍 Verifying prerequisites..."
if ! command -v ntfsfix &> /dev/null; then
    echo "[ERROR] ntfs-3g not found. Install with: sudo apt install ntfs-3g"
    exit 1
fi

# Create mount points
sudo mkdir -p /mnt/win /mnt/virtio

echo "[INFO] 🔧 Fixing NTFS and mounting Windows partition..."
sudo ntfsfix "$TARGET_DEVICE"
sudo mount -t ntfs-3g "$TARGET_DEVICE" /mnt/win

echo "[INFO] 📀 Mounting VirtIO ISO..."
sudo mount -o loop /opt/migration-tools/virtio-win-0.1.240.iso /mnt/virtio

echo "[INFO] 💾 Using MICROSOFT DISM for proper driver installation..."

# Create Windows PE environment for DISM
echo "[INFO] Creating DISM command script for Windows..."
cat > /mnt/win/dism-virtio-install.cmd << 'DISMEOF'
@echo off
echo Installing VirtIO drivers using Microsoft DISM...

REM Mount the Windows image (already mounted as C: in this case)
echo Adding VirtIO storage drivers...
dism /Image:C:\ /Add-Driver /Driver:D:\viostor\w10\amd64\viostor.inf /ForceUnsigned
dism /Image:C:\ /Add-Driver /Driver:D:\vioscsi\w10\amd64\vioscsi.inf /ForceUnsigned

echo Adding VirtIO network drivers...
dism /Image:C:\ /Add-Driver /Driver:D:\NetKVM\w10\amd64\netkvm.inf /ForceUnsigned

echo Adding VirtIO balloon driver...
dism /Image:C:\ /Add-Driver /Driver:D:\Balloon\w10\amd64\balloon.inf /ForceUnsigned

echo Adding VirtIO input drivers...
dism /Image:C:\ /Add-Driver /Driver:D:\vioinput\w10\amd64\vioinput.inf /ForceUnsigned

echo Adding VirtIO serial drivers...
dism /Image:C:\ /Add-Driver /Driver:D:\vioser\w10\amd64\vioser.inf /ForceUnsigned

echo Verifying installed drivers...
dism /Image:C:\ /Get-Drivers

echo VirtIO driver installation complete!
DISMEOF

echo "[INFO] 🎯 Alternative: Use Windows Recovery Environment approach..."
cat > /mnt/win/recovery-virtio-fix.cmd << 'RECOVEOF'
@echo off
echo Loading VirtIO drivers in Windows Recovery Environment...
echo Run this script from Windows Recovery Environment Command Prompt

REM First load the driver so Windows can see the disk
echo Loading viostor driver...
drvload D:\viostor\w10\amd64\viostor.inf

REM Then inject it permanently into the Windows installation
echo Injecting viostor driver into Windows...
dism /image:C:\ /add-driver /driver:D:\viostor\w10\amd64\viostor.inf

echo Injecting vioscsi driver...
dism /image:C:\ /add-driver /driver:D:\vioscsi\w10\amd64\vioscsi.inf

echo Injecting network driver...
dism /image:C:\ /add-driver /driver:D:\NetKVM\w10\amd64\netkvm.inf

echo VirtIO drivers injected! Restart Windows.
RECOVEOF

echo "[SUCCESS] ✅ DISM scripts created on Windows volume"
echo "  📄 /Windows/dism-virtio-install.cmd - For offline DISM injection"
echo "  📄 /Windows/recovery-virtio-fix.cmd - For Windows Recovery Environment"

# Cleanup
sudo umount /mnt/virtio /mnt/win 2>/dev/null || true

echo ""
echo "🎉 PROPER MICROSOFT DISM SOLUTION PREPARED!"
echo "============================================"
echo ""
echo "Option 1: Use Windows Recovery Environment (RECOMMENDED):"
echo "  1. Boot VM with VirtIO (will fail)"
echo "  2. Enter Windows Recovery Environment"
echo "  3. Open Command Prompt"
echo "  4. Mount VirtIO ISO as D: drive"  
echo "  5. Run: C:\\Windows\\recovery-virtio-fix.cmd"
echo "  6. Restart - should boot with VirtIO!"
echo ""
echo "Option 2: Cross-platform DISM (if available):"
echo "  - Run the dism-virtio-install.cmd script"
echo ""
echo "🚀 This uses Microsoft's PROPER driver installation method!"
