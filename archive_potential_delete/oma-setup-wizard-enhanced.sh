#!/bin/bash
# Enhanced OMA Setup Wizard - Professional deployment interface for OSSEA-Migrate OMA
# Features: Super admin access control, VMA status monitoring, network configuration

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
OMA_CONFIG="/opt/ossea-migrate/oma-config.conf"
VENDOR_ACCESS_FILE="/opt/ossea-migrate/.vendor-access"

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

trap 'restart_wizard' INT TERM QUIT HUP
trap 'restart_wizard' TSTP STOP

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
        echo "$(date): Vendor access granted" | sudo tee -a "$VENDOR_ACCESS_FILE" > /dev/null
        return 0
    else
        return 1
    fi
}

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

# Check VMA status
check_vma_status() {
    VMA_COUNT=0
    VMA_STATUS="❌ No VMA Connected"
    VMA_DETAILS=""
    
    # Check if VMA tunnel is active
    if systemctl is-active ssh-tunnel.service > /dev/null 2>&1 || netstat -tln | grep -q ":9081" 2>/dev/null; then
        # Try to connect to VMA API through tunnel
        if curl -s --connect-timeout 3 http://localhost:9081/api/v1/health > /dev/null 2>&1; then
            VMA_COUNT=1
            VMA_STATUS="✅ VMA Connected"
            
            # Get VMA details
            VMA_INFO=$(curl -s --connect-timeout 3 http://localhost:9081/api/v1/health 2>/dev/null || echo "{}")
            VMA_VERSION=$(echo "$VMA_INFO" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown")
            VMA_DETAILS="Version: $VMA_VERSION"
        else
            VMA_STATUS="🟡 VMA Tunnel Active (API not responding)"
        fi
    fi
    
    # Check for multiple VMA connections (future enhancement)
    # This would check for multiple tunnel services or VMA registrations
    if [ -f "/opt/ossea-migrate/vma-registry.conf" ]; then
        REGISTERED_VMAS=$(cat /opt/ossea-migrate/vma-registry.conf | wc -l 2>/dev/null || echo "0")
        if [ "$REGISTERED_VMAS" -gt 1 ]; then
            VMA_COUNT=$REGISTERED_VMAS
            VMA_STATUS="✅ Multiple VMAs ($VMA_COUNT registered)"
        fi
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
        
        # Check VMware credentials functionality
        if curl -s --connect-timeout 3 http://localhost:8082/api/v1/vmware-credentials > /dev/null 2>&1; then
            VMWARE_CREDS_STATUS="✅ Available"
        else
            VMWARE_CREDS_STATUS="❌ Not Available"
        fi
    else
        OMA_API_HEALTH="❌ Not responding"
        VMWARE_CREDS_STATUS="❌ API Down"
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
        
        # Check database statistics
        VM_COUNT=$(mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM vm_replication_contexts;" 2>/dev/null | tail -1 || echo "0")
        JOB_COUNT=$(mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM replication_jobs;" 2>/dev/null | tail -1 || echo "0")
        CREDS_COUNT=$(mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM vmware_credentials;" 2>/dev/null | tail -1 || echo "0")
    else
        DB_HEALTH="❌ Not responding"
        VM_COUNT="N/A"
        JOB_COUNT="N/A" 
        CREDS_COUNT="N/A"
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
    echo -e "${CYAN}Professional interface for network configuration and service management.${NC}"
    echo ""
    
    # Get current information
    get_network_info
    check_services
    check_service_health
    check_vma_status
    
    # Display network information
    echo -e "${YELLOW}📡 Network Configuration:${NC}"
    echo -e "   ┌─────────────────────────────────────────────────┐"
    echo -e "   │ IP Address: ${BOLD}${OMA_IP}${NC}                         │"
    echo -e "   │ Interface: ${BOLD}${OMA_INTERFACE}${NC}                            │"
    echo -e "   │ Gateway: ${BOLD}${OMA_GATEWAY}${NC}                       │"
    echo -e "   │ DNS Server: ${BOLD}${OMA_DNS}${NC}                        │"
    echo -e "   │ Configuration: ${BOLD}${DHCP_STATUS}${NC}                         │"
    echo -e "   └─────────────────────────────────────────────────┘"
    echo ""
    
    # Display VMA status
    echo -e "${YELLOW}🖥️  VMA Status:${NC}"
    echo -e "   ┌─────────────────────────────────────────────────┐"
    echo -e "   │ Status: ${BOLD}${VMA_STATUS}${NC}                    │"
    if [ "$VMA_COUNT" -gt 0 ]; then
        echo -e "   │ ${VMA_DETAILS}                                    │"
    fi
    echo -e "   └─────────────────────────────────────────────────┘"
    echo ""
    
    # Display service status
    echo -e "${YELLOW}🚀 Service Status:${NC}"
    echo -e "   ┌─────────────────────────────────────────────────┐"
    echo -e "   │ $(get_status_icon $OMA_API_STATUS) OMA API Service        [${OMA_API_STATUS^^}]      │"
    echo -e "   │ $(get_status_icon $VOLUME_DAEMON_STATUS) Volume Daemon          [${VOLUME_DAEMON_STATUS^^}]      │"
    echo -e "   │ $(get_status_icon $NBD_SERVER_STATUS) NBD Server             [${NBD_SERVER_STATUS^^}]      │"
    echo -e "   │ $(get_status_icon $MARIADB_STATUS) MariaDB Database       [${MARIADB_STATUS^^}]      │"
    echo -e "   │ $(get_status_icon $GUI_STATUS) Migration GUI          [${GUI_STATUS^^}]      │"
    echo -e "   └─────────────────────────────────────────────────┘"
    echo ""
    
    # Display system statistics
    echo -e "${YELLOW}📊 System Statistics:${NC}"
    echo -e "   ┌─────────────────────────────────────────────────┐"
    echo -e "   │ VM Contexts: ${BOLD}${VM_COUNT}${NC}                             │"
    echo -e "   │ Migration Jobs: ${BOLD}${JOB_COUNT}${NC}                          │"
    echo -e "   │ VMware Credentials: ${BOLD}${CREDS_COUNT}${NC}                    │"
    echo -e "   │ VMware Creds API: ${VMWARE_CREDS_STATUS}             │"
    echo -e "   └─────────────────────────────────────────────────┘"
    echo ""
    
    # Display access information
    echo -e "${YELLOW}🌐 Access Information:${NC}"
    echo -e "   Web Interface: ${BOLD}http://${OMA_IP}:3001${NC} (${GUI_HEALTH})"
    echo -e "   API Endpoint: ${BOLD}http://${OMA_IP}:8082${NC} (${OMA_API_HEALTH})"
    echo -e "   VMware Credentials: ${BOLD}http://${OMA_IP}:3001/settings/ossea${NC}"
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

# Enhanced network configuration
configure_network() {
    clear
    echo -e "${BLUE}${BOLD}🔧 Network Configuration${NC}"
    echo "=============================="
    echo ""
    
    echo -e "${YELLOW}Current Configuration:${NC}"
    echo -e "   IP: ${BOLD}${OMA_IP}${NC}"
    echo -e "   Gateway: ${BOLD}${OMA_GATEWAY}${NC}"
    echo -e "   DNS: ${BOLD}${OMA_DNS}${NC}"
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
        1) configure_static_network ;;
        2) configure_dhcp_network ;;
        3) configure_dns_settings ;;
        4) return ;;
        *) 
            echo -e "${RED}Invalid option${NC}"
            sleep 2
            configure_network
            ;;
    esac
}

# Configure DHCP network
configure_dhcp_network() {
    echo ""
    echo -e "${BOLD}🔧 DHCP Configuration${NC}"
    echo "====================="
    echo ""
    
    echo -e "${YELLOW}⚠️  This will configure the OMA to use DHCP for IP assignment.${NC}"
    echo -e "${YELLOW}The OMA IP address may change after reboot.${NC}"
    echo ""
    read -p "Continue with DHCP configuration? (y/N): " apply_dhcp
    
    if [[ $apply_dhcp =~ ^[Yy]$ ]]; then
        # Create DHCP netplan configuration
        cat > /tmp/01-dhcp-config.yaml << EOF
network:
  version: 2
  renderer: networkd
  ethernets:
    $OMA_INTERFACE:
      dhcp4: true
      dhcp6: false
EOF
        
        # Backup and apply
        sudo cp /etc/netplan/01-netcfg.yaml /etc/netplan/01-netcfg.yaml.backup 2>/dev/null || true
        sudo cp /tmp/01-dhcp-config.yaml /etc/netplan/01-netcfg.yaml
        
        echo -e "${YELLOW}⚠️  Applying DHCP configuration...${NC}"
        echo -e "${CYAN}OMA will receive IP from DHCP server after reboot.${NC}"
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

# Configure DNS settings
configure_dns_settings() {
    echo ""
    echo -e "${BOLD}🔧 DNS Configuration${NC}"
    echo "===================="
    echo ""
    
    echo -e "Current DNS: ${BOLD}${OMA_DNS}${NC}"
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
        # Update DNS in current netplan config
        sudo sed -i "s/- $OMA_DNS/- $new_dns/g" /etc/netplan/01-netcfg.yaml 2>/dev/null || true
        
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

# Show detailed VMA status
show_vma_details() {
    clear
    echo -e "${BLUE}${BOLD}🖥️  VMA Status Details${NC}"
    echo "========================="
    echo ""
    
    check_vma_status
    
    echo -e "${YELLOW}VMA Connection Status:${NC}"
    echo -e "   Status: ${BOLD}${VMA_STATUS}${NC}"
    echo -e "   Count: ${BOLD}${VMA_COUNT}${NC}"
    if [ -n "$VMA_DETAILS" ]; then
        echo -e "   Details: ${BOLD}${VMA_DETAILS}${NC}"
    fi
    echo ""
    
    # Check tunnel services
    echo -e "${YELLOW}🔗 Tunnel Services:${NC}"
    if systemctl is-active ssh-tunnel.service > /dev/null 2>&1; then
        echo -e "   SSH Tunnel: ${GREEN}✅ Active${NC}"
        systemctl status ssh-tunnel.service --no-pager -l | head -5
    else
        echo -e "   SSH Tunnel: ${RED}❌ Inactive${NC}"
    fi
    echo ""
    
    # Check VMA API connectivity
    echo -e "${YELLOW}🔍 VMA API Connectivity:${NC}"
    if curl -s --connect-timeout 3 http://localhost:9081/api/v1/health > /dev/null 2>&1; then
        echo -e "   VMA API: ${GREEN}✅ Reachable via tunnel${NC}"
        VMA_HEALTH=$(curl -s --connect-timeout 3 http://localhost:9081/api/v1/health)
        echo -e "   Health: ${GREEN}$VMA_HEALTH${NC}"
    else
        echo -e "   VMA API: ${RED}❌ Not reachable${NC}"
        echo -e "   Tunnel Port 9081: $(netstat -tln | grep :9081 > /dev/null && echo '✅ Listening' || echo '❌ Not listening')"
    fi
    echo ""
    
    # Future: Multiple VMA registry
    if [ -f "/opt/ossea-migrate/vma-registry.conf" ]; then
        echo -e "${YELLOW}📋 Registered VMAs:${NC}"
        cat /opt/ossea-migrate/vma-registry.conf 2>/dev/null || echo "   No VMAs registered"
    else
        echo -e "${YELLOW}📋 VMA Registry:${NC}"
        echo -e "   Registry file not found (single VMA mode)"
    fi
    echo ""
    
    read -p "Press Enter to return to main menu..."
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
    
    echo -e "${YELLOW}🔍 Detailed Service Information:${NC}"
    echo ""
    echo -e "${CYAN}OMA API Service (oma-api.service):${NC}"
    echo -e "   Status: $(systemctl is-active oma-api.service 2>/dev/null || echo 'inactive')"
    echo -e "   Health: ${OMA_API_HEALTH}"
    echo -e "   VMware Credentials: ${VMWARE_CREDS_STATUS}"
    echo -e "   Uptime: $(systemctl show oma-api.service -p ActiveEnterTimestamp --value 2>/dev/null | cut -d' ' -f1-2 || echo 'Unknown')"
    echo ""
    
    echo -e "${CYAN}Volume Daemon (volume-daemon.service):${NC}"
    echo -e "   Status: $(systemctl is-active volume-daemon.service 2>/dev/null || echo 'inactive')"
    echo -e "   Health: ${VOLUME_DAEMON_HEALTH}"
    echo -e "   Uptime: $(systemctl show volume-daemon.service -p ActiveEnterTimestamp --value 2>/dev/null | cut -d' ' -f1-2 || echo 'Unknown')"
    echo ""
    
    echo -e "${CYAN}Migration GUI (migration-gui.service):${NC}"
    echo -e "   Status: $(systemctl is-active migration-gui.service 2>/dev/null || echo 'inactive')"
    echo -e "   Health: ${GUI_HEALTH}"
    echo -e "   Uptime: $(systemctl show migration-gui.service -p ActiveEnterTimestamp --value 2>/dev/null | cut -d' ' -f1-2 || echo 'Unknown')"
    echo ""
    
    echo -e "${CYAN}Database (mariadb.service):${NC}"
    echo -e "   Status: $(systemctl is-active mariadb.service 2>/dev/null || echo 'inactive')"
    echo -e "   Health: ${DB_HEALTH}"
    echo -e "   VM Contexts: ${VM_COUNT}"
    echo -e "   Migration Jobs: ${JOB_COUNT}"
    echo -e "   VMware Credentials: ${CREDS_COUNT}"
    echo ""
    
    echo -e "${YELLOW}Press Enter to return to main menu, or 'q' to quit...${NC}"
    read -t 30 response || echo -e "\n${YELLOW}Timeout - returning to menu${NC}"
}

# Vendor shell access with enhanced security
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
            echo -e "${CYAN}Entering administrative shell...${NC}"
            echo ""
            echo "OSSEA-Migrate OMA - Vendor Shell Access"
            echo "======================================="
            echo "Access granted: $(date)"
            echo "User: Vendor Support"
            echo "System: $(hostname) ($(hostname -I | awk '{print $1}'))"
            echo ""
            echo "Type 'exit' to return to setup wizard"
            echo ""
            
            # Set vendor environment
            export PS1="[VENDOR@OMA \W]$ "
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
                echo "$(date): Failed vendor access attempt from $(who am i)" | sudo tee -a /var/log/vendor-access.log > /dev/null
                
                sleep 5
                return
            fi
        fi
    done
}

# Restart services
restart_services() {
    echo ""
    echo -e "${YELLOW}🔄 Restarting OSSEA-Migrate services...${NC}"
    echo ""
    
    services=("mariadb" "volume-daemon" "nbd-server" "oma-api" "migration-gui")
    
    for service in "${services[@]}"; do
        echo -e "   Restarting ${service}..."
        sudo systemctl restart "${service}.service"
        sleep 2
        
        if systemctl is-active "${service}.service" > /dev/null 2>&1; then
            echo -e "   ${GREEN}✅ ${service} restarted successfully${NC}"
        else
            echo -e "   ${RED}❌ ${service} failed to restart${NC}"
        fi
    done
    
    echo ""
    echo -e "${GREEN}✅ Service restart completed${NC}"
    sleep 3
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
        echo "   1. Network Configuration (IP, DNS, Gateway)"
        echo "   2. View VMA Status & Connectivity"
        echo "   3. View Detailed Service Status"
        echo "   4. Access OSSEA-Migrate GUI"
        echo "   5. Restart All Services"
        echo "   6. Vendor Shell Access (Support Only)"
        echo "   7. Reboot System"
        echo ""
        echo -e "${CYAN}🔒 Note: Shell access restricted to vendor support personnel${NC}"
        echo -e "${CYAN}📡 Use Ctrl+] to interrupt (wizard will restart automatically)${NC}"
        echo ""
        
        read -p "Select option (1-7): " choice
        
        case $choice in
            1)
                configure_network
                ;;
            2)
                show_vma_details
                ;;
            3)
                show_service_details
                ;;
            4)
                echo ""
                echo -e "${GREEN}🌐 Opening OSSEA-Migrate GUI...${NC}"
                echo -e "${CYAN}Access the web interface at: ${BOLD}http://${OMA_IP}:3001${NC}"
                echo -e "${CYAN}VMware Credentials: ${BOLD}http://${OMA_IP}:3001/settings/ossea${NC}"
                echo ""
                echo -e "${YELLOW}If GUI is not accessible:${NC}"
                echo "   - Check service status (option 3)"
                echo "   - Restart services (option 5)"
                echo "   - Verify network connectivity (option 1)"
                echo ""
                read -p "Press Enter to continue..."
                ;;
            5)
                restart_services
                ;;
            6)
                vendor_shell_access
                ;;
            7)
                echo ""
                echo -e "${YELLOW}🔄 Rebooting OMA system...${NC}"
                echo -e "${CYAN}System will restart and wizard will load automatically.${NC}"
                echo ""
                read -p "Press Enter to reboot..."
                sudo reboot
                ;;
            *)
                echo -e "${RED}Invalid option. Please select 1-7.${NC}"
                sleep 2
                ;;
        esac
    done
}

# Initialize and run
echo "🚀 Initializing Enhanced OSSEA-Migrate OMA Configuration Wizard..."
echo "🔒 Security: Shell access restricted to vendor support"
echo "📡 Features: Network config, VMA status, service management"
sleep 2

# Ensure we're running with proper permissions (can be root for systemd service)
if [ "$(whoami)" != "oma_admin" ] && [ "$(whoami)" != "oma" ] && [ "$(whoami)" != "root" ]; then
    echo -e "${RED}❌ Wizard must run as oma_admin, oma, or root user${NC}"
    exit 1
fi

main_menu
