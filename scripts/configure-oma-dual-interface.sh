#!/bin/bash
# OMA Dual Interface Configuration Script
# Configure NBD server to bind to specific interface

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${BOLD}🔧 OMA Dual Interface Configuration${NC}"
echo ""

# Get current interfaces
echo "Current network interfaces:"
ip addr show | grep -E "inet.*global" | awk '{print $2, $7}' | head -5
echo ""

# Get Volume Target IP
read -p "Enter OMA Volume Target IP (for NBD exports): " VOLUME_TARGET_IP

if [[ ! $VOLUME_TARGET_IP =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}❌ Invalid IP address format${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}📋 Configuring NBD server to bind to $VOLUME_TARGET_IP...${NC}"

# Backup current NBD config
if [ -f /etc/nbd-server/config ]; then
    sudo cp /etc/nbd-server/config /etc/nbd-server/config.backup-$(date +%Y%m%d-%H%M%S)
    echo "✅ NBD config backed up"
fi

# Update NBD server main config
sudo tee /etc/nbd-server/config > /dev/null << EOF
[generic]
user = nbd
group = nbd
port = 10809
bind = $VOLUME_TARGET_IP
allowlist = true
includedir = /etc/nbd-server/conf.d

# Logging
logfile = /var/log/nbd-server.log
loglevel = 3

# Performance tuning for high-speed transfers
max_connections = 10
timeout = 30
EOF

echo "✅ NBD server config updated to bind to $VOLUME_TARGET_IP"

# Create systemd override for interface binding
sudo mkdir -p /etc/systemd/system/nbd-server.service.d/
sudo tee /etc/systemd/system/nbd-server.service.d/interface-binding.conf > /dev/null << EOF
[Unit]
Description=Network Block Device Server (Interface Bound)

[Service]
Environment=NBD_BIND_IP=$VOLUME_TARGET_IP
# Ensure NBD server starts after network interfaces are up
After=network-online.target
Wants=network-online.target
EOF

echo "✅ NBD server systemd override created"

# Update Volume Daemon to use correct interface for NBD exports
echo ""
echo "Updating Volume Daemon configuration..."

# Create Volume Daemon override for dual interface
sudo mkdir -p /etc/systemd/system/volume-daemon.service.d/
sudo tee /etc/systemd/system/volume-daemon.service.d/dual-interface.conf > /dev/null << EOF
[Service]
Environment=NBD_EXPORT_HOST=$VOLUME_TARGET_IP
Environment=NBD_EXPORT_PORT=10809
EOF

echo "✅ Volume Daemon configured for dual interface"

# Reload systemd and restart services
echo ""
echo "Reloading systemd and restarting services..."
sudo systemctl daemon-reload

echo "Stopping NBD server..."
sudo systemctl stop nbd-server.service

echo "Stopping Volume Daemon..."
sudo systemctl stop volume-daemon.service

sleep 3

echo "Starting Volume Daemon..."
sudo systemctl start volume-daemon.service

echo "Starting NBD server..."
sudo systemctl start nbd-server.service

sleep 3

# Verify services
echo ""
echo "Service status:"
if systemctl is-active nbd-server.service >/dev/null 2>&1; then
    echo -e "- NBD Server: ${GREEN}Active${NC}"
else
    echo -e "- NBD Server: ${RED}Failed${NC}"
fi

if systemctl is-active volume-daemon.service >/dev/null 2>&1; then
    echo -e "- Volume Daemon: ${GREEN}Active${NC}"
else
    echo -e "- Volume Daemon: ${RED}Failed${NC}"
fi

# Test NBD binding
echo ""
echo "Testing NBD server binding..."
if timeout 5 nc -z "$VOLUME_TARGET_IP" 10809 2>/dev/null; then
    echo -e "${GREEN}✅ NBD server listening on $VOLUME_TARGET_IP:10809${NC}"
else
    echo -e "${RED}❌ NBD server not accessible on $VOLUME_TARGET_IP:10809${NC}"
fi

echo ""
echo -e "${GREEN}🎉 OMA Dual Interface Configuration Complete!${NC}"
echo ""
echo "Configuration Summary:"
echo "- NBD Server bound to: $VOLUME_TARGET_IP:10809"
echo "- Volume Daemon configured for dual interface"
echo "- Services restarted and active"
echo ""
echo "Next steps:"
echo "1. Configure VMA to use dual interface"
echo "2. Set OMA Manager IP in VMA wizard"
echo "3. Set OMA Volume Target IP in VMA wizard"
echo "4. Test migration with dual interface setup"


