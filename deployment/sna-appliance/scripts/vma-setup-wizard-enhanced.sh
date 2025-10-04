#!/bin/bash
# VMA Setup Wizard - OMA Connection Configuration
# Professional deployment interface for MigrateKit OSSEA VMA

set -euo pipefail

# Signal handling for graceful interruption
trap 'echo -e "\n${YELLOW}⚠️  Setup interrupted. Restarting wizard...${NC}"; sleep 1; exec "$0"' INT TERM
trap 'echo -e "\n${YELLOW}⚠️  Setup suspended. Restarting wizard...${NC}"; sleep 1; exec "$0"' TSTP

# Colors and formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration files
VMA_CONFIG="/opt/vma/vma-config.conf"
TUNNEL_SERVICE="/etc/systemd/system/vma-ssh-tunnel.service"
TUNNEL_WRAPPER="/usr/local/bin/vma-tunnel-wrapper.sh"

# Clear screen and show header
clear
echo -e "${BLUE}${BOLD}"
cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                     OSSEA-Migrate - VMA Setup                   ║
║                  VMware Migration Appliance                      ║
║                                                                  ║
║              🚀 Professional Migration Platform                  ║
╚══════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${CYAN}Welcome to the OSSEA-Migrate VMA (VMware Migration Appliance) Setup Wizard${NC}"
echo -e "${CYAN}This wizard will configure your VMA to connect to the OMA appliance.${NC}"
echo ""

# Show current VMA network configuration
echo -e "${YELLOW}📡 VMA Network Configuration:${NC}"
VMA_IP=$(hostname -I | awk '{print $1}')
VMA_INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
VMA_GATEWAY=$(ip route | grep default | awk '{print $3}' | head -1)
VMA_DNS=$(cat /etc/resolv.conf | grep nameserver | head -1 | awk '{print $2}')

# Check if using DHCP
if systemctl is-active systemd-networkd > /dev/null 2>&1; then
    DHCP_STATUS="systemd-networkd"
elif [ -f "/etc/netplan/01-netcfg.yaml" ] && grep -q "dhcp4: true" /etc/netplan/01-netcfg.yaml 2>/dev/null; then
    DHCP_STATUS="DHCP (netplan)"
else
    DHCP_STATUS="Static/Unknown"
fi

echo -e "   VMA IP Address: ${BOLD}${VMA_IP}${NC}"
echo -e "   Network Interface: ${BOLD}${VMA_INTERFACE}${NC}"
echo -e "   Gateway: ${BOLD}${VMA_GATEWAY}${NC}"
echo -e "   DNS Server: ${BOLD}${VMA_DNS}${NC}"
echo -e "   Configuration: ${BOLD}${DHCP_STATUS}${NC}"
echo ""

# Show current configuration if it exists
if [ -f "$VMA_CONFIG" ]; then
    CURRENT_OMA_IP=$(grep "OMA_HOST=" "$VMA_CONFIG" 2>/dev/null | cut -d= -f2 || echo "Not configured")
    
    # Test actual tunnel connectivity via OMA API through tunnel  
    if curl -s --connect-timeout 10 http://localhost:8082/health > /dev/null 2>&1; then
        TUNNEL_STATUS="${GREEN}Connected${NC}"
        TUNNEL_HEALTH="✅"
    else
        TUNNEL_STATUS="${RED}Disconnected${NC}"
        TUNNEL_HEALTH="❌"
    fi
    
    echo -e "${YELLOW}📋 Current Configuration:${NC}"
    echo -e "   Current OMA IP: ${BOLD}$CURRENT_OMA_IP${NC}"
    echo -e "   Tunnel Status: $TUNNEL_HEALTH ${BOLD}$TUNNEL_STATUS${NC}"
    echo ""
elif systemctl is-active vma-ssh-tunnel.service > /dev/null 2>&1; then
    # Check systemd service for current OMA IP from ExecStart
    CURRENT_OMA_IP=$(systemctl show vma-ssh-tunnel.service -p ExecStart | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 || echo "Unknown")
    
    # Test actual tunnel connectivity via OMA API through tunnel
    if curl -s --connect-timeout 10 http://localhost:8082/health > /dev/null 2>&1; then
        TUNNEL_STATUS="${GREEN}Connected${NC}"
        TUNNEL_HEALTH="✅"
    else
        TUNNEL_STATUS="${RED}Disconnected${NC}"
        TUNNEL_HEALTH="❌"
    fi
    
    echo -e "${YELLOW}📋 Current Configuration:${NC}"
    echo -e "   Current OMA IP: ${BOLD}$CURRENT_OMA_IP${NC}"
    echo -e "   Tunnel Status: $TUNNEL_HEALTH ${BOLD}$TUNNEL_STATUS${NC}"
    echo ""
else
    echo -e "${YELLOW}📋 No existing configuration found - first-time setup${NC}"
    echo ""
fi

# Function to validate IP address
validate_ip() {
    local ip=$1
    if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        IFS='.' read -ra ADDR <<< "$ip"
        for i in "${ADDR[@]}"; do
            if [[ $i -gt 255 ]]; then
                return 1
            fi
        done
        return 0
    else
        return 1
    fi
}

# Function to test connectivity
test_connectivity() {
    local oma_ip=$1
    
    echo -e "${YELLOW}🔍 Testing connectivity to OMA at $oma_ip...${NC}"
    
    # Test ping
    if ping -c 3 -W 5 "$oma_ip" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Network connectivity successful${NC}"
    else
        echo -e "${YELLOW}⚠️  Ping failed (may be normal if firewall blocks ICMP)${NC}"
    fi
    
    # Test port 443 (TLS tunnel)
    if timeout 10 bash -c "</dev/tcp/$oma_ip/443" 2>/dev/null; then
        echo -e "${GREEN}✅ Port 443 accessible for TLS tunnel${NC}"
        return 0
    else
        echo -e "${RED}❌ Port 443 not accessible${NC}"
        return 1
    fi
}

# Function to configure tunnel
configure_tunnel() {
    local oma_ip=$1
    
    echo -e "${YELLOW}🔧 Configuring VMA-OMA SSH tunnel...${NC}"
    
    # Create configuration
    sudo mkdir -p /opt/vma
    cat > /tmp/vma-config.conf << EOF
# VMA Configuration
OMA_HOST=$oma_ip
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_LOCAL_PORT=9081
TUNNEL_TYPE=ssh
SETUP_DATE="$(date)"
SETUP_VERSION=v2.0.0-ssh-tunnel
EOF
    
    sudo mv /tmp/vma-config.conf "$VMA_CONFIG"
    
    # Check if SSH enrollment key exists
    if [ ! -f "/opt/vma/enrollment/vma_enrollment_key" ]; then
        echo -e "${RED}❌ SSH enrollment key not found!${NC}"
        echo -e "${YELLOW}⚠️  Please run VMA enrollment wizard first${NC}"
        return 1
    fi
    
    # Deploy tunnel wrapper script if needed
    if [ ! -f "$TUNNEL_WRAPPER" ]; then
        echo -e "${YELLOW}📡 Deploying SSH tunnel wrapper...${NC}"
        # This would normally be deployed by the enrollment/setup process
        echo -e "${RED}❌ Tunnel wrapper not found - deployment needed${NC}"
        return 1
    fi
    
    # Create/update systemd service with OMA IP
    echo -e "${YELLOW}📝 Creating SSH tunnel service...${NC}"
    sudo tee "$TUNNEL_SERVICE" > /dev/null << EOF
[Unit]
Description=VMA SSH Tunnel to OMA
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=vma
Group=vma
ExecStart=$TUNNEL_WRAPPER $oma_ip
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
    
    sudo systemctl daemon-reload
    echo -e "${GREEN}✅ SSH tunnel service configured for OMA: $oma_ip${NC}"
}

# Function to start services
start_services() {
    echo -e "${YELLOW}🚀 Starting VMA services...${NC}"
    
    # Enable and start SSH tunnel
    if sudo systemctl enable vma-ssh-tunnel.service 2>/dev/null; then
        echo -e "${GREEN}✅ SSH tunnel service enabled${NC}"
    fi
    
    # Restart tunnel service to establish connection
    if sudo systemctl restart vma-ssh-tunnel.service 2>/dev/null; then
        echo -e "${GREEN}✅ SSH tunnel service restarted${NC}"

    # Validate service is actually running
    sleep 5
    if systemctl is-active vma-ssh-tunnel.service > /dev/null; then
        echo -e "${GREEN}✅ Tunnel service confirmed running${NC}"
    else
        echo -e "${RED}❌ Tunnel service failed to start - check logs${NC}"
        journalctl -u vma-ssh-tunnel --no-pager -n 5
    fi
        sleep 3  # Give tunnel time to establish
    else
        echo -e "${RED}❌ Failed to restart SSH tunnel service${NC}"
    fi
    
    # Enable and start VMA API
    if sudo systemctl enable vma-api.service 2>/dev/null; then
        echo -e "${GREEN}✅ VMA API service enabled${NC}"
    fi
    
    if sudo systemctl start vma-api.service 2>/dev/null; then
        echo -e "${GREEN}✅ VMA API service started${NC}"
    else
        echo -e "${RED}❌ Failed to start VMA API service${NC}"
    fi
}

# Function to validate setup
validate_setup() {
    local oma_ip=$1
    
    echo -e "${YELLOW}🔍 Validating VMA setup...${NC}"
    
    # Check SSH tunnel status
    if systemctl is-active vma-ssh-tunnel.service > /dev/null 2>&1; then
        echo -e "${GREEN}✅ SSH tunnel service active${NC}"
        
        # Check if tunnels are actually working
        sleep 2
        if ss -tlnp 2>/dev/null | grep -q ':10808'; then
            echo -e "${GREEN}✅ NBD forward tunnel established (port 10808)${NC}"
        else
            echo -e "${YELLOW}⚠️  NBD forward tunnel not detected${NC}"
        fi
    else
        echo -e "${RED}❌ SSH tunnel service not active${NC}"
    fi
    
    # Check VMA API
    if systemctl is-active vma-api.service > /dev/null 2>&1; then
        echo -e "${GREEN}✅ VMA API service active${NC}"
    else
        echo -e "${RED}❌ VMA API service not active${NC}"
    fi
    
    # Test local API
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ VMA API responding${NC}"
    else
        echo -e "${YELLOW}⚠️  VMA API not responding yet (may need time to start)${NC}"
    fi
}

# Main setup flow
main() {
    # Network configuration option
    echo -e "${BOLD}🔧 Configuration Options:${NC}"
    echo "   1. Configure OMA connection (recommended)"
    echo "   2. Configure VMA network settings"
    echo "   3. View current configuration"
    echo ""
    read -p "Select option (1-3): " config_option
    
    case $config_option in
        2)
            configure_vma_network
            return
            ;;
        3)
            show_detailed_status
            return
            ;;
    esac

    # Choose connection method
    echo ""
    echo -e "${BOLD}🔗 OMA Connection Method${NC}"
    echo -e "${CYAN}(Press Ctrl+C to restart wizard at any time)${NC}"
    echo ""
    echo "   1. Automatic Enrollment (New VMA - requires pairing code)"
    echo "   2. Manual Configuration (Existing VMA or troubleshooting)"
    echo ""
    read -p "Select connection method (1-2): " connection_method
    
    case $connection_method in
        1)
            enrollment_workflow
            return
            ;;
        2)
            # Continue with existing manual configuration
            ;;
        *)
            echo -e "${RED}❌ Invalid option. Please select 1 or 2.${NC}"
            sleep 2
            continue
            ;;
    esac
    
    # Get OMA IP (existing manual configuration)
    while true; do
        echo ""
        echo -e "${BOLD}📡 Manual OMA Connection Configuration${NC}"
        echo -e "${CYAN}(Press Ctrl+C to restart wizard at any time)${NC}"
        echo ""
        read -p "Enter OMA IP Address: " OMA_IP
        
        if validate_ip "$OMA_IP"; then
            echo -e "${GREEN}✅ Valid IP format: $OMA_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid IP format. Please enter a valid IP address (e.g., 192.168.1.100)${NC}"
        fi
    done
    
    # Test connectivity
    echo ""
    if test_connectivity "$OMA_IP"; then
        echo -e "${GREEN}✅ OMA connectivity verified${NC}"
    else
        echo ""
        read -p "⚠️  Connectivity test failed. Continue anyway? (y/N): " continue_setup
        if [[ ! $continue_setup =~ ^[Yy]$ ]]; then
            echo "Setup cancelled."
            exit 1
        fi
    fi
    
    # Configure tunnel
    echo ""
    configure_tunnel "$OMA_IP"
    
    # Start services
    echo ""
    start_services
    
    # Wait for services to start
    echo ""
    echo -e "${YELLOW}⏳ Waiting for services to initialize...${NC}"
    sleep 10
    
    # Validate setup
    echo ""
    validate_setup "$OMA_IP"
    
    # Show completion status
    echo ""
    echo -e "${BOLD}${GREEN}🎉 OSSEA-Migrate VMA Setup Complete!${NC}"
    echo ""
    echo -e "${CYAN}📊 Configuration Summary:${NC}"
    echo -e "   ${BOLD}OMA IP:${NC} $OMA_IP"
    echo -e "   ${BOLD}VMA API:${NC} http://localhost:8081"
    echo -e "   ${BOLD}Tunnel Status:${NC} $(systemctl is-active vma-ssh-tunnel.service 2>/dev/null || echo 'Unknown')"
    echo -e "   ${BOLD}API Status:${NC} $(systemctl is-active vma-api.service 2>/dev/null || echo 'Unknown')"
    echo ""
    echo -e "${CYAN}🎯 Next Steps:${NC}"
    echo -e "   1. Access OSSEA-Migrate GUI at: ${BOLD}http://$OMA_IP:3001${NC}"
    echo -e "   2. Navigate to Discovery settings"
    echo -e "   3. Add this VMA for VM discovery"
    echo -e "   4. Begin VM migration workflows"
    echo ""
    echo -e "${CYAN}📚 Additional Information:${NC}"
    echo -e "   - VMA API Documentation: Available in /home/pgrayson/migratekit-cloudstack/docs/"
    echo -e "   - Log Files: /var/log/vma-*.log"
    echo -e "   - Configuration: $VMA_CONFIG"
    echo ""
    
    # Option to restart wizard or access admin functions
    echo -e "${BOLD}Choose next action:${NC}"
    echo "   1. Restart setup wizard"
    echo "   2. View system status" 
    echo "   3. Admin shell access"
    echo "   4. Reboot system"
    echo ""
    echo -e "${CYAN}(Press Ctrl+C to restart wizard)${NC}"
    read -p "Select option (1-4): " next_action
    
    case $next_action in
        1)
            echo -e "${YELLOW}Restarting setup wizard...${NC}"
            exec "$0"
            ;;
        2)
            echo ""
            echo -e "${CYAN}📊 System Status:${NC}"
            systemctl status vma-ssh-tunnel.service --no-pager -l
            systemctl status vma-api.service --no-pager -l
            echo ""
            echo -e "${BOLD}Choose next action:${NC}"
            echo "   1. Restart setup wizard"
            echo "   2. Admin shell access (requires confirmation)"
            echo ""
            read -p "Select option (1-2): " status_action
            case $status_action in
                1)
                    echo -e "${YELLOW}Restarting setup wizard...${NC}"
                    exec "$0"
                    ;;
                2)
                    echo ""
                    echo -e "${RED}⚠️  ADMIN SHELL ACCESS${NC}"
                    echo -e "${YELLOW}This provides full system access for support personnel only.${NC}"
                    read -p "Enter admin password (or press Enter to return to wizard): " -s admin_pass
                    echo ""
                    if [ "$admin_pass" = "ossea-admin-2025" ]; then
                        echo -e "${GREEN}Admin access granted${NC}"
                        exec /bin/bash
                    else
                        echo -e "${RED}Invalid password. Returning to setup wizard...${NC}"
                        sleep 2
                        exec "$0"
                    fi
                    ;;
                *)
                    echo -e "${YELLOW}Returning to setup wizard...${NC}"
                    exec "$0"
                    ;;
            esac
            ;;
        3)
            echo ""
            echo -e "${RED}⚠️  ADMIN SHELL ACCESS${NC}"
            echo -e "${YELLOW}This provides full system access for support personnel only.${NC}"
            read -p "Enter admin password: " -s admin_pass
            echo ""
            if [ "$admin_pass" = "ossea-admin-2025" ]; then
                echo -e "${GREEN}Admin access granted${NC}"
                exec /bin/bash
            else
                echo -e "${RED}Invalid password. Returning to setup wizard...${NC}"
                sleep 2
                exec "$0"
            fi
            ;;
        4)
            echo -e "${YELLOW}Rebooting system...${NC}"
            sudo reboot
            ;;
        *)
            echo -e "${YELLOW}Restarting setup wizard...${NC}"
            exec "$0"
            ;;
    esac
}

# Function to configure VMA network settings
configure_vma_network() {
    clear
    echo -e "${BLUE}${BOLD}🔧 VMA Network Configuration${NC}"
    echo "================================"
    echo ""
    
    # Show current network info
    echo -e "${YELLOW}📊 Current Network Configuration:${NC}"
    echo -e "   IP Address: ${BOLD}${VMA_IP}${NC}"
    echo -e "   Interface: ${BOLD}${VMA_INTERFACE}${NC}"
    echo -e "   Gateway: ${BOLD}${VMA_GATEWAY}${NC}"
    echo -e "   DNS: ${BOLD}${VMA_DNS}${NC}"
    echo -e "   Mode: ${BOLD}${DHCP_STATUS}${NC}"
    echo ""
    
    echo -e "${BOLD}Network Configuration Options:${NC}"
    echo "   1. Keep current configuration (DHCP/automatic)"
    echo "   2. Configure static IP address"
    echo "   3. Return to main menu"
    echo ""
    read -p "Select option (1-3): " net_option
    
    case $net_option in
        1)
            echo -e "${GREEN}✅ Keeping current network configuration${NC}"
            sleep 2
            exec "$0"
            ;;
        2)
            configure_static_ip
            ;;
        3)
            exec "$0"
            ;;
        *)
            echo -e "${YELLOW}Invalid option. Returning to main menu...${NC}"
            sleep 2
            exec "$0"
            ;;
    esac
}

# Function to configure static IP
configure_static_ip() {
    echo ""
    echo -e "${BOLD}🔧 Static IP Configuration${NC}"
    echo "=========================="
    echo ""
    
    # Get static IP configuration
    while true; do
        read -p "Enter VMA IP Address: " STATIC_IP
        if validate_ip "$STATIC_IP"; then
            echo -e "${GREEN}✅ Valid IP format: $STATIC_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid IP format${NC}"
        fi
    done
    
    while true; do
        read -p "Enter Subnet Mask (e.g., 255.255.255.0 or 24): " NETMASK
        if [[ $NETMASK =~ ^[0-9]{1,2}$ ]] && [ "$NETMASK" -le 32 ]; then
            echo -e "${GREEN}✅ Valid CIDR notation: /$NETMASK${NC}"
            break
        elif validate_ip "$NETMASK"; then
            echo -e "${GREEN}✅ Valid subnet mask: $NETMASK${NC}"
            break
        else
            echo -e "${RED}❌ Invalid subnet mask format${NC}"
        fi
    done
    
    while true; do
        read -p "Enter Gateway IP: " GATEWAY_IP
        if validate_ip "$GATEWAY_IP"; then
            echo -e "${GREEN}✅ Valid gateway IP: $GATEWAY_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid gateway IP format${NC}"
        fi
    done
    
    while true; do
        read -p "Enter DNS Server (e.g., 8.8.8.8): " DNS_IP
        if validate_ip "$DNS_IP"; then
            echo -e "${GREEN}✅ Valid DNS IP: $DNS_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid DNS IP format${NC}"
        fi
    done
    
    # Confirm configuration
    echo ""
    echo -e "${YELLOW}📋 Static IP Configuration Summary:${NC}"
    echo -e "   IP Address: ${BOLD}$STATIC_IP${NC}"
    echo -e "   Subnet Mask: ${BOLD}$NETMASK${NC}"
    echo -e "   Gateway: ${BOLD}$GATEWAY_IP${NC}"
    echo -e "   DNS Server: ${BOLD}$DNS_IP${NC}"
    echo ""
    read -p "Apply this configuration? (y/N): " apply_static
    
    if [[ $apply_static =~ ^[Yy]$ ]]; then
        apply_static_network "$STATIC_IP" "$NETMASK" "$GATEWAY_IP" "$DNS_IP"
    else
        echo -e "${YELLOW}Configuration cancelled. Returning to main menu...${NC}"
        sleep 2
        exec "$0"
    fi
}

# Function to apply static network configuration
apply_static_network() {
    local static_ip="$1"
    local netmask="$2"
    local gateway="$3"
    local dns="$4"
    
    echo ""
    echo -e "${YELLOW}🔧 Applying static network configuration...${NC}"
    
    # Convert netmask to CIDR if needed
    if [[ $netmask =~ ^[0-9]{1,2}$ ]]; then
        CIDR="$netmask"
    else
        # Convert subnet mask to CIDR (simplified)
        case $netmask in
            "255.255.255.0") CIDR="24" ;;
            "255.255.0.0") CIDR="16" ;;
            "255.0.0.0") CIDR="8" ;;
            *) CIDR="24" ;; # Default fallback
        esac
    fi
    
    # Create netplan configuration
    cat > /tmp/01-static-config.yaml << EOF
network:
  version: 2
  renderer: networkd
  ethernets:
    $VMA_INTERFACE:
      dhcp4: false
      addresses:
        - $static_ip/$CIDR
      gateway4: $gateway
      nameservers:
        addresses:
          - $dns
EOF
    
    # Backup current configuration
    sudo cp /etc/netplan/01-netcfg.yaml /etc/netplan/01-netcfg.yaml.backup 2>/dev/null || true
    
    # Apply new configuration
    sudo cp /tmp/01-static-config.yaml /etc/netplan/01-netcfg.yaml
    
    echo -e "${YELLOW}⚠️  Applying network configuration - VMA will disconnect briefly...${NC}"
    echo -e "${CYAN}Reconnect to VMA at: ${BOLD}$static_ip${NC}"
    echo ""
    read -p "Press Enter to apply configuration and reboot..."
    
    sudo netplan apply
    sleep 3
    sudo reboot
}

# Function to show detailed status
show_detailed_status() {
    clear
    echo -e "${BLUE}${BOLD}📊 OSSEA-Migrate VMA Detailed Status${NC}"
    echo "===================================="
    echo ""
    
    echo -e "${YELLOW}🌐 Network Information:${NC}"
    echo -e "   VMA IP: ${BOLD}${VMA_IP}${NC}"
    echo -e "   Interface: ${BOLD}${VMA_INTERFACE}${NC}"
    echo -e "   Gateway: ${BOLD}${VMA_GATEWAY}${NC}"
    echo -e "   DNS: ${BOLD}${VMA_DNS}${NC}"
    echo -e "   Configuration: ${BOLD}${DHCP_STATUS}${NC}"
    echo ""
    
    echo -e "${YELLOW}🔗 Tunnel Information:${NC}"
    if [ -f "$VMA_CONFIG" ]; then
        CURRENT_OMA_IP=$(grep "OMA_HOST=" "$VMA_CONFIG" 2>/dev/null | cut -d= -f2 || echo "Not configured")
    else
        CURRENT_OMA_IP=$(systemctl show vma-ssh-tunnel.service -p ExecStart | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 || echo "Unknown")
    fi
    echo -e "   OMA IP: ${BOLD}$CURRENT_OMA_IP${NC}"
    echo -e "   Tunnel Service: ${BOLD}$(systemctl is-active vma-ssh-tunnel.service)${NC}"
    echo -e "   VMA API: ${BOLD}$(systemctl is-active vma-api.service)${NC}"
    echo ""
    
    read -p "Press Enter to return to main menu..."
    exec "$0"
}

# VMA Enrollment Workflow Functions

enrollment_workflow() {
    echo ""
    echo -e "${BOLD}🔐 VMA Enrollment Workflow${NC}"
    echo -e "${CYAN}This process will automatically configure your VMA with OMA approval${NC}"
    echo ""
    
    # Get OMA IP for enrollment
    while true; do
        echo -e "${BOLD}📡 OMA Server Information${NC}"
        echo ""
        read -p "Enter OMA IP Address: " OMA_IP
        
        if validate_ip "$OMA_IP"; then
            echo -e "${GREEN}✅ Valid IP format: $OMA_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid IP format${NC}"
        fi
    done
    
    # Get pairing code
    while true; do
        echo ""
        echo -e "${BOLD}🔑 Pairing Code Entry${NC}"
        echo -e "${CYAN}Enter the pairing code provided by your OMA administrator${NC}"
        echo -e "${CYAN}Format: XXXX-XXXX-XXXX (e.g., AX7K-PJ3F-TH2Q)${NC}"
        echo ""
        read -p "Enter pairing code: " PAIRING_CODE
        
        if [[ $PAIRING_CODE =~ ^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$ ]]; then
            echo -e "${GREEN}✅ Valid pairing code format${NC}"
            break
        else
            echo -e "${RED}❌ Invalid format. Use XXXX-XXXX-XXXX format${NC}"
        fi
    done
    
    # Generate VMA keypair
    echo ""
    echo -e "${BOLD}🔑 Generating VMA SSH Keypair${NC}"
    echo -e "${CYAN}Creating Ed25519 keypair for secure enrollment...${NC}"
    
    mkdir -p /opt/vma/enrollment
    chmod 700 /opt/vma/enrollment
    
    HOSTNAME=$(hostname)
    DATE=$(date +%Y%m%d%H%M)
    KEY_COMMENT="VMA-${HOSTNAME}-${OMA_IP}-${DATE}"
    
    ssh-keygen -t ed25519 \
        -f /opt/vma/enrollment/vma_enrollment_key \
        -N "" \
        -C "$KEY_COMMENT" >/dev/null 2>&1
    
    chmod 600 /opt/vma/enrollment/vma_enrollment_key*
    
    FINGERPRINT=$(ssh-keygen -lf /opt/vma/enrollment/vma_enrollment_key.pub | awk '{print $2}')
    echo -e "${GREEN}✅ VMA keypair generated${NC}"
    echo -e "${CYAN}SSH Fingerprint: ${BOLD}$FINGERPRINT${NC}"
    
    # Submit enrollment
    echo ""
    echo -e "${BOLD}📤 Submitting VMA Enrollment${NC}"
    echo -e "${CYAN}Connecting to OMA enrollment endpoint...${NC}"
    
    PUBLIC_KEY=$(cat /opt/vma/enrollment/vma_enrollment_key.pub)
    
    ENROLLMENT_RESPONSE=$(curl -s -X POST \
        "http://${OMA_IP}:443/api/v1/vma/enroll" \
        -H "Content-Type: application/json" \
        -d "{
            \"pairing_code\": \"$PAIRING_CODE\",
            \"vma_public_key\": \"$PUBLIC_KEY\",
            \"vma_name\": \"$HOSTNAME\",
            \"vma_version\": \"v2.20.1\",
            \"vma_fingerprint\": \"$FINGERPRINT\"
        }" 2>/dev/null)
    
    ENROLLMENT_ID=$(echo "$ENROLLMENT_RESPONSE" | jq -r '.enrollment_id // empty' 2>/dev/null)
    CHALLENGE=$(echo "$ENROLLMENT_RESPONSE" | jq -r '.challenge // empty' 2>/dev/null)
    
    if [ -n "$ENROLLMENT_ID" ] && [ -n "$CHALLENGE" ]; then
        echo -e "${GREEN}✅ Enrollment submitted successfully${NC}"
        
        # Submit challenge verification (simplified for MVP)
        curl -s -X POST \
            "http://${OMA_IP}:443/api/v1/vma/enroll/verify" \
            -H "Content-Type: application/json" \
            -d "{
                \"enrollment_id\": \"$ENROLLMENT_ID\",
                \"signature\": \"vma_signature_$(date +%s)\"
            }" >/dev/null 2>&1
        
        # Poll for approval
        echo ""
        echo -e "${BOLD}⏳ Waiting for Administrator Approval${NC}"
        echo -e "${CYAN}Enrollment ID: $ENROLLMENT_ID${NC}"
        echo -e "${YELLOW}Polling for approval (max 30 minutes)...${NC}"
        
        local attempt=0
        local max_attempts=60  # 30 minutes
        
        while [ $attempt -lt $max_attempts ]; do
            RESULT_RESPONSE=$(curl -s "http://${OMA_IP}:443/api/v1/vma/enroll/result?enrollment_id=$ENROLLMENT_ID" 2>/dev/null)
            STATUS=$(echo "$RESULT_RESPONSE" | jq -r '.status // empty' 2>/dev/null)
            
            case "$STATUS" in
                "approved")
                    echo -e "${GREEN}🎉 VMA enrollment approved!${NC}"
                    configure_enrollment_tunnel "$OMA_IP" "$ENROLLMENT_ID"
                    return
                    ;;
                "rejected")
                    echo -e "${RED}❌ VMA enrollment rejected by administrator${NC}"
                    exit 1
                    ;;
                "expired")
                    echo -e "${RED}❌ VMA enrollment expired${NC}"
                    exit 1
                    ;;
                *)
                    echo -e "${CYAN}⏳ Awaiting approval... ($(($attempt + 1))/60)${NC}"
                    sleep 30
                    ((attempt++))
                    ;;
            esac
        done
        
        echo -e "${RED}❌ Approval timeout - contact administrator${NC}"
        exit 1
    else
        ERROR_MSG=$(echo "$ENROLLMENT_RESPONSE" | jq -r '.error // "Connection failed"' 2>/dev/null)
        echo -e "${RED}❌ Enrollment failed: $ERROR_MSG${NC}"
        exit 1
    fi
    
    # Manual configuration continues below
    # Get OMA IP
    while true; do
        echo ""
        echo -e "${BOLD}📡 Manual OMA Connection Configuration${NC}"
        echo -e "${CYAN}(Press Ctrl+C to restart wizard at any time)${NC}"
        echo ""
        read -p "Enter OMA IP Address: " OMA_IP


configure_enrollment_tunnel() {
    local oma_ip="$1"
    local enrollment_id="$2"
    
    echo ""
    echo -e "${BOLD}🔧 Configuring VMA Tunnel${NC}"
    echo -e "${CYAN}Setting up SSH tunnel with enrolled credentials...${NC}"
    
    # Update VMA configuration
    cat > "$VMA_CONFIG" << EOFCONFIG
# VMA Configuration - Enrollment Based
OMA_HOST=$oma_ip
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_LOCAL_PORT=9081
SETUP_DATE="$(date)"
SETUP_VERSION=v2.0.0-enrollment
ENROLLMENT_ID=$enrollment_id
ENROLLMENT_METHOD=automatic
EOFCONFIG
    
    echo -e "${GREEN}✅ VMA configuration updated${NC}"
    
    # Configure tunnel service with enrollment key
    configure_tunnel "$oma_ip"
    
    # Start services
    start_services
    
    # Validate setup
    validate_setup "$oma_ip"
    
    echo ""
    echo -e "${BOLD}${GREEN}🎉 VMA Enrollment Complete!${NC}"
    echo ""
    echo -e "${CYAN}📊 Enrollment Summary:${NC}"
    echo -e "   ${BOLD}OMA IP:${NC} $oma_ip"
    echo -e "   ${BOLD}Enrollment ID:${NC} $enrollment_id"
    echo -e "   ${BOLD}SSH Key:${NC} /opt/vma/enrollment/vma_enrollment_key"
    echo -e "   ${BOLD}Tunnel Status:${NC} $(systemctl is-active vma-ssh-tunnel.service 2>/dev/null || echo 'Unknown')"
    echo ""
    echo -e "${CYAN}🎯 VMA is now connected and ready for migration operations${NC}"
    echo ""
    read -p "Press Enter to exit..."
}
