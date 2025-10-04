#!/bin/bash
set -euo pipefail

# Production SSH Tunnel Deployment Script
# Deploys complete bidirectional SSH tunnel infrastructure

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

OMA_IP="${1:-}"
VMA_IP="${2:-}"

if [[ -z "$OMA_IP" || -z "$VMA_IP" ]]; then
    echo "Usage: $0 <OMA_IP> <VMA_IP>"
    echo "Example: $0 10.245.246.125 10.0.100.232"
    exit 1
fi

echo "🚀 Deploying Production SSH Tunnel Infrastructure"
echo "   OMA: $OMA_IP"
echo "   VMA: $VMA_IP"
echo ""

# 1. Clean existing deployment on VMA
echo "🧹 Cleaning existing VMA deployment..."
ssh -i ~/.ssh/vma_232_key vma@$VMA_IP "
sudo systemctl stop vma-ssh-tunnel.service 2>/dev/null || true
sudo systemctl disable vma-ssh-tunnel.service 2>/dev/null || true
sudo rm -f /etc/systemd/system/vma-ssh-tunnel.service
sudo rm -f /usr/local/bin/vma-tunnel-wrapper.sh
sudo systemctl daemon-reload
echo '✅ VMA cleanup complete'
"

# 2. Deploy OMA SSH tunnel infrastructure
echo "📡 Setting up OMA SSH tunnel infrastructure..."
scp -i ~/.ssh/cloudstack_key "$PROJECT_ROOT/source/current/oma/scripts/setup-oma-ssh-tunnel.sh" pgrayson@$OMA_IP:/tmp/
ssh -i ~/.ssh/cloudstack_key pgrayson@$OMA_IP "sudo bash /tmp/setup-oma-ssh-tunnel.sh"

# 3. Verify VMA enrollment key exists
echo "🔑 Verifying VMA enrollment status..."
if ! ssh -i ~/.ssh/vma_232_key vma@$VMA_IP "test -f /opt/vma/enrollment/vma_enrollment_key"; then
    echo "❌ VMA enrollment key not found!"
    echo "   Run enrollment wizard first:"
    echo "   ssh -i ~/.ssh/vma_232_key vma@$VMA_IP"
    echo "   sudo /opt/vma/enrollment/vma-enrollment-wizard.sh"
    exit 1
fi

# 4. Get VMA public key and add to OMA with hardened restrictions
echo "🔒 Configuring hardened SSH key authentication..."
VMA_PUBLIC_KEY=$(ssh -i ~/.ssh/vma_232_key vma@$VMA_IP "cat /opt/vma/enrollment/vma_enrollment_key.pub")
ssh -i ~/.ssh/cloudstack_key pgrayson@$OMA_IP "echo 'no-pty,no-X11-forwarding,no-agent-forwarding,no-user-rc,permitlisten=\"127.0.0.1:9081\",command=\"/bin/true\" $VMA_PUBLIC_KEY' | sudo tee /var/lib/vma_tunnel/.ssh/authorized_keys"
echo "✅ Hardened SSH key configured"

# 5. Deploy VMA wrapper script
echo "📡 Deploying VMA tunnel wrapper..."
scp -i ~/.ssh/vma_232_key "$PROJECT_ROOT/source/current/vma/scripts/vma-tunnel-wrapper.sh" vma@$VMA_IP:/tmp/
ssh -i ~/.ssh/vma_232_key vma@$VMA_IP "
sudo cp /tmp/vma-tunnel-wrapper.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/vma-tunnel-wrapper.sh
echo '✅ Wrapper script deployed'
"

# 6. Create and deploy VMA systemd service
echo "⚙️  Creating VMA systemd service..."
ssh -i ~/.ssh/vma_232_key vma@$VMA_IP "
sudo tee /etc/systemd/system/vma-ssh-tunnel.service << 'EOF'
[Unit]
Description=VMA SSH Tunnel to OMA
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=vma
Group=vma
ExecStart=/usr/local/bin/vma-tunnel-wrapper.sh $OMA_IP
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
"

# 7. Enable and start VMA tunnel service
echo "🚀 Starting VMA tunnel service..."
ssh -i ~/.ssh/vma_232_key vma@$VMA_IP "
sudo systemctl daemon-reload
sudo systemctl enable vma-ssh-tunnel.service
sudo systemctl start vma-ssh-tunnel.service
echo '✅ Service started'
"

# 8. Wait for tunnel to establish
echo "⏳ Waiting for tunnel to establish..."
sleep 10

# 9. Test bidirectional connectivity
echo ""
echo "🧪 Testing bidirectional tunnel..."
echo ""

# Test reverse direction (OMA → VMA)
echo "   Testing OMA → VMA (reverse tunnel)..."
if ssh -i ~/.ssh/cloudstack_key pgrayson@$OMA_IP "curl -s -f http://127.0.0.1:9081/api/v1/health" >/dev/null 2>&1; then
    echo "   ✅ Reverse tunnel working"
else
    echo "   ❌ Reverse tunnel failed"
    echo "   📋 Checking VMA service status..."
    ssh -i ~/.ssh/vma_232_key vma@$VMA_IP "sudo systemctl status vma-ssh-tunnel.service --no-pager -l"
    exit 1
fi

echo ""
echo "🎉 Production SSH Tunnel Deployment Complete!"
echo ""
echo "📊 Architecture Summary:"
echo "   VMA ($VMA_IP) ←→ SSH Tunnel (Port 443) ←→ OMA ($OMA_IP)"
echo "   Reverse:  OMA:9081 → VMA API (port 8081)"
echo ""
echo "🔧 Management Commands:"
echo "   VMA Tunnel Status: ssh -i ~/.ssh/vma_232_key vma@$VMA_IP 'sudo systemctl status vma-ssh-tunnel'"
echo "   VMA Tunnel Logs:   ssh -i ~/.ssh/vma_232_key vma@$VMA_IP 'sudo journalctl -u vma-ssh-tunnel -f'"
echo "   Test Reverse:      ssh -i ~/.ssh/cloudstack_key pgrayson@$OMA_IP 'curl http://127.0.0.1:9081/api/v1/health'"
echo ""
echo "🔒 Security Features:"
echo "   ✅ SSH on port 443 only"
echo "   ✅ Surgically restricted SSH access (no PTY, no X11, no agent forwarding)"
echo "   ✅ Limited port forwarding (9081 only)"
echo "   ✅ Forced command execution (/bin/true)"
echo "   ✅ Public key authentication only"
echo "   ✅ Systemd service with auto-restart"