#!/bin/bash
set -euo pipefail

# OMA SSH Tunnel Infrastructure Setup
# Configures OMA to accept secure SSH reverse tunnels from VMAs

echo "🔧 Setting up OMA SSH tunnel infrastructure"

# 1. Create vma_tunnel user if it doesn't exist
if ! id vma_tunnel &>/dev/null; then
    echo "👤 Creating vma_tunnel user..."
    sudo useradd -r -s /bin/bash -d /var/lib/vma_tunnel -m vma_tunnel
    echo "✅ vma_tunnel user created"
else
    echo "✅ vma_tunnel user already exists"
fi

# 2. Set up SSH directory with correct permissions
sudo mkdir -p /var/lib/vma_tunnel/.ssh
sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh
sudo chmod 700 /var/lib/vma_tunnel/.ssh

# 3. Create empty authorized_keys file
sudo touch /var/lib/vma_tunnel/.ssh/authorized_keys
sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh/authorized_keys
sudo chmod 600 /var/lib/vma_tunnel/.ssh/authorized_keys

# 4. Configure SSH daemon for port 443 and vma_tunnel restrictions
if ! grep -q "Port 443" /etc/ssh/sshd_config; then
    echo "🔧 Adding SSH port 443..."
    echo "Port 443" | sudo tee -a /etc/ssh/sshd_config
fi

if ! grep -q "Match User vma_tunnel" /etc/ssh/sshd_config; then
    echo "🔒 Adding vma_tunnel security restrictions..."
    sudo tee -a /etc/ssh/sshd_config << 'EOF'

# VMA Tunnel Security - Production Hardening
Match User vma_tunnel
    AuthenticationMethods publickey
    PubkeyAuthentication yes
    PasswordAuthentication no
    KbdInteractiveAuthentication no
    PermitTTY no
    X11Forwarding no
    AllowTcpForwarding yes
    AllowStreamLocalForwarding no
    GatewayPorts no
    PermitOpen 127.0.0.1:10809
    PermitListen 127.0.0.1:9081
EOF
fi

# 5. Reload SSH daemon
echo "🔄 Reloading SSH daemon..."
sudo systemctl reload ssh

# 6. Verify SSH is listening on port 443
if ss -tlnp | grep -q ":443.*ssh"; then
    echo "✅ SSH daemon listening on port 443"
else
    echo "❌ SSH daemon not listening on port 443"
    exit 1
fi

echo "🎉 OMA SSH tunnel infrastructure setup complete!"
echo "📋 Next steps:"
echo "   1. Run VMA enrollment to generate SSH keys"
echo "   2. Add VMA public key to /var/lib/vma_tunnel/.ssh/authorized_keys"
echo "   3. Test tunnel connection from VMA"
