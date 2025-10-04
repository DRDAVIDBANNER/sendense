#!/bin/bash
# VMA Setup Wizard - Dual Interface Support
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

# Load current configuration
load_config() {
    if [ -f "$VMA_CONFIG" ]; then
        source "$VMA_CONFIG" 2>/dev/null || true
    fi
}

# Save configuration
save_config() {
    cat > "$VMA_CONFIG" << EOF
OMA_MANAGER_HOST=${OMA_MANAGER_HOST:-}
OMA_VOLUME_TARGET_HOST=${OMA_VOLUME_TARGET_HOST:-}
TUNNEL_TYPE=ssh
USER_TYPE=vma
SETUP_DATE=$(date)
EOF
}

# Clear screen and show header
show_header() {
    clear
    echo -e "${BLUE}${BOLD}"
    cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                 OSSEA-Migrate - VMA Setup (Dual Interface)      ║
║                  VMware Migration Appliance                      ║
║                                                                  ║
║              🚀 Professional Migration Platform                  ║
╚══════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
    echo -e "${CYAN}VMA Setup Wizard - Dual Interface Configuration${NC}"
    echo ""
}

# Show current status
show_status() {
    load_config
    
    echo -e "${YELLOW}📡 Current Configuration:${NC}"
    echo "- VMA IP: $(hostname -I | awk '{print $1}')"
    echo "- OMA Manager: ${OMA_MANAGER_HOST:-${RED}Not Set${NC}}"
    echo "- OMA Volume Target: ${OMA_VOLUME_TARGET_HOST:-${RED}Not Set${NC}}"
    
    # Check SSH tunnel status
    if systemctl is-active vma-ssh-tunnel.service >/dev/null 2>&1; then
        echo -e "- SSH Tunnel: ${GREEN}Active${NC}"
    else
        echo -e "- SSH Tunnel: ${RED}Inactive${NC}"
    fi
    
    # Check VMA API status
    if systemctl is-active vma-api.service >/dev/null 2>&1; then
        echo -e "- VMA API: ${GREEN}Active${NC}"
    else
        echo -e "- VMA API: ${RED}Inactive${NC}"
    fi
    
    echo ""
}

# Configure network settings
configure_network() {
    echo -e "${BOLD}🔧 Network Configuration${NC}"
    echo ""
    
    echo "Current network settings:"
    ip addr show | grep -E "inet.*global" | head -3
    echo ""
    
    echo "Network configuration options:"
    echo "1. Keep current settings"
    echo "2. Configure static IP"
    echo "3. Configure DHCP"
    echo ""
    
    read -p "Select option [1-3]: " net_choice
    
    case $net_choice in
        1)
            echo "✅ Keeping current network settings"
            ;;
        2)
            configure_static_ip
            ;;
        3)
            configure_dhcp
            ;;
        *)
            echo "Invalid option"
            return 1
            ;;
    esac
}

configure_static_ip() {
    echo "Static IP Configuration:"
    read -p "Enter IP address (e.g., 10.0.100.233): " STATIC_IP
    read -p "Enter subnet mask (e.g., 24): " SUBNET_MASK
    read -p "Enter gateway (e.g., 10.0.100.1): " GATEWAY
    read -p "Enter DNS server (e.g., 8.8.8.8): " DNS_SERVER
    
    INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    
    # Create netplan config
    sudo tee /etc/netplan/01-vma-static.yaml > /dev/null << EOF
network:
  version: 2
  ethernets:
    $INTERFACE:
      addresses:
        - $STATIC_IP/$SUBNET_MASK
      gateway4: $GATEWAY
      nameservers:
        addresses:
          - $DNS_SERVER
EOF
    
    sudo netplan apply
    echo "✅ Static IP configured"
}

configure_dhcp() {
    INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    
    sudo tee /etc/netplan/01-vma-dhcp.yaml > /dev/null << EOF
network:
  version: 2
  ethernets:
    $INTERFACE:
      dhcp4: true
EOF
    
    sudo netplan apply
    echo "✅ DHCP configured"
}

# Configure OMA Manager IP (SSH tunnel)
configure_oma_manager() {
    echo -e "${BOLD}🔧 OMA Manager Configuration${NC}"
    echo ""
    echo "The OMA Manager handles API calls, enrollment, and management traffic."
    echo "This will be accessed via SSH tunnel on port 443."
    echo ""
    
    read -p "Enter OMA Manager IP: " OMA_MANAGER_IP
    
    if [[ ! $OMA_MANAGER_IP =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo -e "${RED}❌ Invalid IP address format${NC}"
        return 1
    fi
    
    # Update VMA config
    load_config
    OMA_MANAGER_HOST="$OMA_MANAGER_IP"
    save_config
    
    # Update SSH tunnel service
    if [ -f "$TUNNEL_SERVICE" ]; then
        sudo systemctl stop vma-ssh-tunnel.service 2>/dev/null || true
        sudo sed -i "s|ExecStart=.*|ExecStart=/usr/local/bin/vma-ssh-tunnel-wrapper.sh $OMA_MANAGER_IP|" "$TUNNEL_SERVICE"
        sudo systemctl daemon-reload
    fi
    
    echo -e "${GREEN}✅ OMA Manager IP set to: $OMA_MANAGER_IP${NC}"
}

# Configure OMA Volume Target IP (Direct NBD)
configure_oma_volume_target() {
    echo -e "${BOLD}🔧 OMA Volume Target Configuration${NC}"
    echo ""
    echo "The OMA Volume Target handles high-speed data migration traffic."
    echo "This will be accessed directly (no tunnel) for maximum performance."
    echo ""
    
    read -p "Enter OMA Volume Target IP: " OMA_VOLUME_TARGET_IP
    
    if [[ ! $OMA_VOLUME_TARGET_IP =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo -e "${RED}❌ Invalid IP address format${NC}"
        return 1
    fi
    
    # Update VMA config
    load_config
    OMA_VOLUME_TARGET_HOST="$OMA_VOLUME_TARGET_IP"
    save_config
    
    # Update VMA API environment
    sudo mkdir -p /etc/systemd/system/vma-api.service.d/
    sudo tee /etc/systemd/system/vma-api.service.d/dual-interface.conf > /dev/null << EOF
[Service]
Environment=OMA_MANAGER_HOST=$OMA_MANAGER_HOST
Environment=OMA_VOLUME_TARGET_HOST=$OMA_VOLUME_TARGET_IP
Environment=CLOUDSTACK_API_URL=http://localhost:8082
Environment=NBD_TARGET_HOST=$OMA_VOLUME_TARGET_IP
EOF
    
    echo -e "${GREEN}✅ OMA Volume Target IP set to: $OMA_VOLUME_TARGET_IP${NC}"
}

# Test connectivity to both interfaces
test_connectivity() {
    echo -e "${BOLD}🔧 Connectivity Testing${NC}"
    echo ""
    
    load_config
    
    if [ -z "${OMA_MANAGER_HOST:-}" ]; then
        echo -e "${RED}❌ OMA Manager IP not configured${NC}"
        return 1
    fi
    
    if [ -z "${OMA_VOLUME_TARGET_HOST:-}" ]; then
        echo -e "${RED}❌ OMA Volume Target IP not configured${NC}"
        return 1
    fi
    
    echo "Testing OMA Manager connectivity (SSH tunnel)..."
    if timeout 5 nc -z "$OMA_MANAGER_HOST" 443 2>/dev/null; then
        echo -e "${GREEN}✅ OMA Manager reachable on port 443${NC}"
    else
        echo -e "${RED}❌ OMA Manager not reachable on port 443${NC}"
    fi
    
    echo ""
    echo "Testing OMA Volume Target connectivity (Direct NBD)..."
    if timeout 5 nc -z "$OMA_VOLUME_TARGET_HOST" 10809 2>/dev/null; then
        echo -e "${GREEN}✅ OMA Volume Target reachable on port 10809${NC}"
    else
        echo -e "${RED}❌ OMA Volume Target not reachable on port 10809${NC}"
    fi
    
    echo ""
    echo "Testing SSH tunnel establishment..."
    if systemctl is-active vma-ssh-tunnel.service >/dev/null 2>&1; then
        if timeout 5 curl -s http://localhost:8082/health >/dev/null 2>&1; then
            echo -e "${GREEN}✅ SSH tunnel working - OMA API accessible${NC}"
        else
            echo -e "${YELLOW}⚠️  SSH tunnel active but OMA API not responding${NC}"
        fi
    else
        echo -e "${RED}❌ SSH tunnel not active${NC}"
    fi
}

# Restart services
restart_services() {
    echo -e "${BOLD}🔧 Service Management${NC}"
    echo ""
    
    echo "Restarting VMA services..."
    
    echo "Stopping services..."
    sudo systemctl stop vma-ssh-tunnel.service 2>/dev/null || true
    sudo systemctl stop vma-api.service 2>/dev/null || true
    
    sleep 2
    
    echo "Starting SSH tunnel..."
    sudo systemctl start vma-ssh-tunnel.service
    sleep 3
    
    echo "Starting VMA API..."
    sudo systemctl start vma-api.service
    sleep 2
    
    echo ""
    echo "Service status:"
    if systemctl is-active vma-ssh-tunnel.service >/dev/null 2>&1; then
        echo -e "- SSH Tunnel: ${GREEN}Active${NC}"
    else
        echo -e "- SSH Tunnel: ${RED}Failed${NC}"
    fi
    
    if systemctl is-active vma-api.service >/dev/null 2>&1; then
        echo -e "- VMA API: ${GREEN}Active${NC}"
    else
        echo -e "- VMA API: ${RED}Failed${NC}"
    fi
    
    echo -e "${GREEN}✅ Service restart completed${NC}"
}

# Protected shell access
shell_access() {
    echo -e "${BOLD}🔧 Protected Shell Access${NC}"
    echo ""
    echo -e "${YELLOW}⚠️  This provides direct shell access for advanced configuration.${NC}"
    echo -e "${YELLOW}⚠️  Use with caution - incorrect changes can break the VMA.${NC}"
    echo ""
    
    read -p "Are you sure you want shell access? (yes/no): " confirm
    if [ "$confirm" = "yes" ]; then
        echo -e "${CYAN}Starting protected shell session...${NC}"
        echo -e "${CYAN}Type 'exit' to return to the wizard.${NC}"
        echo ""
        bash
    else
        echo "Shell access cancelled"
    fi
}

# Main menu loop
main_menu() {
    while true; do
        show_header
        show_status
        
        echo -e "${BOLD}📋 Configuration Options:${NC}"
        echo ""
        echo "1. Configure Network (IP, DNS, Gateway)"
        echo "2. Set OMA Manager IP (API/Management)"
        echo "3. Set OMA Volume Target IP (NBD/Data)"
        echo "4. Test Connectivity (Both interfaces)"
        echo "5. Restart Services (VMA API, SSH Tunnel)"
        echo "6. Shell Access (Protected)"
        echo "7. Exit Wizard"
        echo ""
        
        read -p "Select option [1-7]: " choice
        echo ""
        
        case $choice in
            1)
                configure_network
                ;;
            2)
                configure_oma_manager
                ;;
            3)
                configure_oma_volume_target
                ;;
            4)
                test_connectivity
                ;;
            5)
                restart_services
                ;;
            6)
                shell_access
                ;;
            7)
                echo -e "${GREEN}👋 Exiting VMA Setup Wizard${NC}"
                exit 0
                ;;
            *)
                echo -e "${RED}❌ Invalid option. Please select 1-7.${NC}"
                ;;
        esac
        
        echo ""
        read -p "Press Enter to continue..."
    done
}

# Initialize configuration file if it doesn't exist
if [ ! -f "$VMA_CONFIG" ]; then
    touch "$VMA_CONFIG"
    chmod 644 "$VMA_CONFIG"
fi

# Start the wizard
main_menu


