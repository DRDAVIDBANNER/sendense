#!/bin/bash
# Deploy OMA Custom Boot Setup
# Configures OMA to boot directly to network configuration and service status wizard

set -euo pipefail

echo "🚀 Deploying OMA Custom Boot Setup"
echo "=================================="

# Create OMA directory
echo "📋 Creating OMA setup directory..."
sudo mkdir -p /opt/ossea-migrate

# Install setup wizard
echo "🔧 Installing OMA setup wizard..."
sudo cp oma-setup-wizard.sh /opt/ossea-migrate/oma-setup-wizard.sh
sudo chmod +x /opt/ossea-migrate/oma-setup-wizard.sh
sudo chown root:root /opt/ossea-migrate/oma-setup-wizard.sh

# Install auto-login service
echo "🔧 Installing OMA auto-login service..."
sudo cp oma-autologin.service /etc/systemd/system/
sudo systemctl daemon-reload

echo ""
echo "✅ OMA Custom Boot Setup Installation Complete"
echo ""
echo "🎯 To activate custom boot:"
echo "   sudo systemctl disable getty@tty1.service"
echo "   sudo systemctl enable oma-autologin.service"
echo ""
echo "🔄 To revert to standard login:"
echo "   sudo systemctl disable oma-autologin.service"
echo "   sudo systemctl enable getty@tty1.service"
echo ""
echo "📊 After activation, OMA will boot to:"
echo "   - Network configuration interface"
echo "   - Service status dashboard"
echo "   - Professional OSSEA-Migrate branding"
echo "   - Access URL guidance"






