#!/bin/bash
set -e

echo "🔍 VirtIO Boot Diagnostics"
echo "========================="

DEVICE="$1"
PARTITION="$2"

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <device> <partition_number>"
    echo "Example: $0 /dev/vdc 2"
    exit 1
fi

TARGET_DEVICE="${DEVICE}${PARTITION}"
echo "[INFO] Target: $TARGET_DEVICE"

sudo mkdir -p /mnt/win
sudo mount -t ntfs-3g "$TARGET_DEVICE" /mnt/win

echo ""
echo "1. Checking OEM .inf files..."
echo "=============================="
ls -la /mnt/win/Windows/inf/oem*.inf 2>/dev/null || echo "❌ No OEM .inf files found!"

echo ""
echo "2. Checking if CriticalDeviceDatabase exists..."
echo "============================================="
hivexsh -r /mnt/win/Windows/System32/config/SYSTEM << 'HIVEX_EOF'
cd \ControlSet001\Control
ls
quit
HIVEX_EOF

echo ""
echo "3. Checking current VirtIO registry entries..."
echo "=============================================="
hivexsh -r /mnt/win/Windows/System32/config/SYSTEM << 'HIVEX_EOF'
cd \ControlSet001\Services\viostor
lsval
quit
HIVEX_EOF

echo ""
echo "4. Checking DriverStore VirtIO files..."
echo "======================================"
find /mnt/win/Windows/System32/DriverStore/FileRepository -name "*viostor*" -o -name "*vioscsi*" 2>/dev/null | head -5

sudo umount /mnt/win
echo ""
echo "Diagnostics complete!"
