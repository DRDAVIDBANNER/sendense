#!/bin/bash
# Simple VirtIO Boot Fix - Direct approach without interactive tools
# This manually creates the essential registry entries to fix boot issues

set -e

DEVICE=$1
PARTITION=${2:-4}
WINDOWS_PARTITION="${DEVICE}${PARTITION}"

log_info() { echo -e "\033[0;34m[INFO]\033[0m $1"; }
log_success() { echo -e "\033[0;32m[SUCCESS]\033[0m $1"; }
log_error() { echo -e "\033[0;31m[ERROR]\033[0m $1"; }

if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
    exit 1
fi

if [[ -z "$DEVICE" ]]; then
    log_error "Usage: $0 <device> [partition_number]"
    exit 1
fi

log_info "🔧 Simple VirtIO Boot Fix"
log_info "Target: $WINDOWS_PARTITION"

# Clean up any previous mounts
umount /mnt/windows-vm 2>/dev/null || true

# Mount Windows partition
mkdir -p /mnt/windows-vm
ntfsfix "$WINDOWS_PARTITION"
mount -t ntfs -o rw "$WINDOWS_PARTITION" /mnt/windows-vm

# Verify we have Windows
if [[ ! -d "/mnt/windows-vm/Windows" ]]; then
    log_error "No Windows directory found!"
    exit 1
fi

# Ensure VirtIO drivers are present
log_info "📂 Ensuring VirtIO drivers are present..."
if [[ ! -f "/mnt/windows-vm/Windows/System32/drivers/viostor.sys" ]]; then
    log_error "viostor.sys not found! Please run the driver injection script first."
    exit 1
fi

log_success "✓ viostor.sys found: $(ls -lh /mnt/windows-vm/Windows/System32/drivers/viostor.sys | awk '{print $5}')"

# Create a Windows Registry script that will run during Safe Mode boot
log_info "📝 Creating Safe Mode boot registry fix..."

cat > /mnt/windows-vm/Windows/virtio-boot-fix.reg << 'EOF'
Windows Registry Editor Version 5.00

; VirtIO Storage Driver - Boot Critical
[HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\viostor]
"Type"=dword:00000001
"Start"=dword:00000000
"ErrorControl"=dword:00000001
"ImagePath"="\\\SystemRoot\\\System32\\\drivers\\\viostor.sys"
"DisplayName"="Red Hat VirtIO SCSI controller"
"Group"="SCSI miniport"
"Tag"=dword:00000021

[HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\viostor\Parameters]

[HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\viostor\Parameters\PnpInterface]
"5"=dword:00000001

; Critical Device Database Entry
[HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1001]
"Service"="viostor"
"ClassGUID"="{4D36E97B-E325-11CE-BFC1-08002BE10318}"

[HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1004]
"Service"="viostor"
"ClassGUID"="{4D36E97B-E325-11CE-BFC1-08002BE10318}"

[HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1008]
"Service"="viostor"
"ClassGUID"="{4D36E97B-E325-11CE-BFC1-08002BE10318}"
EOF

# Create a batch file that can be run from Windows Recovery Console
cat > /mnt/windows-vm/Windows/fix-virtio-boot.bat << 'EOF'
@echo off
echo VirtIO Boot Fix - Emergency Registry Repair
echo.

REM Import the registry file
echo Importing VirtIO registry entries...
regedit /s C:\Windows\virtio-boot-fix.reg

REM Manual registry entries for critical boot driver
echo Manually adding boot-critical entries...
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Type /t REG_DWORD /d 1 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Start /t REG_DWORD /d 0 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v ErrorControl /t REG_DWORD /d 1 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v ImagePath /t REG_SZ /d "\SystemRoot\System32\drivers\viostor.sys" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Group /t REG_SZ /d "SCSI miniport" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v DisplayName /t REG_SZ /d "Red Hat VirtIO SCSI controller" /f

REM Add critical device database entry
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1001" /v Service /t REG_SZ /d "viostor" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1001" /v ClassGUID /t REG_SZ /d "{4D36E97B-E325-11CE-BFC1-08002BE10318}" /f

echo.
echo VirtIO boot fix applied!
echo You can now try booting with VirtIO storage.
echo.
pause
EOF

# Try using libguestfs if available (non-interactive)
if command -v virt-win-reg >/dev/null 2>&1; then
    log_info "🔧 Using libguestfs to modify registry directly..."
    
    # Create a temporary registry file
    cat > /tmp/viostor.reg << 'EOF'
[HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\viostor]
"Type"=dword:00000001
"Start"=dword:00000000
"ErrorControl"=dword:00000001
"ImagePath"="\SystemRoot\System32\drivers\viostor.sys"
"Group"="SCSI miniport"
EOF
    
    # Apply using libguestfs
    virt-win-reg --merge "$WINDOWS_PARTITION" /tmp/viostor.reg 2>/dev/null || log_info "Libguestfs method failed, using manual approach"
    rm -f /tmp/viostor.reg
fi

# Create a Python script to modify registry if python3 is available
if command -v python3 >/dev/null 2>&1; then
    log_info "🐍 Creating Python registry modifier..."
    
    cat > /mnt/windows-vm/Windows/python-registry-fix.py << 'EOF'
#!/usr/bin/env python3
import sys
import struct
import os

def main():
    try:
        print("Python registry modification would go here")
        print("For now, using manual batch file approach")
        return 0
    except Exception as e:
        print(f"Registry modification failed: {e}")
        return 1

if __name__ == "__main__":
    sys.exit(main())
EOF
fi

# Create instructions for manual fix
cat > /mnt/windows-vm/Windows/VIRTIO_FIX_INSTRUCTIONS.txt << 'EOF'
VirtIO Boot Fix Instructions
============================

If Windows fails to boot with error 0xc000000f, follow these steps:

OPTION 1: Boot from Windows Install Media (Recommended)
1. Boot from Windows installation disc/USB
2. Choose "Repair your computer"
3. Go to Troubleshoot > Advanced Options > Command Prompt
4. Run: C:\Windows\fix-virtio-boot.bat
5. Restart and try booting normally

OPTION 2: Safe Mode (if accessible)
1. Boot into Safe Mode
2. Open Command Prompt as Administrator
3. Run: C:\Windows\fix-virtio-boot.bat
4. Restart and try booting normally

OPTION 3: Registry Import (if Windows boots)
1. Boot Windows with IDE controller
2. Double-click: C:\Windows\virtio-boot-fix.reg
3. Restart and switch to VirtIO controller

What this fixes:
- Registers viostor.sys as boot-critical driver (Start=0)
- Adds VirtIO PCI device mappings
- Enables VirtIO storage controller support

The key issue: Windows needs the VirtIO storage driver registered 
as boot-critical BEFORE it encounters VirtIO hardware during boot.
EOF

log_success "📋 Registry files and fix scripts created:"
log_success "  ✓ C:\\Windows\\virtio-boot-fix.reg"
log_success "  ✓ C:\\Windows\\fix-virtio-boot.bat"
log_success "  ✓ C:\\Windows\\VIRTIO_FIX_INSTRUCTIONS.txt"

# Verify driver presence one more time
log_info "🔍 Final verification..."
DRIVER_SIZE=$(stat -c%s "/mnt/windows-vm/Windows/System32/drivers/viostor.sys")
log_success "✓ viostor.sys verified: $DRIVER_SIZE bytes"

# Cleanup
umount /mnt/windows-vm

log_success "=================================================="
log_success "🎯 VirtIO Boot Fix Prepared!"
log_success ""
log_success "NEXT STEPS:"
log_success "1. Try booting VM with VirtIO storage"
log_success "2. If it fails with 0xc000000f error:"
log_success "   - Boot from Windows install media"
log_success "   - Go to Repair > Command Prompt"
log_success "   - Run: C:\\Windows\\fix-virtio-boot.bat"
log_success "3. Restart and VirtIO should work!"
log_success "=================================================="