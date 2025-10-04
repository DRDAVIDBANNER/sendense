#!/bin/bash

# Script to completely disable authentication for development
# This removes the token requirement entirely

set -e

echo "🔓 Disabling authentication for OMA API (development mode)"

# Step 1: Restart OMA API with authentication disabled
echo ""
echo "🔄 Step 1: Restarting OMA API with --auth=false..."

sudo systemctl stop oma-api

# Update the service to run with --auth=false
sudo systemctl edit --full oma-api << 'EOF'
[Unit]
Description=OMA Migration API Server
Documentation=OMA API for VMware to OSSEA migration operations
After=network-online.target mariadb.service
Wants=network-online.target
Requires=network-online.target

[Service]
Type=simple
User=pgrayson
Group=pgrayson
WorkingDirectory=/home/pgrayson/migratekit-cloudstack

# Service executable with authentication disabled
ExecStart=/opt/migratekit/bin/oma-api \
  --port=8082 \
  --auth=false \
  --db-type=mariadb \
  --db-host=localhost \
  --db-port=3306 \
  --db-name=migratekit_oma \
  --db-user=oma_user \
  --db-pass=oma_password

Restart=always
RestartSec=10
StartLimitInterval=60
StartLimitBurst=3

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=oma-api

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl start oma-api

echo "✅ OMA API restarted with authentication disabled"

# Step 2: Update GUI to remove authentication headers
echo ""
echo "🎨 Step 2: Updating GUI to remove authentication headers..."

GUI_DASHBOARD_DIR="/home/pgrayson/migration-dashboard"

if [ -d "$GUI_DASHBOARD_DIR" ]; then
    cd "$GUI_DASHBOARD_DIR"
    
    # Update replicate route
    if [ -f "src/app/api/replicate/route.ts" ]; then
        echo "Removing auth headers from replicate route..."
        sed -i "/Authorization.*Bearer/d" src/app/api/replicate/route.ts
        echo "✅ Replicate route updated"
    fi
    
    # Update migrations route  
    if [ -f "src/app/api/migrations/route.ts" ]; then
        echo "Removing auth headers from migrations route..."
        sed -i "/Authorization.*Bearer/d" src/app/api/migrations/route.ts
        echo "✅ Migrations route updated"
    fi
    
    # Update discover route if it exists
    if [ -f "src/app/api/discover/route.ts" ]; then
        echo "Removing auth headers from discover route..."
        sed -i "/Authorization.*Bearer/d" src/app/api/discover/route.ts
        echo "✅ Discover route updated"
    fi
    
    echo "🔄 Restarting GUI service..."
    sudo systemctl restart migration-gui
    echo "✅ GUI service restarted"
    
else
    echo "❌ GUI dashboard directory not found: $GUI_DASHBOARD_DIR"
fi

# Step 3: Update VMA client to skip authentication
echo ""
echo "🛠️  Step 3: Creating VMA no-auth environment..."

VMA_ENV_FILE="/home/pgrayson/vma_noauth.env"
cat > "$VMA_ENV_FILE" << 'EOF'
# VMA No Authentication Configuration  
# For development use only
export VMA_OMA_BASE_URL="http://localhost:8082"
export VMA_SKIP_AUTH="true"
export VMA_APPLIANCE_ID="vma-001"
EOF

echo "✅ VMA no-auth environment file created: $VMA_ENV_FILE"
echo "   To use: source $VMA_ENV_FILE"

echo ""
echo "🎉 COMPLETED: Authentication disabled for development!"
echo ""
echo "📋 Summary:"
echo "   - OMA API: Running with --auth=false"
echo "   - GUI routes: Authentication headers removed"
echo "   - VMA config: Set to skip authentication"
echo ""
echo "🧪 Test the configuration:"
echo "   curl http://localhost:8082/health (no auth required)"
echo "   curl http://localhost:3001 (no auth required)"
echo ""
echo "⚠️  WARNING: This is for development only!"
echo "   Re-enable authentication before production deployment"



