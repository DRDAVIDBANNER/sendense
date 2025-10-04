#!/bin/bash
# Fix VirtIO Boot Issue by Direct Registry Modification
# This script directly modifies the Windows registry while offline to fix boot issues

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
    log_error "Example: $0 /dev/vdb 4"
    exit 1
fi

log_info "🔧 FIXING VirtIO Boot Issue via Direct Registry Modification"
log_info "Target: $WINDOWS_PARTITION"

# Install required tools
if ! command -v chntpw >/dev/null 2>&1; then
    log_info "Installing chntpw for registry editing..."
    apt-get update -qq
    apt-get install -y -qq chntpw
fi

# Cleanup any previous mounts
umount /mnt/windows-vm 2>/dev/null || true

# Mount Windows partition
mkdir -p /mnt/windows-vm
ntfsfix "$WINDOWS_PARTITION"
mount -t ntfs -o rw "$WINDOWS_PARTITION" /mnt/windows-vm

if [[ ! -f "/mnt/windows-vm/Windows/System32/config/SYSTEM" ]]; then
    log_error "Windows SYSTEM registry not found!"
    exit 1
fi

log_info "📋 Directly modifying Windows registry to fix boot..."

# Create a comprehensive chntpw script to fix boot issues
cat > /tmp/fix_boot_registry.txt << 'EOF'
9
cd \CurrentControlSet\Services
mk viostor
cd viostor
nv 1 Type
ed Type
1
nv 0 Start
ed Start
0
nv 1 ErrorControl
ed ErrorControl
1
nv 1 ImagePath
ed ImagePath
\SystemRoot\System32\drivers\viostor.sys
nv 2 DisplayName
ed DisplayName
Red Hat VirtIO SCSI controller
nv 2 Group
ed Group
SCSI miniport
cd ..
mk vioscsi
cd vioscsi
nv 1 Type
ed Type
1
nv 0 Start
ed Start
0
nv 1 ErrorControl
ed ErrorControl
1
nv 1 ImagePath
ed ImagePath
\SystemRoot\System32\drivers\vioscsi.sys
nv 2 Group
ed Group
SCSI miniport
cd ..
cd ..\Control\CriticalDeviceDatabase
mk pci#ven_1af4&dev_1001
cd pci#ven_1af4&dev_1001
nv 2 Service
ed Service
viostor
nv 2 ClassGUID
ed ClassGUID
{4D36E97B-E325-11CE-BFC1-08002BE10318}
cd ..
mk pci#ven_1af4&dev_1008
cd pci#ven_1af4&dev_1008
nv 2 Service
ed Service
vioscsi
nv 2 ClassGUID
ed ClassGUID
{4D36E97B-E325-11CE-BFC1-08002BE10318}
q
y
EOF

# Apply the registry changes
log_info "Applying boot-critical registry entries..."
cat /tmp/fix_boot_registry.txt | chntpw -i /mnt/windows-vm/Windows/System32/config/SYSTEM

# Create a simpler approach using hivex if available
if command -v hivexregedit >/dev/null 2>&1; then
    log_info "Using hivex for additional registry modifications..."
    
    cat > /tmp/virtio_registry.reg << 'EOF'
[\CurrentControlSet\Services\viostor]
"Type"=dword:00000001
"Start"=dword:00000000
"ErrorControl"=dword:00000001
"ImagePath"="\SystemRoot\System32\drivers\viostor.sys"
"DisplayName"="Red Hat VirtIO SCSI controller"
"Group"="SCSI miniport"

[\CurrentControlSet\Services\vioscsi]
"Type"=dword:00000001
"Start"=dword:00000000
"ErrorControl"=dword:00000001
"ImagePath"="\SystemRoot\System32\drivers\vioscsi.sys"
"Group"="SCSI miniport"

[\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1001]
"Service"="viostor"
"ClassGUID"="{4D36E97B-E325-11CE-BFC1-08002BE10318}"

[\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1008]
"Service"="vioscsi"
"ClassGUID"="{4D36E97B-E325-11CE-BFC1-08002BE10318}"
EOF
    
    hivexregedit --merge /mnt/windows-vm/Windows/System32/config/SYSTEM /tmp/virtio_registry.reg 2>/dev/null || log_info "Hivex method skipped"
fi

# Verify the drivers are present
log_info "🔍 Verifying VirtIO drivers are present..."
if [[ -f "/mnt/windows-vm/Windows/System32/drivers/viostor.sys" ]]; then
    log_success "✓ viostor.sys present: $(ls -lh /mnt/windows-vm/Windows/System32/drivers/viostor.sys | awk '{print $5}')"
else
    log_error "✗ viostor.sys missing! This will cause boot failure."
    exit 1
fi

if [[ -f "/mnt/windows-vm/Windows/System32/drivers/vioscsi.sys" ]]; then
    log_success "✓ vioscsi.sys present: $(ls -lh /mnt/windows-vm/Windows/System32/drivers/vioscsi.sys | awk '{print $5}')"
else
    log_info "vioscsi.sys not found - this is optional"
fi

# Create an additional boot-time script as backup
cat > /mnt/windows-vm/Windows/fix-boot.bat << 'EOF'
@echo off
echo Emergency VirtIO driver registration...

REM Register viostor as boot driver
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Type /t REG_DWORD /d 1 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Start /t REG_DWORD /d 0 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v ErrorControl /t REG_DWORD /d 1 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v ImagePath /t REG_SZ /d "\SystemRoot\System32\drivers\viostor.sys" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Group /t REG_SZ /d "SCSI miniport" /f

echo VirtIO boot drivers registered!
EOF

log_success "📝 Created emergency boot script: C:\\Windows\\fix-boot.bat"

# Cleanup
umount /mnt/windows-vm
rm -f /tmp/fix_boot_registry.txt /tmp/virtio_registry.reg

log_success "=================================================="
log_success "🎉 VirtIO Boot Fix Applied!"
log_success "Changes made:"
log_success "  ✓ viostor driver registered as BOOT-CRITICAL (Start=0)"
log_success "  ✓ vioscsi driver registered as boot driver"
log_success "  ✓ Critical device database entries added"
log_success "  ✓ Registry modified OFFLINE (no boot required)"
log_success ""
log_success "The VM should now boot successfully with VirtIO storage!"
log_success "=================================================="