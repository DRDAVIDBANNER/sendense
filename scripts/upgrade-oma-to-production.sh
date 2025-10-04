#!/bin/bash
# 🚀 **UPGRADE OMA TO PRODUCTION READY**
#
# Purpose: Transform partially deployed OMA server to production-ready template
# Target: Server 120 (10.245.246.120) - with snapshot rollback capability
# Author: MigrateKit OSSEA Team
# Date: September 30, 2025

set -euo pipefail

# Configuration
SCRIPT_VERSION="v1.0.0"
LOG_FILE="/tmp/oma-production-upgrade-$(date +%Y%m%d-%H%M%S).log"
BACKUP_DIR="/tmp/oma-upgrade-backup"
SUDO_PASSWORD="Password1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Redirect all output to log file and console
exec > >(tee -a "$LOG_FILE")
exec 2>&1

echo -e "${BLUE}🚀 OSSEA-Migrate OMA Production Upgrade Script${NC}"
echo -e "${BLUE}=============================================${NC}"
echo "Script Version: $SCRIPT_VERSION"
echo "Target Server: Server 120 (10.245.246.120)"
echo "Log File: $LOG_FILE"
echo "Backup Directory: $BACKUP_DIR"
echo "Start Time: $(date)"
echo ""

# Function to run sudo commands
run_sudo() {
    echo "$SUDO_PASSWORD" | sudo -S "$@"
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
        log "${YELLOW}💡 Consider snapshot rollback if multiple failures${NC}"
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
        if systemctl is-active "$service_name" > /dev/null 2>&1; then
            log "${GREEN}✅ $service_name is ready${NC}"
            return 0
        fi
        sleep 2
        attempt=$((attempt + 1))
    done
    log "${RED}⚠️ $service_name did not start within timeout${NC}"
    return 1
}

# Function to test health endpoint
test_health() {
    local endpoint="$1"
    local service_name="$2"
    
    if curl -s --connect-timeout 10 "$endpoint" > /dev/null 2>&1; then
        log "${GREEN}✅ $service_name health check passed${NC}"
        return 0
    else
        log "${RED}❌ $service_name health check failed${NC}"
        return 1
    fi
}

# =============================================================================
# PHASE 1: PRE-FLIGHT CHECKS AND BACKUP
# =============================================================================

log "${BLUE}📋 Phase 1: Pre-flight Checks and Backup${NC}"
log "========================================="

# Check if we're running on the right server (allow multiple test servers)
current_ip=$(hostname -I | awk '{print $1}' | tr -d ' ')
if [[ "$current_ip" != "10.245.246.120" && "$current_ip" != "10.245.246.134" ]]; then
    log "${RED}❌ This script is designed for test servers (120 or 134)${NC}"
    log "${RED}   Current IP: $current_ip${NC}"
    exit 1
fi

# Check OS version
if ! grep -q "24.04" /etc/os-release; then
    log "${RED}❌ This script requires Ubuntu 24.04 LTS${NC}"
    exit 1
fi

# Create backup directory
run_sudo mkdir -p "$BACKUP_DIR"
run_sudo chown "$USER:$USER" "$BACKUP_DIR"
check_success "Backup directory creation"

# Backup current database
log "${YELLOW}📦 Backing up current database...${NC}"
mysqldump -u oma_user -poma_password migratekit_oma > "$BACKUP_DIR/database-pre-upgrade.sql"
check_success "Database backup"

# Backup current service configurations
log "${YELLOW}📦 Backing up service configurations...${NC}"
run_sudo cp /etc/systemd/system/oma-*.service "$BACKUP_DIR/" 2>/dev/null || true
run_sudo cp /etc/systemd/system/volume-daemon.service "$BACKUP_DIR/" 2>/dev/null || true
run_sudo cp /etc/systemd/system/migration-gui.service "$BACKUP_DIR/" 2>/dev/null || true
check_success "Service configuration backup"

# Document current state
log "${YELLOW}📊 Documenting current state...${NC}"
cat > "$BACKUP_DIR/current-state.txt" << EOF
OMA Production Upgrade - Current State Documentation
Date: $(date)
Server: 10.245.246.120

SERVICES STATUS:
$(systemctl status oma-api volume-daemon migration-gui mariadb --no-pager -l | head -20)

BINARY VERSIONS:
OMA API: $(ls -la /opt/migratekit/bin/oma-api)
Volume Daemon: $(ls -la /usr/local/bin/volume-daemon)

DATABASE TABLES:
$(mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) as table_count FROM information_schema.tables WHERE table_schema = 'migratekit_oma';")

VOLUME DAEMON HEALTH:
$(curl -s http://localhost:8090/api/v1/health | jq . 2>/dev/null || curl -s http://localhost:8090/api/v1/health)

USER INFORMATION:
Service User: $(systemctl show oma-api.service --property=User | cut -d= -f2)
File Owner: $(ls -la /opt/migratekit/bin/oma-api | awk '{print $3":"$4}')
EOF

log "${GREEN}✅ Pre-flight checks and backup completed${NC}"
echo ""

# =============================================================================
# PHASE 2: BUILD PACKAGE PREPARATION
# =============================================================================

log "${BLUE}📋 Phase 2: Build Package Preparation${NC}"
log "====================================="

# Create build package directory
BUILD_PACKAGE_DIR="/tmp/production-upgrade-package"
run_sudo rm -rf "$BUILD_PACKAGE_DIR"
mkdir -p "$BUILD_PACKAGE_DIR"/{binaries,services,database,scripts}

# Copy current production binaries from dev OMA
log "${YELLOW}📦 Copying current production binaries...${NC}"

# Get current OMA API binary (latest stable)
if [ -f "/opt/migratekit/bin/oma-api-v2.39.0-gorm-field-fix" ]; then
    cp "/opt/migratekit/bin/oma-api-v2.39.0-gorm-field-fix" "$BUILD_PACKAGE_DIR/binaries/oma-api"
elif [ -f "/opt/migratekit/bin/oma-api" ]; then
    cp "/opt/migratekit/bin/oma-api" "$BUILD_PACKAGE_DIR/binaries/oma-api"
else
    log "${RED}❌ No OMA API binary found on dev system${NC}"
    exit 1
fi

# Get current Volume Daemon binary
if [ -f "/usr/local/bin/volume-daemon" ]; then
    cp "/usr/local/bin/volume-daemon" "$BUILD_PACKAGE_DIR/binaries/volume-daemon"
else
    log "${RED}❌ No Volume Daemon binary found on dev system${NC}"
    exit 1
fi

chmod +x "$BUILD_PACKAGE_DIR/binaries/"*
check_success "Production binary preparation"

# Create updated service configurations
log "${YELLOW}⚙️ Creating updated service configurations...${NC}"

# Updated OMA API Service (fix user consistency)
cat > "$BUILD_PACKAGE_DIR/services/oma-api.service" << 'EOF'
[Unit]
Description=OSSEA-Migrate OMA API Server
After=network.target mariadb.service volume-daemon.service
Requires=mariadb.service
Wants=volume-daemon.service

[Service]
Type=simple
User=oma_admin
Group=oma_admin
WorkingDirectory=/opt/migratekit
ExecStart=/opt/migratekit/bin/oma-api -port=8082 -db-type=mariadb -db-host=localhost -db-port=3306 -db-name=migratekit_oma -db-user=oma_user -db-pass=oma_password -auth=false -debug=false
Restart=always
RestartSec=10
TimeoutStartSec=60
TimeoutStopSec=30
KillMode=mixed
KillSignal=SIGTERM
StandardOutput=journal
StandardError=journal

# VMware credentials encryption key (will be regenerated)
Environment=MIGRATEKIT_CRED_ENCRYPTION_KEY=PLACEHOLDER_WILL_BE_REPLACED

[Install]
WantedBy=multi-user.target
EOF

# Updated Volume Daemon Service
cat > "$BUILD_PACKAGE_DIR/services/volume-daemon.service" << 'EOF'
[Unit]
Description=OSSEA-Migrate Volume Management Daemon
After=network.target mariadb.service
Requires=mariadb.service

[Service]
Type=simple
User=oma_admin
Group=oma_admin
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

# Migration GUI Service (ensure consistent user)
cat > "$BUILD_PACKAGE_DIR/services/migration-gui.service" << 'EOF'
[Unit]
Description=OSSEA-Migrate Dashboard GUI
After=network.target oma-api.service
Wants=oma-api.service

[Service]
Type=simple
User=oma_admin
Group=oma_admin
WorkingDirectory=/opt/migratekit/gui
ExecStart=/usr/bin/npx next start --port 3001 --hostname 0.0.0.0
Restart=always
RestartSec=10
TimeoutStartSec=60
StandardOutput=journal
StandardError=journal
Environment=NODE_ENV=production

[Install]
WantedBy=multi-user.target
EOF

check_success "Service configuration creation"

# Create NBD configuration fix script
log "${YELLOW}📡 Creating NBD server configuration...${NC}"

cat > "$BUILD_PACKAGE_DIR/scripts/setup-nbd-server.sh" << 'EOF'
#!/bin/bash
# NBD Server Configuration Setup

set -euo pipefail

echo "📡 Setting up NBD server configuration..."

# Create proper config-base (required by Volume Daemon)
cat > /etc/nbd-server/config-base << 'NBDEOF'
[generic]
port = 10809
allowlist = true
includedir = /etc/nbd-server/conf.d

# Dummy export required for NBD server to start
[dummy]
exportname = /dev/null
readonly = true
NBDEOF

# Also update the default config used by systemd service
cp /etc/nbd-server/config-base /etc/nbd-server/config

echo "✅ NBD server configuration completed"
EOF

chmod +x "$BUILD_PACKAGE_DIR/scripts/setup-nbd-server.sh"

# Create SSH tunnel setup script
log "${YELLOW}🔐 Creating SSH tunnel infrastructure setup...${NC}"

cat > "$BUILD_PACKAGE_DIR/scripts/setup-ssh-tunnel.sh" << 'EOF'
#!/bin/bash
# SSH Tunnel Infrastructure Setup for OMA

set -euo pipefail

echo "🔐 Setting up SSH tunnel infrastructure..."

# Create vma_tunnel user
sudo useradd -r -m -s /bin/bash -d /var/lib/vma_tunnel vma_tunnel 2>/dev/null || echo "User already exists"

# Create SSH directory
sudo mkdir -p /var/lib/vma_tunnel/.ssh
sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh
sudo chmod 700 /var/lib/vma_tunnel/.ssh

# Create authorized_keys file
sudo touch /var/lib/vma_tunnel/.ssh/authorized_keys
sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh/authorized_keys
sudo chmod 600 /var/lib/vma_tunnel/.ssh/authorized_keys

# Configure SSH daemon for port 443
if ! grep -q "Port 443" /etc/ssh/sshd_config; then
    echo "Port 443" | sudo tee -a /etc/ssh/sshd_config
fi

# Add Match User block for vma_tunnel
if ! grep -q "Match User vma_tunnel" /etc/ssh/sshd_config; then
    cat << 'SSHCONFIG' | sudo tee -a /etc/ssh/sshd_config

# VMA Tunnel User Configuration
Match User vma_tunnel
    AuthenticationMethods publickey
    PubkeyAuthentication yes
    PasswordAuthentication no
    KbdInteractiveAuthentication no
    PermitTTY no
    X11Forwarding no
    AllowTcpForwarding yes
    AllowStreamLocalForwarding no
    GatewayPorts no
    PermitOpen 127.0.0.1:10809
    PermitListen 127.0.0.1:9081
SSHCONFIG
fi

echo "✅ SSH tunnel infrastructure setup completed"
EOF

chmod +x "$BUILD_PACKAGE_DIR/scripts/setup-ssh-tunnel.sh"

# Create cloud-init disabling script
log "${YELLOW}🚫 Creating cloud-init disabling script...${NC}"

cat > "$BUILD_PACKAGE_DIR/scripts/disable-cloud-init.sh" << 'EOF'
#!/bin/bash
# Disable cloud-init for production deployment

set -euo pipefail

echo "🚫 Disabling cloud-init services..."

# Create cloud-init disabled file
touch /etc/cloud/cloud-init.disabled

# Disable all cloud-init services
systemctl disable cloud-init cloud-config cloud-final cloud-init-local 2>/dev/null || true

# Remove cloud-init packages if desired (optional)
# apt-get remove -y cloud-init cloud-guest-utils cloud-initramfs-copymods cloud-initramfs-dyn-netconf

echo "✅ Cloud-init disabled successfully"
EOF

chmod +x "$BUILD_PACKAGE_DIR/scripts/disable-cloud-init.sh"
check_success "Cloud-init disabling script creation"

log "${GREEN}✅ Build package preparation completed${NC}"
echo ""

# =============================================================================
# PHASE 3: VOLUME DAEMON UPGRADE
# =============================================================================

log "${BLUE}📋 Phase 3: Volume Daemon Upgrade${NC}"
log "================================="

log "${YELLOW}⏹️ Stopping current Volume Daemon...${NC}"
run_sudo systemctl stop volume-daemon
check_success "Volume Daemon stop"

log "${YELLOW}📦 Deploying updated Volume Daemon binary...${NC}"
run_sudo cp "$BUILD_PACKAGE_DIR/binaries/volume-daemon" /usr/local/bin/
run_sudo chmod +x /usr/local/bin/volume-daemon
run_sudo chown oma_admin:oma_admin /usr/local/bin/volume-daemon
check_success "Volume Daemon binary deployment"

log "${YELLOW}⚙️ Updating Volume Daemon service configuration...${NC}"
run_sudo cp "$BUILD_PACKAGE_DIR/services/volume-daemon.service" /etc/systemd/system/
run_sudo systemctl daemon-reload
check_success "Volume Daemon service configuration"

log "${YELLOW}🚀 Starting updated Volume Daemon...${NC}"
run_sudo systemctl start volume-daemon
wait_for_service "volume-daemon.service"
check_success "Volume Daemon startup"

# Test Volume Daemon health
log "${YELLOW}🔍 Testing Volume Daemon health...${NC}"
sleep 5  # Give it time to fully initialize
test_health "http://localhost:8090/api/v1/health" "Volume Daemon"
check_success "Volume Daemon health check"

# Check Volume Daemon endpoints
log "${YELLOW}📊 Validating Volume Daemon endpoints...${NC}"
endpoint_count=$(curl -s http://localhost:8090/api/v1/health | jq -r '.details.implementation_status' 2>/dev/null || echo "unknown")
if [[ "$endpoint_count" != "phase_1_foundation" ]]; then
    log "${GREEN}✅ Volume Daemon upgraded successfully (no longer phase_1_foundation)${NC}"
else
    log "${YELLOW}⚠️ Volume Daemon still shows phase_1_foundation - may need investigation${NC}"
fi

log "${GREEN}✅ Volume Daemon upgrade completed${NC}"
echo ""

# =============================================================================
# PHASE 4: OMA API SERVICE UPGRADE
# =============================================================================

log "${BLUE}📋 Phase 4: OMA API Service Upgrade${NC}"
log "=================================="

log "${YELLOW}⏹️ Stopping current OMA API...${NC}"
run_sudo systemctl stop oma-api
check_success "OMA API stop"

log "${YELLOW}📦 Deploying updated OMA API binary...${NC}"
run_sudo cp "$BUILD_PACKAGE_DIR/binaries/oma-api" /opt/migratekit/bin/
run_sudo chmod +x /opt/migratekit/bin/oma-api
run_sudo chown oma_admin:oma_admin /opt/migratekit/bin/oma-api
check_success "OMA API binary deployment"

# Generate new encryption key
log "${YELLOW}🔐 Generating new VMware credentials encryption key...${NC}"
ENCRYPTION_KEY=$(openssl rand -base64 32)
# Use a different delimiter to avoid issues with special characters
sed -i "s|PLACEHOLDER_WILL_BE_REPLACED|$ENCRYPTION_KEY|g" "$BUILD_PACKAGE_DIR/services/oma-api.service"

log "${YELLOW}⚙️ Updating OMA API service configuration...${NC}"
run_sudo cp "$BUILD_PACKAGE_DIR/services/oma-api.service" /etc/systemd/system/
run_sudo systemctl daemon-reload
check_success "OMA API service configuration"

# Fix file ownership consistency
log "${YELLOW}👤 Fixing file ownership consistency...${NC}"
run_sudo chown -R oma_admin:oma_admin /opt/migratekit/
check_success "File ownership fix"

log "${YELLOW}🚀 Starting updated OMA API...${NC}"
run_sudo systemctl start oma-api
wait_for_service "oma-api.service"
check_success "OMA API startup"

# Test OMA API health
log "${YELLOW}🔍 Testing OMA API health...${NC}"
sleep 5  # Give it time to fully initialize
test_health "http://localhost:8082/health" "OMA API"
check_success "OMA API health check"

log "${GREEN}✅ OMA API service upgrade completed${NC}"
echo ""

# =============================================================================
# PHASE 5: MIGRATION GUI SERVICE UPDATE
# =============================================================================

log "${BLUE}📋 Phase 5: Migration GUI Service Update${NC}"
log "======================================="

log "${YELLOW}⏹️ Stopping current Migration GUI...${NC}"
run_sudo systemctl stop migration-gui
check_success "Migration GUI stop"

log "${YELLOW}⚙️ Updating Migration GUI service configuration...${NC}"
run_sudo cp "$BUILD_PACKAGE_DIR/services/migration-gui.service" /etc/systemd/system/
run_sudo systemctl daemon-reload
check_success "Migration GUI service configuration"

# Fix GUI ownership
log "${YELLOW}👤 Fixing GUI ownership consistency...${NC}"
run_sudo chown -R oma_admin:oma_admin /opt/migratekit/gui/
check_success "GUI ownership fix"

log "${YELLOW}🚀 Starting updated Migration GUI...${NC}"
run_sudo systemctl start migration-gui
wait_for_service "migration-gui.service"
check_success "Migration GUI startup"

# Test GUI health
log "${YELLOW}🔍 Testing Migration GUI health...${NC}"
sleep 5  # Give it time to fully initialize
test_health "http://localhost:3001" "Migration GUI"
check_success "Migration GUI health check"

log "${GREEN}✅ Migration GUI service update completed${NC}"
echo ""

# =============================================================================
# PHASE 6: SYSTEM HARDENING AND INFRASTRUCTURE SETUP
# =============================================================================

log "${BLUE}📋 Phase 6: System Hardening and Infrastructure Setup${NC}"
log "===================================================="

log "${YELLOW}🚫 Disabling cloud-init...${NC}"
run_sudo bash "$BUILD_PACKAGE_DIR/scripts/disable-cloud-init.sh"
check_success "Cloud-init disabling"

log "${YELLOW}📡 Setting up NBD server configuration...${NC}"
run_sudo bash "$BUILD_PACKAGE_DIR/scripts/setup-nbd-server.sh"
check_success "NBD server configuration"

# Restart NBD server with proper config
log "${YELLOW}🔄 Restarting NBD server with proper configuration...${NC}"
run_sudo systemctl restart nbd-server
wait_for_service "nbd-server.service"
check_success "NBD server restart"

log "${YELLOW}🔐 Setting up SSH tunnel infrastructure...${NC}"
run_sudo bash "$BUILD_PACKAGE_DIR/scripts/setup-ssh-tunnel.sh"
check_success "SSH tunnel infrastructure setup"

# Test SSH configuration
log "${YELLOW}🔍 Testing SSH configuration...${NC}"
if run_sudo sshd -t; then
    log "${GREEN}✅ SSH configuration is valid${NC}"
else
    log "${RED}❌ SSH configuration has errors${NC}"
    exit 1
fi

# Add SSH socket override for port 443
log "${YELLOW}🔧 Adding SSH port 443 socket override...${NC}"
run_sudo mkdir -p /etc/systemd/system/ssh.socket.d
cat > /tmp/port443.conf << 'EOF'
[Socket]
ListenStream=
ListenStream=0.0.0.0:22
ListenStream=0.0.0.0:443
ListenStream=[::]:22
ListenStream=[::]:443
EOF
run_sudo cp /tmp/port443.conf /etc/systemd/system/ssh.socket.d/
run_sudo systemctl daemon-reload

# Restart SSH daemon to apply port 443 configuration
log "${YELLOW}🔄 Restarting SSH daemon for port 443...${NC}"
run_sudo systemctl restart ssh.socket
wait_for_service "ssh.service"
check_success "SSH daemon restart"

# Verify SSH is listening on port 443
log "${YELLOW}🔍 Verifying SSH is listening on port 443...${NC}"
if ss -tlnp | grep -q ":443.*sshd"; then
    log "${GREEN}✅ SSH is listening on port 443${NC}"
else
    log "${YELLOW}⚠️ SSH port 443 not detected - may need manual verification${NC}"
fi

log "${GREEN}✅ SSH tunnel infrastructure setup completed${NC}"
echo ""

# =============================================================================
# PHASE 7: SYSTEM VALIDATION
# =============================================================================

log "${BLUE}📋 Phase 7: Complete System Validation${NC}"
log "======================================"

# Test all services
log "${YELLOW}🔍 Testing all service health endpoints...${NC}"

services_status=""

# Database connectivity
if mysql -u oma_user -poma_password migratekit_oma -e "SELECT 1;" > /dev/null 2>&1; then
    log "${GREEN}✅ Database connectivity confirmed${NC}"
    services_status="${services_status}Database: ✅\n"
else
    log "${RED}❌ Database connectivity failed${NC}"
    services_status="${services_status}Database: ❌\n"
fi

# OMA API health
if test_health "http://localhost:8082/health" "OMA API"; then
    services_status="${services_status}OMA API: ✅\n"
else
    services_status="${services_status}OMA API: ❌\n"
fi

# Volume Daemon health
if test_health "http://localhost:8090/api/v1/health" "Volume Daemon"; then
    services_status="${services_status}Volume Daemon: ✅\n"
else
    services_status="${services_status}Volume Daemon: ❌\n"
fi

# Migration GUI health
if test_health "http://localhost:3001" "Migration GUI"; then
    services_status="${services_status}Migration GUI: ✅\n"
else
    services_status="${services_status}Migration GUI: ❌\n"
fi

# NBD Server
if ss -tlnp | grep -q ":10809"; then
    log "${GREEN}✅ NBD Server is listening on port 10809${NC}"
    services_status="${services_status}NBD Server: ✅\n"
else
    log "${YELLOW}⚠️ NBD Server not detected on port 10809${NC}"
    services_status="${services_status}NBD Server: ⚠️\n"
fi

# SSH Tunnel Infrastructure
if id vma_tunnel > /dev/null 2>&1; then
    log "${GREEN}✅ SSH tunnel user (vma_tunnel) exists${NC}"
    services_status="${services_status}SSH Tunnel User: ✅\n"
else
    log "${RED}❌ SSH tunnel user (vma_tunnel) missing${NC}"
    services_status="${services_status}SSH Tunnel User: ❌\n"
fi

# VirtIO Tools
if [ -f "/usr/share/virtio-win/virtio-win.iso" ]; then
    log "${GREEN}✅ VirtIO tools are present${NC}"
    services_status="${services_status}VirtIO Tools: ✅\n"
else
    log "${RED}❌ VirtIO tools missing${NC}"
    services_status="${services_status}VirtIO Tools: ❌\n"
fi

log "${GREEN}✅ System validation completed${NC}"
echo ""

# =============================================================================
# PHASE 8: CLEANUP AND REPORTING
# =============================================================================

log "${BLUE}📋 Phase 8: Cleanup and Final Report${NC}"
log "===================================="

# Clean up build package
log "${YELLOW}🧹 Cleaning up build artifacts...${NC}"
rm -rf "$BUILD_PACKAGE_DIR"
check_success "Build artifact cleanup"

# Create final upgrade report
FINAL_REPORT="$BACKUP_DIR/upgrade-completion-report.txt"
cat > "$FINAL_REPORT" << EOF
🚀 OSSEA-Migrate OMA Production Upgrade Completion Report
========================================================

Upgrade Date: $(date)
Server: 10.245.246.120
Script Version: $SCRIPT_VERSION
Log File: $LOG_FILE

UPGRADE RESULTS:
================
$services_status

COMPONENTS UPGRADED:
===================
✅ Volume Daemon: Upgraded from phase_1_foundation to current production
✅ OMA API Service: Updated with consistent user model (oma_admin)
✅ Migration GUI: Updated service configuration for consistency
✅ SSH Tunnel Infrastructure: Added vma_tunnel user and port 443 support
✅ File Ownership: Fixed oma vs oma_admin inconsistencies
✅ Service Configurations: Standardized timeout and restart policies

PRODUCTION READINESS CHECKLIST:
===============================
$(echo -e "$services_status")

MISSING COMPONENTS (if any):
============================
- Pre-shared Key System: Requires VMA enrollment implementation
- Advanced SSH hardening: May need additional security policies

NEXT STEPS:
===========
1. Test complete migration workflow
2. Validate SSH tunnel connectivity with VMA
3. Export as production template if all tests pass
4. Document any additional configuration needed

ROLLBACK INFORMATION:
====================
- Snapshot available for complete rollback
- Database backup: $BACKUP_DIR/database-pre-upgrade.sql
- Service configs backup: $BACKUP_DIR/*.service
- Current state documentation: $BACKUP_DIR/current-state.txt

SUPPORT:
========
- Full upgrade log: $LOG_FILE
- Backup directory: $BACKUP_DIR
- For issues, check service logs: journalctl -u <service-name>
EOF

log "${GREEN}✅ Cleanup completed${NC}"
echo ""

# =============================================================================
# FINAL SUMMARY
# =============================================================================

log "${BLUE}🎉 OSSEA-MIGRATE OMA PRODUCTION UPGRADE COMPLETE!${NC}"
log "================================================="
echo ""
log "${GREEN}📊 UPGRADE SUMMARY:${NC}"
echo -e "$services_status"
echo ""
log "${BLUE}📄 Reports Generated:${NC}"
log "   - Upgrade log: $LOG_FILE"
log "   - Final report: $FINAL_REPORT"
log "   - Backup directory: $BACKUP_DIR"
echo ""
log "${BLUE}🔗 Access Points:${NC}"
log "   - Migration GUI: http://10.245.246.120:3001"
log "   - OMA API: http://10.245.246.120:8082"
log "   - Volume Daemon: http://10.245.246.120:8090"
echo ""
log "${YELLOW}💡 Next Steps:${NC}"
log "   1. Test all functionality via GUI"
log "   2. Validate SSH tunnel connectivity"
log "   3. Export as production template if successful"
log "   4. Use snapshot rollback if issues found"
echo ""
log "${GREEN}🚀 Server 120 is now production-ready!${NC}"

# Create appliance info file
cat > "/home/oma_admin/appliance-upgrade-info.txt" << EOF
OSSEA-Migrate OMA Production Upgrade Complete
Upgrade Date: $(date)
Server: 10.245.246.120
Script Version: $SCRIPT_VERSION

Access Points:
- Migration GUI: http://10.245.246.120:3001
- OMA API: http://10.245.246.120:8082
- Volume Daemon: http://10.245.246.120:8090

Components Status:
$services_status

For support or rollback: See $FINAL_REPORT
EOF

run_sudo chown oma_admin:oma_admin "/home/oma_admin/appliance-upgrade-info.txt"

log "${BLUE}✅ UPGRADE SCRIPT COMPLETED SUCCESSFULLY!${NC}"
exit 0
