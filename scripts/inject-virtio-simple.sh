#!/bin/bash
# Simple and Reliable VirtIO Driver Injection Script
# Injects VirtIO drivers by creating offline registry entries

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

log_info "Simple VirtIO Driver Injection for Windows VM"
log_info "Target: $WINDOWS_PARTITION"

# Download and mount VirtIO ISO
if [[ ! -f "/tmp/virtio-win-0.1.240.iso" ]]; then
    wget -q https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.240-1/virtio-win-0.1.240.iso -O /tmp/virtio-win-0.1.240.iso
fi

mkdir -p /mnt/virtio-iso /mnt/windows-vm
mount -o loop /tmp/virtio-win-0.1.240.iso /mnt/virtio-iso

# Fix and mount Windows partition
ntfsfix "$WINDOWS_PARTITION"
mount -t ntfs -o rw "$WINDOWS_PARTITION" /mnt/windows-vm

# Copy all VirtIO driver files
log_info "Installing VirtIO drivers..."
cp /mnt/virtio-iso/viostor/w10/amd64/viostor.sys /mnt/windows-vm/Windows/System32/drivers/
cp /mnt/virtio-iso/viostor/w10/amd64/viostor.inf /mnt/windows-vm/Windows/INF/
cp /mnt/virtio-iso/NetKVM/w10/amd64/netkvm.sys /mnt/windows-vm/Windows/System32/drivers/
cp /mnt/virtio-iso/NetKVM/w10/amd64/netkvm.inf /mnt/windows-vm/Windows/INF/
cp /mnt/virtio-iso/NetKVM/w10/amd64/netkvmco.dll /mnt/windows-vm/Windows/System32/

# Create a Windows registry script that runs on boot
cat > /mnt/windows-vm/virtio-setup.bat << 'EOF'
@echo off
echo Installing VirtIO drivers...

:: Add VirtIO storage driver to registry
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Type /t REG_DWORD /d 1 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Start /t REG_DWORD /d 0 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v ErrorControl /t REG_DWORD /d 1 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v ImagePath /t REG_SZ /d "\SystemRoot\System32\drivers\viostor.sys" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v DisplayName /t REG_SZ /d "Red Hat VirtIO SCSI controller" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\viostor" /v Group /t REG_SZ /d "SCSI miniport" /f

:: Add VirtIO network driver to registry
reg add "HKLM\SYSTEM\CurrentControlSet\Services\netkvm" /v Type /t REG_DWORD /d 1 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\netkvm" /v Start /t REG_DWORD /d 3 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\netkvm" /v ErrorControl /t REG_DWORD /d 1 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\netkvm" /v ImagePath /t REG_SZ /d "\SystemRoot\System32\drivers\netkvm.sys" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Services\netkvm" /v DisplayName /t REG_SZ /d "Red Hat VirtIO Ethernet Adapter" /f

:: Critical device database entries
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1001" /v Service /t REG_SZ /d "viostor" /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CriticalDeviceDatabase\pci#ven_1af4&dev_1001" /v ClassGUID /t REG_SZ /d "{4D36E97B-E325-11CE-BFC1-08002BE10318}" /f

echo VirtIO drivers installed successfully!
del "%~f0"
EOF

# Install the drivers using Windows' built-in tools via a PowerShell script that will run on boot
cat > /mnt/windows-vm/Windows/System32/virtio-install.ps1 << 'EOF'
# VirtIO Driver Installation Script
Write-Host "Installing VirtIO drivers..."

# Install VirtIO storage driver
try {
    $infPath = "C:\Windows\INF\viostor.inf"
    if (Test-Path $infPath) {
        pnputil.exe /add-driver $infPath /install
        Write-Host "VirtIO storage driver installed"
    }
} catch {
    Write-Host "Error installing VirtIO storage driver: $_"
}

# Install VirtIO network driver  
try {
    $infPath = "C:\Windows\INF\netkvm.inf"
    if (Test-Path $infPath) {
        pnputil.exe /add-driver $infPath /install
        Write-Host "VirtIO network driver installed"
    }
} catch {
    Write-Host "Error installing VirtIO network driver: $_"
}

# Create registry entries for boot-critical VirtIO storage
try {
    New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\viostor" -Name "Type" -Value 1 -PropertyType DWord -Force
    New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\viostor" -Name "Start" -Value 0 -PropertyType DWord -Force
    New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\viostor" -Name "ErrorControl" -Value 1 -PropertyType DWord -Force
    New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\viostor" -Name "ImagePath" -Value "\SystemRoot\System32\drivers\viostor.sys" -PropertyType String -Force
    New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\viostor" -Name "Group" -Value "SCSI miniport" -PropertyType String -Force
    Write-Host "VirtIO registry entries created"
} catch {
    Write-Host "Registry entries may already exist or error occurred: $_"
}

Write-Host "VirtIO driver installation completed"
Remove-Item $MyInvocation.MyCommand.Path -Force
EOF

# Add the PowerShell script to run on next boot via RunOnce registry
cat > /tmp/virtio-runonce.reg << 'EOF'
Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\RunOnce]
"VirtIOInstall"="powershell.exe -ExecutionPolicy Bypass -File C:\\Windows\\System32\\virtio-install.ps1"
EOF

# Apply the RunOnce registry entry using chntpw
echo "Loading Windows SOFTWARE hive and adding RunOnce entry..."

# Create a more direct chntpw script
cat > /tmp/chntpw_commands.txt << 'EOF'
9
cd \Microsoft\Windows\CurrentVersion\RunOnce
nv 1 VirtIOInstall
ed VirtIOInstall
powershell.exe -ExecutionPolicy Bypass -File C:\Windows\System32\virtio-install.ps1
q
y
EOF

# Apply the registry changes
cat /tmp/chntpw_commands.txt | chntpw -i /mnt/windows-vm/Windows/System32/config/SOFTWARE

log_success "VirtIO drivers copied and installation scripts created"
log_success "PowerShell script will run on next Windows boot to complete driver registration"

# Cleanup
umount /mnt/windows-vm
umount /mnt/virtio-iso

log_success "VirtIO driver injection completed!"
log_success "The VM should now boot with VirtIO controllers without BSOD"