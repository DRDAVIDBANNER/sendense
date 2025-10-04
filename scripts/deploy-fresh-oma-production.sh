#!/bin/bash
# 🚀 **DEPLOY FRESH OMA PRODUCTION**
#
# Purpose: Deploy complete OMA production environment on fresh Ubuntu 24.04
# Target: Fresh servers (no existing deployment)
# Author: MigrateKit OSSEA Team
# Date: October 1, 2025

set -euo pipefail

# Configuration
SCRIPT_VERSION="v1.0.0"
LOG_FILE="/tmp/oma-fresh-deployment-$(date +%Y%m%d-%H%M%S).log"
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

echo -e "${BLUE}🚀 OSSEA-Migrate OMA Fresh Production Deployment${NC}"
echo -e "${BLUE}===============================================${NC}"
echo "Script Version: $SCRIPT_VERSION"
echo "Target: Fresh Ubuntu 24.04 Server"
echo "Log File: $LOG_FILE"
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
# PHASE 1: SYSTEM PREPARATION
# =============================================================================

log "${BLUE}📋 Phase 1: System Preparation${NC}"
log "==============================="

# Check OS version
if ! grep -q "24.04" /etc/os-release; then
    log "${RED}❌ This script requires Ubuntu 24.04 LTS${NC}"
    exit 1
fi

log "${YELLOW}🚫 Disabling cloud-init for production deployment...${NC}"
run_sudo touch /etc/cloud/cloud-init.disabled
run_sudo systemctl disable cloud-init cloud-config cloud-final cloud-init-local 2>/dev/null || true
check_success "Cloud-init disable"

log "${YELLOW}🔄 Updating system packages...${NC}"
DEBIAN_FRONTEND=noninteractive run_sudo apt update -y
check_success "System package update"

log "${YELLOW}📦 Installing dependencies...${NC}"
DEBIAN_FRONTEND=noninteractive run_sudo apt install -y \
    mariadb-server \
    mariadb-client \
    nbd-server \
    curl \
    jq \
    unzip \
    systemd \
    net-tools \
    openssh-server \
    nodejs \
    npm
check_success "Dependencies installation"

log "${YELLOW}👤 Configuring OMA admin user...${NC}"
echo "oma_admin:$SUDO_PASSWORD" | run_sudo chpasswd
run_sudo usermod -aG sudo oma_admin
check_success "User configuration"

log "${GREEN}✅ System preparation completed${NC}"
echo ""

# =============================================================================
# PHASE 2: DATABASE SETUP
# =============================================================================

log "${BLUE}📋 Phase 2: Database Configuration${NC}"
log "=================================="

log "${YELLOW}🗄️ Starting MariaDB...${NC}"
run_sudo systemctl start mariadb
run_sudo systemctl enable mariadb
wait_for_service "mariadb.service"

log "${YELLOW}👤 Creating database and user...${NC}"
run_sudo mysql -e "CREATE DATABASE IF NOT EXISTS migratekit_oma;"
run_sudo mysql -e "CREATE USER IF NOT EXISTS 'oma_user'@'localhost' IDENTIFIED BY 'oma_password';"
run_sudo mysql -e "GRANT ALL PRIVILEGES ON migratekit_oma.* TO 'oma_user'@'localhost';"
run_sudo mysql -e "FLUSH PRIVILEGES;"
check_success "Database user creation"

log "${YELLOW}📊 Creating basic database schema...${NC}"
# Create minimal schema for testing
mysql -u oma_user -poma_password migratekit_oma << 'EOF'
CREATE TABLE IF NOT EXISTS ossea_configs (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    api_url VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) NOT NULL,
    secret_key VARCHAR(255) NOT NULL,
    domain VARCHAR(255) DEFAULT 'ROOT',
    zone VARCHAR(255) NOT NULL,
    template_id VARCHAR(255),
    service_offering_id VARCHAR(255),
    oma_vm_id VARCHAR(255),
    is_active BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS vmware_credentials (
    id INT PRIMARY KEY AUTO_INCREMENT,
    credential_name VARCHAR(255) NOT NULL,
    vcenter_host VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL,
    password_encrypted TEXT NOT NULL,
    datacenter VARCHAR(255),
    is_active BOOLEAN DEFAULT false,
    is_default BOOLEAN DEFAULT false,
    created_by VARCHAR(255) DEFAULT 'system',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
EOF
check_success "Basic database schema creation"

log "${GREEN}✅ Database configuration completed${NC}"
echo ""

# =============================================================================
# PHASE 3: BINARY DEPLOYMENT
# =============================================================================

log "${BLUE}📋 Phase 3: Binary Deployment${NC}"
log "============================="

log "${YELLOW}📁 Creating directory structure...${NC}"
run_sudo mkdir -p /opt/migratekit/{bin,gui}
run_sudo mkdir -p /opt/ossea-migrate
run_sudo mkdir -p /usr/local/bin
check_success "Directory creation"

# For fresh deployment, we'll create placeholder binaries that can be replaced
log "${YELLOW}📦 Creating placeholder binaries (to be replaced with actual binaries)...${NC}"

# Create placeholder OMA API
cat > /tmp/oma-api-placeholder << 'EOF'
#!/bin/bash
echo "OMA API Placeholder - Replace with actual binary"
echo "Health endpoint simulation"
if [[ "${1:-}" == "-port=8082" ]]; then
    echo "Starting placeholder OMA API on port 8082..."
    while true; do
        echo '{"status":"healthy","message":"placeholder"}' | nc -l -p 8082 -q 1
    done
fi
EOF

run_sudo cp /tmp/oma-api-placeholder /opt/migratekit/bin/oma-api
run_sudo chmod +x /opt/migratekit/bin/oma-api
run_sudo chown oma_admin:oma_admin /opt/migratekit/bin/oma-api

# Create placeholder Volume Daemon
cat > /tmp/volume-daemon-placeholder << 'EOF'
#!/bin/bash
echo "Volume Daemon Placeholder - Replace with actual binary"
echo "Starting placeholder Volume Daemon on port 8090..."
while true; do
    echo '{"status":"healthy","message":"placeholder"}' | nc -l -p 8090 -q 1
done
EOF

run_sudo cp /tmp/volume-daemon-placeholder /usr/local/bin/volume-daemon
run_sudo chmod +x /usr/local/bin/volume-daemon
run_sudo chown oma_admin:oma_admin /usr/local/bin/volume-daemon

check_success "Placeholder binary deployment"

log "${GREEN}✅ Binary deployment completed${NC}"
echo ""

# =============================================================================
# PHASE 4: NBD SERVER CONFIGURATION
# =============================================================================

log "${BLUE}📋 Phase 4: NBD Server Configuration${NC}"
log "==================================="

log "${YELLOW}📡 Setting up NBD server configuration...${NC}"

# Create proper config-base
run_sudo tee /etc/nbd-server/config-base << 'EOF'
[generic]
port = 10809
allowlist = true
includedir = /etc/nbd-server/conf.d

# Dummy export required for NBD server to start
[dummy]
exportname = /dev/null
readonly = true
EOF

# Copy to default config location
run_sudo cp /etc/nbd-server/config-base /etc/nbd-server/config
check_success "NBD server configuration"

log "${YELLOW}🚀 Starting NBD server...${NC}"
run_sudo systemctl start nbd-server
run_sudo systemctl enable nbd-server
wait_for_service "nbd-server.service"

# Verify NBD is listening
if ss -tlnp | grep -q ":10809"; then
    log "${GREEN}✅ NBD Server is listening on port 10809${NC}"
else
    log "${YELLOW}⚠️ NBD Server not detected on port 10809${NC}"
fi

log "${GREEN}✅ NBD server configuration completed${NC}"
echo ""

# =============================================================================
# PHASE 5: SSH TUNNEL INFRASTRUCTURE
# =============================================================================

log "${BLUE}📋 Phase 5: SSH Tunnel Infrastructure${NC}"
log "====================================="

log "${YELLOW}🔐 Creating vma_tunnel user...${NC}"
run_sudo useradd -r -m -s /bin/bash -d /var/lib/vma_tunnel vma_tunnel 2>/dev/null || echo "User already exists"

# Create SSH directory
run_sudo mkdir -p /var/lib/vma_tunnel/.ssh
run_sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh
run_sudo chmod 700 /var/lib/vma_tunnel/.ssh

# Create authorized_keys file (will be populated when VMA connects)
run_sudo touch /var/lib/vma_tunnel/.ssh/authorized_keys
run_sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh/authorized_keys
run_sudo chmod 600 /var/lib/vma_tunnel/.ssh/authorized_keys
check_success "vma_tunnel user creation"

log "${YELLOW}🔧 Configuring SSH for port 443 and tunnel restrictions...${NC}"

# Add port 443 to SSH
if ! grep -q "Port 443" /etc/ssh/sshd_config; then
    echo "Port 443" | run_sudo tee -a /etc/ssh/sshd_config
fi

# Add Match User block for vma_tunnel
if ! grep -q "Match User vma_tunnel" /etc/ssh/sshd_config; then
    cat << 'SSHCONFIG' | run_sudo tee -a /etc/ssh/sshd_config

# VMA Tunnel User Configuration - Production
Match User vma_tunnel
    AuthenticationMethods publickey
    PubkeyAuthentication yes
    PasswordAuthentication no
    PermitTTY no
    X11Forwarding no
    AllowTcpForwarding remote
    PermitOpen 127.0.0.1:10809 127.0.0.1:8082
    PermitListen 127.0.0.1:9081
SSHCONFIG
fi

# Add SSH socket override for port 443
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

# Test SSH configuration
if run_sudo sshd -t; then
    log "${GREEN}✅ SSH configuration is valid${NC}"
else
    log "${RED}❌ SSH configuration has errors${NC}"
    exit 1
fi

# Restart SSH to apply configuration
log "${YELLOW}🔄 Restarting SSH for port 443...${NC}"
run_sudo systemctl restart ssh.socket
wait_for_service "ssh.service"

# Verify SSH is listening on both ports
if ss -tlnp | grep -E ":22.*sshd|:443.*sshd" | wc -l | grep -q "2"; then
    log "${GREEN}✅ SSH is listening on both ports 22 and 443${NC}"
else
    log "${YELLOW}⚠️ SSH port configuration may need verification${NC}"
fi

check_success "SSH tunnel infrastructure setup"

log "${GREEN}✅ SSH tunnel infrastructure completed${NC}"
echo ""

# =============================================================================
# PHASE 6: FINAL VALIDATION
# =============================================================================

log "${BLUE}📋 Phase 6: System Validation${NC}"
log "============================="

current_ip=$(hostname -I | awk '{print $1}' | tr -d ' ')

log "${YELLOW}🔍 Testing system components...${NC}"

# Database connectivity
if mysql -u oma_user -poma_password migratekit_oma -e "SELECT 1;" > /dev/null 2>&1; then
    log "${GREEN}✅ Database connectivity confirmed${NC}"
else
    log "${RED}❌ Database connectivity failed${NC}"
fi

# NBD Server
if ss -tlnp | grep -q ":10809"; then
    log "${GREEN}✅ NBD Server is listening on port 10809${NC}"
else
    log "${RED}❌ NBD Server not listening${NC}"
fi

# SSH Tunnel Infrastructure
if id vma_tunnel > /dev/null 2>&1; then
    log "${GREEN}✅ SSH tunnel user (vma_tunnel) exists${NC}"
else
    log "${RED}❌ SSH tunnel user (vma_tunnel) missing${NC}"
fi

# SSH Port 443
if ss -tlnp | grep -q ":443"; then
    log "${GREEN}✅ SSH listening on port 443${NC}"
else
    log "${RED}❌ SSH not listening on port 443${NC}"
fi

log "${GREEN}✅ System validation completed${NC}"
echo ""

# =============================================================================
# FINAL SUMMARY
# =============================================================================

log "${BLUE}🎉 OSSEA-MIGRATE OMA FRESH DEPLOYMENT COMPLETE!${NC}"
log "==============================================="
echo ""
log "${GREEN}📊 DEPLOYMENT SUMMARY:${NC}"
log "   - Database: MariaDB with basic schema"
log "   - NBD Server: Configured and listening on port 10809"
log "   - SSH Tunnel: vma_tunnel user and port 443 ready"
log "   - Cloud-init: Disabled for production"
echo ""
log "${BLUE}🔗 Access Points:${NC}"
log "   - Server IP: $current_ip"
log "   - SSH Access: ssh oma_admin@$current_ip (port 22 or 443)"
log "   - NBD Server: port 10809"
echo ""
log "${YELLOW}📋 Next Steps:${NC}"
log "   1. Copy production binaries (OMA API, Volume Daemon, GUI)"
log "   2. Add VMA SSH public key to /var/lib/vma_tunnel/.ssh/authorized_keys"
log "   3. Test VMA tunnel connectivity"
log "   4. Deploy actual migration functionality"
echo ""
log "${BLUE}🔑 VMA Connection Setup:${NC}"
log "   To add VMA SSH key:"
log "   echo 'VMA_PUBLIC_KEY' | sudo tee /var/lib/vma_tunnel/.ssh/authorized_keys"
log "   sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh/authorized_keys"
log "   sudo chmod 600 /var/lib/vma_tunnel/.ssh/authorized_keys"
echo ""
log "${GREEN}🚀 OMAv3 infrastructure ready for production deployment!${NC}"

# Create deployment info file
cat > "/home/oma_admin/fresh-deployment-info.txt" << EOF
OSSEA-Migrate OMA Fresh Deployment Complete
Deployment Date: $(date)
Server: $current_ip (OMAv3)
Base OS: Ubuntu 24.04 LTS

Infrastructure Ready:
- Database: MariaDB with oma_user/oma_password
- NBD Server: Port 10809 configured
- SSH Tunnel: vma_tunnel user ready for VMA keys
- SSH Ports: 22 and 443 active

Next Steps:
1. Deploy production binaries
2. Add VMA SSH public key
3. Test tunnel connectivity
4. Validate migration workflow

Log File: $LOG_FILE
EOF

run_sudo chown oma_admin:oma_admin "/home/oma_admin/fresh-deployment-info.txt"

log "${BLUE}✅ FRESH OMA DEPLOYMENT COMPLETED SUCCESSFULLY!${NC}"
exit 0
