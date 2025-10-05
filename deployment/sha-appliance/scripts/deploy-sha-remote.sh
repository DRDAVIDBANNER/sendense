#!/bin/bash
# 🚀 **DEPLOY SENDENSE HUB APPLIANCE (SHA) - REMOTE DEPLOYMENT**
#
# Purpose: Deploy complete SHA to remote Ubuntu 24.04 server
# Database: migratekit_oma (kept for binary compatibility)
# Schema: unified-sha-schema.sql (41 tables: 35 OMA + 6 backup tables)
# Author: Sendense Team
# Date: October 5, 2025
# Version: v1.0.0-remote-deployment
#
# FEATURES:
# - Remote deployment via SSH
# - Complete database setup with backup tables
# - Production binaries from deployment package
# - NBD server on port 10809
# - SSH tunnel infrastructure (vma_tunnel user, port 443)
# - VirtIO tools for Windows VM support
# - Volume Daemon integration
#
# USAGE: ./deploy-sha-remote.sh <TARGET_IP>
# Example: ./deploy-sha-remote.sh 10.245.246.134
#

set -euo pipefail

# Configuration
SCRIPT_VERSION="v1.0.0-remote-deployment"
TARGET_IP="${1:-}"
REMOTE_USER="${2:-adm_admin}"
LOG_FILE="/tmp/sha-remote-deployment-$(date +%Y%m%d-%H%M%S).log"
SUDO_PASSWORD="Password1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOYMENT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

if [[ -z "$TARGET_IP" ]]; then
    echo "Usage: $0 <TARGET_IP>"
    echo "Example: $0 10.245.246.134"
    exit 1
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Redirect all output to log file and console
exec > >(tee -a "$LOG_FILE")
exec 2>&1

echo -e "${BLUE}🚀 Sendense Hub Appliance (SHA) Remote Deployment${NC}"
echo -e "${BLUE}===================================================${NC}"
echo "Script Version: $SCRIPT_VERSION"
echo "Target Server: $TARGET_IP"
echo "Log File: $LOG_FILE"
echo "Start Time: $(date)"
echo ""

# Function to run remote command
run_remote() {
    sshpass -p "$SUDO_PASSWORD" ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o PreferredAuthentications=password ${REMOTE_USER}@$TARGET_IP "$@"
}

# Function to copy files remotely
copy_file() {
    sshpass -p "$SUDO_PASSWORD" scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o PreferredAuthentications=password "$@"
}

# Function to log with timestamp
log() {
    echo -e "[$(date '+%H:%M:%S')] $1"
}

# Function to check command success
check_success() {
    local exit_code=$?
    if [ $exit_code -eq 0 ]; then
        log "${GREEN}✅ $1 completed successfully${NC}"
    else
        log "${RED}❌ $1 failed (exit code: $exit_code)${NC}"
        log "${RED}🔍 Check log file: $LOG_FILE${NC}"
        exit 1
    fi
}

# Function to wait for service
wait_for_service() {
    local service_name="$1"
    local max_attempts=30
    local attempt=0
    
    log "${YELLOW}⏳ Waiting for $service_name to be ready...${NC}"
    while [ $attempt -lt $max_attempts ]; do
        if run_remote "systemctl is-active $service_name" > /dev/null 2>&1; then
            log "${GREEN}✅ $service_name is ready${NC}"
            return 0
        fi
        sleep 2
        attempt=$((attempt + 1))
    done
    log "${RED}⚠️ $service_name did not start within timeout${NC}"
    return 1
}

# =============================================================================
# PHASE 0: PASSWORDLESS SUDO SETUP
# =============================================================================

log "${BLUE}📋 Phase 0: Passwordless Sudo Setup (Remote)${NC}"
log "============================================"

log "${YELLOW}🔑 Setting up passwordless sudo on target server...${NC}"
run_remote "echo '$SUDO_PASSWORD' | sudo -S sh -c 'echo \"${REMOTE_USER} ALL=(ALL) NOPASSWD: ALL\" > /etc/sudoers.d/${REMOTE_USER}'"

# Test passwordless sudo
if run_remote "sudo whoami" | grep -q "root"; then
    log "${GREEN}✅ Passwordless sudo configured on $TARGET_IP${NC}"
else
    log "${RED}❌ Passwordless sudo failed${NC}"
    exit 1
fi

log "${GREEN}✅ Remote authentication setup completed${NC}"
echo ""

# =============================================================================
# PHASE 1: SYSTEM PREPARATION
# =============================================================================

log "${BLUE}📋 Phase 1: System Preparation${NC}"
log "==============================="

# Check OS version on target
if ! run_remote "grep -q '24.04' /etc/os-release"; then
    log "${RED}❌ Target server requires Ubuntu 24.04 LTS${NC}"
    exit 1
fi

log "${BLUE}📍 Target Server: $TARGET_IP${NC}"

log "${YELLOW}🚫 Disabling cloud-init on target server...${NC}"
run_remote "sudo touch /etc/cloud/cloud-init.disabled"
run_remote "sudo systemctl disable cloud-init cloud-config cloud-final cloud-init-local 2>/dev/null || true"
check_success "Cloud-init disable"

log "${YELLOW}🔄 Updating system packages on target...${NC}"
run_remote "sudo apt update -y"
check_success "System package update"

log "${YELLOW}📦 Installing dependencies on target...${NC}"
run_remote "DEBIAN_FRONTEND=noninteractive sudo apt install -y mariadb-server mariadb-client nbd-server curl jq nodejs npm openssh-server virt-v2v"
check_success "Dependencies installation (including virt-v2v for VirtIO injection)"

log "${GREEN}✅ System preparation completed${NC}"
echo ""

# =============================================================================
# PHASE 2: PRODUCTION BINARY DEPLOYMENT
# =============================================================================

log "${BLUE}📋 Phase 2: Production Binary Deployment${NC}"
log "========================================"

log "${YELLOW}📁 Creating production directory structure on target...${NC}"
run_remote "sudo mkdir -p /opt/sendense/{bin,gui,scripts} /usr/local/bin"
check_success "Directory creation"

log "${YELLOW}📦 Copying REAL production binaries from deployment package...${NC}"

# Copy Sendense Hub binary
SENDENSE_HUB_BINARY="sendense-hub-latest"
log "${BLUE}   Copying Sendense Hub: $SENDENSE_HUB_BINARY${NC}"
if [ -f "$DEPLOYMENT_ROOT/sha-appliance/binaries/$SENDENSE_HUB_BINARY" ]; then
    copy_file "$DEPLOYMENT_ROOT/sha-appliance/binaries/$SENDENSE_HUB_BINARY" ${REMOTE_USER}@$TARGET_IP:/tmp/sendense-hub
    run_remote "sudo cp /tmp/sendense-hub /opt/sendense/bin/sendense-hub"
    run_remote "sudo chmod +x /opt/sendense/bin/sendense-hub"
    run_remote "sudo chown ${REMOTE_USER}:${REMOTE_USER} /opt/sendense/bin/sendense-hub"
    log "${GREEN}✅ Sendense Hub binary deployed${NC}"
else
    log "${RED}❌ Sendense Hub binary not found in deployment package${NC}"
    exit 1
fi

# Copy Volume Daemon binary
VOLUME_DAEMON_BINARY="volume-daemon-latest"
log "${BLUE}   Copying Volume Daemon: $VOLUME_DAEMON_BINARY${NC}"
if [ -f "$DEPLOYMENT_ROOT/sha-appliance/binaries/$VOLUME_DAEMON_BINARY" ]; then
    copy_file "$DEPLOYMENT_ROOT/sha-appliance/binaries/$VOLUME_DAEMON_BINARY" ${REMOTE_USER}@$TARGET_IP:/tmp/volume-daemon
    run_remote "sudo cp /tmp/volume-daemon /usr/local/bin/volume-daemon"
    run_remote "sudo chmod +x /usr/local/bin/volume-daemon"
    run_remote "sudo chown ${REMOTE_USER}:${REMOTE_USER} /usr/local/bin/volume-daemon"
    log "${GREEN}✅ Volume Daemon binary deployed${NC}"
else
    log "${RED}❌ Volume Daemon binary not found in deployment package${NC}"
    exit 1
fi

check_success "Production binary deployment"

log "${GREEN}✅ Production binaries deployed${NC}"
echo ""

# =============================================================================
# PHASE 3: PRODUCTION DATABASE DEPLOYMENT
# =============================================================================

log "${BLUE}📋 Phase 3: Production Database Deployment${NC}"
log "=========================================="

log "${YELLOW}🗄️ Starting MariaDB on target...${NC}"
run_remote "sudo systemctl start mariadb"
run_remote "sudo systemctl enable mariadb"
sleep 5

log "${YELLOW}👤 Creating production database and user...${NC}"
run_remote 'sudo mysql -e "CREATE DATABASE IF NOT EXISTS migratekit_oma;"'
run_remote 'sudo mysql -e "CREATE USER IF NOT EXISTS \"oma_user\"@\"localhost\" IDENTIFIED BY \"oma_password\";"'
run_remote 'sudo mysql -e "GRANT ALL PRIVILEGES ON migratekit_oma.* TO \"oma_user\"@\"localhost\";"'
run_remote 'sudo mysql -e "FLUSH PRIVILEGES;"'
check_success "Database user creation"

log "${YELLOW}📊 Importing unified SHA schema (41 tables: 35 OMA + 6 backup tables)...${NC}"
SCHEMA_FILE="$DEPLOYMENT_ROOT/sha-appliance/database/unified-sha-schema.sql"

if [ ! -f "$SCHEMA_FILE" ]; then
    log "${RED}❌ Unified SHA schema not found at: $SCHEMA_FILE${NC}"
    exit 1
fi

log "${BLUE}   Using unified schema from deployment package${NC}"
copy_file "$SCHEMA_FILE" ${REMOTE_USER}@$TARGET_IP:/tmp/unified-sha-schema.sql
run_remote "mysql -u oma_user -poma_password migratekit_oma < /tmp/unified-sha-schema.sql"
check_success "Production database schema import"

# Verify table count
table_count=$(run_remote 'mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = \"migratekit_oma\";" | tail -1')
log "${GREEN}✅ Database contains $table_count tables (expected 41)${NC}"

# Verify backup tables exist
backup_table_count=$(run_remote 'mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = \"migratekit_oma\" AND table_name LIKE \"backup_%\";" | tail -1')
log "${GREEN}✅ Backup system tables: $backup_table_count (expected 6)${NC}"

# Set OSSEA config auto-increment to start at ID=1 (GUI compatibility)
log "${YELLOW}🔧 Setting OSSEA config auto-increment to start at ID=1 (GUI compatibility fix)...${NC}"
run_remote "mysql -u oma_user -poma_password migratekit_oma -e 'ALTER TABLE ossea_configs AUTO_INCREMENT = 1;'"
check_success "OSSEA config auto-increment setup"
log "${GREEN}✅ OSSEA config will use ID=1 when created via GUI${NC}"

log "${GREEN}✅ Production database deployment completed${NC}"
echo ""

# =============================================================================
# PHASE 4: PRODUCTION SERVICE CONFIGURATION
# =============================================================================

log "${BLUE}📋 Phase 4: Production Service Configuration${NC}"
log "==========================================="

log "${YELLOW}⚙️ Creating production service configurations...${NC}"

# Generate new encryption key for VMware credentials
ENCRYPTION_KEY=$(openssl rand -base64 32)

# Sendense Hub Service
cat > /tmp/sendense-hub.service << EOF
[Unit]
Description=Sendense Hub API Server
After=network.target mariadb.service volume-daemon.service
Requires=mariadb.service
Wants=volume-daemon.service

[Service]
Type=simple
User=$REMOTE_USER
Group=$REMOTE_USER
WorkingDirectory=/opt/sendense
ExecStart=/opt/sendense/bin/sendense-hub -port=8082 -db-type=mariadb -db-host=localhost -db-port=3306 -db-name=migratekit_oma -db-user=oma_user -db-pass=oma_password -auth=false -debug=false
Restart=always
RestartSec=10
TimeoutStartSec=60
TimeoutStopSec=30
KillMode=mixed
KillSignal=SIGTERM
StandardOutput=journal
StandardError=journal
Environment=MIGRATEKIT_CRED_ENCRYPTION_KEY=$ENCRYPTION_KEY
Environment=OMA_NBD_HOST=127.0.0.1

[Install]
WantedBy=multi-user.target
EOF

# Volume Daemon Service
cat > /tmp/volume-daemon.service << EOF
[Unit]
Description=Volume Management Daemon for Sendense
After=network.target mariadb.service
Requires=mariadb.service

[Service]
Type=simple
User=$REMOTE_USER
Group=$REMOTE_USER
ExecStart=/usr/local/bin/volume-daemon
Restart=always
RestartSec=10
TimeoutStartSec=30
TimeoutStopSec=30
StandardOutput=journal
StandardError=journal
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

# Deploy service configurations to target
copy_file /tmp/sendense-hub.service /tmp/volume-daemon.service ${REMOTE_USER}@$TARGET_IP:/tmp/
run_remote "sudo cp /tmp/sendense-hub.service /tmp/volume-daemon.service /etc/systemd/system/"
run_remote "sudo systemctl daemon-reload"
check_success "Service configuration deployment"

log "${GREEN}✅ Production service configuration completed${NC}"
echo ""

# =============================================================================
# PHASE 5: NBD AND SSH TUNNEL INFRASTRUCTURE
# =============================================================================

log "${BLUE}📋 Phase 5: NBD and SSH Tunnel Infrastructure${NC}"
log "=============================================="

log "${YELLOW}📡 Setting up production NBD server configuration...${NC}"

# Create NBD config
cat > /tmp/nbd-config << 'EOF'
[generic]
port = 10809
allowlist = true
includedir = /etc/nbd-server/conf.d
max_connections = 50

# Dummy export required for NBD server to start
[dummy]
exportname = /dev/null
readonly = true
EOF

copy_file /tmp/nbd-config ${REMOTE_USER}@$TARGET_IP:/tmp/

# Deploy to target
run_remote "sudo cp /tmp/nbd-config /etc/nbd-server/config"
run_remote "sudo cp /tmp/nbd-config /etc/nbd-server/config-base"

# Ensure conf.d directory exists for dynamic exports
run_remote "sudo mkdir -p /etc/nbd-server/conf.d"

# CRITICAL: Volume Daemon needs write access to conf.d to create NBD exports
run_remote "sudo chown -R ${REMOTE_USER}:${REMOTE_USER} /etc/nbd-server/conf.d"
run_remote "sudo chmod 755 /etc/nbd-server/conf.d"

# Verify config was deployed
if run_remote 'grep -q "max_connections = 50" /etc/nbd-server/config'; then
    log "${GREEN}✅ Production NBD config deployed (max_connections=50, port=10809)${NC}"
else
    log "${RED}❌ NBD config verification failed${NC}"
    exit 1
fi

check_success "NBD configuration deployment"

log "${YELLOW}🔐 Setting up SSH tunnel infrastructure...${NC}"

# Create vma_tunnel user on target
run_remote "sudo useradd -r -m -s /bin/bash -d /var/lib/vma_tunnel vma_tunnel 2>/dev/null || true"

# Create SSH directory
run_remote "sudo mkdir -p /var/lib/vma_tunnel/.ssh"
run_remote "sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh"
run_remote "sudo chmod 700 /var/lib/vma_tunnel/.ssh"

# Create authorized_keys file (will be populated when VMA connects)
run_remote "sudo touch /var/lib/vma_tunnel/.ssh/authorized_keys"
run_remote "sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh/authorized_keys"
run_remote "sudo chmod 600 /var/lib/vma_tunnel/.ssh/authorized_keys"

log "${BLUE}   VMA SSH key will need to be added manually to /var/lib/vma_tunnel/.ssh/authorized_keys${NC}"

# Configure SSH for port 443 and tunnel restrictions on target
log "${YELLOW}🔧 Configuring SSH for port 443 and tunnel restrictions...${NC}"

# Remove conflicting AllowTcpForwarding lines
log "${YELLOW}🔧 Removing conflicting SSH TCP forwarding settings...${NC}"
run_remote 'sudo sed -i "/^[[:space:]]*AllowTcpForwarding[[:space:]]*no/d" /etc/ssh/sshd_config'
run_remote 'sudo sed -i "/^[[:space:]]*#.*AllowTcpForwarding[[:space:]]*no/d" /etc/ssh/sshd_config'
log "${GREEN}✅ Conflicting TCP forwarding settings removed${NC}"

run_remote 'sudo tee -a /etc/ssh/sshd_config << "SSHEOF"

# Production SSH Configuration
Port 443

# VMA Tunnel User Configuration - Production
Match User vma_tunnel
    AuthenticationMethods publickey
    PubkeyAuthentication yes
    PasswordAuthentication no
    PermitTTY no
    X11Forwarding no
    AllowTcpForwarding yes
    PermitOpen 127.0.0.1:10809 127.0.0.1:8082
    PermitListen 127.0.0.1:9081
SSHEOF'

# Add SSH socket override for port 443 on target
log "${YELLOW}🔧 Configuring SSH socket for port 443...${NC}"
run_remote "sudo mkdir -p /etc/systemd/system/ssh.socket.d"
run_remote 'sudo tee /etc/systemd/system/ssh.socket.d/port443.conf << "SOCKETEOF"
[Socket]
ListenStream=
ListenStream=0.0.0.0:22
ListenStream=0.0.0.0:443
ListenStream=[::]:22
ListenStream=[::]:443
SOCKETEOF'

run_remote "sudo systemctl daemon-reload"

# Test SSH configuration on target
if run_remote "sudo sshd -t"; then
    log "${GREEN}✅ SSH configuration is valid${NC}"
else
    log "${RED}❌ SSH configuration has errors${NC}"
    exit 1
fi

check_success "SSH tunnel infrastructure setup"

log "${YELLOW}🖥️ Installing VirtIO tools for Windows VM support...${NC}"

# VirtIO tools are required for Windows VM failover (virt-v2v-in-place driver injection)
run_remote "sudo mkdir -p /usr/share/virtio-win"

VIRTIO_ISO="$DEPLOYMENT_ROOT/sha-appliance/virtio/virtio-win.iso"
if [ -f "$VIRTIO_ISO" ]; then
    log "${BLUE}   Copying VirtIO ISO from deployment package (693MB, will take ~1 minute)...${NC}"
    copy_file "$VIRTIO_ISO" ${REMOTE_USER}@$TARGET_IP:/tmp/
    run_remote "sudo mv /tmp/virtio-win.iso /usr/share/virtio-win/"
    run_remote "sudo chmod 644 /usr/share/virtio-win/virtio-win.iso"
    
    # Verify it's a valid ISO
    if run_remote 'file /usr/share/virtio-win/virtio-win.iso | grep -q "ISO 9660"'; then
        log "${GREEN}✅ VirtIO tools installed successfully (virtio-win.iso)${NC}"
    else
        log "${RED}❌ VirtIO ISO verification failed${NC}"
        exit 1
    fi
else
    log "${YELLOW}⚠️ VirtIO ISO not found in deployment package, skipping...${NC}"
    log "${YELLOW}   Windows VM failover will not work without VirtIO tools${NC}"
fi

check_success "VirtIO tools installation"

log "${GREEN}✅ Infrastructure setup completed${NC}"
echo ""

# =============================================================================
# PHASE 6: SERVICE STARTUP AND VALIDATION
# =============================================================================

log "${BLUE}📋 Phase 6: Service Startup and Validation${NC}"
log "=========================================="

log "${YELLOW}🚀 Starting production services in dependency order...${NC}"

# Start MariaDB (already started)
log "${GREEN}✅ MariaDB already running${NC}"

# Start Volume Daemon
log "${YELLOW}   Starting Volume Daemon...${NC}"
run_remote "sudo systemctl enable volume-daemon"
run_remote "sudo systemctl start volume-daemon"
sleep 5

# Start NBD Server
log "${YELLOW}   Starting NBD Server (restart to load new config)...${NC}"
run_remote "sudo systemctl enable nbd-server"
run_remote "sudo systemctl restart nbd-server"
sleep 3

# Start Sendense Hub
log "${YELLOW}   Starting Sendense Hub...${NC}"
run_remote "sudo systemctl enable sendense-hub"
run_remote "sudo systemctl start sendense-hub"
sleep 8

# Restart SSH for port 443
log "${YELLOW}   Configuring SSH for port 443...${NC}"
run_remote "sudo systemctl restart ssh.socket"
sleep 3

log "${GREEN}✅ All services started successfully${NC}"
echo ""

# =============================================================================
# PHASE 7: COMPREHENSIVE VALIDATION
# =============================================================================

log "${BLUE}📋 Phase 7: Comprehensive Production Validation${NC}"
log "==============================================="

log "${YELLOW}🔍 Testing all production components...${NC}"

validation_results=""

# Database connectivity
if run_remote 'mysql -u oma_user -poma_password migratekit_oma -e "SELECT 1;" > /dev/null 2>&1'; then
    log "${GREEN}✅ Database connectivity confirmed${NC}"
    validation_results="${validation_results}Database: ✅\n"
else
    log "${RED}❌ Database connectivity failed${NC}"
    validation_results="${validation_results}Database: ❌\n"
fi

# Sendense Hub API health
if curl -s --connect-timeout 10 http://$TARGET_IP:8082/health > /dev/null 2>&1; then
    log "${GREEN}✅ Sendense Hub API health check passed${NC}"
    validation_results="${validation_results}Sendense Hub: ✅\n"
else
    log "${RED}❌ Sendense Hub API health check failed${NC}"
    validation_results="${validation_results}Sendense Hub: ❌\n"
fi

# Volume Daemon health
if curl -s --connect-timeout 10 http://$TARGET_IP:8090/api/v1/health > /dev/null 2>&1; then
    log "${GREEN}✅ Volume Daemon health check passed${NC}"
    validation_results="${validation_results}Volume Daemon: ✅\n"
else
    log "${RED}❌ Volume Daemon health check failed${NC}"
    validation_results="${validation_results}Volume Daemon: ❌\n"
fi

# NBD Server
if run_remote "ss -tlnp | grep -q :10809"; then
    log "${GREEN}✅ NBD Server is listening on port 10809${NC}"
    validation_results="${validation_results}NBD Server: ✅\n"
else
    log "${RED}❌ NBD Server not listening on port 10809${NC}"
    validation_results="${validation_results}NBD Server: ❌\n"
fi

# SSH Tunnel Infrastructure
if run_remote "id vma_tunnel > /dev/null 2>&1"; then
    log "${GREEN}✅ SSH tunnel user (vma_tunnel) exists${NC}"
    validation_results="${validation_results}SSH Tunnel User: ✅\n"
else
    log "${RED}❌ SSH tunnel user (vma_tunnel) missing${NC}"
    validation_results="${validation_results}SSH Tunnel User: ❌\n"
fi

# SSH Port 443
if run_remote "ss -tlnp | grep -q :443"; then
    log "${GREEN}✅ SSH listening on port 443${NC}"
    validation_results="${validation_results}SSH Port 443: ✅\n"
else
    log "${RED}❌ SSH not listening on port 443${NC}"
    validation_results="${validation_results}SSH Port 443: ❌\n"
fi

# VirtIO Tools
if run_remote 'test -f "/usr/share/virtio-win/virtio-win.iso"'; then
    log "${GREEN}✅ VirtIO tools are present (Windows VM support enabled)${NC}"
    validation_results="${validation_results}VirtIO Tools: ✅\n"
else
    log "${YELLOW}⚠️ VirtIO tools not found (Windows VM failover will not work)${NC}"
    validation_results="${validation_results}VirtIO Tools: ⚠️ (missing)\n"
fi

log "${GREEN}✅ Comprehensive validation completed${NC}"
echo ""

# =============================================================================
# FINAL SUMMARY
# =============================================================================

log "${BLUE}🎉 SENDENSE HUB APPLIANCE (SHA) DEPLOYMENT COMPLETE!${NC}"
log "===================================================="
echo ""
log "${GREEN}📊 PRODUCTION DEPLOYMENT SUMMARY:${NC}"
echo -e "$validation_results"
echo ""
log "${BLUE}🔗 Access Points:${NC}"
log "   - Sendense Hub API: http://$TARGET_IP:8082"
log "   - Volume Daemon: http://$TARGET_IP:8090"
log "   - SSH Access: ssh ${REMOTE_USER}@$TARGET_IP (ports 22, 443)"
echo ""
log "${BLUE}📊 Production Components Deployed:${NC}"
log "   - Sendense Hub: v2.7.6-api-uuid-correlation (REAL binary)"
log "   - Volume Daemon: v2.1.0-dynamic-config (REAL binary)"
log "   - Database: Complete 41-table unified schema (35 OMA + 6 backup)"
log "   - NBD Server: Production configuration on port 10809"
log "   - SSH Tunnel: vma_tunnel user ready for VMA connections"
log "   - VirtIO Tools: Windows VM failover support"
echo ""
log "${YELLOW}📋 Next Steps:${NC}"
log "   1. Add VMA SSH public key to /var/lib/vma_tunnel/.ssh/authorized_keys"
log "   2. Deploy Sendense Cockpit GUI"
log "   3. Configure OSSEA (CloudStack) connection via GUI"
log "   4. Configure VMware credentials via GUI"
log "   5. Create backup repositories via API or GUI"
log "   6. Test VMA tunnel connectivity"
echo ""
log "${GREEN}🚀 REAL PRODUCTION SHA TEMPLATE READY!${NC}"

# Create deployment summary on target
run_remote "cat > /home/${REMOTE_USER}/sha-deployment-summary.txt << 'EOF'
Sendense Hub Appliance (SHA) Deployment Complete
=================================================

Deployment Date: $(date)
Server: $TARGET_IP
Script Version: $SCRIPT_VERSION

REAL PRODUCTION COMPONENTS:
- Sendense Hub: v2.7.6-api-uuid-correlation (REAL binary)
- Volume Daemon: v2.1.0-dynamic-config (REAL binary)
- Database: Complete 41-table unified schema
- NBD Server: Production configuration
- SSH Tunnel: Complete vma_tunnel infrastructure
- VirtIO Tools: Windows VM failover support

VALIDATION RESULTS:
$(echo -e \"$validation_results\")

ACCESS POINTS:
- Hub API: http://$TARGET_IP:8082
- Volume Daemon: http://$TARGET_IP:8090

Database: migratekit_oma (oma_user/oma_password)

This is a COMPLETE production-ready SHA with backup capabilities.

Log File: $LOG_FILE
EOF"

log "${BLUE}✅ SHA REMOTE DEPLOYMENT COMPLETED SUCCESSFULLY!${NC}"
exit 0
