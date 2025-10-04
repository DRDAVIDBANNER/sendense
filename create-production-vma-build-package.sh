#!/bin/bash
# Production VMA Build Package Creator
# Creates complete VMA appliance build package with latest binaries and enhanced wizard

set -euo pipefail

BUILD_DIR="/tmp/production-vma-build"
STABLE_MIGRATEKIT="/home/pgrayson/migratekit-cloudstack/source/current/migratekit/migratekit-v2.20.1-chunk-size-fix"
STABLE_VMA_API="/home/pgrayson/migratekit-cloudstack/source/current/vma-api-server/vma-api-server-v1.10.4-progress-fixed"

echo "📦 Creating Production VMA Build Package"
echo "========================================"

# Verify stable binaries exist
if [ ! -f "$STABLE_MIGRATEKIT" ]; then
    echo "❌ Stable migratekit binary not found: $STABLE_MIGRATEKIT"
    echo "🔍 Available migratekit binaries:"
    find /home/pgrayson/migratekit-cloudstack -name "migratekit-v*" -type f | head -5
    exit 1
fi

if [ ! -f "$STABLE_VMA_API" ]; then
    echo "❌ Stable VMA API binary not found: $STABLE_VMA_API"
    echo "🔍 Available VMA API binaries:"
    find /home/pgrayson/migratekit-cloudstack -name "vma-api-server-v*" -type f | head -5
    exit 1
fi

# Clean and create build directory
sudo rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"/{binaries,services,scripts,config}

# PHASE 1: STABLE PRODUCTION BINARIES
echo "📋 Collecting STABLE production binaries..."

# Use latest stable migratekit binary
cp "$STABLE_MIGRATEKIT" "$BUILD_DIR/binaries/migratekit"
chmod +x "$BUILD_DIR/binaries/migratekit"

# Use latest stable VMA API binary  
cp "$STABLE_VMA_API" "$BUILD_DIR/binaries/vma-api-server"
chmod +x "$BUILD_DIR/binaries/vma-api-server"

echo "✅ STABLE binaries collected"

# PHASE 2: ENHANCED VMA WIZARD
echo "🎨 Creating enhanced VMA setup wizard..."

# Copy the production VMA wizard
cp vma-setup-wizard-production.sh "$BUILD_DIR/scripts/vma-setup-wizard.sh"
chmod +x "$BUILD_DIR/scripts/vma-setup-wizard.sh"

echo "✅ Enhanced VMA wizard created"

# Create a backup basic wizard for reference
cat > "$BUILD_DIR/scripts/vma-setup-wizard-basic.sh" << 'BASIC_EOF'
#!/bin/bash
# Basic VMA Setup Wizard - Simple OMA Connection Configuration

set -euo pipefail

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
TUNNEL_SERVICE="/etc/systemd/system/vma-tunnel-enhanced-v2.service"
VENDOR_ACCESS_FILE="/opt/vma/.vendor-access"

# Enhanced signal handling with TTY recovery
restart_wizard() {
    echo -e "\n${YELLOW}⚠️  Wizard interrupted. Recovering TTY and restarting...${NC}"
    
    # Reset TTY to sane state
    if [ -t 0 ]; then
        stty sane 2>/dev/null || true
        stty echo 2>/dev/null || true
        stty icanon 2>/dev/null || true
        reset 2>/dev/null || true
    fi
    
    # Clear screen and restart
    clear 2>/dev/null || true
    sleep 1
    exec "$0"
}

trap 'restart_wizard' INT TERM QUIT HUP TSTP STOP

# Disable job control to prevent backgrounding
set +m

# Initialize TTY to proper state
if [ -t 0 ]; then
    stty sane 2>/dev/null || true
    stty echo 2>/dev/null || true
    stty icanon 2>/dev/null || true
fi

# Super admin access control
check_vendor_access() {
    local password="$1"
    local vendor_hash="7c4a8d09ca3762af61e59520943dc26494f8941b"  # SHA1 of vendor password
    local input_hash=$(echo -n "$password" | sha1sum | cut -d' ' -f1)
    
    if [ "$input_hash" = "$vendor_hash" ]; then
        # Create vendor access marker
        echo "$(date): VMA vendor access granted" | sudo tee -a "$VENDOR_ACCESS_FILE" > /dev/null
        return 0
    else
        return 1
    fi
}

# Validate IP address
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

# Get VMA network information
get_vma_network_info() {
    VMA_IP=$(hostname -I | awk '{print $1}')
    VMA_INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    VMA_GATEWAY=$(ip route | grep default | awk '{print $3}' | head -1)
    VMA_DNS=$(cat /etc/resolv.conf | grep nameserver | head -1 | awk '{print $2}')
    
    # Check network configuration method
    if systemctl is-active systemd-networkd > /dev/null 2>&1; then
        DHCP_STATUS="systemd-networkd"
    elif [ -f "/etc/netplan/01-netcfg.yaml" ] && grep -q "dhcp4: true" /etc/netplan/01-netcfg.yaml 2>/dev/null; then
        DHCP_STATUS="DHCP (netplan)"
    else
        DHCP_STATUS="Static/Unknown"
    fi
}

# Check VMA services
check_vma_services() {
    VMA_API_STATUS=$(systemctl is-active vma-api.service 2>/dev/null || echo "inactive")
    TUNNEL_STATUS=$(systemctl is-active vma-tunnel-enhanced-v2.service 2>/dev/null || echo "inactive")
    
    # Check VMA API health
    if curl -s --connect-timeout 5 http://localhost:8081/api/v1/health > /dev/null 2>&1; then
        VMA_API_HEALTH="✅ Healthy"
    else
        VMA_API_HEALTH="❌ Not responding"
    fi
    
    # Check OMA connectivity through tunnel
    if curl -s --connect-timeout 5 http://localhost:8082/health > /dev/null 2>&1; then
        OMA_CONNECTIVITY="✅ Connected"
    else
        OMA_CONNECTIVITY="❌ Not connected"
    fi
}

# Get current OMA configuration
get_oma_config() {
    # Set service paths
    VMA_CONFIG="/opt/vma/vma-config.conf"
    TUNNEL_SERVICE="/etc/systemd/system/vma-tunnel-enhanced-v2.service"
    
    if [ -f "$VMA_CONFIG" ]; then
        CURRENT_OMA_IP=$(grep "OMA_HOST=" "$VMA_CONFIG" 2>/dev/null | cut -d= -f2 || echo "Not configured")
    elif systemctl is-active vma-tunnel-enhanced-v2.service > /dev/null 2>&1; then
        CURRENT_OMA_IP=$(systemctl show vma-tunnel-enhanced-v2.service -p Environment | grep -o 'OMA_HOST=[^[:space:]]*' | cut -d= -f2 || echo "Unknown")
    else
        CURRENT_OMA_IP="Not configured"
    fi
}

# Display main interface
show_main_interface() {
    clear
    echo -e "${BLUE}${BOLD}"
    cat << 'BANNER'
╔══════════════════════════════════════════════════════════════════╗
║                     OSSEA-Migrate - VMA Setup                   ║
║                  VMware Migration Appliance                      ║
║                                                                  ║
║              🚀 Professional Migration Platform                  ║
╚══════════════════════════════════════════════════════════════════╝
BANNER
    echo -e "${NC}"
    
    echo -e "${CYAN}Welcome to OSSEA-Migrate VMA (VMware Migration Appliance) Configuration${NC}"
    echo -e "${CYAN}Professional interface for OMA connection and service management.${NC}"
    echo ""
    
    # Get current information
    get_vma_network_info
    check_vma_services
    get_oma_config
    
    # Set tunnel service variable for status display
    TUNNEL_SERVICE="/etc/systemd/system/vma-tunnel-enhanced-v2.service"
    
    # Display VMA network information
    echo -e "${YELLOW}📡 VMA Network Configuration:${NC}"
    echo -e "   ┌─────────────────────────────────────────────────┐"
    echo -e "   │ VMA IP Address: ${BOLD}${VMA_IP}${NC}                    │"
    echo -e "   │ Interface: ${BOLD}${VMA_INTERFACE}${NC}                            │"
    echo -e "   │ Gateway: ${BOLD}${VMA_GATEWAY}${NC}                       │"
    echo -e "   │ DNS Server: ${BOLD}${VMA_DNS}${NC}                        │"
    echo -e "   │ Configuration: ${BOLD}${DHCP_STATUS}${NC}                 │"
    echo -e "   └─────────────────────────────────────────────────┘"
    echo ""
    
    # Display OMA connection status
    echo -e "${YELLOW}🔗 OMA Connection Status:${NC}"
    echo -e "   ┌─────────────────────────────────────────────────┐"
    echo -e "   │ OMA IP: ${BOLD}${CURRENT_OMA_IP}${NC}                       │"
    echo -e "   │ Tunnel Status: ${BOLD}${TUNNEL_STATUS^^}${NC}               │"
    echo -e "   │ OMA Connectivity: ${OMA_CONNECTIVITY}              │"
    echo -e "   └─────────────────────────────────────────────────┘"
    echo ""
    
    # Display service status
    echo -e "${YELLOW}🚀 VMA Service Status:${NC}"
    echo -e "   ┌─────────────────────────────────────────────────┐"
    echo -e "   │ $(get_status_icon $VMA_API_STATUS) VMA API Service        [${VMA_API_STATUS^^}]      │"
    echo -e "   │ $(get_status_icon $TUNNEL_STATUS) SSH Tunnel             [${TUNNEL_STATUS^^}]      │"
    echo -e "   └─────────────────────────────────────────────────┘"
    echo ""
    
    # Display access information
    echo -e "${YELLOW}🌐 Access Information:${NC}"
    echo -e "   VMA API: ${BOLD}http://${VMA_IP}:8081${NC} (${VMA_API_HEALTH})"
    echo -e "   OMA GUI: ${BOLD}http://${CURRENT_OMA_IP}:3001${NC} (via tunnel)"
    echo -e "   Tunnel Port: ${BOLD}localhost:8082${NC} → OMA API"
    echo ""
}

# Get status icon for services
get_status_icon() {
    case $1 in
        "active") echo "🟢" ;;
        "inactive") echo "🔴" ;;
        "failed") echo "❌" ;;
        *) echo "🟡" ;;
    esac
}

# Configure OMA connection
configure_oma_connection() {
    clear
    echo -e "${BLUE}${BOLD}🔗 OMA Connection Configuration${NC}"
    echo "===================================="
    echo ""
    
    echo -e "${YELLOW}Current OMA Configuration:${NC}"
    echo -e "   OMA IP: ${BOLD}${CURRENT_OMA_IP}${NC}"
    echo -e "   Connectivity: ${OMA_CONNECTIVITY}"
    echo ""
    
    # Get new OMA IP
    while true; do
        read -p "📡 Enter OMA IP Address (or 'cancel'): " NEW_OMA_IP
        
        if [ "$NEW_OMA_IP" = "cancel" ]; then
            return
        elif validate_ip "$NEW_OMA_IP"; then
            echo -e "${GREEN}✅ Valid IP format: $NEW_OMA_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid IP format${NC}"
        fi
    done
    
    # Test connectivity
    echo ""
    echo -e "${YELLOW}🔍 Testing connectivity to OMA at $NEW_OMA_IP...${NC}"
    if ping -c 3 "$NEW_OMA_IP" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Network connectivity to OMA successful${NC}"
    else
        echo -e "${YELLOW}⚠️  Warning: Cannot ping OMA (may be normal if firewall blocks ping)${NC}"
    fi
    
    # Test OMA API port
    if curl -s --connect-timeout 5 "http://$NEW_OMA_IP:3001" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ OMA GUI port 3001 accessible${NC}"
    else
        echo -e "${YELLOW}⚠️  OMA GUI port 3001 not accessible (may need configuration)${NC}"
    fi
    
    echo ""
    read -p "Configure VMA to connect to OMA at $NEW_OMA_IP? (y/N): " confirm_config
    
    if [[ $confirm_config =~ ^[Yy]$ ]]; then
        apply_oma_configuration "$NEW_OMA_IP"
    else
        echo -e "${YELLOW}Configuration cancelled.${NC}"
        sleep 2
    fi
}

# Apply OMA configuration
apply_oma_configuration() {
    local oma_ip="$1"
    
    echo ""
    echo -e "${YELLOW}🔧 Configuring VMA-OMA connection...${NC}"
    
    # Save configuration
    cat > "$VMA_CONFIG" << EOF
# VMA Configuration - Generated $(date)
OMA_HOST=$oma_ip
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_LOCAL_PORT=8082
TUNNEL_REMOTE_PORT=8082
EOF
    
    # Update tunnel service environment
    if [ -f "/etc/systemd/system/vma-tunnel-enhanced-v2.service" ]; then
        sudo sed -i "s/Environment=OMA_HOST=.*/Environment=OMA_HOST=$oma_ip/" "/etc/systemd/system/vma-tunnel-enhanced-v2.service" 2>/dev/null || true
        sudo systemctl daemon-reload
    fi
    
    # Restart tunnel service
    echo -e "\${YELLOW}🔄 Restarting tunnel service...\${NC}"
    sudo systemctl stop vma-tunnel-enhanced-v2.service 2>/dev/null || true
    sudo systemctl start vma-tunnel-enhanced-v2.service
    sleep 5
    
    # Restart VMA API
    echo -e "\${YELLOW}🔄 Restarting VMA API service...\${NC}"
    sudo systemctl restart vma-api.service
    sleep 3
    
    # Test connectivity
    echo -e "\${YELLOW}🔍 Testing VMA-OMA connectivity...\${NC}"
    sleep 5
    
    if curl -s --connect-timeout 10 http://localhost:8082/health > /dev/null 2>&1; then
        echo -e "\${GREEN}✅ VMA-OMA tunnel established successfully\${NC}"
        echo -e "\${GREEN}✅ OMA API accessible via tunnel\${NC}"
    else
        echo -e "\${RED}❌ VMA-OMA tunnel connection failed\${NC}"
        echo -e "\${YELLOW}Check OMA firewall and network configuration\${NC}"
    fi
    
    echo ""
    echo -e "\${GREEN}✅ VMA Configuration Complete!\${NC}"
    echo ""
    echo -e "\${CYAN}📊 Connection Summary:\${NC}"
    echo -e "   VMA IP: \${BOLD}\${VMA_IP}\${NC}"
    echo -e "   OMA IP: \${BOLD}$oma_ip\${NC}"
    echo -e "   Tunnel: VMA:8082 → OMA:8082"
    echo -e "   Status: \$(systemctl is-active vma-tunnel-enhanced-v2.service)"
    echo ""
    echo -e "\${CYAN}🎯 Next Steps:\${NC}"
    echo -e "   1. Access OMA GUI: \${BOLD}http://$oma_ip:3001\${NC}"
    echo -e "   2. Add VMA credentials in Discovery settings"
    echo -e "   3. Begin VM discovery and migration"
    echo ""
    
    read -p "Press Enter to continue..."
}

# Show detailed service status
show_service_details() {
    clear
    echo -e "${BLUE}${BOLD}📊 VMA Service Details${NC}"
    echo "========================="
    echo ""
    
    get_vma_network_info
    check_vma_services
    
    echo -e "${YELLOW}🔍 Detailed Service Information:${NC}"
    echo ""
    
    echo -e "${CYAN}VMA API Service (vma-api.service):${NC}"
    systemctl status vma-api.service --no-pager -l | head -8
    echo -e "   Health: ${VMA_API_HEALTH}"
    echo ""
    
    echo -e "${CYAN}SSH Tunnel (vma-tunnel-enhanced-v2.service):${NC}"
    systemctl status vma-tunnel-enhanced-v2.service --no-pager -l | head -8
    echo -e "   OMA Connectivity: ${OMA_CONNECTIVITY}"
    echo ""
    
    # Show tunnel ports
    echo -e "${CYAN}🔗 Tunnel Port Status:${NC}"
    echo -e "   Local Port 8082: $(netstat -tln | grep :8082 > /dev/null && echo '✅ Listening' || echo '❌ Not listening')"
    echo -e "   VMA API Port 8081: $(netstat -tln | grep :8081 > /dev/null && echo '✅ Listening' || echo '❌ Not listening')"
    echo ""
    
    read -p "Press Enter to return to main menu..."
}

# Configure VMA network settings
configure_vma_network() {
    clear
    echo -e "${BLUE}${BOLD}🔧 VMA Network Configuration${NC}"
    echo "================================="
    echo ""
    
    echo -e "${YELLOW}Current VMA Network:${NC}"
    echo -e "   IP: ${BOLD}${VMA_IP}${NC}"
    echo -e "   Gateway: ${BOLD}${VMA_GATEWAY}${NC}"
    echo -e "   DNS: ${BOLD}${VMA_DNS}${NC}"
    echo -e "   Mode: ${BOLD}${DHCP_STATUS}${NC}"
    echo ""
    
    echo -e "${BOLD}Network Configuration Options:${NC}"
    echo "   1. Configure Static IP"
    echo "   2. Switch to DHCP"
    echo "   3. Update DNS Settings"
    echo "   4. Return to main menu"
    echo ""
    
    read -p "Select option (1-4): " net_choice
    
    case $net_choice in
        1) configure_static_vma_network ;;
        2) configure_dhcp_vma_network ;;
        3) configure_vma_dns ;;
        4) return ;;
        *) 
            echo -e "${RED}Invalid option${NC}"
            sleep 2
            configure_vma_network
            ;;
    esac
}

# Configure static VMA network
configure_static_vma_network() {
    echo ""
    echo -e "${BOLD}🔧 VMA Static IP Configuration${NC}"
    echo "=============================="
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
        read -p "Enter Gateway IP: " GATEWAY_IP
        if validate_ip "$GATEWAY_IP"; then
            echo -e "${GREEN}✅ Valid gateway IP: $GATEWAY_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid gateway IP format${NC}"
        fi
    done
    
    while true; do
        read -p "Enter DNS Server: " DNS_IP
        if validate_ip "$DNS_IP"; then
            echo -e "${GREEN}✅ Valid DNS IP: $DNS_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid DNS IP format${NC}"
        fi
    done
    
    # Apply static configuration
    echo ""
    echo -e "${YELLOW}📋 VMA Static IP Configuration:${NC}"
    echo -e "   IP Address: ${BOLD}$STATIC_IP${NC}"
    echo -e "   Gateway: ${BOLD}$GATEWAY_IP${NC}"
    echo -e "   DNS Server: ${BOLD}$DNS_IP${NC}"
    echo ""
    read -p "Apply this configuration? (y/N): " apply_static
    
    if [[ $apply_static =~ ^[Yy]$ ]]; then
        # Create netplan configuration for VMA
        cat > /tmp/vma-static-config.yaml << NETPLAN_EOF
network:
  version: 2
  renderer: networkd
  ethernets:
    $VMA_INTERFACE:
      dhcp4: false
      addresses:
        - $STATIC_IP/24
      gateway4: $GATEWAY_IP
      nameservers:
        addresses:
          - $DNS_IP
NETPLAN_EOF
        
        sudo cp /etc/netplan/01-netcfg.yaml /etc/netplan/01-netcfg.yaml.backup 2>/dev/null || true
        sudo cp /tmp/vma-static-config.yaml /etc/netplan/01-netcfg.yaml
        
        echo -e "${YELLOW}⚠️  Applying VMA network configuration...${NC}"
        echo -e "${CYAN}VMA will be accessible at: ${BOLD}$STATIC_IP${NC}"
        echo ""
        read -p "Press Enter to apply configuration and reboot..."
        
        sudo netplan apply
        sleep 3
        sudo reboot
    else
        echo -e "${YELLOW}Configuration cancelled.${NC}"
        sleep 2
    fi
}

# Configure DHCP for VMA
configure_dhcp_vma_network() {
    echo ""
    echo -e "${BOLD}🔧 VMA DHCP Configuration${NC}"
    echo "========================="
    echo ""
    
    echo -e "${YELLOW}⚠️  This will configure VMA to use DHCP for IP assignment.${NC}"
    echo -e "${YELLOW}The VMA IP address may change after reboot.${NC}"
    echo ""
    read -p "Continue with DHCP configuration? (y/N): " apply_dhcp
    
    if [[ $apply_dhcp =~ ^[Yy]$ ]]; then
        # Create DHCP netplan configuration
        cat > /tmp/vma-dhcp-config.yaml << NETPLAN_EOF
network:
  version: 2
  renderer: networkd
  ethernets:
    $VMA_INTERFACE:
      dhcp4: true
      dhcp6: false
NETPLAN_EOF
        
        sudo cp /etc/netplan/01-netcfg.yaml /etc/netplan/01-netcfg.yaml.backup 2>/dev/null || true
        sudo cp /tmp/vma-dhcp-config.yaml /etc/netplan/01-netcfg.yaml
        
        echo -e "${YELLOW}⚠️  Applying DHCP configuration...${NC}"
        echo -e "${CYAN}VMA will receive IP from DHCP server after reboot.${NC}"
        echo ""
        read -p "Press Enter to apply configuration and reboot..."
        
        sudo netplan apply
        sleep 3
        sudo reboot
    else
        echo -e "${YELLOW}DHCP configuration cancelled.${NC}"
        sleep 2
    fi
}

# Configure VMA DNS
configure_vma_dns() {
    echo ""
    echo -e "${BOLD}🔧 VMA DNS Configuration${NC}"
    echo "========================"
    echo ""
    
    echo -e "Current DNS: ${BOLD}${VMA_DNS}${NC}"
    echo ""
    
    while true; do
        read -p "Enter new DNS server IP (or 'cancel'): " new_dns
        if [ "$new_dns" = "cancel" ]; then
            return
        elif validate_ip "$new_dns"; then
            echo -e "${GREEN}✅ Valid DNS IP: $new_dns${NC}"
            break
        else
            echo -e "${RED}❌ Invalid DNS IP format${NC}"
        fi
    done
    
    echo ""
    read -p "Apply DNS configuration? (y/N): " apply_dns
    
    if [[ $apply_dns =~ ^[Yy]$ ]]; then
        sudo sed -i "s/- $VMA_DNS/- $new_dns/g" /etc/netplan/01-netcfg.yaml 2>/dev/null || true
        
        echo -e "${YELLOW}🔧 Applying DNS configuration...${NC}"
        sudo netplan apply
        sleep 2
        echo -e "${GREEN}✅ DNS updated to $new_dns${NC}"
        sleep 3
    else
        echo -e "${YELLOW}DNS configuration cancelled.${NC}"
        sleep 2
    fi
}

# Restart VMA services
restart_vma_services() {
    echo ""
    echo -e "${YELLOW}🔄 Restarting VMA services...${NC}"
    echo ""
    
    services=("vma-tunnel-enhanced-v2" "vma-api")
    
    for service in "${services[@]}"; do
        echo -e "   Restarting ${service}..."
        sudo systemctl restart "${service}.service"
        sleep 3
        
        if systemctl is-active "${service}.service" > /dev/null 2>&1; then
            echo -e "   ${GREEN}✅ ${service} restarted successfully${NC}"
        else
            echo -e "   ${RED}❌ ${service} failed to restart${NC}"
        fi
    done
    
    echo ""
    echo -e "${GREEN}✅ VMA service restart completed${NC}"
    sleep 3
}

# Vendor shell access
vendor_shell_access() {
    echo ""
    echo -e "${RED}${BOLD}🔒 VENDOR ACCESS CONTROL${NC}"
    echo -e "${YELLOW}This provides full system access for OSSEA-Migrate support personnel only.${NC}"
    echo -e "${YELLOW}Unauthorized access is logged and monitored.${NC}"
    echo ""
    
    # Three-strike system
    local attempts=0
    local max_attempts=3
    
    while [ $attempts -lt $max_attempts ]; do
        read -p "Enter vendor access code: " -s vendor_pass
        echo ""
        
        if check_vendor_access "$vendor_pass"; then
            echo -e "${GREEN}✅ Vendor access granted${NC}"
            echo -e "${CYAN}Entering VMA administrative shell...${NC}"
            echo ""
            echo "OSSEA-Migrate VMA - Vendor Shell Access"
            echo "======================================="
            echo "Access granted: $(date)"
            echo "User: Vendor Support"
            echo "VMA System: $(hostname) ($(hostname -I | awk '{print $1}'))"
            echo ""
            echo "Type 'exit' to return to setup wizard"
            echo ""
            
            # Set vendor environment
            export PS1="[VENDOR@VMA \W]$ "
            export VENDOR_ACCESS="true"
            
            # Start vendor shell
            exec /bin/bash --login
        else
            attempts=$((attempts + 1))
            remaining=$((max_attempts - attempts))
            
            echo -e "${RED}❌ Invalid vendor access code${NC}"
            
            if [ $remaining -gt 0 ]; then
                echo -e "${YELLOW}$remaining attempts remaining${NC}"
                echo ""
            else
                echo -e "${RED}🚨 Maximum attempts exceeded. Access denied.${NC}"
                echo -e "${YELLOW}Returning to setup wizard in 5 seconds...${NC}"
                
                # Log failed access attempt
                echo "$(date): Failed VMA vendor access attempt from $(who am i)" | sudo tee -a /var/log/vma-vendor-access.log > /dev/null
                
                sleep 5
                return
            fi
        fi
    done
}

# Main menu
main_menu() {
    while true; do
        # Ensure TTY is in proper state for input
        if [ -t 0 ]; then
            stty sane 2>/dev/null || true
            stty echo 2>/dev/null || true
        fi
        
        show_main_interface
        
        echo -e "${BOLD}🔧 Configuration Options:${NC}"
        echo "   1. Configure OMA Connection"
        echo "   2. Configure VMA Network Settings"
        echo "   3. View Detailed Service Status"
        echo "   4. Restart VMA Services"
        echo "   5. Vendor Shell Access (Support Only)"
        echo "   6. Reboot VMA System"
        echo ""
        echo -e "${CYAN}🔒 Note: Shell access restricted to vendor support personnel${NC}"
        echo -e "${CYAN}📡 Use Ctrl+] to interrupt (wizard will restart automatically)${NC}"
        echo ""
        
        read -p "Select option (1-6): " choice
        
        case $choice in
            1)
                configure_oma_connection
                ;;
            2)
                configure_vma_network
                ;;
            3)
                show_service_details
                ;;
            4)
                restart_vma_services
                ;;
            5)
                vendor_shell_access
                ;;
            6)
                echo ""
                echo -e "${YELLOW}🔄 Rebooting VMA system...${NC}"
                echo -e "${CYAN}System will restart and wizard will load automatically.${NC}"
                echo ""
                read -p "Press Enter to reboot..."
                sudo reboot
                ;;
            *)
                echo -e "${RED}Invalid option. Please select 1-6.${NC}"
                sleep 2
                ;;
        esac
    done
}

# Initialize and run
echo "🚀 Initializing Enhanced OSSEA-Migrate VMA Configuration Wizard..."
echo "🔒 Security: Shell access restricted to vendor support"
echo "📡 Features: OMA connection, network config, service management"
sleep 2

# Ensure we're running with proper permissions
if [ "$(whoami)" != "pgrayson" ] && [ "$(whoami)" != "root" ]; then
    echo -e "${RED}❌ Wizard must run as pgrayson or root user${NC}"
    exit 1
fi

main_menu
BASIC_EOF

chmod +x "$BUILD_DIR/scripts/vma-setup-wizard-basic.sh"

# PHASE 3: VMA SERVICE CONFIGURATIONS
echo "⚙️ Creating VMA service configurations..."

# Enhanced VMA autologin service
cat > "$BUILD_DIR/services/vma-autologin.service" << 'EOF'
[Unit]
Description=OSSEA-Migrate VMA Custom Boot Experience
Documentation=https://github.com/DRDAVIDBANNER/X-Vire
After=multi-user.target network.target
Wants=network.target
DefaultDependencies=no
Conflicts=getty@tty1.service

[Service]
Type=simple
User=pgrayson
Group=pgrayson
ExecStart=/opt/vma/setup-wizard.sh
StandardInput=tty-force
StandardOutput=tty
TTYPath=/dev/tty1
TTYReset=yes
TTYVTDisallocate=yes
KillMode=process
IgnoreSIGPIPE=no
SendSIGHUP=yes

# Environment variables
Environment=HOME=/home/pgrayson
Environment=USER=pgrayson
Environment=TERM=xterm-256color

# Security settings
NoNewPrivileges=false
PrivateTmp=false

[Install]
WantedBy=multi-user.target
EOF

# VMA API service
cat > "$BUILD_DIR/services/vma-api.service" << 'EOF'
[Unit]
Description=VMA Migration API Server
Documentation=https://github.com/DRDAVIDBANNER/X-Vire
After=network.target
Wants=network.target

[Service]
Type=simple
User=pgrayson
Group=pgrayson
WorkingDirectory=/home/pgrayson
ExecStart=/home/pgrayson/migratekit-cloudstack/vma-api-server -port=8081
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Environment variables
Environment=HOME=/home/pgrayson
Environment=USER=pgrayson

[Install]
WantedBy=multi-user.target
EOF

# Enhanced tunnel service template
cat > "$BUILD_DIR/services/vma-tunnel-enhanced-v2.service" << 'EOF'
[Unit]
Description=Enhanced VMA-OMA SSH Tunnel (Bidirectional)
Documentation=https://github.com/DRDAVIDBANNER/X-Vire
After=network.target
Wants=network.target

[Service]
Type=simple
User=pgrayson
Group=pgrayson
WorkingDirectory=/home/pgrayson

# Environment - OMA_HOST will be configured during setup
Environment=OMA_HOST=CONFIGURE_DURING_SETUP
Environment=SSH_KEY=/home/pgrayson/.ssh/cloudstack_key

# Enhanced tunnel command with health checks and auto-recovery
ExecStart=/bin/bash -c 'while true; do ssh -i ${SSH_KEY} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ServerAliveInterval=60 -o ServerAliveCountMax=3 -o ExitOnForwardFailure=yes -L 8082:localhost:8082 -R 9081:localhost:8081 pgrayson@${OMA_HOST} "echo VMA tunnel established to ${OMA_HOST}; while true; do sleep 60; done"; echo "Tunnel disconnected, retrying in 10 seconds..."; sleep 10; done'

Restart=always
RestartSec=15
StartLimitInterval=60
StartLimitBurst=5

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vma-tunnel

[Install]
WantedBy=multi-user.target
EOF

echo "✅ VMA service configurations created"

# PHASE 4: VMA CONFIGURATION SCRIPTS
echo "🔧 Creating VMA configuration scripts..."

# VMA configuration helper
cat > "$BUILD_DIR/scripts/configure-oma-connection.sh" << 'EOF'
#!/bin/bash
# Configure VMA-OMA Connection Helper Script

OMA_IP="$1"
VMA_IP=$(hostname -I | awk '{print $1}')
VMA_CONFIG="/opt/vma/vma-config.conf"

echo "🔧 Configuring VMA-OMA connection..."
echo "   VMA IP: $VMA_IP"
echo "   OMA IP: $OMA_IP"

# Create VMA configuration
cat > "$VMA_CONFIG" << CONFIG_EOF
# VMA Configuration - Generated $(date)
OMA_HOST=$OMA_IP
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_LOCAL_PORT=8082
TUNNEL_REMOTE_PORT=8082
CONFIG_EOF

# Update tunnel service environment
sudo sed -i "s/Environment=OMA_HOST=.*/Environment=OMA_HOST=$OMA_IP/" /etc/systemd/system/vma-tunnel-enhanced-v2.service
sudo systemctl daemon-reload

echo "✅ VMA-OMA connection configured"
EOF

chmod +x "$BUILD_DIR/scripts/configure-oma-connection.sh"

echo "✅ VMA configuration scripts created"

# PHASE 5: Create VMA deployment script
echo "📜 Creating VMA deployment script..."
cat > "$BUILD_DIR/deploy-production-vma.sh" << 'DEPLOY_SCRIPT'
#!/bin/bash
# Production VMA Deployment Script
# Deploys STABLE binaries with enhanced wizard and tunnel configuration

set -euo pipefail

SUDO_PASSWORD="Password1"
BUILD_DIR="/tmp/production-vma-build"
VMA_USER="pgrayson"
LOG_FILE="/tmp/vma-deployment.log"

# Redirect all output to log file and console
exec > >(tee -a "$LOG_FILE")
exec 2>&1

echo "🚀 OSSEA-Migrate VMA Production Deployment"
echo "=========================================="
echo "Deployment Date: $(date)"
echo "Target User: $VMA_USER"
echo "Log File: $LOG_FILE"
echo ""

# Function to run sudo commands with password
run_sudo() {
    echo "$SUDO_PASSWORD" | sudo -S "$@"
}

# Function to check command success
check_success() {
    local exit_code=$?
    if [ $exit_code -eq 0 ]; then
        echo "✅ $1 completed successfully"
    else
        echo "❌ $1 failed (exit code: $exit_code)"
        echo "🔍 Check log file: $LOG_FILE"
        exit 1
    fi
}

# PHASE 1: Pre-flight validation
echo "📋 Phase 1: Pre-flight Validation"
echo "=================================="

# Verify OS version
if ! grep -q "24.04\|22.04" /etc/os-release; then
    echo "❌ This script requires Ubuntu 22.04 or 24.04 LTS"
    exit 1
fi

# Verify build package
if [ ! -d "$BUILD_DIR" ]; then
    echo "❌ Build directory not found: $BUILD_DIR"
    echo "Please transfer the VMA build package first"
    exit 1
fi

# Verify required files
required_files=(
    "$BUILD_DIR/binaries/migratekit"
    "$BUILD_DIR/binaries/vma-api-server"
    "$BUILD_DIR/scripts/vma-setup-wizard.sh"
    "$BUILD_DIR/scripts/configure-oma-connection.sh"
    "$BUILD_DIR/services/vma-api.service"
    "$BUILD_DIR/services/vma-autologin.service"
    "$BUILD_DIR/services/vma-tunnel-enhanced-v2.service"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Required file missing: $file"
        exit 1
    fi
done

echo "✅ Pre-flight validation passed"
echo ""

# PHASE 2: System preparation
echo "📋 Phase 2: System Preparation"
echo "=============================="

echo "🔄 Updating system packages..."
run_sudo apt update -y
check_success "System package update"

echo "📦 Installing VMA dependencies..."
DEBIAN_FRONTEND=noninteractive run_sudo apt install -y \
    openssh-client \
    curl \
    jq \
    net-tools \
    systemd \
    golang-go
check_success "Dependencies installation"

echo "👤 Configuring VMA user..."
run_sudo usermod -aG sudo "$VMA_USER" 2>/dev/null || true
check_success "User configuration"

echo "✅ System preparation completed"
echo ""

# PHASE 3: Directory structure and binary deployment
echo "📋 Phase 3: Binary Deployment"
echo "============================="

echo "📁 Creating VMA directory structure..."
run_sudo mkdir -p /opt/vma
run_sudo mkdir -p /home/pgrayson/migratekit-cloudstack
check_success "Directory creation"

echo "📦 Deploying VMA production binaries..."

# Deploy migratekit
run_sudo cp binaries/migratekit /home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel
run_sudo chmod +x /home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel
run_sudo chown "$VMA_USER:$VMA_USER" /home/pgrayson/migratekit-cloudstack/migratekit-tls-tunnel
check_success "Migratekit deployment"

# Deploy VMA API server
run_sudo cp binaries/vma-api-server /home/pgrayson/migratekit-cloudstack/vma-api-server
run_sudo chmod +x /home/pgrayson/migratekit-cloudstack/vma-api-server
run_sudo chown "$VMA_USER:$VMA_USER" /home/pgrayson/migratekit-cloudstack/vma-api-server
check_success "VMA API deployment"

# Deploy setup wizard
run_sudo cp scripts/vma-setup-wizard.sh /opt/vma/setup-wizard.sh
run_sudo chmod +x /opt/vma/setup-wizard.sh
run_sudo chown "$VMA_USER:$VMA_USER" /opt/vma/setup-wizard.sh
check_success "VMA setup wizard deployment"

# Deploy configuration helper
run_sudo cp scripts/configure-oma-connection.sh /opt/vma/
run_sudo chmod +x /opt/vma/configure-oma-connection.sh
run_sudo chown "$VMA_USER:$VMA_USER" /opt/vma/configure-oma-connection.sh
check_success "VMA configuration helper deployment"

echo "✅ Binary deployment completed"
echo ""

# PHASE 4: Service configuration
echo "📋 Phase 4: Service Configuration"
echo "================================="

echo "⚙️ Installing systemd services..."
run_sudo cp "$BUILD_DIR/services/"*.service /etc/systemd/system/
run_sudo systemctl daemon-reload
check_success "Service installation"

echo "🚀 Enabling VMA services..."
run_sudo systemctl enable vma-api vma-tunnel-enhanced-v2 vma-autologin
check_success "Service enablement"

echo "🚫 Disabling standard login..."
run_sudo systemctl disable getty@tty1 2>/dev/null || true
check_success "Standard login disable"

echo "✅ Service configuration completed"
echo ""

# PHASE 5: Service startup
echo "📋 Phase 5: Service Startup"
echo "=========================="

echo "🚀 Starting VMA API service..."
run_sudo systemctl start vma-api
sleep 5

# Note: Tunnel service will be configured via wizard
echo "ℹ️  Tunnel service will be configured via setup wizard"

echo "✅ Service startup completed"
echo ""

# PHASE 6: Health validation
echo "📋 Phase 6: Health Validation"
echo "============================="

VMA_IP=$(hostname -I | awk '{print $1}')
echo "🔍 Testing VMA service health on $VMA_IP..."

# VMA API health
if curl -s --connect-timeout 5 http://localhost:8081/api/v1/health > /dev/null 2>&1; then
    echo "✅ VMA API health check passed"
else
    echo "⚠️ VMA API health check failed"
fi

# Service status check
echo ""
echo "📊 Service Status:"
for service in vma-api vma-autologin; do
    status=$(systemctl is-active "$service.service" 2>/dev/null || echo "inactive")
    if [ "$status" = "active" ]; then
        echo "   ✅ $service: $status"
    else
        echo "   ❌ $service: $status"
    fi
done

echo ""
echo "✅ Health validation completed"
echo ""

# PHASE 7: Finalization
echo "📋 Phase 7: Finalization"
echo "======================="

echo "🧹 Cleaning up build artifacts..."
rm -rf "$BUILD_DIR" 2>/dev/null || true
check_success "Build artifact cleanup"

echo "✅ Finalization completed"
echo ""

# PHASE 8: Final VMA information
echo "🎉 OSSEA-Migrate VMA Production Deployment Complete!"
echo "===================================================="
echo ""
echo "📊 VMA Appliance Information:"
echo "   Appliance: OSSEA-Migrate VMA v1.0 (Production)"
echo "   OS: $(lsb_release -d | cut -f2)"
echo "   VMA IP Address: $VMA_IP"
echo "   Deployment Date: $(date)"
echo "   User Account: $VMA_USER"
echo ""
echo "🌐 VMA Services:"
echo "   VMA API: http://$VMA_IP:8081"
echo "   Setup Wizard: Available on console/reboot"
echo ""
echo "🚀 Deployed Features:"
echo "   ✅ Latest migratekit with sparse block optimization"
echo "   ✅ VMA API server with progress tracking"
echo "   ✅ Enhanced setup wizard with vendor access control"
echo "   ✅ Professional custom boot experience"
echo "   ✅ OMA connection management"
echo "   ✅ Network configuration capabilities"
echo ""
echo "📋 Next Steps:"
echo "   1. Reboot VMA to activate custom boot wizard"
echo "   2. Configure OMA connection via wizard"
echo "   3. Test VMA-OMA tunnel connectivity"
echo "   4. Begin VM discovery from OMA"
echo ""
echo "🎯 VMA Ready For:"
echo "   - Enterprise VMware migration operations"
echo "   - Professional customer deployment"
echo "   - OMA appliance connectivity"
echo ""

# Create VMA info file
cat > "/home/$VMA_USER/vma-appliance-info.txt" << EOF
OSSEA-Migrate VMA Production Appliance v1.0
Deployment Date: $(date)
Base OS: $(lsb_release -d | cut -f2)
VMA IP Address: $VMA_IP

VMA Services:
- VMA API: http://$VMA_IP:8081
- Setup Wizard: Console boot interface

Deployed Components:
- Migratekit: Latest with sparse block optimization
- VMA API: Production with progress tracking
- Enhanced Wizard: Vendor access control + TTY recovery
- Custom Boot: Professional OSSEA-Migrate experience

Service Status: $(date)
$(systemctl is-active vma-api vma-autologin | paste -sd' ')

For support: https://github.com/DRDAVIDBANNER/X-Vire
EOF

run_sudo chown "$VMA_USER:$VMA_USER" "/home/$VMA_USER/vma-appliance-info.txt"

echo ""
echo "📄 Deployment log: $LOG_FILE"
echo "📄 VMA info: /home/$VMA_USER/vma-appliance-info.txt"
echo ""
echo "✅ PRODUCTION OSSEA-MIGRATE VMA APPLIANCE READY!"
DEPLOY_SCRIPT

chmod +x "$BUILD_DIR/deploy-production-vma.sh"

echo ""
echo "✅ PRODUCTION VMA BUILD PACKAGE COMPLETE!"
echo ""
echo "📦 Build package location: $BUILD_DIR"
echo "📊 Package contents:"
echo "   - Latest migratekit: sparse block optimization + NBD compatibility"
echo "   - Latest VMA API: v1.10.4 with progress tracking"
echo "   - Enhanced VMA wizard: vendor access control + TTY recovery"
echo "   - Complete service configurations"
echo "   - OMA connection management scripts"
echo "   - Professional custom boot experience"
echo ""
echo "🚀 Ready for VMA production deployment!"
echo ""
echo "📋 Next Steps:"
echo "   1. Create Ubuntu 22.04 VM in VMware (4GB RAM, 2 vCPU, 20GB disk)"
echo "   2. Transfer build package: scp -r $BUILD_DIR/ user@vma-vm:/tmp/"
echo "   3. Deploy: ssh user@vma-vm 'cd /tmp/production-vma-build && sudo ./deploy-production-vma.sh'"
echo "   4. Test functionality and export as VMware OVA"
echo ""
echo "🔒 SOURCE CODE ISOLATION: No source code or docs included - production binaries only!"
