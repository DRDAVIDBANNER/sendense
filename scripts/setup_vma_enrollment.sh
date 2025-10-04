#!/bin/bash
# Setup VMA Enrollment System for OMA Deployment
# Generates SSH keys and configures vma_tunnel user

set -euo pipefail

echo "🔑 Setting up VMA Enrollment System"
echo "==================================="

# Create vma_tunnel system user
echo "👥 Creating vma_tunnel system user..."
if ! id vma_tunnel >/dev/null 2>&1; then
    sudo useradd -r -m -d /var/lib/vma_tunnel -s /bin/false -c "VMA SSH Tunnel User" vma_tunnel
    echo "✅ Created vma_tunnel system user"
else
    echo "ℹ️  vma_tunnel user already exists"
fi

# Setup SSH directory with proper permissions
echo "📁 Setting up SSH directory structure..."
sudo mkdir -p /var/lib/vma_tunnel/.ssh
sudo chown -R vma_tunnel:vma_tunnel /var/lib/vma_tunnel
sudo chmod 700 /var/lib/vma_tunnel/.ssh

# Generate unique SSH key for this OMA instance
echo "🔑 Generating SSH key for VMA enrollment system..."
HOSTNAME=$(hostname)
DATE=$(date +%Y%m%d%H%M)
KEY_COMMENT="VMA-Enrollment-${HOSTNAME}-${DATE}"

sudo -u vma_tunnel ssh-keygen -t ed25519 \
    -f /var/lib/vma_tunnel/.ssh/vma_enrollment_key \
    -N "" \
    -C "${KEY_COMMENT}"

# Create authorized_keys with the generated key and proper restrictions
echo "🔒 Configuring authorized_keys with tunnel restrictions..."
sudo tee /var/lib/vma_tunnel/.ssh/authorized_keys > /dev/null << EOF
# VMA Enrollment System - Auto-generated key for ${HOSTNAME}
# Generated: $(date)
# Restrictions: Tunnel access only (ports 10809, 8081)
command="/usr/local/sbin/oma_tunnel_wrapper.sh",restrict,permitopen="127.0.0.1:10809",permitopen="127.0.0.1:8081" $(sudo cat /var/lib/vma_tunnel/.ssh/vma_enrollment_key.pub) # VMA enrollment key - ${HOSTNAME}
EOF

# Set proper permissions
sudo chmod 600 /var/lib/vma_tunnel/.ssh/authorized_keys
sudo chown vma_tunnel:vma_tunnel /var/lib/vma_tunnel/.ssh/authorized_keys

# Create tunnel wrapper script
echo "🔧 Creating VMA tunnel wrapper script..."
sudo tee /usr/local/sbin/oma_tunnel_wrapper.sh > /dev/null << 'EOF'
#!/bin/bash
# OMA Tunnel Wrapper Script for VMA SSH Connections
# Logs VMA connections and allows tunnel forwarding

# Create log file if it doesn't exist
sudo touch /var/log/vma-connections.log
sudo chmod 644 /var/log/vma-connections.log

# Log connection details
if [ -n "$SSH_CLIENT" ]; then
    echo "$(date): VMA tunnel connection from $SSH_CLIENT" >> /var/log/vma-connections.log
fi

# Log the command being executed  
if [ $# -gt 0 ]; then
    echo "$(date): VMA tunnel command: $*" >> /var/log/vma-connections.log
fi

# Allow SSH tunnel forwarding by executing the command
exec "$@"
EOF

sudo chmod 755 /usr/local/sbin/oma_tunnel_wrapper.sh

# Configure sudo permissions for VMA enrollment SSH automation
echo "🔐 Configuring VMA enrollment sudo permissions..."
sudo tee /etc/sudoers.d/oma-vma-enrollment > /dev/null << 'EOF'
# VMA Enrollment System - Allow oma user to manage vma_tunnel user
# Required for SSH key automation during VMA approval workflow

# Allow oma user to create vma_tunnel user and setup SSH
oma ALL=(root) NOPASSWD: /usr/sbin/useradd -r -m -d /var/lib/vma_tunnel -s /bin/false -c VMA\ SSH\ Tunnel\ User vma_tunnel
oma ALL=(root) NOPASSWD: /bin/mkdir -p /var/lib/vma_tunnel/.ssh
oma ALL=(root) NOPASSWD: /bin/chown vma_tunnel /var/lib/vma_tunnel
oma ALL=(root) NOPASSWD: /bin/chown vma_tunnel /var/lib/vma_tunnel/.ssh
oma ALL=(root) NOPASSWD: /bin/chown vma_tunnel /var/lib/vma_tunnel/.ssh/authorized_keys
oma ALL=(root) NOPASSWD: /bin/chmod 700 /var/lib/vma_tunnel/.ssh
oma ALL=(root) NOPASSWD: /bin/chmod 600 /var/lib/vma_tunnel/.ssh/authorized_keys
oma ALL=(root) NOPASSWD: /usr/bin/tee /usr/local/sbin/oma_tunnel_wrapper.sh
oma ALL=(root) NOPASSWD: /bin/chmod 755 /usr/local/sbin/oma_tunnel_wrapper.sh
EOF

sudo chmod 0440 /etc/sudoers.d/oma-vma-enrollment

# Validate sudoers configuration
if sudo visudo -c >/dev/null 2>&1; then
    echo "✅ Sudoers configuration validated"
else
    echo "❌ Sudoers configuration validation failed"
    exit 1
fi

echo ""
echo "✅ VMA Enrollment System Setup Complete"
echo "======================================="
echo "🔑 SSH Key: /var/lib/vma_tunnel/.ssh/vma_enrollment_key.pub"
echo "👤 User: vma_tunnel"
echo "🔒 Restrictions: Tunnel access only (ports 10809, 8081)"
echo "📋 Next: Deploy OMA API with VMA enrollment endpoints"
echo ""
echo "🔑 Public Key for Reference:"
sudo cat /var/lib/vma_tunnel/.ssh/vma_enrollment_key.pub
echo ""





