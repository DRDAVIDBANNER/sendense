#!/bin/bash
# VMA Setup Wizard - Clean Production Version
# Simple, working wizard for VMA configuration

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration
VMA_CONFIG="/opt/vma/vma-config.conf"
TUNNEL_SERVICE="vma-ssh-tunnel.service"

# Functions
validate_ip() {
    local ip="$1"
    [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]] && return 0 || return 1
}

get_oma_ip() {
    if [ -f "$VMA_CONFIG" ]; then
        grep "OMA_HOST=" "$VMA_CONFIG" 2>/dev/null | cut -d= -f2 | tr -d ' '
    else
        echo "Not configured"
    fi
}

get_tunnel_status() {
    if systemctl is-active $TUNNEL_SERVICE >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Active${NC}"
    else
        echo -e "${RED}❌ Inactive${NC}"
    fi
}

get_api_status() {
    if systemctl is-active vma-api.service >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Active${NC}"
    else
        echo -e "${RED}❌ Inactive${NC}"
    fi
}

show_header() {
    clear
    echo -e "${BLUE}${BOLD}"
    cat << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                     OSSEA-Migrate - VMA Setup                   ║
║                  VMware Migration Appliance                      ║
╚══════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
    
    # Show current status
    echo -e "${CYAN}📊 Current Status:${NC}"
    echo -e "   OMA IP: ${BOLD}$(get_oma_ip)${NC}"
    echo -e "   SSH Tunnel: $(get_tunnel_status)"
    echo -e "   VMA API: $(get_api_status)"
    echo ""
}

configure_network() {
    show_header
    echo -e "${BOLD}🌐 Network Configuration${NC}"
    echo ""
    
    # Current network info
    VMA_IP=$(hostname -I | awk '{print $1}')
    VMA_INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    VMA_GATEWAY=$(ip route | grep default | awk '{print $3}' | head -1)
    
    echo -e "${YELLOW}Current Configuration:${NC}"
    echo -e "   IP Address: ${BOLD}$VMA_IP${NC}"
    echo -e "   Interface: ${BOLD}$VMA_INTERFACE${NC}"
    echo -e "   Gateway: ${BOLD}$VMA_GATEWAY${NC}"
    echo ""
    
    echo -e "${CYAN}Network Configuration Options:${NC}"
    echo "   1. Configure Static IP"
    echo "   2. Configure DHCP"
    echo "   3. Configure DNS"
    echo "   4. Back to Main Menu"
    echo ""
    read -p "Select option (1-4): " net_choice
    
    case $net_choice in
        1)
            echo ""
            read -p "Enter IP Address: " NEW_IP
            read -p "Enter Netmask (e.g., 255.255.255.0): " NETMASK
            read -p "Enter Gateway: " GATEWAY
            read -p "Enter DNS Server: " DNS
            
            # Create netplan config
            sudo tee /etc/netplan/01-netcfg.yaml >/dev/null << EOF
network:
  version: 2
  ethernets:
    $VMA_INTERFACE:
      dhcp4: no
      addresses: [$NEW_IP/24]
      gateway4: $GATEWAY
      nameservers:
        addresses: [$DNS]
EOF
            sudo netplan apply
            echo -e "${GREEN}✅ Static IP configured${NC}"
            sleep 2
            ;;
        2)
            sudo tee /etc/netplan/01-netcfg.yaml >/dev/null << EOF
network:
  version: 2
  ethernets:
    $VMA_INTERFACE:
      dhcp4: yes
EOF
            sudo netplan apply
            echo -e "${GREEN}✅ DHCP configured${NC}"
            sleep 2
            ;;
        3)
            read -p "Enter DNS Server: " DNS
            echo "nameserver $DNS" | sudo tee /etc/resolv.conf >/dev/null
            echo -e "${GREEN}✅ DNS configured${NC}"
            sleep 2
            ;;
        *)
            return
            ;;
    esac
}

set_oma_ip() {
    show_header
    echo -e "${BOLD}🔗 Set OMA Connection${NC}"
    echo ""
    
    while true; do
        read -p "Enter OMA IP Address: " OMA_IP
        
        if validate_ip "$OMA_IP"; then
            echo -e "${GREEN}✅ Valid IP: $OMA_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid IP format${NC}"
        fi
    done
    
    # Check if enrollment key exists
    if [ ! -f "/opt/vma/enrollment/vma_enrollment_key" ]; then
        echo ""
        echo -e "${YELLOW}⚠️  Warning: SSH enrollment key not found${NC}"
        echo -e "${YELLOW}    The tunnel will not work without enrollment${NC}"
        echo ""
        read -p "Continue anyway? (y/n): " confirm
        [[ "$confirm" != "y" ]] && return
    fi
    
    # Save config
    sudo tee "$VMA_CONFIG" >/dev/null << EOF
# VMA Configuration
OMA_HOST=$OMA_IP
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_TYPE=ssh
SETUP_DATE=$(date)
SETUP_VERSION=v2.0.0
EOF
    
    echo -e "${GREEN}✅ Configuration saved${NC}"
    
    # Update tunnel service
    if systemctl list-unit-files | grep -q "$TUNNEL_SERVICE"; then
        echo -e "${CYAN}🔄 Updating SSH tunnel service...${NC}"
        
        sudo tee /etc/systemd/system/$TUNNEL_SERVICE >/dev/null << EOF
[Unit]
Description=VMA SSH Tunnel to OMA
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=vma
Group=vma
ExecStart=/usr/local/bin/vma-tunnel-wrapper.sh $OMA_IP
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
        
        sudo systemctl daemon-reload
        sudo systemctl enable $TUNNEL_SERVICE
        sudo systemctl restart $TUNNEL_SERVICE
        
        sleep 2
        echo -e "${GREEN}✅ SSH tunnel service configured${NC}"
    else
        echo -e "${YELLOW}⚠️  Tunnel service not found - run deployment script${NC}"
    fi
    
    echo ""
    read -p "Press Enter to continue..."
}

test_connectivity() {
    show_header
    echo -e "${BOLD}🧪 Testing Connectivity${NC}"
    echo ""
    
    OMA_IP=$(get_oma_ip)
    
    if [ "$OMA_IP" = "Not configured" ]; then
        echo -e "${RED}❌ OMA IP not configured${NC}"
        echo ""
        read -p "Press Enter to continue..."
        return
    fi
    
    # Test 1: Ping gateway
    echo -e "${CYAN}1. Testing gateway connectivity...${NC}"
    GATEWAY=$(ip route | grep default | awk '{print $3}' | head -1)
    if ping -c 2 -W 2 $GATEWAY >/dev/null 2>&1; then
        echo -e "   ${GREEN}✅ Gateway reachable: $GATEWAY${NC}"
    else
        echo -e "   ${RED}❌ Gateway unreachable: $GATEWAY${NC}"
    fi
    
    # Test 2: OMA port 443
    echo -e "${CYAN}2. Testing OMA port 443...${NC}"
    if timeout 3 bash -c "echo >/dev/tcp/$OMA_IP/443" 2>/dev/null; then
        echo -e "   ${GREEN}✅ OMA port 443 reachable${NC}"
    else
        echo -e "   ${RED}❌ OMA port 443 not reachable${NC}"
    fi
    
    # Test 3: SSH Tunnel Service
    echo -e "${CYAN}3. Checking SSH tunnel service...${NC}"
    if systemctl is-active $TUNNEL_SERVICE >/dev/null 2>&1; then
        echo -e "   ${GREEN}✅ Tunnel service active${NC}"
        
        # Check NBD tunnel
        if ss -tlnp 2>/dev/null | grep -q ':10808'; then
            echo -e "   ${GREEN}✅ NBD forward tunnel (10808) established${NC}"
        else
            echo -e "   ${YELLOW}⚠️  NBD forward tunnel not detected${NC}"
        fi
    else
        echo -e "   ${RED}❌ Tunnel service inactive${NC}"
    fi
    
    # Test 4: VMA API
    echo -e "${CYAN}4. Testing VMA API...${NC}"
    if curl -s --connect-timeout 3 http://localhost:8081/api/v1/health >/dev/null 2>&1; then
        echo -e "   ${GREEN}✅ VMA API responding${NC}"
    else
        echo -e "   ${RED}❌ VMA API not responding${NC}"
    fi
    
    # Test 5: DNS
    echo -e "${CYAN}5. Testing DNS resolution...${NC}"
    if nslookup google.com >/dev/null 2>&1; then
        echo -e "   ${GREEN}✅ DNS working${NC}"
    else
        echo -e "   ${YELLOW}⚠️  DNS not working${NC}"
    fi
    
    echo ""
    read -p "Press Enter to continue..."
}

restart_services() {
    show_header
    echo -e "${BOLD}🔄 Restart Services${NC}"
    echo ""
    
    echo -e "${CYAN}Restarting VMA API...${NC}"
    if sudo systemctl restart vma-api.service; then
        echo -e "${GREEN}✅ VMA API restarted${NC}"
    else
        echo -e "${RED}❌ Failed to restart VMA API${NC}"
    fi
    
    echo -e "${CYAN}Restarting SSH Tunnel...${NC}"
    if sudo systemctl restart $TUNNEL_SERVICE 2>/dev/null; then
        echo -e "${GREEN}✅ SSH Tunnel restarted${NC}"
    else
        echo -e "${YELLOW}⚠️  Tunnel service not found or failed${NC}"
    fi
    
    sleep 2
    echo ""
    echo -e "${GREEN}✅ Services restart complete${NC}"
    echo ""
    read -p "Press Enter to continue..."
}

shell_access() {
    show_header
    echo -e "${BOLD}🔐 Shell Access${NC}"
    echo ""
    echo -e "${YELLOW}⚠️  This will exit the wizard and give you a shell${NC}"
    echo ""
    read -p "Continue? (y/n): " confirm
    
    if [ "$confirm" = "y" ]; then
        echo ""
        echo -e "${GREEN}Starting shell... (type 'exit' to return)${NC}"
        echo ""
        bash
    fi
}

# Main loop
while true; do
    show_header
    
    echo -e "${BOLD}🔧 VMA Configuration Menu${NC}"
    echo ""
    echo "   1. 🌐 Configure Network"
    echo "   2. 🔗 Set OMA IP/URL"
    echo "   3. 🧪 Test Connectivity"
    echo "   4. 🔄 Restart Services"
    echo "   5. 💻 Shell Access"
    echo "   6. 🚪 Exit"
    echo ""
    read -p "Select option (1-6): " choice
    
    case $choice in
        1) configure_network ;;
        2) set_oma_ip ;;
        3) test_connectivity ;;
        4) restart_services ;;
        5) shell_access ;;
        6) 
            echo ""
            echo -e "${GREEN}👋 Exiting VMA wizard${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}❌ Invalid option${NC}"
            sleep 1
            ;;
    esac
done
