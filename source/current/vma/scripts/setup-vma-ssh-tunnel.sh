#!/bin/bash
set -euo pipefail

# VMA SSH Tunnel Setup Script
# Sets up secure SSH reverse tunnel to OMA on port 443

OMA_IP="${1:-}"
if [[ -z "$OMA_IP" ]]; then
    echo "Usage: $0 <OMA_IP_ADDRESS>"
    echo "Example: $0 10.245.246.125"
    exit 1
fi

echo "🔧 Setting up VMA SSH tunnel to OMA: $OMA_IP"

# 1. Ensure enrollment key exists
if [[ ! -f /opt/vma/enrollment/vma_enrollment_key ]]; then
    echo "❌ ERROR: VMA enrollment key not found at /opt/vma/enrollment/vma_enrollment_key"
    echo "Run VMA enrollment wizard first!"
    exit 1
fi

# 2. Set correct permissions on enrollment key
sudo chown vma:vma /opt/vma/enrollment/vma_enrollment_key*
sudo chmod 600 /opt/vma/enrollment/vma_enrollment_key
sudo chmod 644 /opt/vma/enrollment/vma_enrollment_key.pub

# 3. Install systemd service
sudo cp /home/vma/migratekit-cloudstack/source/current/vma/scripts/vma-ssh-tunnel.service /etc/systemd/system/
sudo sed -i "s/OMA_IP_ADDRESS/$OMA_IP/g" /etc/systemd/system/vma-ssh-tunnel.service

# 4. Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable vma-ssh-tunnel.service
sudo systemctl start vma-ssh-tunnel.service

# 5. Verify tunnel
echo "⏳ Waiting for tunnel to establish..."
sleep 5

if sudo systemctl is-active --quiet vma-ssh-tunnel.service; then
    echo "✅ VMA SSH tunnel service is running"
    echo "📊 Service status:"
    sudo systemctl status vma-ssh-tunnel.service --no-pager -l
else
    echo "❌ VMA SSH tunnel service failed to start"
    echo "📋 Service logs:"
    sudo journalctl -u vma-ssh-tunnel.service --no-pager -l
    exit 1
fi

echo "🎉 VMA SSH tunnel setup complete!"
echo "🔗 Reverse tunnel: OMA:9081 → VMA:8081"


