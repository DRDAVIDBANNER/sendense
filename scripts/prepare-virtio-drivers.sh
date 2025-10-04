#!/bin/bash
# prepare-virtio-drivers.sh - Extract and package drivers from working VM
# This script should be run once to prepare the driver package for future use

set -e

echo "🔧 Preparing VirtIO Driver Package from Working VM"
echo "================================================="

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <working_vm_device> <working_vm_partition>"
    echo "Example: $0 /dev/vdd 3"
    echo ""
    echo "This extracts drivers from a working libvirt VM for future use"
    exit 1
fi

WORKING_DEVICE="$1$2"
echo "[INFO] Extracting drivers from: $WORKING_DEVICE"

# Mount working VM
sudo mkdir -p /mnt/working-vm-extract
sudo mount -t ntfs-3g "$WORKING_DEVICE" /mnt/working-vm-extract

# Create driver package
echo "[INFO] Creating driver package..."
sudo mkdir -p /tmp/virtio-driver-package/DriverStore/FileRepository
sudo mkdir -p /tmp/virtio-driver-package/System32/drivers

# Copy VirtIO drivers
sudo cp -r /mnt/working-vm-extract/Windows/System32/DriverStore/FileRepository/vio* /tmp/virtio-driver-package/DriverStore/FileRepository/ 2>/dev/null || true
sudo cp -r /mnt/working-vm-extract/Windows/System32/DriverStore/FileRepository/netkvm* /tmp/virtio-driver-package/DriverStore/FileRepository/ 2>/dev/null || true
sudo cp -r /mnt/working-vm-extract/Windows/System32/DriverStore/FileRepository/balloon* /tmp/virtio-driver-package/DriverStore/FileRepository/ 2>/dev/null || true

sudo cp /mnt/working-vm-extract/Windows/System32/drivers/vio*.sys /tmp/virtio-driver-package/System32/drivers/ 2>/dev/null || true
sudo cp /mnt/working-vm-extract/Windows/System32/drivers/netkvm.sys /tmp/virtio-driver-package/System32/drivers/ 2>/dev/null || true

# Package for future use
cd /tmp && sudo tar -czf /opt/migration-tools/working-vm-drivers.tar.gz virtio-driver-package/

# Cleanup
sudo umount /mnt/working-vm-extract
sudo rm -rf /tmp/virtio-driver-package

PACKAGE_COUNT=$(tar -tzf /opt/migration-tools/working-vm-drivers.tar.gz | grep -E '\.(sys|inf)$' | wc -l)

echo "[SUCCESS] VirtIO driver package created!"
echo "  📦 Location: /opt/migration-tools/working-vm-drivers.tar.gz"
echo "  📊 Files packaged: $PACKAGE_COUNT"
echo ""
echo "✅ Ready for use with complete-virtio-solution.sh"
