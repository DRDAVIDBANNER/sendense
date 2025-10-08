#!/bin/bash
# Deploy SSH Tunnel Configuration to SHA (Hub Appliance)
# Version: 1.1.0
# Date: 2025-10-07

set -euo pipefail

echo "🚀 Deploying Sendense SSH Tunnel Configuration to SHA"
echo "======================================================"

# Check if running with sudo
if [ "$EUID" -ne 0 ]; then 
    echo "❌ ERROR: This script must be run with sudo"
    exit 1
fi

# Backup existing sshd_config
BACKUP_FILE="/etc/ssh/sshd_config.backup-$(date +%Y%m%d-%H%M%S)"
echo "📦 Backing up current sshd_config to: $BACKUP_FILE"
cp /etc/ssh/sshd_config "$BACKUP_FILE"

# Check if vma_tunnel configuration already exists
if grep -q "Match User vma_tunnel" /etc/ssh/sshd_config; then
    echo "⚠️  WARNING: vma_tunnel configuration already exists in sshd_config"
    echo "    Please manually review and update if needed"
    echo "    Config snippet location: $(pwd)/sshd_config.snippet"
    exit 0
fi

# Append configuration
echo ""
echo "📝 Adding Sendense tunnel configuration to sshd_config..."
cat sshd_config.snippet >> /etc/ssh/sshd_config

# Test configuration
echo "✅ Testing SSH configuration..."
if ! sshd -t; then
    echo "❌ ERROR: SSH configuration test failed!"
    echo "    Restoring backup..."
    cp "$BACKUP_FILE" /etc/ssh/sshd_config
    exit 1
fi

# Reload SSH
echo "🔄 Reloading SSH daemon..."
systemctl reload sshd

echo ""
echo "✅ SUCCESS: SHA SSH tunnel configuration deployed"
echo ""
echo "Next steps:"
echo "1. Ensure vma_tunnel user exists with Ed25519 key"
echo "2. Deploy SNA tunnel service"
echo "3. Test tunnel connectivity"
echo ""
echo "Backup location: $BACKUP_FILE"

