#!/bin/bash
# Deploy enhanced SSH tunnel service to VMA
# This script installs the improved tunnel with keep-alive and monitoring

set -euo pipefail

VMA_HOST="pgrayson@10.0.100.231"
SSH_KEY="$HOME/.ssh/cloudstack_key"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🚀 Deploying Enhanced SSH Tunnel Service to VMA"
echo "   VMA: $VMA_HOST"
echo "   Project: $PROJECT_ROOT"

# Function to run commands on VMA
run_on_vma() {
    ssh -i "$SSH_KEY" "$VMA_HOST" "$@"
}

# Copy enhanced script to VMA
echo "📁 Copying enhanced tunnel script to VMA..."
scp -i "$SSH_KEY" "$SCRIPT_DIR/enhanced-ssh-tunnel.sh" "$VMA_HOST:/tmp/"

# Copy service file to VMA
echo "📁 Copying systemd service file to VMA..."
scp -i "$SSH_KEY" "$SCRIPT_DIR/vma-tunnel-enhanced-v2.service" "$VMA_HOST:/tmp/"

# Install on VMA
echo "🔧 Installing enhanced tunnel service on VMA..."
run_on_vma 'bash -s' <<'EOF'
set -euo pipefail

echo "🛑 Stopping existing tunnel service..."
sudo systemctl stop vma-tunnel-enhanced || true

echo "📝 Installing enhanced script..."
sudo mkdir -p /home/pgrayson/migratekit-cloudstack/scripts
sudo cp /tmp/enhanced-ssh-tunnel.sh /home/pgrayson/migratekit-cloudstack/scripts/
sudo chmod +x /home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel.sh
sudo chown pgrayson:pgrayson /home/pgrayson/migratekit-cloudstack/scripts/enhanced-ssh-tunnel.sh

echo "📝 Installing enhanced systemd service..."
sudo cp /tmp/vma-tunnel-enhanced-v2.service /etc/systemd/system/
sudo systemctl daemon-reload

echo "🔄 Starting enhanced tunnel service..."
sudo systemctl enable vma-tunnel-enhanced-v2
sudo systemctl start vma-tunnel-enhanced-v2

echo "✅ Enhanced tunnel service installed and started"

# Show status
echo "📊 Service status:"
sudo systemctl status vma-tunnel-enhanced-v2 --no-pager -l
EOF

# Test tunnel connectivity
echo "🧪 Testing tunnel connectivity..."
sleep 10  # Give tunnel time to establish

if run_on_vma "curl --connect-timeout 5 --max-time 10 -s http://localhost:8082/health" >/dev/null; then
    echo "✅ Tunnel connectivity test passed"
else
    echo "❌ Tunnel connectivity test failed"
    echo "📋 Service logs:"
    run_on_vma "sudo journalctl -u vma-tunnel-enhanced-v2 --no-pager -l --since='5 minutes ago'"
    exit 1
fi

echo ""
echo "🎉 Enhanced SSH Tunnel Deployment Complete!"
echo ""
echo "📊 Service Management Commands:"
echo "   Status:  ssh $VMA_HOST 'sudo systemctl status vma-tunnel-enhanced-v2'"
echo "   Logs:    ssh $VMA_HOST 'sudo journalctl -u vma-tunnel-enhanced-v2 -f'"
echo "   Restart: ssh $VMA_HOST 'sudo systemctl restart vma-tunnel-enhanced-v2'"
echo ""
echo "🔍 Tunnel Health Check:"
echo "   curl http://localhost:9081/api/v1/health  # (from OMA)"
echo "   ssh $VMA_HOST 'curl http://localhost:8082/health'  # (from VMA)"
