#!/bin/bash
# OMA Setup Wizard - Network Configuration and Service Management
# Professional deployment interface for OSSEA-Migrate OMA

set -euo pipefail

# Colors and formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Signal handling for graceful interruption
trap 'echo -e "\n${YELLOW}⚠️  Setup interrupted. Restarting wizard...${NC}"; sleep 1; exec "$0"' INT TERM
trap 'echo -e "\n${YELLOW}⚠️  Setup suspended. Restarting wizard...${NC}"; sleep 1; exec "$0"' TSTP

# Configuration files
OMA_CONFIG="/opt/ossea-migrate/oma-config.conf"

# Get current network information
get_network_info() {
    OMA_IP=$(hostname -I | awk '{print $1}')
    OMA_INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    OMA_GATEWAY=$(ip route | grep default | awk '{print $3}' | head -1)
    OMA_DNS=$(cat /etc/resolv.conf | grep nameserver | head -1 | awk '{print $2}')
    
    # Check if using DHCP
    if [ -f "/etc/netplan/01-netcfg.yaml" ] && grep -q "dhcp4: true" /etc/netplan/01-netcfg.yaml 2>/dev/null; then
        DHCP_STATUS="DHCP"
    else
        DHCP_STATUS="Static"
    fi
}

# Check service status
check_services() {
    OMA_API_STATUS=$(systemctl is-active oma-api.service 2>/dev/null || echo "inactive")
    VOLUME_DAEMON_STATUS=$(systemctl is-active volume-daemon.service 2>/dev/null || echo "inactive")
    NBD_SERVER_STATUS=$(systemctl is-active nbd-server.service 2>/dev/null || echo "inactive")
    MARIADB_STATUS=$(systemctl is-active mariadb.service 2>/dev/null || echo "inactive")
    GUI_STATUS=$(systemctl is-active migration-gui.service 2>/dev/null || echo "inactive")
}

# Check service health
check_service_health() {
    # OMA API health
    if curl -s --connect-timeout 5 http://localhost:8082/health > /dev/null 2>&1; then
        OMA_API_HEALTH="✅ Healthy"
    else
        OMA_API_HEALTH="❌ Not responding"
    fi
    
    # Volume Daemon health
    if curl -s --connect-timeout 5 http://localhost:8090/api/v1/health > /dev/null 2>&1; then
        VOLUME_DAEMON_HEALTH="✅ Healthy"
    else
        VOLUME_DAEMON_HEALTH="❌ Not responding"
    fi
    
    # GUI health
    if curl -s --connect-timeout 5 http://localhost:3001 > /dev/null 2>&1; then
        GUI_HEALTH="✅ Healthy"
    else
        GUI_HEALTH="❌ Not responding"
    fi
    
    # Database health
    if mysql -u oma_user -poma_password -e "SELECT 1;" > /dev/null 2>&1; then
        DB_HEALTH="✅ Healthy"
    else
        DB_HEALTH="❌ Not responding"
    fi
}

# Display main interface
show_main_interface() {
    clear
    echo -e "${BLUE}${BOLD}"
    cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                     OSSEA-Migrate - OMA Setup                   ║
║                OSSEA Migration Appliance Control                 ║
║                                                                  ║
║              🚀 Professional Migration Platform                  ║
╚══════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
    
    echo -e "${CYAN}Welcome to OSSEA-Migrate OMA (OSSEA Migration Appliance) Configuration${NC}"
    echo -e "${CYAN}This interface provides network configuration and service management.${NC}"
    echo ""
    
    # Get current information
    get_network_info
    check_services
    check_service_health
    
    # Display network information
    echo -e "${YELLOW}📡 Current Network Configuration:${NC}"
    echo -e "   OMA IP Address: ${BOLD}${OMA_IP}${NC}"
    echo -e "   Network Interface: ${BOLD}${OMA_INTERFACE}${NC}"
    echo -e "   Gateway: ${BOLD}${OMA_GATEWAY}${NC}"
    echo -e "   DNS Server: ${BOLD}${OMA_DNS}${NC}"
    echo -e "   Configuration: ${BOLD}${DHCP_STATUS}${NC}"
    echo ""
    
    # Display service status
    echo -e "${YELLOW}🚀 Service Status:${NC}"
    echo -e "   ┌─────────────────────────────────────┐"
    echo -e "   │ $(get_status_icon $OMA_API_STATUS) OMA API Service        [${OMA_API_STATUS^^}]  │"
    echo -e "   │ $(get_status_icon $VOLUME_DAEMON_STATUS) Volume Daemon          [${VOLUME_DAEMON_STATUS^^}]  │"
    echo -e "   │ $(get_status_icon $NBD_SERVER_STATUS) NBD Server             [${NBD_SERVER_STATUS^^}]  │"
    echo -e "   │ $(get_status_icon $MARIADB_STATUS) MariaDB Database       [${MARIADB_STATUS^^}]  │"
    echo -e "   │ $(get_status_icon $GUI_STATUS) Migration GUI          [${GUI_STATUS^^}]  │"
    echo -e "   └─────────────────────────────────────┘"
    echo ""
    
    # Display access information
    echo -e "${YELLOW}🌐 Access Information:${NC}"
    echo -e "   Web Interface: ${BOLD}http://${OMA_IP}:3001${NC}"
    echo -e "   API Endpoint: ${BOLD}http://${OMA_IP}:8082${NC}"
    echo -e "   Health Status: ${BOLD}${OMA_API_HEALTH}${NC}"
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

# Configure static IP
configure_static_network() {
    echo ""
    echo -e "${BOLD}🔧 Static IP Configuration${NC}"
    echo "=========================="
    echo ""
    
    # Get static IP configuration
    while true; do
        read -p "Enter OMA IP Address: " STATIC_IP
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
        echo -e "${YELLOW}Configuration cancelled.${NC}"
        sleep 2
    fi
}

# Apply static network configuration
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
        case $netmask in
            "255.255.255.0") CIDR="24" ;;
            "255.255.0.0") CIDR="16" ;;
            "255.0.0.0") CIDR="8" ;;
            *) CIDR="24" ;;
        esac
    fi
    
    # Create netplan configuration
    cat > /tmp/01-static-config.yaml << EOF
network:
  version: 2
  renderer: networkd
  ethernets:
    $OMA_INTERFACE:
      dhcp4: false
      addresses:
        - $static_ip/$CIDR
      gateway4: $gateway
      nameservers:
        addresses:
          - $dns
EOF
    
    # Backup and apply
    sudo cp /etc/netplan/01-netcfg.yaml /etc/netplan/01-netcfg.yaml.backup 2>/dev/null || true
    sudo cp /tmp/01-static-config.yaml /etc/netplan/01-netcfg.yaml
    
    echo -e "${YELLOW}⚠️  Applying network configuration...${NC}"
    echo -e "${CYAN}OMA will be accessible at: ${BOLD}http://$static_ip:3001${NC}"
    echo ""
    read -p "Press Enter to apply configuration and reboot..."
    
    sudo netplan apply
    sleep 3
    sudo reboot
}

# Show detailed service status
show_service_details() {
    clear
    echo -e "${BLUE}${BOLD}📊 OSSEA-Migrate Service Details${NC}"
    echo "===================================="
    echo ""
    
    get_network_info
    check_services
    check_service_health
    
    echo -e "${YELLOW}🌐 Network Information:${NC}"
    echo -e "   IP Address: ${BOLD}${OMA_IP}${NC}"
    echo -e "   Interface: ${BOLD}${OMA_INTERFACE}${NC}"
    echo -e "   Gateway: ${BOLD}${OMA_GATEWAY}${NC}"
    echo -e "   DNS: ${BOLD}${OMA_DNS}${NC}"
    echo -e "   Configuration: ${BOLD}${DHCP_STATUS}${NC}"
    echo ""
    
    echo -e "${YELLOW}🔍 Detailed Service Information:${NC}"
    echo ""
    echo -e "${CYAN}OMA API Service (oma-api.service):${NC}"
    systemctl status oma-api.service --no-pager -l | head -8
    echo -e "   Health: ${OMA_API_HEALTH}"
    echo ""
    
    echo -e "${CYAN}Volume Daemon (volume-daemon.service):${NC}"
    systemctl status volume-daemon.service --no-pager -l | head -8
    echo -e "   Health: ${VOLUME_DAEMON_HEALTH}"
    echo ""
    
    echo -e "${CYAN}Migration GUI (migration-gui.service):${NC}"
    systemctl status migration-gui.service --no-pager -l | head -8
    echo -e "   Health: ${GUI_HEALTH}"
    echo ""
    
    read -p "Press Enter to return to main menu..."
}

# Main menu
main_menu() {
    while true; do
        show_main_interface
        
        echo -e "${BOLD}🔧 Configuration Options:${NC}"
        echo "   1. Configure network settings"
        echo "   2. View detailed service status"
        echo "   3. Access OSSEA-Migrate GUI"
        echo "   4. Restart services"
        echo "   5. Admin shell access"
        echo "   6. Reboot system"
        echo ""
        echo -e "${CYAN}(Press Ctrl+C to restart wizard)${NC}"
        read -p "Select option (1-6): " choice
        
        case $choice in
            1)
                configure_static_network
                ;;
            2)
                show_service_details
                ;;
            3)
                echo ""
                echo -e "${GREEN}🌐 Opening OSSEA-Migrate GUI...${NC}"
                echo -e "${CYAN}Access the web interface at: ${BOLD}http://${OMA_IP}:3001${NC}"
                echo ""
                echo -e "${YELLOW}If GUI is not accessible:${NC}"
                echo "   - Check service status (option 2)"
                echo "   - Restart services (option 4)"
                echo "   - Verify network connectivity"
                echo ""
                read -p "Press Enter to continue..."
                ;;
            4)
                echo ""
                echo -e "${YELLOW}🔄 Restarting OSSEA-Migrate services...${NC}"
                sudo systemctl restart oma-api.service
                sudo systemctl restart volume-daemon.service
                sudo systemctl restart migration-gui.service
                echo -e "${GREEN}✅ Services restarted${NC}"
                sleep 3
                ;;
            5)
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
                fi
                ;;
            6)
                echo -e "${YELLOW}Rebooting OMA system...${NC}"
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
echo "🚀 Initializing OSSEA-Migrate OMA Configuration Wizard..."
main_menu






