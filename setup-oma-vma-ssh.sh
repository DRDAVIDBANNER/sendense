#!/bin/bash
# OMA VMA SSH Setup Script - MVP Solution
# Pre-generates SSH key infrastructure for VMA enrollment system
# Run this after OMA deployment to enable VMA enrollment

set -euo pipefail

# Colors and formatting
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
║                OSSEA-Migrate OMA VMA SSH Setup                  ║
║                    MVP Enrollment Solution                      ║
║                                                                  ║
║              🔐 Pre-configured SSH Key Setup                    ║
╚══════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${CYAN}Setting up VMA tunnel SSH infrastructure...${NC}"
echo ""

# Configuration
VMA_TUNNEL_USER="vma_tunnel"
SSH_DIR="/home/${VMA_TUNNEL_USER}/.ssh"
SSH_KEY_PATH="${SSH_DIR}/oma_enrollment_key"
AUTHORIZED_KEYS="${SSH_DIR}/authorized_keys"
TUNNEL_WRAPPER="/usr/local/sbin/oma_tunnel_wrapper.sh"

# Check if vma_tunnel user exists
if ! id "$VMA_TUNNEL_USER" >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Creating vma_tunnel user...${NC}"
    sudo useradd -m -s /bin/bash "$VMA_TUNNEL_USER"
    echo -e "${GREEN}✅ vma_tunnel user created${NC}"
else
    echo -e "${GREEN}✅ vma_tunnel user already exists${NC}"
fi

# Create SSH directory with proper permissions
echo -e "${CYAN}📁 Setting up SSH directory...${NC}"
sudo mkdir -p "$SSH_DIR"
sudo chown "${VMA_TUNNEL_USER}:${VMA_TUNNEL_USER}" "$SSH_DIR"
sudo chmod 700 "$SSH_DIR"

# Generate OMA enrollment key (this will be the shared key for all VMAs)
echo -e "${CYAN}🔐 Generating OMA enrollment SSH key...${NC}"
if [ ! -f "$SSH_KEY_PATH" ]; then
    sudo ssh-keygen -t ed25519 -f "$SSH_KEY_PATH" -N "" -C "OMA_enrollment_key_$(date +%Y%m%d)"
    sudo chown "${VMA_TUNNEL_USER}:${VMA_TUNNEL_USER}" "$SSH_KEY_PATH"*
    sudo chmod 600 "$SSH_KEY_PATH"*
    echo -e "${GREEN}✅ OMA enrollment SSH key generated${NC}"
else
    echo -e "${GREEN}✅ OMA enrollment SSH key already exists${NC}"
fi

# Create tunnel wrapper script
echo -e "${CYAN}🔧 Creating tunnel wrapper script...${NC}"
sudo tee "$TUNNEL_WRAPPER" > /dev/null << 'WRAPPER_EOF'
#!/bin/bash
# OMA Tunnel Wrapper Script
# Logs VMA connections and manages tunnel access

LOG_FILE="/var/log/vma-tunnel-connections.log"
CLIENT_IP="${SSH_CLIENT%% *}"

# Log connection
echo "$(date '+%Y-%m-%d %H:%M:%S') - VMA connection from ${CLIENT_IP:-unknown}" >> "$LOG_FILE"

# Execute SSH tunnel (allow port forwarding)
exec "$@"
WRAPPER_EOF

sudo chmod +x "$TUNNEL_WRAPPER"
sudo chown root:root "$TUNNEL_WRAPPER"

# Create authorized_keys with restricted access
echo -e "${CYAN}🔑 Setting up authorized_keys template...${NC}"
PUBLIC_KEY=$(sudo cat "${SSH_KEY_PATH}.pub")
sudo tee "$AUTHORIZED_KEYS" > /dev/null << AUTHKEY_EOF
# VMA Enrollment SSH Keys
# Format: command="wrapper",restrict,permitopen="ports" ssh-ed25519 KEY comment

# Example VMA enrollment key (replace with actual VMA keys):
# command="${TUNNEL_WRAPPER}",restrict,permitopen="127.0.0.1:10809",permitopen="127.0.0.1:8081" ssh-ed25519 AAAAC3... VMA_NAME

# OMA enrollment key (for manual testing):
command="${TUNNEL_WRAPPER}",restrict,permitopen="127.0.0.1:10809",permitopen="127.0.0.1:8081" ${PUBLIC_KEY}
AUTHKEY_EOF

sudo chown "${VMA_TUNNEL_USER}:${VMA_TUNNEL_USER}" "$AUTHORIZED_KEYS"
sudo chmod 600 "$AUTHORIZED_KEYS"

# Configure SSH daemon for VMA tunnel user restrictions
echo -e "${CYAN}🔒 Configuring SSH restrictions...${NC}"
SSH_CONFIG="/etc/ssh/sshd_config.d/vma-tunnel.conf"
sudo tee "$SSH_CONFIG" > /dev/null << 'SSHCONF_EOF'
# VMA Tunnel User Restrictions
Match User vma_tunnel
    PermitTTY no
    X11Forwarding no
    AllowTcpForwarding yes
    PermitOpen 127.0.0.1:10809 127.0.0.1:8081
    ForceCommand /usr/local/sbin/oma_tunnel_wrapper.sh
SSHCONF_EOF

# Test SSH configuration
echo -e "${CYAN}🧪 Testing SSH configuration...${NC}"
if sudo sshd -t; then
    echo -e "${GREEN}✅ SSH configuration is valid${NC}"
    echo -e "${CYAN}⚠️  Restarting SSH service...${NC}"
    sudo systemctl reload ssh
    echo -e "${GREEN}✅ SSH service reloaded${NC}"
else
    echo -e "${RED}❌ SSH configuration error${NC}"
    exit 1
fi

# Create log file with proper permissions
sudo touch /var/log/vma-tunnel-connections.log
sudo chown syslog:adm /var/log/vma-tunnel-connections.log
sudo chmod 644 /var/log/vma-tunnel-connections.log

echo ""
echo -e "${GREEN}${BOLD}🎉 OMA VMA SSH Setup Complete!${NC}"
echo ""
echo -e "${CYAN}📋 Setup Summary:${NC}"
echo -e "   👤 VMA tunnel user: ${BOLD}${VMA_TUNNEL_USER}${NC}"
echo -e "   🔐 SSH key location: ${BOLD}${SSH_KEY_PATH}${NC}"
echo -e "   📁 SSH directory: ${BOLD}${SSH_DIR}${NC}"
echo -e "   🔑 Authorized keys: ${BOLD}${AUTHORIZED_KEYS}${NC}"
echo -e "   🛡️  Tunnel wrapper: ${BOLD}${TUNNEL_WRAPPER}${NC}"
echo -e "   📊 Connection log: ${BOLD}/var/log/vma-tunnel-connections.log${NC}"
echo ""
echo -e "${YELLOW}📋 Next Steps:${NC}"
echo "1. Test VMA enrollment with existing pairing codes"
echo "2. Manually add VMA public keys to authorized_keys after approval"
echo "3. Test tunnel connectivity from approved VMAs"
echo ""
echo -e "${CYAN}🔍 To view SSH key fingerprint:${NC}"
echo "   sudo ssh-keygen -lf ${SSH_KEY_PATH}.pub"
echo ""
echo -e "${CYAN}🔍 To monitor VMA connections:${NC}"
echo "   tail -f /var/log/vma-tunnel-connections.log"
echo ""
echo -e "${YELLOW}⚠️  MVP Note:${NC} For now, manually add VMA public keys to authorized_keys"
echo "   after approval in the GUI. Future versions will automate this."
