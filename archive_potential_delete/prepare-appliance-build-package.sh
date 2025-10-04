#!/bin/bash
# Prepare OSSEA-Migrate OMA Appliance Build Package
# Creates complete build package with production binaries and clean database

set -euo pipefail

BUILD_DIR="/tmp/appliance-build"
echo "📦 Preparing OSSEA-Migrate OMA Appliance Build Package"
echo "====================================================="

# Clean and create build directory
echo "🧹 Preparing build directory..."
sudo rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"/{binaries,services,database,config,scripts}

# Phase 1: Production binaries
echo "📋 Collecting production binaries..."

# OMA API (latest production version)
cp source/current/oma/oma-api-v2.29.3-update-config-fix "$BUILD_DIR/binaries/oma-api"

# Volume Daemon (latest production version)
cp source/current/volume-daemon/volume-daemon-v1.3.2-persistent-naming-fixed "$BUILD_DIR/binaries/volume-daemon"

# Custom boot setup
cp oma-setup-wizard.sh "$BUILD_DIR/scripts/"
cp oma-autologin.service "$BUILD_DIR/services/"

echo "✅ Production binaries collected"

# Phase 2: Migration GUI (production build)
echo "🎨 Building production GUI..."
cd /home/pgrayson/migration-dashboard
npm run build > /dev/null 2>&1
tar -czf "$BUILD_DIR/binaries/migration-gui.tar.gz" .next package.json package-lock.json public src
cd /home/pgrayson/migratekit-cloudstack
echo "✅ Production GUI packaged"

# Phase 3: Database schema (clean, no data)
echo "🗄️ Exporting clean database schema..."

# Export complete schema structure without data
mysqldump -u oma_user -poma_password \
  --no-data \
  --routines \
  --triggers \
  --single-transaction \
  migratekit_oma > "$BUILD_DIR/database/schema-only.sql"

# Add initial required data (OSSEA config template, etc.)
cat >> "$BUILD_DIR/database/initial-data.sql" << 'EOF'
-- Initial data for OSSEA-Migrate OMA appliance

-- Insert default OSSEA configuration template (will be configured via GUI)
INSERT INTO ossea_configs (
  name, api_url, api_key, secret_key, domain, zone, 
  template_id, service_offering_id, oma_vm_id, is_active
) VALUES (
  'production-ossea-template',
  'http://your-cloudstack:8080/client/api',
  'your-api-key-here',
  'your-secret-key-here',
  'OSSEA',
  'your-zone-id-here',
  'your-template-id-here',
  'your-service-offering-id-here',
  'your-oma-vm-id-here',
  false
);

-- Insert VMware credentials template (will be configured via GUI)
INSERT INTO vmware_credentials (
  credential_name, vcenter_host, username, password_encrypted, 
  datacenter, is_active, is_default, created_by
) VALUES (
  'Production-vCenter-Template',
  'your-vcenter-host.local',
  'administrator@vsphere.local',
  'TEMPLATE_ENCRYPTED_PASSWORD',
  'DatabanxDC',
  false,
  false,
  'appliance_setup'
);
EOF

echo "✅ Clean database schema exported"

# Phase 4: Service configurations
echo "⚙️ Collecting service configurations..."

# OMA API service
cat > "$BUILD_DIR/services/oma-api.service" << 'EOF'
[Unit]
Description=OMA Migration API Server
After=network.target mariadb.service
Requires=mariadb.service

[Service]
Type=simple
User=ossea-migrate
Group=ossea-migrate
WorkingDirectory=/opt/migratekit
ExecStart=/opt/migratekit/bin/oma-api -port=8082 -db-type=mariadb -db-host=localhost -db-port=3306 -db-name=migratekit_oma -db-user=oma_user -db-pass=oma_password -auth=false -debug=false
Restart=always
RestartSec=10

# Environment for VMware credentials encryption
Environment=MIGRATEKIT_CRED_ENCRYPTION_KEY=APPLIANCE_WILL_GENERATE_KEY

[Install]
WantedBy=multi-user.target
EOF

# Volume Daemon service
cat > "$BUILD_DIR/services/volume-daemon.service" << 'EOF'
[Unit]
Description=Volume Management Daemon for OSSEA-Migrate
After=network.target mariadb.service
Requires=mariadb.service

[Service]
Type=simple
User=ossea-migrate
Group=ossea-migrate
ExecStart=/usr/local/bin/volume-daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Migration GUI service
cat > "$BUILD_DIR/services/migration-gui.service" << 'EOF'
[Unit]
Description=OSSEA-Migrate Dashboard GUI
After=network.target oma-api.service
Requires=oma-api.service

[Service]
Type=simple
User=ossea-migrate
Group=ossea-migrate
WorkingDirectory=/opt/migratekit/gui
ExecStart=/usr/bin/npm run start -- --port 3001 --hostname 0.0.0.0
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

echo "✅ Service configurations created"

# Phase 5: Build script
echo "📜 Creating appliance build script..."
cp "$0" "$BUILD_DIR/scripts/prepare-build-package.sh"

# Copy the production build script (captures all encountered issues)
cp /home/pgrayson/migratekit-cloudstack/production-oma-appliance-build.sh "$BUILD_DIR/build-oma-appliance.sh"

# Also create a simplified build script for reference
cat > "$BUILD_DIR/simple-build-oma-appliance.sh" << 'EOF'
#!/bin/bash
# OSSEA-Migrate OMA Appliance Build Script
# Run this script on a fresh Ubuntu 24.04 VM to create production OMA appliance

set -euo pipefail

echo "🚀 Building OSSEA-Migrate OMA Appliance"
echo "======================================"

# System preparation
echo "📋 Preparing system..."
sudo apt update && sudo apt upgrade -y
sudo apt install -y mariadb-server nbd-server curl jq unzip systemd net-tools openssh-server nodejs npm

# Create ossea-migrate user
sudo useradd -m -s /bin/bash ossea-migrate || true
sudo usermod -aG sudo ossea-migrate

# Database setup
echo "🗄️ Setting up database..."
sudo mysql -e "CREATE DATABASE IF NOT EXISTS migratekit_oma;"
sudo mysql -e "CREATE USER IF NOT EXISTS 'oma_user'@'localhost' IDENTIFIED BY 'oma_password';"
sudo mysql -e "GRANT ALL PRIVILEGES ON migratekit_oma.* TO 'oma_user'@'localhost';"
sudo mysql -e "FLUSH PRIVILEGES;"

# Import schema
mysql -u oma_user -poma_password migratekit_oma < database/schema-only.sql
mysql -u oma_user -poma_password migratekit_oma < database/initial-data.sql

# Deploy binaries
echo "📦 Deploying binaries..."
sudo mkdir -p /opt/migratekit/{bin,gui}
sudo mkdir -p /usr/local/bin
sudo mkdir -p /opt/ossea-migrate

sudo cp binaries/oma-api /opt/migratekit/bin/
sudo cp binaries/volume-daemon /usr/local/bin/
sudo cp scripts/oma-setup-wizard.sh /opt/ossea-migrate/

# Extract GUI
cd /opt/migratekit/gui
sudo tar -xzf /tmp/appliance-build/binaries/migration-gui.tar.gz
sudo chown -R ossea-migrate:ossea-migrate /opt/migratekit/gui

# Install services
sudo cp services/*.service /etc/systemd/system/
sudo systemctl daemon-reload

# Generate encryption key
ENCRYPTION_KEY=$(openssl rand -base64 32)
sudo sed -i "s/APPLIANCE_WILL_GENERATE_KEY/$ENCRYPTION_KEY/" /etc/systemd/system/oma-api.service

# Enable services
sudo systemctl enable mariadb oma-api volume-daemon nbd-server migration-gui oma-autologin
sudo systemctl disable getty@tty1

# Start services
sudo systemctl start mariadb oma-api volume-daemon nbd-server migration-gui

echo "✅ OSSEA-Migrate OMA Appliance build complete!"
echo "🌐 Access GUI at: http://$(hostname -I | awk '{print $1}'):3001"
EOF

chmod +x "$BUILD_DIR/build-oma-appliance.sh"

echo ""
echo "✅ OSSEA-Migrate OMA Appliance Build Package Complete!"
echo ""
echo "📦 Build package created at: $BUILD_DIR"
echo "📊 Package contents:"
echo "   - Production binaries (OMA API, Volume Daemon, GUI)"
echo "   - Clean database schema with initial templates"
echo "   - Systemd service configurations"
echo "   - Custom boot setup (OSSEA-Migrate branding)"
echo "   - Automated build script"
echo ""
echo "🚀 Next steps:"
echo "   1. Create Ubuntu 24.04 VM in CloudStack (8GB RAM, 4 vCPU, 100GB disk)"
echo "   2. Transfer build package: scp -r $BUILD_DIR ubuntu@new-vm:/tmp/"
echo "   3. Run build script: ssh ubuntu@new-vm 'sudo /tmp/appliance-build/build-oma-appliance.sh'"
echo "   4. Test complete functionality"
echo "   5. Export as CloudStack template for distribution"
