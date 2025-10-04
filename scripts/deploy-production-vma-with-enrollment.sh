#!/bin/bash
# Production VMA Deployment Script - Clean Version
# Deploys complete VMA with enhanced tunnel (no enrollment complexity)

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
║                Production VMA Deployment Script                  ║
║              Enhanced Tunnel Architecture                        ║
║                                                                  ║
║                    🚀 Production Ready                           ║
╚══════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${CYAN}Deploying production VMA with enhanced tunnel system...${NC}"
echo ""

# Phase 1: System Dependencies
echo -e "${BOLD}📦 Phase 1: Installing System Dependencies${NC}"
echo "Installing system dependencies..."
apt-get update
apt-get install -y haveged jq curl openssh-client golang-go nbdkit libnbd-dev nbd-client
systemctl enable haveged
systemctl start haveged
echo -e "${GREEN}✅ System dependencies installed${NC}"
echo ""

# Phase 2: NBD Stack
echo -e "${BOLD}📦 Phase 2: Deploying NBD Stack${NC}"
if [ -f "/tmp/nbdkit-vddk-stack.tar.gz" ]; then
    echo "Extracting NBD stack..."
    tar xzf /tmp/nbdkit-vddk-stack.tar.gz -C /
    
    # VDDK library setup
    mkdir -p /usr/lib/x86_64-linux-gnu/vmware-vix-disklib/lib64
    cd /usr/lib/x86_64-linux-gnu/vmware-vix-disklib/lib64
    ln -sf /usr/lib/vmware-vix-disklib/lib64/libvixDiskLib.so libvixDiskLib.so
    ln -sf /usr/lib/vmware-vix-disklib/lib64/libvixDiskLib.so.8 libvixDiskLib.so.8
    ln -sf /usr/lib/vmware-vix-disklib/lib64/libvixDiskLib.so.8.0.3 libvixDiskLib.so.8.0.3
    ln -sf /usr/lib/vmware-vix-disklib/lib64/libvixDiskLib.so.8.0.3 libvixDiskLib.so.9
    cd - >/dev/null
    
    echo '/usr/lib/vmware-vix-disklib/lib64' > /etc/ld.so.conf.d/vmware-vix-disklib.conf
    ldconfig
    
    echo -e "${GREEN}✅ NBD stack deployed${NC}"
else
    echo -e "${RED}❌ NBD stack not found at /tmp/nbdkit-vddk-stack.tar.gz${NC}"
    exit 1
fi
echo ""

# Phase 3: VMA Directory Structure
echo -e "${BOLD}📁 Phase 3: Creating VMA Directory Structure${NC}"
mkdir -p /opt/vma/{bin,config,enrollment,ssh,scripts,logs}
mkdir -p /var/log/vma
mkdir -p /home/vma/.ssh
chown -R vma:vma /opt/vma /home/vma
chmod 755 /opt/vma/{bin,scripts}
chmod 750 /opt/vma/{config,logs}
chmod 700 /opt/vma/{enrollment,ssh} /home/vma/.ssh
echo -e "${GREEN}✅ VMA directory structure created${NC}"
echo ""

# Phase 4: Deploy migratekit Binary
echo -e "${BOLD}🔧 Phase 4: Deploying migratekit Binary${NC}"
if [ -f "/tmp/migratekit-v2.21.0-hierarchical-sparse-optimization" ]; then
    cp /tmp/migratekit-v2.21.0-hierarchical-sparse-optimization /opt/vma/bin/migratekit
    chmod +x /opt/vma/bin/migratekit
    chown vma:vma /opt/vma/bin/migratekit
    ln -sf /opt/vma/bin/migratekit /usr/local/bin/migratekit
    
    # Create compatibility paths for VMA API server
    mkdir -p /home/pgrayson/migratekit-cloudstack
    ln -sf /opt/vma/bin/migratekit /home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel
    
    echo -e "${GREEN}✅ migratekit binary deployed${NC}"
    echo "   Compatibility paths created for VMA API server"
else
    echo -e "${RED}❌ migratekit binary not found at /tmp/migratekit-v2.21.0-hierarchical-sparse-optimization${NC}"
    exit 1
fi
echo ""

# Phase 5: Deploy SSH Keys
echo -e "${BOLD}🔑 Phase 5: Deploying SSH Keys${NC}"
if [ -f "/tmp/cloudstack_key" ] && [ -f "/tmp/cloudstack_key.pub" ]; then
    cp /tmp/cloudstack_key /home/vma/.ssh/cloudstack_key
    cp /tmp/cloudstack_key.pub /home/vma/.ssh/cloudstack_key.pub
    chown vma:vma /home/vma/.ssh/cloudstack_key*
    chmod 600 /home/vma/.ssh/cloudstack_key
    chmod 644 /home/vma/.ssh/cloudstack_key.pub
    echo -e "${GREEN}✅ SSH keys deployed${NC}"
else
    echo -e "${RED}❌ SSH keys not found in /tmp/${NC}"
    exit 1
fi
echo ""

# Phase 6: Deploy VMA API Server
echo -e "${BOLD}🚀 Phase 6: Deploying VMA API Server${NC}"
if [ -f "/tmp/vma-api-server" ]; then
    cp /tmp/vma-api-server /opt/vma/bin/vma-api-server
    chmod +x /opt/vma/bin/vma-api-server
    chown vma:vma /opt/vma/bin/vma-api-server
    echo -e "${GREEN}✅ VMA API server deployed${NC}"
else
    echo -e "${RED}❌ VMA API server not found in /tmp/${NC}"
    exit 1
fi
echo ""

# Phase 7: Deploy Setup Wizard
echo -e "${BOLD}🧙 Phase 7: Deploying Setup Wizard${NC}"
if [ -f "/tmp/vma-setup-wizard.sh" ]; then
    cp /tmp/vma-setup-wizard.sh /opt/vma/setup-wizard.sh
    chmod +x /opt/vma/setup-wizard.sh
    chown vma:vma /opt/vma/setup-wizard.sh
    echo -e "${GREEN}✅ Setup wizard deployed${NC}"
else
    echo -e "${RED}❌ Setup wizard not found in /tmp/${NC}"
    exit 1
fi
echo ""

# Phase 7: Configure Passwordless Sudo
echo -e "${BOLD}🔧 Phase 7: Configuring VMA User${NC}"
usermod -a -G sudo vma
echo 'vma ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/vma
chmod 440 /etc/sudoers.d/vma
echo -e "${GREEN}✅ Passwordless sudo configured${NC}"
echo ""

# Phase 8: Create Enhanced Tunnel Script
echo -e "${BOLD}🔧 Phase 8: Creating Enhanced Tunnel${NC}"
cat > /opt/vma/scripts/enhanced-ssh-tunnel-remote.sh << 'TUNNELEOF'
#!/bin/bash
set -euo pipefail

OMA_HOST="${OMA_HOST:-10.245.246.125}"
SSH_KEY="/home/vma/.ssh/cloudstack_key"
LOG_FILE="/var/log/vma-tunnel-enhanced.log"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [$$] $*" | tee -a "$LOG_FILE"
}

cleanup_tunnel() {
    log "🧹 Cleaning up existing tunnel processes..."
    pkill -f "ssh.*$OMA_HOST" || true
            sleep 2
    log "✅ Cleanup completed"
}

establish_tunnel() {
    log "🔧 Establishing SSH tunnel to OMA: $OMA_HOST:443..."
    
    ssh -i "$SSH_KEY" \
        -p 443 \
        -R 9081:localhost:8081 \
        -L 8082:localhost:8082 \
        -L 10809:localhost:10809 \
        -L 10808:localhost:10809 \
        -N \
        -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o ServerAliveInterval=30 \
        -o ServerAliveCountMax=3 \
        -o ConnectTimeout=30 \
        -o TCPKeepAlive=yes \
        -o ExitOnForwardFailure=yes \
        -o BatchMode=yes \
        "pgrayson@$OMA_HOST" &
    
    local ssh_pid=$!
    log "🚀 SSH tunnel started with PID: $ssh_pid"
    
    sleep 5
    
    if ! kill -0 $ssh_pid 2>/dev/null; then
        log "❌ SSH tunnel process died immediately"
        return 1
    fi
    
    log "✅ SSH tunnel established"
    return 0
}

main() {
    log "🎯 Starting Enhanced SSH Tunnel Manager"
    log "   Target: $OMA_HOST:443"
    log "   SSH Key: $SSH_KEY"
    
    cleanup_tunnel
    
    while true; do
        if establish_tunnel; then
            log "🔄 Tunnel established, monitoring..."
            
            while true; do
                sleep 60
                if ! pgrep -f "ssh.*$OMA_HOST" >/dev/null; then
                    log "💔 Tunnel process died - restarting"
                    break
                fi
                log "💚 Tunnel health check passed"
            done
        else
            log "❌ Failed to establish tunnel"
        fi
        
        cleanup_tunnel
        log "⏳ Waiting 10 seconds before retry..."
        sleep 10
    done
}

trap 'log "🛑 Received termination signal"; cleanup_tunnel; exit 0' TERM INT
mkdir -p "$(dirname "$LOG_FILE")"
touch "$LOG_FILE"
chown vma:vma "$LOG_FILE"
main
TUNNELEOF

chmod +x /opt/vma/scripts/enhanced-ssh-tunnel-remote.sh
chown vma:vma /opt/vma/scripts/enhanced-ssh-tunnel-remote.sh
echo -e "${GREEN}✅ Enhanced tunnel script created${NC}"
echo ""

# Phase 9: Create Services
echo -e "${BOLD}🚀 Phase 9: Creating VMA Services${NC}"

# VMA API service
cat > /etc/systemd/system/vma-api.service << 'APIEOF'
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
APIEOF

# Enhanced tunnel service
cat > /etc/systemd/system/vma-tunnel-enhanced-v2.service << 'TUNNELSERVICEEOF'
[Unit]
Description=VMA Enhanced SSH Tunnel to OMA
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=vma
Group=vma
WorkingDirectory=/opt/vma
ExecStart=/opt/vma/scripts/enhanced-ssh-tunnel-remote.sh
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
Environment=OMA_HOST=10.245.246.125
Environment=SSH_KEY=/home/vma/.ssh/cloudstack_key

[Install]
WantedBy=multi-user.target
TUNNELSERVICEEOF

# Auto-login wizard service
cat > /etc/systemd/system/vma-autologin.service << 'AUTOLOGINEOF'
[Unit]
Description=VMA Auto-login Setup Wizard
After=multi-user.target network.target
Wants=network.target

[Service]
Type=idle
User=vma
Group=vma
TTYPath=/dev/tty1
ExecStart=/opt/vma/setup-wizard.sh
StandardInput=tty
StandardOutput=tty
StandardError=tty
Restart=no
RemainAfterExit=yes
Environment=HOME=/home/vma
Environment=USER=vma
Environment=TERM=xterm-256color

[Install]
WantedBy=multi-user.target
AUTOLOGINEOF

# Enable services
systemctl daemon-reload
systemctl enable vma-api.service
systemctl enable vma-tunnel-enhanced-v2.service
systemctl disable getty@tty1.service 2>/dev/null || true
systemctl enable vma-autologin.service

echo -e "${GREEN}✅ All VMA services created and enabled${NC}"
echo ""

# Phase 10: Final Setup
echo -e "${BOLD}🔧 Phase 10: Final Configuration${NC}"

# Set all permissions
chown -R vma:vma /opt/vma /home/vma
touch /var/log/vma-tunnel-enhanced.log
chown vma:vma /var/log/vma-tunnel-enhanced.log
chmod 644 /var/log/vma-tunnel-enhanced.log

echo -e "${GREEN}✅ Final configuration complete${NC}"
echo ""

# Summary
echo -e "${BOLD}${GREEN}🎉 Production VMA Deployment Complete!${NC}"
echo ""
echo -e "${CYAN}📋 Deployed Components:${NC}"
echo -e "   ${BOLD}migratekit:${NC} v2.21.0-hierarchical-sparse-optimization"
echo -e "   ${BOLD}Enhanced Tunnel:${NC} vma-tunnel-enhanced-v2.service"
echo -e "   ${BOLD}VMA API:${NC} vma-api.service" 
echo -e "   ${BOLD}Setup Wizard:${NC} /opt/vma/setup-wizard.sh (auto-login)"
echo -e "   ${BOLD}SSH Keys:${NC} cloudstack_key (pgrayson@OMA access)"
echo -e "   ${BOLD}NBD Stack:${NC} nbdkit + VDDK operational"
echo ""
echo -e "${CYAN}🎯 Next Steps:${NC}"
echo -e "   1. Reboot VMA - wizard will auto-load on console"
echo -e "   2. Configure OMA IP: 10.245.246.125"
echo -e "   3. Test tunnel connectivity"
echo -e "   4. Verify migration functionality"
echo ""
echo -e "${BOLD}${GREEN}VMA ready for enhanced tunnel operation!${NC}"
