#!/bin/bash
# Enhanced VMA Setup Wizard - Professional OMA Connection Configuration
# Features: Vendor access control, tunnel management, service monitoring

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
    cat > "$VMA_CONFIG" << CONFIG_EOF
# VMA Configuration - Generated $(date)
OMA_HOST=$oma_ip
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_LOCAL_PORT=8082
TUNNEL_REMOTE_PORT=8082
CONFIG_EOF
    
    # Update tunnel service environment
    if [ -f "$TUNNEL_SERVICE" ]; then
        sudo sed -i "s/Environment=OMA_HOST=.*/Environment=OMA_HOST=$oma_ip/" "$TUNNEL_SERVICE" 2>/dev/null || true
        sudo systemctl daemon-reload
    fi
    
    # Restart tunnel service
    echo -e "${YELLOW}🔄 Restarting tunnel service...${NC}"
    sudo systemctl stop vma-tunnel-enhanced-v2.service 2>/dev/null || true
    sudo systemctl start vma-tunnel-enhanced-v2.service
    sleep 5
    
    # Restart VMA API
    echo -e "${YELLOW}🔄 Restarting VMA API service...${NC}"
    sudo systemctl restart vma-api.service
    sleep 3
    
    # Test connectivity
    echo -e "${YELLOW}🔍 Testing VMA-OMA connectivity...${NC}"
    sleep 5
    
    if curl -s --connect-timeout 10 http://localhost:8082/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ VMA-OMA tunnel established successfully${NC}"
        echo -e "${GREEN}✅ OMA API accessible via tunnel${NC}"
    else
        echo -e "${RED}❌ VMA-OMA tunnel connection failed${NC}"
        echo -e "${YELLOW}Check OMA firewall and network configuration${NC}"
    fi
    
    echo ""
    echo -e "${GREEN}✅ VMA Configuration Complete!${NC}"
    echo ""
    echo -e "${CYAN}📊 Connection Summary:${NC}"
    echo -e "   VMA IP: ${BOLD}${VMA_IP}${NC}"
    echo -e "   OMA IP: ${BOLD}$oma_ip${NC}"
    echo -e "   Tunnel: VMA:8082 → OMA:8082"
    echo -e "   Status: $(systemctl is-active vma-tunnel-enhanced-v2.service)"
    echo ""
    echo -e "${CYAN}🎯 Next Steps:${NC}"
    echo -e "   1. Access OMA GUI: ${BOLD}http://$oma_ip:3001${NC}"
    echo -e "   2. Add VMA credentials in Discovery settings"
    echo -e "   3. Begin VM discovery and migration"
    echo ""
    
    read -p "Press Enter to continue..."
}

# Vendor shell access
vendor_shell_access() {
    echo ""
    echo -e "${RED}${BOLD}🔒 VENDOR ACCESS CONTROL${NC}"
    echo -e "${YELLOW}This provides full system access for OSSEA-Migrate support personnel only.${NC}"
    echo ""
    
    read -p "Enter vendor access code: " -s vendor_pass
    echo ""
    
    if check_vendor_access "$vendor_pass"; then
        echo -e "${GREEN}✅ Vendor access granted${NC}"
        echo -e "${CYAN}Entering VMA administrative shell...${NC}"
        export PS1="[VENDOR@VMA \W]$ "
        exec /bin/bash --login
    else
        echo -e "${RED}❌ Invalid vendor access code${NC}"
        sleep 3
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
    
    echo -e "${CYAN}Welcome to OSSEA-Migrate VMA (VMware Migration Appliance)${NC}"
    echo ""
    
    # Get current information
    get_vma_network_info
    check_vma_services
    get_oma_config
    
    # Display VMA network information
    echo -e "${YELLOW}📡 VMA Network: ${BOLD}${VMA_IP}${NC} (${DHCP_STATUS})"
    echo -e "${YELLOW}🔗 OMA Connection: ${BOLD}${CURRENT_OMA_IP}${NC} (${OMA_CONNECTIVITY})"
    echo -e "${YELLOW}🚀 VMA API: ${VMA_API_HEALTH} | Tunnel: $(get_status_icon $TUNNEL_STATUS)"
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
        echo "   2. View Service Status"
        echo "   3. Restart VMA Services"
        echo "   4. Vendor Shell Access (Support Only)"
        echo "   5. Reboot VMA System"
        echo ""
        echo -e "${CYAN}🔒 Note: Shell access restricted to vendor support${NC}"
        echo ""
        
        read -p "Select option (1-5): " choice
        
        case $choice in
            1) configure_oma_connection ;;
            2) 
                echo ""
                echo "📊 VMA Service Details:"
                systemctl status vma-api vma-tunnel-enhanced-v2 --no-pager -l | head -20
                read -p "Press Enter to continue..."
                ;;
            3)
                echo ""
                echo -e "${YELLOW}🔄 Restarting VMA services...${NC}"
                sudo systemctl restart vma-api vma-tunnel-enhanced-v2
                echo -e "${GREEN}✅ Services restarted${NC}"
                sleep 3
                ;;
            4) vendor_shell_access ;;
            5)
                echo -e "${YELLOW}🔄 Rebooting VMA...${NC}"
                sudo reboot
                ;;
            *)
                echo -e "${RED}Invalid option${NC}"
                sleep 2
                ;;
        esac
    done
}

# Initialize and run
echo "🚀 Initializing OSSEA-Migrate VMA Configuration Wizard..."
sleep 2

# Ensure we're running with proper permissions
if [ "$(whoami)" != "pgrayson" ] && [ "$(whoami)" != "root" ]; then
    echo -e "${RED}❌ Wizard must run as pgrayson or root user${NC}"
    exit 1
fi

main_menu






