#!/bin/bash
# VMA Setup Wizard - Simple version with enrollment
# Professional deployment interface for MigrateKit OSSEA VMA

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

# Functions
validate_ip() {
    local ip="$1"
    if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        IFS='.' read -ra ADDR <<< "$ip"
        for i in "${ADDR[@]}"; do
            if [[ $i -lt 0 ]] || [[ $i -gt 255 ]]; then
                return 1
            fi
        done
        return 0
    else
        return 1
    fi
}

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

echo -e "${CYAN}Welcome to the OSSEA-Migrate VMA Setup Wizard${NC}"
echo -e "${CYAN}This wizard will configure your VMA to connect to the OMA appliance.${NC}"
echo ""

# Show current configuration if it exists
if [ -f "$VMA_CONFIG" ]; then
    CURRENT_OMA_IP=$(grep "OMA_HOST=" "$VMA_CONFIG" 2>/dev/null | cut -d= -f2 || echo "Not configured")
    echo -e "${YELLOW}📋 Current Configuration:${NC}"
    echo -e "   Current OMA IP: ${BOLD}$CURRENT_OMA_IP${NC}"
    echo -e "   Tunnel Status: ${BOLD}$(systemctl is-active vma-tunnel-enhanced-v2.service 2>/dev/null)${NC}"
    echo ""
fi

# Main menu
while true; do
    echo -e "${BOLD}🔧 VMA Setup Options${NC}"
    echo ""
    echo "   0. 🔐 VMA Enrollment (NEW VMA - requires pairing code from admin)"
    echo "   1. 🔧 Manual OMA Configuration (Existing VMA or troubleshooting)"
    echo "   2. 📊 Show Current Status"
    echo "   3. 🔄 Restart Services"
    echo "   4. 🌐 Test Connectivity"
    echo "   5. 🚪 Exit"
    echo ""
    read -p "Select option (0-5): " choice
    
    case $choice in
        0)
            echo ""
            echo -e "${CYAN}🔐 Starting VMA enrollment workflow...${NC}"
            echo ""
            if [ -f "/opt/vma/vma-enrollment.sh" ]; then
                /opt/vma/vma-enrollment.sh
            else
                echo -e "${RED}❌ Enrollment script not found${NC}"
                echo "Please ensure /opt/vma/vma-enrollment.sh is installed"
                read -p "Press Enter to continue..."
            fi
            ;;
        1)
            manual_configuration
            ;;
        2)
            show_status
            ;;
        3)
            restart_services
            ;;
        4)
            test_connectivity_menu
            ;;
        5)
            echo ""
            echo -e "${GREEN}👋 Exiting VMA setup wizard${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}❌ Invalid option. Please select 0-5.${NC}"
            sleep 2
            ;;
    esac
done

manual_configuration() {
    echo ""
    echo -e "${BOLD}🔧 Manual OMA Configuration${NC}"
    
    # Get OMA IP
    while true; do
        echo ""
        read -p "Enter OMA IP Address: " OMA_IP
        
        if validate_ip "$OMA_IP"; then
            echo -e "${GREEN}✅ Valid IP format: $OMA_IP${NC}"
            break
        else
            echo -e "${RED}❌ Invalid IP format${NC}"
        fi
    done
    
    # Update configuration
    cat > "$VMA_CONFIG" << EOF
# VMA Configuration - Manual Setup
OMA_HOST=$OMA_IP
OMA_PORT=443
VMA_API_PORT=8081
TUNNEL_LOCAL_PORT=9081
SETUP_DATE=$(date)
SETUP_VERSION=v2.0.0-manual
SETUP_METHOD=manual
EOF
    
    # Set tunnel environment
    sudo systemctl set-environment OMA_HOST="$OMA_IP"
    
    # Restart services
    echo -e "${CYAN}🔄 Restarting services...${NC}"
    sudo systemctl restart vma-tunnel-enhanced-v2.service
    sudo systemctl restart vma-api.service
    
    sleep 5
    echo -e "${GREEN}✅ Manual configuration complete${NC}"
    read -p "Press Enter to continue..."
}

show_status() {
    echo ""
    echo -e "${BOLD}📊 VMA System Status${NC}"
    echo ""
    
    if [ -f "$VMA_CONFIG" ]; then
        OMA_IP=$(grep "OMA_HOST=" "$VMA_CONFIG" | cut -d= -f2)
        echo -e "${CYAN}Configuration:${NC}"
        echo -e "   OMA IP: ${BOLD}$OMA_IP${NC}"
        echo -e "   Setup Method: ${BOLD}$(grep "SETUP_METHOD=" "$VMA_CONFIG" | cut -d= -f2 || echo 'Unknown')${NC}"
    fi
    
    echo -e "${CYAN}Services:${NC}"
    echo -e "   VMA API: ${BOLD}$(systemctl is-active vma-api.service)${NC}"
    echo -e "   Tunnel: ${BOLD}$(systemctl is-active vma-tunnel-enhanced-v2.service)${NC}"
    
    echo ""
    read -p "Press Enter to continue..."
}

restart_services() {
    echo ""
    echo -e "${BOLD}🔄 Restarting VMA Services${NC}"
    
    sudo systemctl restart vma-api.service
    sudo systemctl restart vma-tunnel-enhanced-v2.service
    
    sleep 3
    echo -e "${GREEN}✅ Services restarted${NC}"
    read -p "Press Enter to continue..."
}

test_connectivity_menu() {
    echo ""
    echo -e "${BOLD}🌐 Testing Connectivity${NC}"
    
    if [ -f "$VMA_CONFIG" ]; then
        OMA_IP=$(grep "OMA_HOST=" "$VMA_CONFIG" | cut -d= -f2)
        echo -e "${CYAN}Testing connection to OMA: $OMA_IP${NC}"
        
        if curl -s --connect-timeout 10 http://localhost:8082/health >/dev/null 2>&1; then
            echo -e "${GREEN}✅ OMA API accessible via tunnel${NC}"
        else
            echo -e "${RED}❌ OMA API not accessible${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No configuration found${NC}"
    fi
    
    echo ""
    read -p "Press Enter to continue..."
}
