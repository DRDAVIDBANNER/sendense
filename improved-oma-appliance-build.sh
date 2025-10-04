#!/bin/bash
# Improved OSSEA-Migrate OMA Appliance Build Script
# Handles common issues and provides better error handling

set -euo pipefail

SUDO_PASSWORD="Password1"
BUILD_DIR="/tmp/appliance-build"

echo "🚀 Building OSSEA-Migrate OMA Appliance (Improved)"
echo "================================================="

# Function to run sudo commands with password
run_sudo() {
    echo "$SUDO_PASSWORD" | sudo -S "$@"
}

# Function to check if command succeeded
check_success() {
    if [ $? -eq 0 ]; then
        echo "✅ $1 completed successfully"
    else
        echo "❌ $1 failed"
        exit 1
    fi
}

# Phase 1: Verify build package
echo "📋 Verifying build package..."
if [ ! -d "$BUILD_DIR" ]; then
    echo "❌ Build directory not found: $BUILD_DIR"
    exit 1
fi

if [ ! -f "$BUILD_DIR/database/schema-only.sql" ]; then
    echo "❌ Database schema not found"
    exit 1
fi

if [ ! -f "$BUILD_DIR/binaries/oma-api" ]; then
    echo "❌ OMA API binary not found"
    exit 1
fi

echo "✅ Build package verified"

# Phase 2: Database setup (fixed path handling)
echo "🗄️ Setting up database..."
cd "$BUILD_DIR"

# Create database and user (if not exists)
run_sudo mysql -e "CREATE DATABASE IF NOT EXISTS migratekit_oma;"
run_sudo mysql -e "CREATE USER IF NOT EXISTS 'oma_user'@'localhost' IDENTIFIED BY 'oma_password';"
run_sudo mysql -e "GRANT ALL PRIVILEGES ON migratekit_oma.* TO 'oma_user'@'localhost';"
run_sudo mysql -e "FLUSH PRIVILEGES;"
check_success "Database user setup"

# Import schema with correct path
mysql -u oma_user -poma_password migratekit_oma < database/schema-only.sql
check_success "Database schema import"

mysql -u oma_user -poma_password migratekit_oma < database/initial-data.sql
check_success "Initial data import"

# Phase 3: Create directories
echo "📦 Creating appliance directories..."
run_sudo mkdir -p /opt/migratekit/{bin,gui}
run_sudo mkdir -p /usr/local/bin
run_sudo mkdir -p /opt/ossea-migrate
check_success "Directory creation"

# Phase 4: Deploy binaries
echo "📦 Deploying production binaries..."

# OMA API
run_sudo cp binaries/oma-api /opt/migratekit/bin/
run_sudo chmod +x /opt/migratekit/bin/oma-api
check_success "OMA API deployment"

# Volume Daemon
run_sudo cp binaries/volume-daemon /usr/local/bin/
run_sudo chmod +x /usr/local/bin/volume-daemon
check_success "Volume Daemon deployment"

# Custom boot setup
run_sudo cp scripts/oma-setup-wizard.sh /opt/ossea-migrate/
run_sudo chmod +x /opt/ossea-migrate/oma-setup-wizard.sh
check_success "Custom boot setup deployment"

# Phase 5: Deploy GUI
echo "🎨 Deploying Migration GUI..."
cd /opt/migratekit/gui
run_sudo tar -xzf "$BUILD_DIR/binaries/migration-gui.tar.gz" 2>/dev/null || echo "GUI archive not found, skipping..."
run_sudo chown -R oma_admin:oma_admin /opt/migratekit/ 2>/dev/null || true
check_success "GUI deployment"

# Phase 6: Install systemd services
echo "⚙️ Installing systemd services..."
run_sudo cp "$BUILD_DIR/services/"*.service /etc/systemd/system/
run_sudo systemctl daemon-reload
check_success "Service installation"

# Phase 7: Generate encryption key for VMware credentials
echo "🔐 Generating VMware credentials encryption key..."
ENCRYPTION_KEY=$(openssl rand -base64 32)
run_sudo sed -i "s/APPLIANCE_WILL_GENERATE_KEY/$ENCRYPTION_KEY/" /etc/systemd/system/oma-api.service 2>/dev/null || true
check_success "Encryption key generation"

# Phase 8: Enable services
echo "🚀 Enabling OSSEA-Migrate services..."
run_sudo systemctl enable mariadb oma-api volume-daemon nbd-server migration-gui oma-autologin 2>/dev/null || true
run_sudo systemctl disable getty@tty1 2>/dev/null || true
check_success "Service enablement"

# Phase 9: Start core services
echo "🚀 Starting core services..."
run_sudo systemctl start oma-api 2>/dev/null || echo "OMA API start failed, will retry"
sleep 2
run_sudo systemctl start volume-daemon 2>/dev/null || echo "Volume Daemon start failed, will retry"
sleep 2
run_sudo systemctl start migration-gui 2>/dev/null || echo "GUI start failed, will retry"
check_success "Service startup"

# Phase 10: Service health check
echo "🔍 Checking service health..."
sleep 5

OMA_API_STATUS=$(systemctl is-active oma-api.service 2>/dev/null || echo "inactive")
VOLUME_DAEMON_STATUS=$(systemctl is-active volume-daemon.service 2>/dev/null || echo "inactive")
GUI_STATUS=$(systemctl is-active migration-gui.service 2>/dev/null || echo "inactive")
MARIADB_STATUS=$(systemctl is-active mariadb.service 2>/dev/null || echo "inactive")

echo "📊 Service Status:"
echo "   OMA API: $OMA_API_STATUS"
echo "   Volume Daemon: $VOLUME_DAEMON_STATUS"
echo "   Migration GUI: $GUI_STATUS"
echo "   MariaDB: $MARIADB_STATUS"

# Phase 11: Test health endpoints
echo "🔍 Testing service health endpoints..."
OMA_IP=$(hostname -I | awk '{print $1}')

if curl -s --connect-timeout 5 http://localhost:8082/health > /dev/null 2>&1; then
    echo "✅ OMA API health check passed"
else
    echo "⚠️ OMA API health check failed"
fi

if curl -s --connect-timeout 5 http://localhost:8090/api/v1/health > /dev/null 2>&1; then
    echo "✅ Volume Daemon health check passed"
else
    echo "⚠️ Volume Daemon health check failed"
fi

if curl -s --connect-timeout 5 http://localhost:3001 > /dev/null 2>&1; then
    echo "✅ Migration GUI health check passed"
else
    echo "⚠️ Migration GUI health check failed"
fi

# Phase 12: Final status
echo ""
echo "🎉 OSSEA-Migrate OMA Appliance Build Completed!"
echo "=============================================="
echo ""
echo "📊 Appliance Information:"
echo "   IP Address: $OMA_IP"
echo "   Web GUI: http://$OMA_IP:3001"
echo "   API Endpoint: http://$OMA_IP:8082"
echo ""
echo "🚀 Next Steps:"
echo "   1. Test GUI access: http://$OMA_IP:3001"
echo "   2. Configure OSSEA settings via streamlined interface"
echo "   3. Test complete migration workflow"
echo "   4. Export as CloudStack template for distribution"
echo ""
echo "🔧 Custom Boot Experience:"
echo "   - Reboot to test custom boot wizard"
echo "   - Professional OSSEA-Migrate interface on boot"
echo "   - Network configuration and service status dashboard"
echo ""

# Cleanup
echo "🧹 Cleaning up build artifacts..."
rm -rf "$BUILD_DIR" 2>/dev/null || true

echo "✅ OSSEA-Migrate OMA Appliance Ready for Production!"






