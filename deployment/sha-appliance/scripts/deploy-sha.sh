#!/bin/bash
# deploy-sha.sh - Sendense Hub Appliance Deployment Script
# Version: v3.0.1
# Last Updated: 2025-10-04
# Compatible with: SHA v3.0.1, Volume Daemon v1.2.1, Sendense Cockpit v1.2.0
# Tested on: Ubuntu 22.04 LTS, RHEL 9

set -euo pipefail

SCRIPT_VERSION="v3.0.1"
REQUIRED_SHA_VERSION="v3.0.1"
REQUIRED_VOLUME_DAEMON_VERSION="v1.2.1"
REQUIRED_GUI_VERSION="v1.2.0"

echo "🚀 Sendense Hub Appliance (SHA) Deployment Script ${SCRIPT_VERSION}"
echo "📦 Target SHA Version: ${REQUIRED_SHA_VERSION}"
echo "⚙️ Volume Daemon Version: ${REQUIRED_VOLUME_DAEMON_VERSION}"
echo "🖥️ Cockpit GUI Version: ${REQUIRED_GUI_VERSION}"

# Deployment configuration
INSTALL_DIR="/opt/sendense"
BINARY_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sendense"
LOG_DIR="/var/log/sendense"
DATA_DIR="/var/lib/sendense"

SHA_BINARY="sendense-hub-v${REQUIRED_SHA_VERSION}-linux-amd64-20251004-abc123ef"
VOLUME_DAEMON_BINARY="volume-daemon-v${REQUIRED_VOLUME_DAEMON_VERSION}-linux-amd64-20251004-def456ab"

# Validation functions
validate_system() {
    echo "🔍 Validating system requirements..."
    
    # Check OS
    if ! command -v systemctl &> /dev/null; then
        echo "❌ systemd required but not found"
        exit 1
    fi
    
    # Check minimum resources
    TOTAL_RAM=$(free -g | awk '/^Mem:/{print $2}')
    if [[ $TOTAL_RAM -lt 8 ]]; then
        echo "⚠️ Warning: Less than 8GB RAM detected ($TOTAL_RAM GB)"
        echo "   Minimum recommended: 8GB for SHA"
    fi
    
    # Check disk space
    AVAILABLE_SPACE=$(df /var/lib -BG | awk 'NR==2{print $4}' | sed 's/G//')
    if [[ $AVAILABLE_SPACE -lt 100 ]]; then
        echo "❌ Insufficient disk space: ${AVAILABLE_SPACE}GB available, 100GB+ required"
        exit 1
    fi
    
    echo "✅ System validation passed"
}

validate_binaries() {
    echo "🔍 Validating binary integrity..."
    
    cd "$(dirname "$0")/../binaries"
    
    if [[ ! -f "CHECKSUMS.sha256" ]]; then
        echo "❌ CHECKSUMS.sha256 not found"
        exit 1
    fi
    
    sha256sum -c CHECKSUMS.sha256 || {
        echo "❌ Binary checksum validation failed"
        exit 1
    }
    
    echo "✅ Binary validation passed"
}

install_sendense_hub() {
    echo "📦 Installing Sendense Hub Appliance..."
    
    # Create directories
    sudo mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    sudo mkdir -p "$DATA_DIR"/{backups,repositories,temp}
    
    # Install binaries
    sudo cp "../binaries/$SHA_BINARY" "$BINARY_DIR/sendense-hub"
    sudo cp "../binaries/$VOLUME_DAEMON_BINARY" "$BINARY_DIR/volume-daemon"
    sudo chmod +x "$BINARY_DIR/sendense-hub" "$BINARY_DIR/volume-daemon"
    
    # Install configuration
    sudo cp ../configs/sendense-hub.service /etc/systemd/system/
    sudo cp ../configs/volume-daemon.service /etc/systemd/system/
    sudo cp ../configs/config-templates/* "$CONFIG_DIR/"
    
    # Install GUI
    if [[ -f "../gui/sendense-cockpit-v${REQUIRED_GUI_VERSION}.tar.gz" ]]; then
        sudo tar -xzf "../gui/sendense-cockpit-v${REQUIRED_GUI_VERSION}.tar.gz" -C "$INSTALL_DIR/"
        echo "✅ Sendense Cockpit GUI installed"
    fi
    
    echo "✅ Sendense Hub Appliance installed"
}

configure_services() {
    echo "⚙️ Configuring Sendense services..."
    
    # Reload systemd
    sudo systemctl daemon-reload
    
    # Enable services
    sudo systemctl enable sendense-hub
    sudo systemctl enable volume-daemon
    
    # Start services
    sudo systemctl start volume-daemon
    sleep 5
    sudo systemctl start sendense-hub
    
    echo "✅ Services configured and started"
}

validate_deployment() {
    echo "🔍 Validating deployment..."
    
    # Check service status
    if ! sudo systemctl is-active --quiet sendense-hub; then
        echo "❌ Sendense Hub service not running"
        sudo systemctl status sendense-hub
        exit 1
    fi
    
    if ! sudo systemctl is-active --quiet volume-daemon; then
        echo "❌ Volume Daemon service not running"
        sudo systemctl status volume-daemon
        exit 1
    fi
    
    # Check API health
    sleep 10
    if ! curl -f http://localhost:8082/health > /dev/null 2>&1; then
        echo "❌ Sendense Hub API health check failed"
        curl http://localhost:8082/health || true
        exit 1
    fi
    
    if ! curl -f http://localhost:8090/api/v1/health > /dev/null 2>&1; then
        echo "❌ Volume Daemon health check failed"  
        curl http://localhost:8090/api/v1/health || true
        exit 1
    fi
    
    echo "✅ Deployment validation passed"
}

# Main deployment flow
main() {
    echo "🎯 Starting Sendense Hub Appliance (SHA) Deployment"
    echo "📅 Date: $(date)"
    echo "🖥️ Host: $(hostname)"
    echo "👤 User: $(whoami)"
    
    validate_system
    validate_binaries
    install_sendense_hub
    configure_services
    validate_deployment
    
    echo ""
    echo "🎉 Sendense Hub Appliance deployment completed successfully!"
    echo ""
    echo "📊 Service Status:"
    echo "   • Sendense Hub: $(sudo systemctl is-active sendense-hub)"
    echo "   • Volume Daemon: $(sudo systemctl is-active volume-daemon)"
    echo ""
    echo "🌐 Access URLs:"
    echo "   • Sendense Cockpit GUI: http://$(hostname -I | awk '{print $1}'):3001"
    echo "   • SHA API: http://$(hostname -I | awk '{print $1}'):8082/health"
    echo "   • Volume Daemon API: http://$(hostname -I | awk '{print $1}'):8090/api/v1/health"
    echo ""
    echo "📚 Next Steps:"
    echo "   1. Configure platform connections (VMware vCenter, CloudStack)"
    echo "   2. Deploy Sendense Node Appliances (SNA) to source platforms"
    echo "   3. Test backup operations"
    echo ""
    echo "📖 Documentation:"
    echo "   • Admin Guide: /opt/sendense/docs/admin-guide/"
    echo "   • API Reference: /opt/sendense/docs/api-reference/"
    echo "   • Troubleshooting: /opt/sendense/docs/troubleshooting/"
}

# Execute main deployment
main "$@"

