#!/bin/bash
# prepare-virtio-installer.sh - Drop VirtIO installer onto Windows VM for manual installation
# Usage: ./prepare-virtio-installer.sh /dev/vdc 2

set -e

echo "🎯 VirtIO Installer Deployment Script"
echo "====================================="

DEVICE="$1"
PARTITION="$2"

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <device> <partition_number>"
    echo "Example: $0 /dev/vdc 2"
    echo ""
    echo "This script places the VirtIO installer into C:\\temp for manual installation"
    echo "Because fucking Windows won't let us do proper driver injection from Linux!"
    exit 1
fi

TARGET_DEVICE="${DEVICE}${PARTITION}"
echo "[INFO] Target Windows partition: $TARGET_DEVICE"

# Create mount point
sudo mkdir -p /mnt/win

echo "[INFO] 🔧 Fixing NTFS and mounting Windows..."
sudo ntfsfix "$TARGET_DEVICE"
sudo mount -t ntfs-3g "$TARGET_DEVICE" /mnt/win

echo "[INFO] 📁 Creating C:\\temp directory..."
sudo mkdir -p /mnt/win/temp

# Download COMPLETE VirtIO installer (includes QEMU agent + all drivers)
INSTALLER_PATH="/opt/migration-tools/virtio-win-guest-tools.exe"
if [ ! -f "$INSTALLER_PATH" ]; then
    echo "[INFO] 📥 Downloading COMPLETE VirtIO installer (includes QEMU agent)..."
    sudo mkdir -p /opt/migration-tools
    
    # Try multiple sources for the COMPLETE installer
    echo "[INFO] Trying Red Hat fedorapeople.org (primary source)..."
    sudo wget -O "$INSTALLER_PATH" \
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win-guest-tools.exe" \
        || {
            echo "[INFO] Primary source failed, trying latest release..."
            sudo wget -O "$INSTALLER_PATH" \
                "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/latest-virtio/virtio-win-guest-tools.exe"
        } || {
            echo "[INFO] Trying archive.virtio-win.org..."
            sudo wget -O "$INSTALLER_PATH" \
                "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.240-1/virtio-win-guest-tools.exe"
        } || {
            echo "[ERROR] Failed to download VirtIO installer from all sources!"
            exit 1
        }
fi

# Verify we got the COMPLETE installer (should be 20MB+)
INSTALLER_SIZE=$(stat -c%s "$INSTALLER_PATH" 2>/dev/null || echo "0")
if [ "$INSTALLER_SIZE" -lt 15000000 ]; then  # Less than 15MB indicates incomplete download
    echo "[ERROR] Downloaded installer seems incomplete (${INSTALLER_SIZE} bytes)"
    echo "[INFO] Removing and retrying download..."
    sudo rm -f "$INSTALLER_PATH"
    exit 1
fi

echo "[INFO] ✅ Complete VirtIO installer ready (${INSTALLER_SIZE} bytes)"

echo "[INFO] 📦 Copying VirtIO installer to C:\\temp..."
sudo cp "$INSTALLER_PATH" /mnt/win/temp/

# Create instructions file
echo "[INFO] 📝 Creating installation instructions..."
cat > /tmp/virtio-install-instructions.txt << 'EOF'
COMPLETE VirtIO + QEMU Guest Agent Installation
==============================================

WHAT'S INCLUDED IN THIS INSTALLER:
- ✅ VirtIO Storage Drivers (viostor, vioscsi) - CRITICAL for boot
- ✅ VirtIO Network Drivers (NetKVM) - Paravirtualized networking  
- ✅ VirtIO Balloon Driver - Memory management
- ✅ VirtIO Serial, Input, RNG drivers
- ✅ QEMU Guest Agent - VM management integration
- ✅ All Windows-compatible VirtIO drivers

INSTALLATION STEPS:
==================
1. Boot this Windows VM with IDE controller (NOT VirtIO yet!)
2. Login to Windows normally  
3. Navigate to C:\temp\ (or use desktop shortcut)
4. Run "virtio-win-guest-tools.exe" as Administrator
5. Install with DEFAULT settings (installs everything)
6. RESTART when the installer prompts you
7. After restart, SHUTDOWN the VM completely
8. Change VM storage controller from IDE to VirtIO
9. Boot - Windows will now work perfectly with VirtIO!

FILES DEPLOYED:
==============
- C:\temp\virtio-win-guest-tools.exe (COMPLETE installer)
- C:\temp\virtio-install-instructions.txt (this file)
- C:\temp\virtio-shortcut.bat (quick launcher)
- Desktop shortcut: "Install VirtIO Drivers.bat"

VERIFICATION:
============
After installation, check Device Manager for:
- "Red Hat VirtIO SCSI controller" (storage)
- "Red Hat VirtIO Ethernet Adapter" (network)
- "VirtIO Balloon Driver" (memory)
- QEMU Guest Agent service should be running

PERFORMANCE BENEFITS:
====================
- 10x faster disk I/O performance
- Native paravirtualized networking  
- Lower CPU overhead
- Better memory management
- Industry-standard KVM integration

This approach works because we use Windows' native driver 
installation instead of trying to hack drivers from Linux!
EOF

sudo cp /tmp/virtio-install-instructions.txt /mnt/win/temp/

echo "[INFO] 🔧 Creating desktop shortcut for easy access..."
cat > /tmp/virtio-shortcut.bat << 'EOF'
@echo off
echo Opening VirtIO installer...
cd /d C:\temp
start virtio-win-guest-tools.exe
EOF

sudo cp /tmp/virtio-shortcut.bat /mnt/win/temp/
sudo cp /tmp/virtio-shortcut.bat "/mnt/win/Users/Public/Desktop/Install VirtIO Drivers.bat" 2>/dev/null || true

# Verify the installer was copied correctly (before unmounting)
COPIED_SIZE=$(stat -c%s "/mnt/win/temp/virtio-win-guest-tools.exe" 2>/dev/null || echo "0")

echo "[INFO] 🧹 Unmounting Windows volume..."
sudo umount /mnt/win

echo ""
echo "✅ COMPLETE VirtIO + QEMU INSTALLER DEPLOYED!"
echo "=============================================="
echo ""
echo "📦 COMPLETE INSTALLER DETAILS:"
echo "   Size: $(echo $COPIED_SIZE | numfmt --to=iec)B (${COPIED_SIZE} bytes)"
echo "   Location: C:\\temp\\virtio-win-guest-tools.exe"
echo "   Includes: ALL VirtIO drivers + QEMU Guest Agent"
echo ""
echo "📁 Files placed in C:\\temp:"
echo "   • virtio-win-guest-tools.exe (COMPLETE installer)"
echo "   • virtio-install-instructions.txt (detailed guide)"
echo "   • virtio-shortcut.bat (quick launcher)"
echo ""
echo "🎯 NEXT STEPS:"
echo "1. Boot Windows VM with IDE controller"
echo "2. Login and run C:\\temp\\virtio-win-guest-tools.exe AS ADMINISTRATOR"
echo "3. Install with DEFAULT settings (installs EVERYTHING)"
echo "4. RESTART when prompted by installer"
echo "5. Shutdown VM and switch to VirtIO controller"
echo "6. Boot - enjoy 10x performance improvement!"
echo ""
echo "✨ WHAT GETS INSTALLED:"
echo "   • VirtIO Storage (viostor, vioscsi) - Critical for boot"
echo "   • VirtIO Network (NetKVM) - Paravirtualized networking"
echo "   • VirtIO Balloon - Memory management"
echo "   • QEMU Guest Agent - VM integration"
echo "   • ALL other VirtIO drivers for complete compatibility"
echo ""
echo "💡 Pro tip: Desktop shortcut created for easy access"