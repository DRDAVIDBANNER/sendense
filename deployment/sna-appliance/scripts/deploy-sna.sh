#!/bin/bash
# deploy-sna.sh - Sendense Node Appliance Deployment Script
# Version: v2.1.5
# Last Updated: 2025-10-04
# Compatible with: SNA-VMware v2.1.5, SNA-CloudStack v1.0.3
# Tested on: Ubuntu 22.04 LTS, RHEL 9

set -euo pipefail

SCRIPT_VERSION="v2.1.5"
PLATFORM="${1:-}"
SHA_HOST="${2:-}"

if [[ -z "$PLATFORM" || -z "$SHA_HOST" ]]; then
    echo "Usage: $0 <platform> <sha_host>"
    echo "Platforms: vmware, cloudstack, hyperv, aws, azure, nutanix"
    echo "Example: $0 vmware 10.0.1.100"
    exit 1
fi

echo "🚀 Sendense Node Appliance (SNA) Deployment Script ${SCRIPT_VERSION}"
echo "🌐 Platform: $PLATFORM"
echo "🎯 SHA Host: $SHA_HOST"

# Platform-specific binary selection
case $PLATFORM in
    "vmware")
        SNA_BINARY="sendense-node-vmware-v2.1.5-linux-amd64-20251004-ghi789cd"
        REQUIRED_DEPS=("vddk-libs" "nbdkit" "ssh-client")
        ;;
    "cloudstack")
        SNA_BINARY="sendense-node-cloudstack-v1.0.3-linux-amd64-20251004-jkl012ef"
        REQUIRED_DEPS=("libvirt-clients" "qemu-utils" "ssh-client")
        ;;
    "hyperv")
        echo "❌ Hyper-V SNA deployment not yet implemented (Phase 5B)"
        echo "   Reference: project-goals/modules/06-hyperv-source.md"
        exit 1
        ;;
    *)
        echo "❌ Unsupported platform: $PLATFORM"
        echo "   Available: vmware, cloudstack"
        echo "   Planned: hyperv, aws, azure, nutanix (see project-goals/)"
        exit 1
        ;;
esac

validate_platform_requirements() {
    echo "🔍 Validating $PLATFORM platform requirements..."
    
    case $PLATFORM in
        "vmware")
            # Check for VDDK dependencies
            if [[ ! -d "../dependencies/vddk-libs" ]]; then
                echo "❌ VMware VDDK libraries not found"
                echo "   Required for VMware platform access"
                echo "   Please install VDDK in dependencies/vddk-libs/"
                exit 1
            fi
            ;;
        "cloudstack") 
            # Check for libvirt access
            if ! command -v virsh &> /dev/null; then
                echo "❌ libvirt tools not found"
                echo "   Required for CloudStack KVM access"
                echo "   Install: apt install libvirt-clients qemu-utils"
                exit 1
            fi
            ;;
    esac
    
    echo "✅ Platform requirements validated"
}

install_sna() {
    echo "📦 Installing Sendense Node Appliance ($PLATFORM)..."
    
    # Create directories
    sudo mkdir -p /opt/sendense/{bin,config,ssh,temp}
    
    # Install binary
    sudo cp "../binaries/$SNA_BINARY" /usr/local/bin/sendense-node
    sudo chmod +x /usr/local/bin/sendense-node
    
    # Install platform-specific configuration
    sudo cp "../configs/platform-configs/${PLATFORM}-config.yaml" /opt/sendense/config/
    
    # Install service file
    sudo cp "../configs/sendense-node.service" /etc/systemd/system/
    
    # Setup SSH tunnel keys
    if [[ ! -f "/opt/sendense/ssh/tunnel_key" ]]; then
        ssh-keygen -t ed25519 -f /opt/sendense/ssh/tunnel_key -N '' -C "sendense-node@$(hostname)"
        echo "🔑 SSH tunnel key generated: /opt/sendense/ssh/tunnel_key.pub"
        echo "   Add this public key to SHA at: $SHA_HOST"
        cat /opt/sendense/ssh/tunnel_key.pub
    fi
    
    echo "✅ SNA installed for $PLATFORM platform"
}

configure_connection_to_sha() {
    echo "🔗 Configuring connection to Sendense Hub Appliance..."
    
    # Update configuration with SHA details
    sudo tee /opt/sendense/config/connection.yaml > /dev/null <<EOF
sha:
  host: "$SHA_HOST"
  port: 443
  ssh_key: "/opt/sendense/ssh/tunnel_key"
  api_endpoint: "https://$SHA_HOST:8082"
  
platform:
  type: "$PLATFORM"
  config_file: "/opt/sendense/config/${PLATFORM}-config.yaml"
  
performance:
  max_concurrent_jobs: 5
  bandwidth_limit_mbps: 1000
  backup_window: "22:00-06:00"
EOF
    
    # Test connection to SHA
    if ! ssh -i /opt/sendense/ssh/tunnel_key -o ConnectTimeout=5 sendense@$SHA_HOST exit 2>/dev/null; then
        echo "⚠️ Warning: Cannot connect to SHA at $SHA_HOST"
        echo "   Ensure public key is added to SHA authorized_keys"
        echo "   Public key: /opt/sendense/ssh/tunnel_key.pub"
    else
        echo "✅ SSH connection to SHA verified"
    fi
}

start_services() {
    echo "🚀 Starting Sendense Node services..."
    
    sudo systemctl daemon-reload
    sudo systemctl enable sendense-node
    sudo systemctl start sendense-node
    
    # Wait for service startup
    sleep 10
    
    # Verify service health
    if ! sudo systemctl is-active --quiet sendense-node; then
        echo "❌ Sendense Node service failed to start"
        sudo systemctl status sendense-node
        exit 1
    fi
    
    echo "✅ Sendense Node Appliance operational"
}

# Main deployment function
main() {
    echo "🎯 Starting Sendense Node Appliance (SNA) Deployment"
    echo "📅 Date: $(date)"
    echo "🖥️ Host: $(hostname)"
    echo "🌐 Platform: $PLATFORM"
    echo "🎯 SHA Target: $SHA_HOST"
    
    validate_platform_requirements
    install_sna
    configure_connection_to_sha
    start_services
    
    echo ""
    echo "🎉 Sendense Node Appliance ($PLATFORM) deployment completed!"
    echo ""
    echo "📊 Service Status:"
    echo "   • Sendense Node: $(sudo systemctl is-active sendense-node)"
    echo ""
    echo "🔗 Connection:"
    echo "   • SHA Host: $SHA_HOST:443 (SSH tunnel)"
    echo "   • Platform: $PLATFORM"
    echo ""
    echo "📚 Next Steps:"
    echo "   1. Configure $PLATFORM credentials on SHA"
    echo "   2. Test VM discovery from $PLATFORM" 
    echo "   3. Run first backup operation"
    echo ""
    echo "🔧 Management:"
    echo "   • Status: systemctl status sendense-node"
    echo "   • Logs: journalctl -u sendense-node -f"
    echo "   • Config: /opt/sendense/config/"
}

# Execute deployment
main "$@"
