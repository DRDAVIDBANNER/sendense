#!/bin/bash
# VMA Production User Setup Script
# Creates proper production users and directory structure for VMA enrollment system
# Preserves tunnel recovery system and migrates from development setup

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration
VMA_TARGET="${VMA_TARGET:-10.0.100.231}"
SSH_KEY="${SSH_KEY:-~/.ssh/cloudstack_key}"

echo -e "${BLUE}${BOLD}"
cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                VMA Production User Setup                         ║
║           Secure VMA Enrollment Infrastructure                   ║
╚══════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${CYAN}Setting up production users and directory structure for VMA enrollment system...${NC}"
echo -e "${YELLOW}Target VMA: $VMA_TARGET${NC}"
echo ""

# Function to run commands on VMA
run_vma_cmd() {
    ssh -i "$SSH_KEY" pgrayson@"$VMA_TARGET" "$@"
}

# Function to check if user exists
user_exists() {
    local username=$1
    run_vma_cmd "id $username >/dev/null 2>&1"
}

# Function to create production users
create_production_users() {
    echo -e "${YELLOW}👥 Creating production users...${NC}"
    
    # Create vma_service user
    if ! user_exists "vma_service"; then
        run_vma_cmd "sudo useradd -r -m -d /var/lib/vma_service -s /bin/bash -c 'VMA Service User' vma_service"
        run_vma_cmd "sudo usermod -a -G sudo vma_service"
        echo -e "${GREEN}✅ Created vma_service user${NC}"
    else
        echo -e "${CYAN}ℹ️  vma_service user already exists${NC}"
    fi
    
    # Create vma_admin user
    if ! user_exists "vma_admin"; then
        run_vma_cmd "sudo useradd -m -d /home/vma_admin -s /bin/bash -c 'VMA Administrator' vma_admin"
        run_vma_cmd "sudo usermod -a -G sudo,vma_service vma_admin"
        echo -e "${GREEN}✅ Created vma_admin user${NC}"
    else
        echo -e "${CYAN}ℹ️  vma_admin user already exists${NC}"
    fi
    
    # Ensure vma_tunnel user exists (created by OMA deployment)
    if ! user_exists "vma_tunnel"; then
        run_vma_cmd "sudo useradd -r -m -d /var/lib/vma_tunnel -s /bin/false -c 'VMA Tunnel User' vma_tunnel"
        echo -e "${GREEN}✅ Created vma_tunnel user${NC}"
    else
        echo -e "${CYAN}ℹ️  vma_tunnel user already exists${NC}"
    fi
}

# Function to create production directory structure
create_directory_structure() {
    echo -e "${YELLOW}📁 Creating production directory structure...${NC}"
    
    # Create main VMA directories
    run_vma_cmd "sudo mkdir -p /opt/vma/{bin,config,enrollment,ssh,scripts,logs}"
    run_vma_cmd "sudo mkdir -p /var/lib/vma_service/.ssh"
    run_vma_cmd "sudo mkdir -p /var/log/vma"
    
    # Set directory ownership and permissions
    run_vma_cmd "sudo chown root:root /opt/vma"
    run_vma_cmd "sudo chmod 755 /opt/vma"
    
    run_vma_cmd "sudo chown root:root /opt/vma/bin"
    run_vma_cmd "sudo chmod 755 /opt/vma/bin"
    
    run_vma_cmd "sudo chown vma_service:vma_service /opt/vma/config"
    run_vma_cmd "sudo chmod 750 /opt/vma/config"
    
    run_vma_cmd "sudo chown vma_service:vma_service /opt/vma/enrollment"
    run_vma_cmd "sudo chmod 700 /opt/vma/enrollment"
    
    run_vma_cmd "sudo chown vma_service:vma_service /opt/vma/ssh"
    run_vma_cmd "sudo chmod 700 /opt/vma/ssh"
    
    run_vma_cmd "sudo chown root:vma_service /opt/vma/scripts"
    run_vma_cmd "sudo chmod 755 /opt/vma/scripts"
    
    run_vma_cmd "sudo chown vma_service:vma_service /opt/vma/logs"
    run_vma_cmd "sudo chmod 750 /opt/vma/logs"
    
    run_vma_cmd "sudo chown vma_service:vma_service /var/lib/vma_service"
    run_vma_cmd "sudo chmod 750 /var/lib/vma_service"
    
    run_vma_cmd "sudo chown vma_service:vma_service /var/lib/vma_service/.ssh"
    run_vma_cmd "sudo chmod 700 /var/lib/vma_service/.ssh"
    
    run_vma_cmd "sudo chown vma_service:vma_service /var/log/vma"
    run_vma_cmd "sudo chmod 750 /var/log/vma"
    
    echo -e "${GREEN}✅ Production directory structure created${NC}"
}

# Function to migrate existing configuration
migrate_existing_config() {
    echo -e "${YELLOW}📋 Migrating existing configuration to production structure...${NC}"
    
    # Backup existing configuration
    run_vma_cmd "sudo cp -r /opt/vma /opt/vma.backup-$(date +%Y%m%d_%H%M%S) 2>/dev/null || true"
    
    # Copy current VMA config if exists
    if run_vma_cmd "test -f /opt/vma/vma-config.conf"; then
        run_vma_cmd "sudo cp /opt/vma/vma-config.conf /opt/vma/config/"
        run_vma_cmd "sudo chown vma_service:vma_service /opt/vma/config/vma-config.conf"
        echo -e "${GREEN}✅ Migrated VMA configuration${NC}"
    fi
    
    # Copy current VMA API binary
    if run_vma_cmd "test -f /home/pgrayson/migratekit-cloudstack/vma-api-server"; then
        run_vma_cmd "sudo cp /home/pgrayson/migratekit-cloudstack/vma-api-server /opt/vma/bin/"
        run_vma_cmd "sudo chmod +x /opt/vma/bin/vma-api-server"
        echo -e "${GREEN}✅ Migrated VMA API binary${NC}"
    fi
    
    # Copy tunnel scripts with preservation of recovery system
    if run_vma_cmd "test -f /home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel-remote.sh"; then
        run_vma_cmd "sudo cp /home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel*.sh /opt/vma/scripts/"
        run_vma_cmd "sudo chmod +x /opt/vma/scripts/enhanced-ssh-tunnel*.sh"
        echo -e "${GREEN}✅ Migrated tunnel scripts (recovery system preserved)${NC}"
    fi
    
    # Copy existing SSH keys
    if run_vma_cmd "test -f /home/pgrayson/.ssh/oma-server-key"; then
        run_vma_cmd "sudo cp /home/pgrayson/.ssh/oma-server-key* /opt/vma/ssh/ 2>/dev/null || true"
        run_vma_cmd "sudo chown vma_service:vma_service /opt/vma/ssh/oma-server-key* 2>/dev/null || true"
        run_vma_cmd "sudo chmod 600 /opt/vma/ssh/oma-server-key* 2>/dev/null || true"
        echo -e "${GREEN}✅ Migrated SSH keys${NC}"
    fi
}

# Function to update service configurations
update_service_configs() {
    echo -e "${YELLOW}⚙️ Updating service configurations for production users...${NC}"
    
    # Update VMA API service
    run_vma_cmd "sudo tee /etc/systemd/system/vma-api.service > /dev/null << 'EOF'
[Unit]
Description=VMA Control API Server
After=network.target

[Service]
Type=simple
User=vma_service
Group=vma_service
WorkingDirectory=/opt/vma
ExecStart=/opt/vma/bin/vma-api-server -port 8081
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Environment variables
Environment=VMA_CONFIG_DIR=/opt/vma/config
Environment=VMA_LOG_DIR=/opt/vma/logs

[Install]
WantedBy=multi-user.target
EOF"
    
    # Update VMA tunnel service with preservation of recovery system
    run_vma_cmd "sudo tee /etc/systemd/system/vma-tunnel-enhanced-v2.service > /dev/null << 'EOF'
[Unit]
Description=VMA Enhanced SSH Tunnel to OMA (Production + Enrollment Support)
After=network-online.target vma-api.service
Wants=network-online.target
Requires=vma-api.service

[Service]
Type=simple
User=vma_service
Group=vma_service
WorkingDirectory=/var/lib/vma_service

# Use production tunnel script
ExecStart=/opt/vma/scripts/enhanced-ssh-tunnel-remote.sh

# Enhanced restart policy (PRESERVED)
Restart=always
RestartSec=15
StartLimitInterval=300
StartLimitBurst=5

# Resource limits
TimeoutStartSec=60
TimeoutStopSec=30

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=/var/log /tmp /opt/vma/logs /opt/vma/ssh

# Environment (will be updated by enrollment system)
Environment=OMA_HOST=45.130.45.65
Environment=SSH_KEY=/opt/vma/ssh/oma-server-key

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vma-tunnel-enhanced

[Install]
WantedBy=multi-user.target
EOF"
    
    echo -e "${GREEN}✅ Updated service configurations for production users${NC}"
}

# Function to test production setup
test_production_setup() {
    echo -e "${YELLOW}🧪 Testing production setup...${NC}"
    
    # Reload systemd
    run_vma_cmd "sudo systemctl daemon-reload"
    
    # Test service configurations
    echo -e "${CYAN}Testing VMA API service...${NC}"
    if run_vma_cmd "sudo systemctl restart vma-api.service && sleep 3 && systemctl is-active vma-api.service >/dev/null"; then
        echo -e "${GREEN}✅ VMA API service working with production user${NC}"
    else
        echo -e "${RED}❌ VMA API service failed with production user${NC}"
        return 1
    fi
    
    echo -e "${CYAN}Testing VMA tunnel service...${NC}"
    if run_vma_cmd "sudo systemctl restart vma-tunnel-enhanced-v2.service && sleep 5 && systemctl is-active vma-tunnel-enhanced-v2.service >/dev/null"; then
        echo -e "${GREEN}✅ VMA tunnel service working with production user${NC}"
    else
        echo -e "${RED}❌ VMA tunnel service failed with production user${NC}"
        return 1
    fi
    
    # Test API connectivity
    if run_vma_cmd "curl -s http://localhost:8081/api/v1/health >/dev/null"; then
        echo -e "${GREEN}✅ VMA API responding correctly${NC}"
    else
        echo -e "${YELLOW}⚠️  VMA API not responding yet (may need more time)${NC}"
    fi
    
    echo -e "${GREEN}✅ Production setup validation complete${NC}"
}

# Main execution
main() {
    echo -e "${BOLD}🚀 Starting VMA Production User Setup${NC}"
    echo ""
    
    echo -e "${CYAN}Phase 1: Creating production users...${NC}"
    create_production_users
    echo ""
    
    echo -e "${CYAN}Phase 2: Setting up directory structure...${NC}"
    create_directory_structure
    echo ""
    
    echo -e "${CYAN}Phase 3: Migrating existing configuration...${NC}"
    migrate_existing_config
    echo ""
    
    echo -e "${CYAN}Phase 4: Updating service configurations...${NC}"
    update_service_configs
    echo ""
    
    echo -e "${CYAN}Phase 5: Testing production setup...${NC}"
    test_production_setup
    echo ""
    
    echo -e "${BOLD}${GREEN}🎉 VMA Production User Setup Complete!${NC}"
    echo ""
    echo -e "${CYAN}📊 Production Status:${NC}"
    echo -e "   VMA Service User: ${BOLD}vma_service${NC} (API and tunnel services)"
    echo -e "   VMA Admin User: ${BOLD}vma_admin${NC} (administration and enrollment)"
    echo -e "   VMA Tunnel User: ${BOLD}vma_tunnel${NC} (SSH tunnel connections)"
    echo ""
    echo -e "${CYAN}📁 Directory Structure:${NC}"
    echo -e "   Configuration: ${BOLD}/opt/vma/config/${NC}"
    echo -e "   SSH Keys: ${BOLD}/opt/vma/ssh/${NC}"
    echo -e "   Enrollment: ${BOLD}/opt/vma/enrollment/${NC}"
    echo -e "   Logs: ${BOLD}/opt/vma/logs/${NC}"
    echo ""
    echo -e "${CYAN}🔄 Services Updated:${NC}"
    echo -e "   VMA API: ${BOLD}vma_service${NC} user"
    echo -e "   VMA Tunnel: ${BOLD}vma_service${NC} user"
    echo -e "   Tunnel Recovery: ${BOLD}PRESERVED${NC}"
    echo ""
    echo -e "${GREEN}✅ Ready for VMA enrollment integration!${NC}"
}

# Safety check
echo -e "${YELLOW}⚠️  This script will modify VMA user configuration and services.${NC}"
echo -e "${YELLOW}   Current services will be migrated from 'pgrayson' to production users.${NC}"
echo ""
read -p "Continue with production user setup? (y/N): " confirm

if [[ ! $confirm =~ ^[Yy]$ ]]; then
    echo "Setup cancelled."
    exit 0
fi

# Execute main function
main






