#!/bin/bash
# Deploy Sendense SSH Tunnel to SNA (Node Appliance)
# Version: 1.1.0
# Date: 2025-10-07
#
# USAGE:
#   Local:  sudo ./deploy-sna-tunnel.sh
#   Remote: sshpass -p 'Password1' ./deploy-sna-tunnel.sh 10.0.100.231

set -euo pipefail

SNA_HOST="${1:-localhost}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Deploying Sendense SSH Tunnel to SNA: $SNA_HOST"
echo "======================================================"

deploy_local() {
    echo "📦 Local deployment mode"
    
    # Check if running with sudo
    if [ "$EUID" -ne 0 ]; then 
        echo "❌ ERROR: This script must be run with sudo for local deployment"
        exit 1
    fi
    
    # Copy script
    echo "📝 Installing tunnel script..."
    cp "$SCRIPT_DIR/sendense-tunnel.sh" /usr/local/bin/
    chmod +x /usr/local/bin/sendense-tunnel.sh
    chown root:root /usr/local/bin/sendense-tunnel.sh
    
    # Copy service
    echo "📝 Installing systemd service..."
    cp "$SCRIPT_DIR/sendense-tunnel.service" /etc/systemd/system/
    chmod 644 /etc/systemd/system/sendense-tunnel.service
    
    # Reload systemd
    echo "🔄 Reloading systemd..."
    systemctl daemon-reload
    
    # Enable and start service
    echo "🚀 Enabling and starting service..."
    systemctl enable sendense-tunnel
    systemctl restart sendense-tunnel
    
    # Wait and check status
    sleep 3
    echo ""
    echo "📊 Service Status:"
    systemctl status sendense-tunnel --no-pager || true
    
    echo ""
    echo "✅ SUCCESS: SNA tunnel deployed locally"
}

deploy_remote() {
    echo "📦 Remote deployment mode to: $SNA_HOST"
    
    # Check for sshpass
    if ! command -v sshpass &> /dev/null; then
        echo "❌ ERROR: sshpass is required for remote deployment"
        exit 1
    fi
    
    # Transfer files
    echo "📤 Transferring files to SNA..."
    sshpass -p 'Password1' scp "$SCRIPT_DIR/sendense-tunnel.sh" \
        "$SCRIPT_DIR/sendense-tunnel.service" \
        vma@"$SNA_HOST":/tmp/
    
    # Deploy on remote
    echo "📝 Installing on SNA..."
    sshpass -p 'Password1' ssh vma@"$SNA_HOST" 'sudo bash -s' << 'REMOTE_SCRIPT'
        # Install script
        sudo mv /tmp/sendense-tunnel.sh /usr/local/bin/
        sudo chmod +x /usr/local/bin/sendense-tunnel.sh
        sudo chown root:root /usr/local/bin/sendense-tunnel.sh
        
        # Install service
        sudo mv /tmp/sendense-tunnel.service /etc/systemd/system/
        sudo chmod 644 /etc/systemd/system/sendense-tunnel.service
        
        # Reload and start
        sudo systemctl daemon-reload
        sudo systemctl enable sendense-tunnel
        sudo systemctl restart sendense-tunnel
        
        echo "Waiting for service to start..."
        sleep 3
REMOTE_SCRIPT
    
    # Check status
    echo ""
    echo "📊 Service Status:"
    sshpass -p 'Password1' ssh vma@"$SNA_HOST" 'sudo systemctl status sendense-tunnel --no-pager' || true
    
    echo ""
    echo "✅ SUCCESS: SNA tunnel deployed remotely to $SNA_HOST"
}

# Main
if [ "$SNA_HOST" == "localhost" ]; then
    deploy_local
else
    deploy_remote
fi

echo ""
echo "Next steps:"
echo "1. Verify tunnel connected: systemctl status sendense-tunnel"
echo "2. Check forwarded ports: netstat -an | grep LISTEN | grep 1010"
echo "3. Test SHA connectivity: curl http://localhost:8082/health"
