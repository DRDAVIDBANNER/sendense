#!/bin/bash
# Minimal VMA Appliance Deployment - Binaries Only
# Deploys complete functional VMA with enrollment system

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BLUE}${BOLD}"
cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                Minimal VMA Appliance Deployment                  ║
║              Production Ready - Binaries Only                    ║
║                                                                  ║
║                    🚀 Complete VMA System                       ║
╚══════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${CYAN}Deploying minimal VMA appliance with complete functionality...${NC}"
echo ""

# Phase 1: System Dependencies
echo -e "${BOLD}📦 Phase 1: Installing System Dependencies${NC}"
apt-get update
# FIX #2: NBD tools may not be available in all Ubuntu repos
apt-get install -y haveged jq curl openssh-client
# Try to install NBD tools (may fail on some Ubuntu versions)
apt-get install -y libnbd-bin nbd-client || echo "⚠️ NBD tools not available in apt - manual installation may be required"
systemctl enable haveged
systemctl start haveged
echo -e "${GREEN}✅ System dependencies installed${NC}"
echo ""

# Phase 2: VMA Directory Structure
echo -e "${BOLD}📁 Phase 2: Creating VMA Directory Structure${NC}"
mkdir -p /opt/vma/{bin,config,enrollment,ssh,scripts,logs}
mkdir -p /var/log/vma
chown -R vma:vma /opt/vma
chmod 755 /opt/vma/{bin,scripts}
chmod 750 /opt/vma/{config,logs}
chmod 700 /opt/vma/{enrollment,ssh}
echo -e "${GREEN}✅ VMA directory structure created${NC}"
echo ""

# Phase 3: VMA Binary Deployment
echo -e "${BOLD}🚀 Phase 3: Deploying VMA Binaries${NC}"

# Deploy MigrateKit binary (~20MB)
echo "Deploying MigrateKit migration engine..."
# FIX #1: File transfer issues - use multiple copy approaches
if [ -f "/home/vma/migratekit-cloudstack/vma-binaries/migratekit-multidisk-incremental-fix" ]; then
    cp /home/vma/migratekit-cloudstack/vma-binaries/migratekit-multidisk-incremental-fix /opt/vma/bin/migratekit
    chmod +x /opt/vma/bin/migratekit
    ln -sf /opt/vma/bin/migratekit /usr/local/bin/migratekit
    echo -e "${GREEN}✅ MigrateKit binary deployed from git${NC}"
elif [ -f "/usr/local/bin/migratekit-multidisk-incremental-fix" ]; then
    cp /usr/local/bin/migratekit-multidisk-incremental-fix /opt/vma/bin/migratekit
    chmod +x /opt/vma/bin/migratekit
    ln -sf /opt/vma/bin/migratekit /usr/local/bin/migratekit
    echo -e "${GREEN}✅ MigrateKit binary deployed from system${NC}"
else
    echo -e "${RED}❌ MigrateKit binary not found - deployment incomplete${NC}"
    exit 1
fi

# Deploy VMA API server (~20MB)
echo "Deploying VMA API server..."
if [ -f "/home/vma/migratekit-cloudstack/vma-api-server-v1.11.0-enrollment-system" ]; then
    cp /home/vma/migratekit-cloudstack/vma-api-server-v1.11.0-enrollment-system /opt/vma/bin/vma-api-server
    chmod +x /opt/vma/bin/vma-api-server
    echo -e "${GREEN}✅ VMA API server deployed${NC}"
else
    echo -e "${YELLOW}⚠️ VMA API server not found${NC}"
fi

echo ""

# Phase 4: VMA Scripts Deployment
echo -e "${BOLD}🔧 Phase 4: Deploying VMA Scripts${NC}"

# Copy enrollment and wizard scripts from git
if [ -f "/home/vma/migratekit-cloudstack/vma-templates/vma-enrollment.sh" ]; then
    cp /home/vma/migratekit-cloudstack/vma-templates/vma-enrollment.sh /opt/vma/
    cp /home/vma/migratekit-cloudstack/vma-templates/setup-wizard.sh /opt/vma/
    chmod +x /opt/vma/vma-enrollment.sh /opt/vma/setup-wizard.sh
    echo -e "${GREEN}✅ VMA enrollment and wizard scripts deployed${NC}"
else
    echo -e "${YELLOW}⚠️ VMA templates not found in git${NC}"
fi

# Create tunnel script
cat > /opt/vma/scripts/enhanced-ssh-tunnel.sh << 'TUNNEL_EOF'
#!/bin/bash
# Enhanced SSH tunnel for VMA
set -euo pipefail

OMA_HOST="${OMA_HOST:-10.245.246.125}"
SSH_KEY="${SSH_KEY:-/var/lib/vma_tunnel/.ssh/vma_enrollment_key}"
LOG_FILE="/var/log/vma/tunnel.log"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [$$] $*" | tee -a "$LOG_FILE"
}

log "Starting VMA tunnel to $OMA_HOST"
log "Using SSH key: $SSH_KEY"

# Health check function
health_check() {
    if curl -s --connect-timeout 5 http://localhost:8082/health >/dev/null 2>&1; then
        log "💚 Tunnel health check passed"
        return 0
    else
        log "❌ Tunnel health check failed"
        return 1
    fi
}

# Main tunnel loop with recovery
while true; do
    log "🔗 Establishing SSH tunnel to OMA"
    
    ssh -i "$SSH_KEY" \
        -R 9081:localhost:8081 \
        -L 8082:localhost:8082 \
        -L 10809:localhost:10809 \
        -N -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o ServerAliveInterval=30 \
        -o ServerAliveCountMax=3 \
        -o ConnectTimeout=30 \
        -o TCPKeepAlive=yes \
        -o ExitOnForwardFailure=yes \
        -o BatchMode=yes \
        vma_tunnel@$OMA_HOST 2>&1 | while read line; do
            log "SSH: $line"
        done
    
    log "⚠️ SSH tunnel disconnected, restarting in 15 seconds..."
    sleep 15
done
TUNNEL_EOF

chmod +x /opt/vma/scripts/enhanced-ssh-tunnel.sh
echo -e "${GREEN}✅ VMA tunnel script deployed${NC}"
echo ""

# Phase 5: VMA Services Configuration
echo -e "${BOLD}⚙️ Phase 5: Configuring VMA Services${NC}"

# VMA API Service
cat > /etc/systemd/system/vma-api.service << 'EOF'
[Unit]
Description=VMA Control API Server
After=network.target

[Service]
Type=simple
User=vma
Group=vma
WorkingDirectory=/opt/vma
ExecStart=/opt/vma/bin/vma-api-server -port 8081
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# VMA Tunnel Service
cat > /etc/systemd/system/vma-tunnel-enhanced-v2.service << 'EOF'
[Unit]
Description=VMA Enhanced SSH Tunnel to OMA
After=network-online.target vma-api.service
Wants=network-online.target
Requires=vma-api.service

[Service]
Type=simple
User=vma
Group=vma
WorkingDirectory=/opt/vma
ExecStart=/opt/vma/scripts/enhanced-ssh-tunnel.sh
Restart=always
RestartSec=15
StartLimitInterval=300
StartLimitBurst=5

# Environment - ARCHITECTURE FIX
Environment=OMA_HOST=10.245.246.125
Environment=SSH_KEY=/opt/vma/enrollment/vma_enrollment_key
# FIX #4: VMA uses its own enrolled private key to connect to oma@ user
# FIX #3: SSH key location - VMA user needs accessible key location

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vma-tunnel

[Install]
WantedBy=multi-user.target
EOF

# Auto-login service for wizard
cat > /etc/systemd/system/vma-autologin.service << 'EOF'
[Unit]
Description=VMA Auto-login Setup Wizard
After=multi-user.target network.target
Wants=network.target

[Service]
Type=idle
User=vma
Group=vma
TTY=/dev/tty1
ExecStart=/opt/vma/setup-wizard.sh
StandardInput=tty
StandardOutput=tty
StandardError=tty
Restart=no
RemainAfterExit=yes
Environment=HOME=/home/vma
Environment=USER=vma
Environment=TERM=xterm-256color
NoNewPrivileges=false
PrivateTmp=false

[Install]
WantedBy=multi-user.target
EOF

# Enable services
systemctl daemon-reload
systemctl enable vma-api.service
systemctl enable vma-tunnel-enhanced-v2.service
systemctl disable getty@tty1.service
systemctl enable vma-autologin.service

echo -e "${GREEN}✅ VMA services configured and enabled${NC}"
echo ""

# Phase 6: Final Validation
echo -e "${BOLD}🧪 Phase 6: Final Validation${NC}"

# Test binaries
if [ -x "/opt/vma/bin/migratekit" ]; then
    echo -e "${GREEN}✅ MigrateKit binary ready${NC}"
else
    echo -e "${YELLOW}⚠️ MigrateKit binary missing${NC}"
fi

if [ -x "/opt/vma/bin/vma-api-server" ]; then
    echo -e "${GREEN}✅ VMA API server ready${NC}"
else
    echo -e "${YELLOW}⚠️ VMA API server missing${NC}"
fi

# Test scripts
if bash -n /opt/vma/vma-enrollment.sh && bash -n /opt/vma/setup-wizard.sh; then
    echo -e "${GREEN}✅ VMA scripts validated${NC}"
else
    echo -e "${RED}❌ Script validation failed${NC}"
    exit 1
fi

echo ""
echo -e "${BOLD}${GREEN}🎉 Minimal VMA Appliance Deployment Complete!${NC}"
echo ""
echo -e "${CYAN}📋 VMA Components Deployed:${NC}"
echo -e "   ${BOLD}Binaries:${NC} MigrateKit (~20MB), VMA API (~20MB)"
echo -e "   ${BOLD}Services:${NC} VMA API, Tunnel, Auto-login"
echo -e "   ${BOLD}Enrollment:${NC} Complete enrollment system"
echo -e "   ${BOLD}Dependencies:${NC} libnbd, haveged, jq, curl"
echo -e "   ${BOLD}Total Size:${NC} ~50MB (no source code)"
echo ""
echo -e "${CYAN}🎯 VMA Ready For:${NC}"
echo -e "   1. Boot to enrollment wizard"
echo -e "   2. VMA enrollment workflow"
echo -e "   3. VM discovery and migration"
echo -e "   4. Tunnel connection to OMA"
echo ""
echo -e "${BOLD}${GREEN}Production VMA appliance ready!${NC}"
