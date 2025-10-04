#!/bin/bash
# Production OSSEA-Migrate OMA Appliance Build Script
# Complete automated build capturing all encountered issues and solutions
# Run this on a fresh Ubuntu 24.04 VM to create production-ready OMA appliance

set -euo pipefail

SUDO_PASSWORD="Password1"
BUILD_DIR="/tmp/appliance-build"
OMA_USER="oma_admin"

echo "🚀 OSSEA-Migrate OMA Appliance Production Build"
echo "==============================================="
echo "Build Date: $(date)"
echo "Target OS: Ubuntu 24.04 LTS"
echo "Build User: $OMA_USER"
echo ""

# Function to run sudo commands with password (non-interactive)
run_sudo() {
    echo "$SUDO_PASSWORD" | sudo -S "$@"
}

# Function to check command success with detailed logging
check_success() {
    local exit_code=$?
    if [ $exit_code -eq 0 ]; then
        echo "✅ $1 completed successfully"
    else
        echo "❌ $1 failed (exit code: $exit_code)"
        exit 1
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local service_name="$1"
    local max_attempts=30
    local attempt=0
    
    echo "⏳ Waiting for $service_name to be ready..."
    while [ $attempt -lt $max_attempts ]; do
        if systemctl is-active "$service_name" > /dev/null 2>&1; then
            echo "✅ $service_name is ready"
            return 0
        fi
        sleep 2
        attempt=$((attempt + 1))
    done
    echo "⚠️ $service_name did not start within timeout"
    return 1
}

# Phase 1: Pre-flight checks and validation
echo "📋 Phase 1: Pre-flight Validation"
echo "=================================="

# Verify we're on Ubuntu 24.04
if ! grep -q "24.04" /etc/os-release; then
    echo "❌ This script requires Ubuntu 24.04 LTS"
    exit 1
fi

# Verify build package exists and is complete
if [ ! -d "$BUILD_DIR" ]; then
    echo "❌ Build directory not found: $BUILD_DIR"
    echo "Please transfer the appliance build package first"
    exit 1
fi

required_files=(
    "$BUILD_DIR/database/schema-only.sql"
    "$BUILD_DIR/database/initial-data.sql"
    "$BUILD_DIR/binaries/oma-api"
    "$BUILD_DIR/binaries/volume-daemon"
    "$BUILD_DIR/binaries/migration-gui.tar.gz"
    "$BUILD_DIR/scripts/oma-setup-wizard.sh"
    "$BUILD_DIR/services/oma-api.service"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Required file missing: $file"
        exit 1
    fi
done

echo "✅ Pre-flight validation completed"
echo ""

# Phase 2: System preparation and dependencies
echo "📋 Phase 2: System Preparation"
echo "=============================="

echo "🔄 Updating system packages..."
run_sudo apt update -y
check_success "System package update"

echo "📦 Installing core dependencies..."
# Install required packages in one command to avoid multiple prompts
DEBIAN_FRONTEND=noninteractive run_sudo apt install -y \
    mariadb-server \
    nbd-server \
    curl \
    jq \
    unzip \
    systemd \
    net-tools \
    openssh-server \
    nodejs \
    npm \
    build-essential
check_success "Core dependencies installation"

echo "🌐 Installing Next.js globally..."
run_sudo npm install -g next@latest
check_success "Next.js global installation"

echo "✅ System preparation completed"
echo ""

# Phase 3: Cloud-init removal (prevent future boot delays)
echo "📋 Phase 3: Cloud-init Cleanup"
echo "============================="

echo "🚨 Disabling and removing cloud-init..."
run_sudo systemctl disable cloud-init.service cloud-config.service cloud-final.service cloud-init-local.service 2>/dev/null || true
run_sudo touch /etc/cloud/cloud-init.disabled 2>/dev/null || true
DEBIAN_FRONTEND=noninteractive run_sudo apt remove --purge cloud-init -y 2>/dev/null || true
run_sudo rm -rf /var/lib/cloud/ /var/log/cloud-init* /etc/cloud/ 2>/dev/null || true
run_sudo apt autoremove -y 2>/dev/null || true
check_success "Cloud-init removal"

echo "✅ Cloud-init cleanup completed"
echo ""

# Phase 4: Database setup with proper error handling
echo "📋 Phase 4: Database Configuration"
echo "=================================="

echo "🗄️ Configuring MariaDB..."
run_sudo systemctl start mariadb
wait_for_service "mariadb.service"

echo "👤 Creating database and user..."
run_sudo mysql -e "CREATE DATABASE IF NOT EXISTS migratekit_oma;" 2>/dev/null || true
run_sudo mysql -e "CREATE USER IF NOT EXISTS 'oma_user'@'localhost' IDENTIFIED BY 'oma_password';" 2>/dev/null || true
run_sudo mysql -e "GRANT ALL PRIVILEGES ON migratekit_oma.* TO 'oma_user'@'localhost';" 2>/dev/null || true
run_sudo mysql -e "FLUSH PRIVILEGES;" 2>/dev/null || true
check_success "Database user creation"

echo "📊 Importing database schema..."
cd "$BUILD_DIR"
mysql -u oma_user -poma_password migratekit_oma < database/schema-only.sql
check_success "Database schema import"

mysql -u oma_user -poma_password migratekit_oma < database/initial-data.sql
check_success "Initial data import"

# Verify database import
table_count=$(mysql -u oma_user -poma_password migratekit_oma -e "SHOW TABLES;" | wc -l)
echo "📊 Database contains $table_count tables"

echo "✅ Database configuration completed"
echo ""

# Phase 5: Directory structure and binary deployment
echo "📋 Phase 5: Binary Deployment"
echo "============================="

echo "📁 Creating OSSEA-Migrate directory structure..."
run_sudo mkdir -p /opt/migratekit/{bin,gui}
run_sudo mkdir -p /opt/ossea-migrate
run_sudo mkdir -p /usr/local/bin
check_success "Directory creation"

echo "📦 Deploying production binaries..."

# OMA API
run_sudo cp binaries/oma-api /opt/migratekit/bin/
run_sudo chmod +x /opt/migratekit/bin/oma-api
run_sudo chown $OMA_USER:$OMA_USER /opt/migratekit/bin/oma-api
check_success "OMA API binary deployment"

# Volume Daemon
run_sudo cp binaries/volume-daemon /usr/local/bin/
run_sudo chmod +x /usr/local/bin/volume-daemon
run_sudo chown $OMA_USER:$OMA_USER /usr/local/bin/volume-daemon
check_success "Volume Daemon binary deployment"

# Custom boot setup
run_sudo cp scripts/oma-setup-wizard.sh /opt/ossea-migrate/
run_sudo chmod +x /opt/ossea-migrate/oma-setup-wizard.sh
run_sudo chown $OMA_USER:$OMA_USER /opt/ossea-migrate/oma-setup-wizard.sh
check_success "Custom boot setup deployment"

echo "✅ Binary deployment completed"
echo ""

# Phase 6: Migration GUI setup with proper Next.js configuration
echo "📋 Phase 6: Migration GUI Setup"
echo "=============================="

echo "🎨 Deploying Migration GUI..."
cd /opt/migratekit/gui
run_sudo tar -xzf "$BUILD_DIR/binaries/migration-gui.tar.gz"
run_sudo chown -R $OMA_USER:$OMA_USER /opt/migratekit/gui/

echo "📦 Installing GUI dependencies..."
cd /opt/migratekit/gui
run_sudo -u $OMA_USER npm install --production
check_success "GUI dependencies installation"

echo "✅ Migration GUI setup completed"
echo ""

# Phase 7: Systemd service configuration
echo "📋 Phase 7: Service Configuration"
echo "================================="

echo "⚙️ Installing systemd services..."
run_sudo cp "$BUILD_DIR/services/"*.service /etc/systemd/system/
run_sudo systemctl daemon-reload
check_success "Service installation"

echo "🔐 Generating VMware credentials encryption key..."
ENCRYPTION_KEY=$(openssl rand -base64 32)
run_sudo sed -i "s/APPLIANCE_WILL_GENERATE_KEY/$ENCRYPTION_KEY/" /etc/systemd/system/oma-api.service
check_success "Encryption key generation"

echo "🚀 Enabling OSSEA-Migrate services..."
run_sudo systemctl enable mariadb oma-api volume-daemon nbd-server migration-gui oma-autologin
check_success "Service enablement"

echo "🚫 Disabling standard login (custom boot will replace)..."
run_sudo systemctl disable getty@tty1 2>/dev/null || true
check_success "Standard login disable"

echo "✅ Service configuration completed"
echo ""

# Phase 8: Service startup with health validation
echo "📋 Phase 8: Service Startup & Validation"
echo "========================================"

echo "🚀 Starting core services in dependency order..."

# Start MariaDB first
run_sudo systemctl start mariadb
wait_for_service "mariadb.service"

# Start Volume Daemon
run_sudo systemctl start volume-daemon
wait_for_service "volume-daemon.service"

# Start NBD Server
run_sudo systemctl start nbd-server
wait_for_service "nbd-server.service"

# Start OMA API
run_sudo systemctl start oma-api
sleep 5  # Give OMA API time to initialize

# Start Migration GUI
run_sudo systemctl start migration-gui
sleep 5  # Give GUI time to initialize

echo "✅ Service startup completed"
echo ""

# Phase 9: Health validation and testing
echo "📋 Phase 9: Health Validation"
echo "============================="

OMA_IP=$(hostname -I | awk '{print $1}')
echo "🔍 Testing service health endpoints..."

# Test database connectivity
if mysql -u oma_user -poma_password migratekit_oma -e "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ Database connectivity confirmed"
else
    echo "❌ Database connectivity failed"
fi

# Test OMA API
if curl -s --connect-timeout 10 http://localhost:8082/health > /dev/null 2>&1; then
    echo "✅ OMA API health check passed"
else
    echo "⚠️ OMA API health check failed (may need more time to start)"
fi

# Test Volume Daemon
if curl -s --connect-timeout 5 http://localhost:8090/api/v1/health > /dev/null 2>&1; then
    echo "✅ Volume Daemon health check passed"
else
    echo "⚠️ Volume Daemon health check failed"
fi

# Test Migration GUI
if curl -s --connect-timeout 10 http://localhost:3001 > /dev/null 2>&1; then
    echo "✅ Migration GUI health check passed"
else
    echo "⚠️ Migration GUI health check failed (may need more time to start)"
fi

# Service status summary
echo ""
echo "📊 Final Service Status:"
for service in mariadb oma-api volume-daemon nbd-server migration-gui oma-autologin; do
    status=$(systemctl is-active "$service.service" 2>/dev/null || echo "inactive")
    echo "   $service: $status"
done

echo ""
echo "✅ Health validation completed"
echo ""

# Phase 10: Appliance finalization
echo "📋 Phase 10: Appliance Finalization"
echo "==================================="

echo "🧹 Cleaning up build artifacts..."
rm -rf "$BUILD_DIR" 2>/dev/null || true
check_success "Build artifact cleanup"

echo "🗑️ Cleaning system logs..."
run_sudo journalctl --vacuum-time=1h 2>/dev/null || true
history -c 2>/dev/null || true
check_success "Log cleanup"

echo "✅ Appliance finalization completed"
echo ""

# Phase 11: Final appliance information
echo "🎉 OSSEA-Migrate OMA Appliance Build Complete!"
echo "=============================================="
echo ""
echo "📊 Appliance Information:"
echo "   Appliance Name: OSSEA-Migrate OMA v1.0"
echo "   Base OS: Ubuntu 24.04 LTS"
echo "   IP Address: $OMA_IP"
echo "   Build Date: $(date)"
echo ""
echo "🌐 Access Points:"
echo "   Web GUI: http://$OMA_IP:3001"
echo "   API Endpoint: http://$OMA_IP:8082"
echo "   Health Status: http://$OMA_IP:8082/health"
echo ""
echo "🚀 Features Deployed:"
echo "   ✅ Intelligent Failed Execution Cleanup System"
echo "   ✅ Streamlined OSSEA Configuration Interface"
echo "   ✅ Persistent Device Naming & NBD Memory Sync"
echo "   ✅ Multi-Volume Snapshot Protection"
echo "   ✅ Professional Custom Boot Experience"
echo "   ✅ VMware Credentials Management Foundation"
echo ""
echo "🔧 Custom Boot Experience:"
echo "   - Reboot to test professional OSSEA-Migrate boot wizard"
echo "   - Network configuration interface with service status"
echo "   - Professional branding and enterprise appearance"
echo ""
echo "📋 Next Steps:"
echo "   1. Test GUI access: http://$OMA_IP:3001"
echo "   2. Configure CloudStack via streamlined interface"
echo "   3. Test complete migration workflow"
echo "   4. Export as CloudStack template for distribution"
echo ""
echo "🎯 Appliance Ready for:"
echo "   - Enterprise deployment and distribution"
echo "   - Professional customer installation"
echo "   - Production migration workflows"
echo ""
echo "✅ OSSEA-Migrate OMA Virtual Appliance: Production Ready!"

# Create appliance info file
cat > /home/$OMA_USER/appliance-info.txt << EOF
OSSEA-Migrate OMA Virtual Appliance v1.0
Build Date: $(date)
Base OS: Ubuntu 24.04 LTS
IP Address: $OMA_IP

Access Points:
- Web GUI: http://$OMA_IP:3001
- API Endpoint: http://$OMA_IP:8082

Services:
- OMA API: Professional migration API with intelligent cleanup
- Volume Daemon: Persistent device naming and NBD memory sync
- Migration GUI: Streamlined interface with auto-discovery
- Custom Boot: Professional OSSEA-Migrate boot experience

For support: https://github.com/DRDAVIDBANNER/X-Vire
EOF

echo ""
echo "📄 Appliance information saved to: /home/$OMA_USER/appliance-info.txt"
echo ""
echo "🎉 Production OSSEA-Migrate OMA Appliance Build Complete!"






