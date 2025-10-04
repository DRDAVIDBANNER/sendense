#!/bin/bash
# 🚀 **DEPLOY VMA PRODUCTION**
#
# Purpose: Deploy COMPLETE production VMA using REAL binaries from package
# Package: Self-contained VMA deployment package
# Target: Fresh Ubuntu 24.04 servers
# Author: MigrateKit OSSEA Team
# Date: October 2, 2025

set -euo pipefail

# Configuration
SCRIPT_VERSION="v2.0.0-complete-with-vddk-and-nbdkit"
TARGET_IP="${1:-}"
SUDO_PASSWORD="Password1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(dirname "$SCRIPT_DIR")"

if [[ -z "$TARGET_IP" ]]; then
    echo "Usage: $0 <TARGET_IP>"
    echo "Example: $0 10.0.100.234"
    exit 1
fi

LOG_FILE="/tmp/vma-production-deployment-$(date +%Y%m%d-%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Redirect all output to log file and console
exec > >(tee -a "$LOG_FILE")
exec 2>&1

echo -e "${BLUE}🚀 OSSEA-Migrate VMA Production Deployment${NC}"
echo -e "${BLUE}==========================================${NC}"
echo "Script Version: $SCRIPT_VERSION"
echo "Target: $TARGET_IP"
echo "Package: $PACKAGE_DIR"
echo "Log File: $LOG_FILE"
echo "Start Time: $(date)"
echo ""

# Function to run remote command
run_remote() {
    sshpass -p "$SUDO_PASSWORD" ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o PreferredAuthentications=password vma@$TARGET_IP "$@"
}

# Function to copy files
copy_file() {
    sshpass -p "$SUDO_PASSWORD" scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o PreferredAuthentications=password "$@"
}

# Function to log with timestamp
log() {
    echo -e "[$(date '+%H:%M:%S')] $1"
}

# =============================================================================
# PHASE 1: PASSWORDLESS SUDO SETUP
# =============================================================================

log "${BLUE}📋 Phase 1: Passwordless Sudo Setup${NC}"
log "==================================="

log "${YELLOW}🔑 Setting up passwordless sudo on target VMA...${NC}"
run_remote "echo '$SUDO_PASSWORD' | sudo -S sh -c 'echo \"vma ALL=(ALL) NOPASSWD: ALL\" > /etc/sudoers.d/vma'"

# Test passwordless sudo
if run_remote "sudo whoami" | grep -q "root"; then
    log "${GREEN}✅ Passwordless sudo configured on $TARGET_IP${NC}"
else
    log "${RED}❌ Passwordless sudo failed${NC}"
    exit 1
fi

log "${GREEN}✅ Authentication setup completed${NC}"
echo ""

# =============================================================================
# PHASE 2: SYSTEM PREPARATION
# =============================================================================

log "${BLUE}📋 Phase 2: System Preparation${NC}"
log "==============================="

# Check OS version
if ! run_remote "grep -q '24.04' /etc/os-release"; then
    log "${RED}❌ Target VMA requires Ubuntu 24.04 LTS${NC}"
    exit 1
fi

log "${BLUE}📍 Target VMA: $TARGET_IP${NC}"

log "${YELLOW}🚫 Disabling cloud-init...${NC}"
run_remote "sudo touch /etc/cloud/cloud-init.disabled"
run_remote "sudo systemctl disable cloud-init cloud-config cloud-final cloud-init-local 2>/dev/null || true"

log "${YELLOW}🔄 Installing VMA dependencies...${NC}"
run_remote "sudo apt update -y"
run_remote "DEBIAN_FRONTEND=noninteractive sudo apt install -y haveged jq curl openssh-client golang-go nbdkit libnbd-dev nbd-client"

log "${YELLOW}⚙️ Starting essential services...${NC}"
run_remote "sudo systemctl enable haveged"
run_remote "sudo systemctl start haveged"

log "${GREEN}✅ System preparation completed${NC}"
echo ""

# =============================================================================
# PHASE 3: VMA DIRECTORY STRUCTURE
# =============================================================================

log "${BLUE}📋 Phase 3: VMA Directory Structure${NC}"
log "=================================="

log "${YELLOW}📁 Creating VMA directory structure...${NC}"
run_remote "sudo mkdir -p /opt/vma/{bin,config,logs,enrollment,ssh,scripts}"
run_remote "sudo mkdir -p /home/vma/.ssh"
run_remote "sudo chown -R vma:vma /opt/vma /home/vma/.ssh"
run_remote "sudo chmod 750 /opt/vma/{config,logs}"
run_remote "sudo chmod 700 /opt/vma/{enrollment,ssh} /home/vma/.ssh"

log "${GREEN}✅ Directory structure created${NC}"
echo ""

# =============================================================================
# PHASE 4: VMA BINARY AND DEPENDENCY DEPLOYMENT
# =============================================================================

log "${BLUE}📋 Phase 4: VMA Binary and Dependency Deployment${NC}"
log "==============================================="

log "${YELLOW}📦 Deploying VMA binaries from package...${NC}"

# Deploy MigrateKit
log "${BLUE}   MigrateKit: migratekit-v2.21.1-chunk-size-fix${NC}"
copy_file "$PACKAGE_DIR/binaries/migratekit" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/migratekit /opt/vma/bin/"
run_remote "sudo chmod +x /opt/vma/bin/migratekit"
run_remote "sudo chown vma:vma /opt/vma/bin/migratekit"

# Deploy VMA API Server
log "${BLUE}   VMA API Server: vma-api-server-multi-disk-debug${NC}"
copy_file "$PACKAGE_DIR/binaries/vma-api-server" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/vma-api-server /opt/vma/bin/"
run_remote "sudo chmod +x /opt/vma/bin/vma-api-server"
run_remote "sudo chown vma:vma /opt/vma/bin/vma-api-server"

# Deploy VMware VDDK Libraries (CRITICAL)
log "${YELLOW}📦 Deploying VMware VDDK libraries (132MB)...${NC}"
copy_file "$PACKAGE_DIR/vddk/vmware-vddk-complete.tar.gz" vma@$TARGET_IP:/tmp/
run_remote "cd /usr/lib && sudo tar xzf /tmp/vmware-vddk-complete.tar.gz"
run_remote "echo '/usr/lib/vmware-vix-disklib/lib64' | sudo tee /etc/ld.so.conf.d/vmware-vix-disklib.conf"

# Create VDDK symlinks (CRITICAL for NBDKit compatibility)
log "${YELLOW}🔗 Creating VDDK compatibility symlinks...${NC}"
run_remote "sudo mkdir -p /usr/lib/x86_64-linux-gnu/vmware-vix-disklib/lib64"
run_remote "cd /usr/lib/x86_64-linux-gnu/vmware-vix-disklib/lib64 && sudo ln -sf /usr/lib/vmware-vix-disklib/lib64/libvixDiskLib.so libvixDiskLib.so && sudo ln -sf /usr/lib/vmware-vix-disklib/lib64/libvixDiskLib.so.8 libvixDiskLib.so.8 && sudo ln -sf /usr/lib/vmware-vix-disklib/lib64/libvixDiskLib.so.8.0.3 libvixDiskLib.so.8.0.3 && sudo ln -sf /usr/lib/vmware-vix-disklib/lib64/libvixDiskLib.so.8.0.3 libvixDiskLib.so.9"
run_remote "sudo ldconfig"

# Deploy NBDKit VDDK Plugin (CRITICAL)
log "${YELLOW}📦 Deploying NBDKit VDDK plugin...${NC}"
copy_file "$PACKAGE_DIR/nbdkit-plugins/nbdkit-vddk-plugin.so" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/nbdkit-vddk-plugin.so /usr/lib/x86_64-linux-gnu/nbdkit/plugins/"

# Create compatibility symlinks and directory structure
run_remote "sudo ln -sf /opt/vma/bin/migratekit /usr/local/bin/migratekit"
run_remote "sudo mkdir -p /home/pgrayson/migratekit-cloudstack"
run_remote "sudo ln -sf /opt/vma/bin/migratekit /home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel"

log "${GREEN}✅ VMA binaries and dependencies deployed${NC}"
echo ""

# =============================================================================
# PHASE 5: SSH KEY DEPLOYMENT
# =============================================================================

log "${BLUE}📋 Phase 5: SSH Key Deployment${NC}"
log "==============================="

log "${YELLOW}🔐 Deploying VMA SSH keys...${NC}"

# Deploy private key
copy_file "$PACKAGE_DIR/keys/cloudstack_key" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/cloudstack_key /home/vma/.ssh/"
run_remote "sudo chmod 600 /home/vma/.ssh/cloudstack_key"
run_remote "sudo chown vma:vma /home/vma/.ssh/cloudstack_key"

# Deploy public key
copy_file "$PACKAGE_DIR/keys/cloudstack_key.pub" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/cloudstack_key.pub /home/vma/.ssh/"
run_remote "sudo chmod 644 /home/vma/.ssh/cloudstack_key.pub"
run_remote "sudo chown vma:vma /home/vma/.ssh/cloudstack_key.pub"

log "${GREEN}✅ SSH keys deployed${NC}"
echo ""

# =============================================================================
# PHASE 6: SERVICE CONFIGURATION
# =============================================================================

log "${BLUE}📋 Phase 6: Service Configuration${NC}"
log "================================="

log "${YELLOW}⚙️ Deploying VMA service configurations...${NC}"

# Deploy VMA API service
copy_file "$PACKAGE_DIR/configs/vma-api.service" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/vma-api.service /etc/systemd/system/"

# Deploy SSH tunnel service
copy_file "$PACKAGE_DIR/configs/vma-ssh-tunnel.service" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/vma-ssh-tunnel.service /etc/systemd/system/"

# Deploy tunnel wrapper
copy_file "$PACKAGE_DIR/configs/vma-tunnel-wrapper.sh" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/vma-tunnel-wrapper.sh /usr/local/bin/"
run_remote "sudo chmod +x /usr/local/bin/vma-tunnel-wrapper.sh"

# Deploy fixed wizard
copy_file "$PACKAGE_DIR/scripts/vma-setup-wizard.sh" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/vma-setup-wizard.sh /opt/vma/setup-wizard.sh"
run_remote "sudo chmod +x /opt/vma/setup-wizard.sh"
run_remote "sudo chown vma:vma /opt/vma/setup-wizard.sh"

# Deploy auto-login service (boots to setup wizard)
copy_file "$PACKAGE_DIR/configs/vma-autologin.service" vma@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/vma-autologin.service /etc/systemd/system/"

# Reload systemd
run_remote "sudo systemctl daemon-reload"

log "${GREEN}✅ Service configurations deployed${NC}"
echo ""

# =============================================================================
# PHASE 7: SERVICE STARTUP
# =============================================================================

log "${BLUE}📋 Phase 7: Service Startup${NC}"
log "=========================="

log "${YELLOW}🚀 Starting VMA services...${NC}"

# Start VMA API
run_remote "sudo systemctl enable vma-api"
run_remote "sudo systemctl start vma-api"
sleep 5

# Configure auto-login (setup wizard on boot)
log "${YELLOW}🖥️ Configuring auto-login to setup wizard...${NC}"
run_remote "sudo systemctl enable vma-autologin"
run_remote "sudo systemctl disable getty@tty1"

log "${GREEN}✅ VMA services started${NC}"
log "${GREEN}✅ Auto-login configured (setup wizard on boot)${NC}"
echo ""

# =============================================================================
# PHASE 8: VALIDATION
# =============================================================================

log "${BLUE}📋 Phase 8: Production Validation${NC}"
log "================================="

log "${YELLOW}🔍 Testing VMA components...${NC}"

validation_results=""

# VMA API health
if curl -s --connect-timeout 10 http://$TARGET_IP:8081/api/v1/health > /dev/null 2>&1; then
    log "${GREEN}✅ VMA API: Working${NC}"
    validation_results="${validation_results}VMA API: ✅\n"
else
    log "${RED}❌ VMA API: Failed${NC}"
    validation_results="${validation_results}VMA API: ❌\n"
fi

# Binary verification
if run_remote "test -x /opt/vma/bin/migratekit"; then
    log "${GREEN}✅ MigrateKit: Deployed${NC}"
    validation_results="${validation_results}MigrateKit: ✅\n"
else
    log "${RED}❌ MigrateKit: Missing${NC}"
    validation_results="${validation_results}MigrateKit: ❌\n"
fi

# VDDK Libraries verification
if run_remote "test -d /usr/lib/vmware-vix-disklib"; then
    log "${GREEN}✅ VDDK Libraries: Deployed${NC}"
    validation_results="${validation_results}VDDK Libraries: ✅\n"
else
    log "${RED}❌ VDDK Libraries: Missing${NC}"
    validation_results="${validation_results}VDDK Libraries: ❌\n"
fi

# NBDKit VDDK Plugin verification
if run_remote "test -f /usr/lib/x86_64-linux-gnu/nbdkit/plugins/nbdkit-vddk-plugin.so"; then
    log "${GREEN}✅ NBDKit VDDK Plugin: Deployed${NC}"
    validation_results="${validation_results}NBDKit Plugin: ✅\n"
else
    log "${RED}❌ NBDKit VDDK Plugin: Missing${NC}"
    validation_results="${validation_results}NBDKit Plugin: ❌\n"
fi

# SSH keys
if run_remote "test -f /home/vma/.ssh/cloudstack_key"; then
    log "${GREEN}✅ SSH Keys: Deployed${NC}"
    validation_results="${validation_results}SSH Keys: ✅\n"
else
    log "${RED}❌ SSH Keys: Missing${NC}"
    validation_results="${validation_results}SSH Keys: ❌\n"
fi

# Wizard
if run_remote "test -x /opt/vma/setup-wizard.sh"; then
    log "${GREEN}✅ Setup Wizard: Deployed${NC}"
    validation_results="${validation_results}Setup Wizard: ✅\n"
else
    log "${RED}❌ Setup Wizard: Missing${NC}"
    validation_results="${validation_results}Setup Wizard: ❌\n"
fi

# Auto-login service
if run_remote "systemctl is-enabled vma-autologin > /dev/null 2>&1"; then
    log "${GREEN}✅ Auto-login: Configured${NC}"
    validation_results="${validation_results}Auto-login: ✅\n"
else
    log "${RED}❌ Auto-login: Missing${NC}"
    validation_results="${validation_results}Auto-login: ❌\n"
fi

log "${GREEN}✅ Validation completed${NC}"
echo ""

# =============================================================================
# FINAL SUMMARY
# =============================================================================

log "${BLUE}🎉 VMA PRODUCTION DEPLOYMENT COMPLETE!${NC}"
log "====================================="
echo ""
log "${GREEN}📊 DEPLOYMENT RESULTS:${NC}"
echo -e "$validation_results"
echo ""
log "${BLUE}🔗 ACCESS POINTS:${NC}"
log "   - VMA API: http://$TARGET_IP:8081"
log "   - SSH Access: ssh vma@$TARGET_IP"
log "   - Setup Wizard: /opt/vma/setup-wizard.sh"
echo ""
log "${BLUE}📦 DEPLOYED COMPONENTS:${NC}"
log "   - MigrateKit: migratekit-v2.21.1-chunk-size-fix"
log "   - VMA API: vma-api-server-multi-disk-debug"
log "   - SSH Keys: cloudstack_key (pre-shared)"
log "   - Setup Wizard: Fixed version with quoted SETUP_DATE"
echo ""
log "${YELLOW}📋 NEXT STEPS:${NC}"
log "   1. Run setup wizard: /opt/vma/setup-wizard.sh"
log "   2. Configure OMA connection"
log "   3. Test tunnel connectivity"
echo ""
log "${GREEN}🚀 VMA PRODUCTION TEMPLATE READY!${NC}"

exit 0
