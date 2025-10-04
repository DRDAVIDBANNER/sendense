#!/bin/bash
# deploy-sha-full.sh - Complete Sendense Hub Appliance (SHA) Deployment
# Version: v1.0.0
# Date: 2025-10-04
#
# This script performs a complete SHA deployment including:
# 1. System dependencies installation
# 2. MariaDB setup
# 3. Binary deployment
# 4. Service configuration
# 5. Database migrations
# 6. System validation

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEPLOYMENT_DIR="$PROJECT_ROOT/deployment/sha-appliance"

INSTALL_DIR="/opt/sendense"
BINARY_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sendense"
LOG_DIR="/var/log/sendense"
DATA_DIR="/var/lib/sendense"

# Database configuration
DB_USER="oma_user"
DB_PASS="oma_password"
DB_NAME="migratekit_oma"
DB_ROOT_PASS="sendense_root_$(date +%s)"  # Generate random root password

echo -e "${BLUE}🚀 Sendense Hub Appliance (SHA) - Full Deployment${NC}"
echo -e "${BLUE}=================================================${NC}"
echo ""
echo "📅 Date: $(date)"
echo "🖥️  Host: $(hostname)"
echo "👤 User: $(whoami)"
echo ""

# Check if running as root or with sudo
if [[ $EUID -ne 0 ]] && ! sudo -n true 2>/dev/null; then
   echo -e "${YELLOW}⚠️  This script requires sudo privileges${NC}"
   echo "   Please run with: sudo ./deploy-sha-full.sh"
   exit 1
fi

# Step 1: Install system dependencies
install_dependencies() {
    echo -e "${YELLOW}📦 Installing system dependencies...${NC}"
    
    # Update package lists
    sudo apt-get update -qq
    
    # Install required packages
    sudo apt-get install -y \
        mariadb-server \
        mariadb-client \
        qemu-utils \
        curl \
        wget \
        jq \
        net-tools \
        vim \
        htop
    
    echo -e "${GREEN}   ✅ System dependencies installed${NC}"
    echo ""
}

# Step 2: Configure MariaDB
configure_mariadb() {
    echo -e "${YELLOW}🗄️  Configuring MariaDB...${NC}"
    
    # Start MariaDB if not running
    sudo systemctl start mariadb || true
    sudo systemctl enable mariadb
    
    # Secure MariaDB installation (automated)
    sudo mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED BY '${DB_ROOT_PASS}';" || true
    sudo mysql -u root -p"${DB_ROOT_PASS}" -e "DELETE FROM mysql.user WHERE User='';" || true
    sudo mysql -u root -p"${DB_ROOT_PASS}" -e "DROP DATABASE IF EXISTS test;" || true
    sudo mysql -u root -p"${DB_ROOT_PASS}" -e "FLUSH PRIVILEGES;" || true
    
    # Create Sendense database and user
    echo "   📝 Creating database and user..."
    sudo mysql -u root -p"${DB_ROOT_PASS}" << EOF
CREATE DATABASE IF NOT EXISTS ${DB_NAME} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'localhost';
FLUSH PRIVILEGES;
EOF
    
    # Save root password for future reference
    echo "$DB_ROOT_PASS" | sudo tee /root/.mariadb_root_password > /dev/null
    sudo chmod 600 /root/.mariadb_root_password
    
    echo -e "${GREEN}   ✅ MariaDB configured${NC}"
    echo "      Database: $DB_NAME"
    echo "      User: $DB_USER"
    echo "      Root password saved to: /root/.mariadb_root_password"
    echo ""
}

# Step 3: Create directory structure
create_directories() {
    echo -e "${YELLOW}📁 Creating directory structure...${NC}"
    
    sudo mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    sudo mkdir -p "$DATA_DIR"/{backups,repositories,temp}
    sudo mkdir -p "$LOG_DIR"/{sha-api,volume-daemon}
    
    # Set permissions
    sudo chmod 755 "$INSTALL_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    sudo chmod 1777 "$DATA_DIR/temp"  # Sticky bit for temp
    
    echo -e "${GREEN}   ✅ Directory structure created${NC}"
    echo ""
}

# Step 4: Deploy binaries
deploy_binaries() {
    echo -e "${YELLOW}🔧 Deploying binaries...${NC}"
    
    cd "$DEPLOYMENT_DIR/binaries"
    
    # Find the latest binaries
    SHA_BINARY=$(ls -1 sendense-hub-*.* 2>/dev/null | grep -v "latest" | sort -V | tail -1)
    VOLUME_DAEMON_BINARY=$(ls -1 volume-daemon-*.* 2>/dev/null | grep -v "latest" | sort -V | tail -1)
    
    if [[ -z "$SHA_BINARY" ]]; then
        echo -e "${RED}❌ SHA API binary not found${NC}"
        echo "   Run: ./scripts/build-sha-binaries.sh first"
        exit 1
    fi
    
    if [[ -z "$VOLUME_DAEMON_BINARY" ]]; then
        echo -e "${RED}❌ Volume Daemon binary not found${NC}"
        echo "   Run: ./scripts/build-sha-binaries.sh first"
        exit 1
    fi
    
    echo "   📦 SHA API: $SHA_BINARY"
    echo "   📦 Volume Daemon: $VOLUME_DAEMON_BINARY"
    
    # Verify checksums
    if [[ -f "CHECKSUMS.sha256" ]]; then
        echo "   🔐 Verifying checksums..."
        sha256sum -c CHECKSUMS.sha256 --ignore-missing || {
            echo -e "${RED}❌ Checksum verification failed${NC}"
            exit 1
        }
    fi
    
    # Install binaries
    sudo cp "$SHA_BINARY" "$BINARY_DIR/sendense-hub"
    sudo cp "$VOLUME_DAEMON_BINARY" "$BINARY_DIR/volume-daemon"
    sudo chmod +x "$BINARY_DIR/sendense-hub" "$BINARY_DIR/volume-daemon"
    
    echo -e "${GREEN}   ✅ Binaries deployed${NC}"
    echo ""
}

# Step 5: Deploy configuration
deploy_configuration() {
    echo -e "${YELLOW}⚙️  Deploying configuration...${NC}"
    
    # Install systemd service files
    sudo cp "$DEPLOYMENT_DIR/configs/sendense-hub.service" /etc/systemd/system/
    sudo cp "$DEPLOYMENT_DIR/configs/volume-daemon.service" /etc/systemd/system/
    
    # Reload systemd
    sudo systemctl daemon-reload
    
    echo -e "${GREEN}   ✅ Configuration deployed${NC}"
    echo ""
}

# Step 6: Run database migrations
run_migrations() {
    echo -e "${YELLOW}🔄 Running database migrations...${NC}"
    
    export DB_USER DB_PASS DB_NAME DB_HOST="localhost"
    export MIGRATION_DIR="$DEPLOYMENT_DIR/migrations"
    
    bash "$DEPLOYMENT_DIR/scripts/run-migrations.sh"
    
    echo -e "${GREEN}   ✅ Database migrations completed${NC}"
    echo ""
}

# Step 7: Start services
start_services() {
    echo -e "${YELLOW}🚀 Starting services...${NC}"
    
    # Enable services
    sudo systemctl enable volume-daemon sendense-hub
    
    # Start Volume Daemon first
    echo "   Starting Volume Daemon..."
    sudo systemctl start volume-daemon
    sleep 3
    
    # Start SHA API
    echo "   Starting Sendense Hub API..."
    sudo systemctl start sendense-hub
    sleep 5
    
    echo -e "${GREEN}   ✅ Services started${NC}"
    echo ""
}

# Step 8: Validate deployment
validate_deployment() {
    echo -e "${YELLOW}🔍 Validating deployment...${NC}"
    
    # Check service status
    if ! sudo systemctl is-active --quiet volume-daemon; then
        echo -e "${RED}❌ Volume Daemon service not running${NC}"
        sudo journalctl -u volume-daemon -n 50 --no-pager
        exit 1
    fi
    echo "   ✅ Volume Daemon: $(sudo systemctl is-active volume-daemon)"
    
    if ! sudo systemctl is-active --quiet sendense-hub; then
        echo -e "${RED}❌ Sendense Hub service not running${NC}"
        sudo journalctl -u sendense-hub -n 50 --no-pager
        exit 1
    fi
    echo "   ✅ Sendense Hub: $(sudo systemctl is-active sendense-hub)"
    
    # Check API endpoints
    echo "   Testing API endpoints..."
    
    sleep 5
    
    if curl -f -s http://localhost:8090/api/v1/health > /dev/null 2>&1; then
        echo "   ✅ Volume Daemon API: http://localhost:8090/api/v1/health"
    else
        echo -e "${RED}❌ Volume Daemon API not responding${NC}"
        exit 1
    fi
    
    if curl -f -s http://localhost:8082/health > /dev/null 2>&1; then
        echo "   ✅ Sendense Hub API: http://localhost:8082/health"
    else
        echo -e "${RED}❌ Sendense Hub API not responding${NC}"
        exit 1
    fi
    
    # Check database connectivity
    if mysql -u "$DB_USER" -p"$DB_PASS" -h localhost -e "SELECT 1" "$DB_NAME" > /dev/null 2>&1; then
        echo "   ✅ Database connectivity verified"
    else
        echo -e "${RED}❌ Cannot connect to database${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}   ✅ Deployment validation passed${NC}"
    echo ""
}

# Main deployment flow
main() {
    echo -e "${BLUE}Starting deployment...${NC}"
    echo ""
    
    install_dependencies
    configure_mariadb
    create_directories
    deploy_binaries
    deploy_configuration
    run_migrations
    start_services
    validate_deployment
    
    echo ""
    echo -e "${GREEN}🎉 Sendense Hub Appliance deployment completed successfully!${NC}"
    echo ""
    echo "📊 Service Status:"
    echo "   • Sendense Hub API: $(sudo systemctl is-active sendense-hub)"
    echo "   • Volume Daemon: $(sudo systemctl is-active volume-daemon)"
    echo "   • MariaDB: $(sudo systemctl is-active mariadb)"
    echo ""
    echo "🌐 API Endpoints:"
    echo "   • SHA API Health: http://localhost:8082/health"
    echo "   • Volume Daemon Health: http://localhost:8090/api/v1/health"
    echo ""
    echo "🗄️  Database:"
    echo "   • Name: $DB_NAME"
    echo "   • User: $DB_USER"
    echo "   • Root password: /root/.mariadb_root_password"
    echo ""
    echo "📝 Logs:"
    echo "   • SHA API: sudo journalctl -u sendense-hub -f"
    echo "   • Volume Daemon: sudo journalctl -u volume-daemon -f"
    echo ""
    echo "🧪 Test Storage System:"
    echo "   cd $PROJECT_ROOT/source/current/oma/storage"
    echo "   go test -v ."
    echo ""
}

# Execute deployment
main "$@"
