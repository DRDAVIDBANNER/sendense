#!/bin/bash
# Deploy VMA Custom Boot Setup
# Configures VMA to boot directly to OMA connection wizard

set -euo pipefail

echo "🚀 Deploying VMA Custom Boot Setup"
echo "=================================="

# Copy to VMA
echo "📋 Copying setup wizard to VMA..."
scp -i ~/.ssh/cloudstack_key vma-setup-wizard.sh pgrayson@10.0.100.231:/tmp/
scp -i ~/.ssh/cloudstack_key vma-autologin.service pgrayson@10.0.100.231:/tmp/

# Install on VMA
echo "🔧 Installing custom boot setup on VMA..."
ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231 << 'EOF'
# Create VMA directory
sudo mkdir -p /opt/vma

# Install setup wizard
sudo cp /tmp/vma-setup-wizard.sh /opt/vma/setup-wizard.sh
sudo chmod +x /opt/vma/setup-wizard.sh
sudo chown root:root /opt/vma/setup-wizard.sh

# Install auto-login service
sudo cp /tmp/vma-autologin.service /etc/systemd/system/
sudo systemctl daemon-reload

echo "✅ VMA custom boot setup installed"
echo ""
echo "🎯 To activate custom boot:"
echo "   sudo systemctl disable getty@tty1.service"
echo "   sudo systemctl enable vma-autologin.service"
echo ""
echo "🔄 To revert to standard login:"
echo "   sudo systemctl disable vma-autologin.service" 
echo "   sudo systemctl enable getty@tty1.service"
EOF

echo ""
echo "✅ VMA Custom Boot Setup Deployment Complete"
echo ""
echo "📋 Manual Activation Steps (run on VMA):"
echo "   1. ssh -i ~/.ssh/cloudstack_key pgrayson@10.0.100.231"
echo "   2. sudo systemctl disable getty@tty1.service"
echo "   3. sudo systemctl enable vma-autologin.service"
echo "   4. sudo reboot"
echo ""
echo "🎯 After reboot, VMA will boot directly to OMA connection wizard"






