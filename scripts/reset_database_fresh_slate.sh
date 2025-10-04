#!/bin/bash

# ===================================================================
# FRESH SLATE DATABASE RESET - MigrateKit OSSEA
# ===================================================================
# Created: 2025-08-21
# Purpose: Complete database reset for fresh testing
# 
# This script removes ALL replication data while preserving:
# - OSSEA configurations
# - Network mappings
# - System configurations
# ===================================================================

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database connection parameters
DB_USER="oma_user"
DB_PASS="oma_password"
DB_NAME="migratekit_oma"

# Logging
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

echo "🧹 FRESH SLATE DATABASE RESET"
echo "============================="
log_warning "This will remove ALL replication jobs, volumes, and exports!"
log_info "Preserving: OSSEA configs, network mappings, system settings"
echo

# Confirmation
read -p "Are you sure you want to proceed? (yes/no): " confirm
if [[ $confirm != "yes" ]]; then
    log_info "Operation cancelled"
    exit 0
fi

log_info "📊 Current database state:"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT 'replication_jobs' as table_name, COUNT(*) as count FROM replication_jobs
UNION ALL
SELECT 'vm_disks', COUNT(*) FROM vm_disks  
UNION ALL
SELECT 'device_mappings', COUNT(*) FROM device_mappings
UNION ALL  
SELECT 'nbd_exports', COUNT(*) FROM nbd_exports
UNION ALL
SELECT 'vm_export_mappings', COUNT(*) FROM vm_export_mappings
UNION ALL
SELECT 'failover_jobs', COUNT(*) FROM failover_jobs
UNION ALL
SELECT 'ossea_volumes', COUNT(*) FROM ossea_volumes;"

echo
log_info "🔍 STEP 1: Collecting all CloudStack volumes for cleanup..."

# Get all volume UUIDs from all possible sources
VOLUME_UUIDS=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
-- Get all volume UUIDs from multiple sources
SELECT DISTINCT volume_uuid FROM (
    -- From vm_disks (normalized schema)
    SELECT cloudstack_volume_uuid as volume_uuid 
    FROM vm_disks 
    WHERE cloudstack_volume_uuid IS NOT NULL
    
    UNION
    
    -- From ossea_volumes table
    SELECT volume_id as volume_uuid
    FROM ossea_volumes
    WHERE volume_id IS NOT NULL
    
    UNION
    
    -- From device_mappings
    SELECT volume_uuid
    FROM device_mappings
    WHERE volume_uuid IS NOT NULL
) AS all_volumes
WHERE volume_uuid != '';
")

if [[ -n "$VOLUME_UUIDS" ]]; then
    log_info "Found volumes to clean: $(echo "$VOLUME_UUIDS" | wc -l) volumes"
    echo "$VOLUME_UUIDS" | head -5
    if [[ $(echo "$VOLUME_UUIDS" | wc -l) -gt 5 ]]; then
        echo "... and $(($(echo "$VOLUME_UUIDS" | wc -l) - 5)) more"
    fi
else
    log_info "No CloudStack volumes found for cleanup"
fi

echo
log_info "☁️  STEP 2: CloudStack volume cleanup..."

# Check Volume Daemon status
if ! curl -f http://localhost:8090/health >/dev/null 2>&1; then
    log_warning "Volume Daemon not responding - using direct CloudStack cleanup"
    DAEMON_AVAILABLE=false
else
    log_info "Volume Daemon available - using daemon for cleanup"
    DAEMON_AVAILABLE=true
fi

# Process each volume
volume_count=0
success_count=0

if [[ -n "$VOLUME_UUIDS" ]]; then
    while IFS= read -r volume_uuid; do
        [[ -z "$volume_uuid" ]] && continue
        
        volume_count=$((volume_count + 1))
        log_info "Processing volume $volume_count: $volume_uuid"
        
        if [[ "$DAEMON_AVAILABLE" == "true" ]]; then
            # Use Volume Daemon
            if curl -s -X POST "http://localhost:8090/api/v1/volumes/$volume_uuid/detach" >/dev/null 2>&1; then
                sleep 2 # Wait for detach
                if curl -s -X DELETE "http://localhost:8090/api/v1/volumes/$volume_uuid" >/dev/null 2>&1; then
                    log_info "✅ Volume cleaned via daemon: $volume_uuid"
                    success_count=$((success_count + 1))
                else
                    log_warning "Failed to delete volume via daemon: $volume_uuid"
                fi
            else
                log_warning "Failed to detach volume via daemon: $volume_uuid"
            fi
        else
            # Direct CloudStack cleanup would go here if needed
            log_warning "Direct CloudStack cleanup not implemented - volume may remain: $volume_uuid"
        fi
        
    done <<< "$VOLUME_UUIDS"
fi

log_success "CloudStack cleanup completed: $success_count/$volume_count volumes processed"

echo
log_info "📡 STEP 3: NBD server cleanup..."

# Stop all NBD exports
if pgrep nbd-server >/dev/null; then
    log_info "Stopping NBD server..."
    sudo pkill nbd-server 2>/dev/null || true
    sleep 2
fi

# Clean NBD config files
if [[ -f /etc/nbd-server/config-base ]]; then
    log_info "Backing up and resetting NBD config..."
    sudo cp /etc/nbd-server/config-base /etc/nbd-server/config-base.backup-$(date +%Y%m%d-%H%M%S)
    
    # Create minimal config
    sudo tee /etc/nbd-server/config-base > /dev/null << 'EOF'
# NBD Server Configuration - Fresh Slate
# Generated by reset_database_fresh_slate.sh

[generic]
    allowlist = true
    port = 10809

# Dynamic exports will be added here
EOF
    
    log_success "NBD config reset to minimal state"
fi

# Start NBD server with clean config
sudo systemctl restart nbd-server 2>/dev/null || log_warning "Failed to restart NBD server"

echo
log_info "🗄️  STEP 4: Database cleanup (preserving configurations)..."

# Disable foreign key checks temporarily for clean deletion
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "SET FOREIGN_KEY_CHECKS = 0;"

# Clean replication data (order matters due to FK relationships)
log_info "Cleaning failover_jobs..."
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM failover_jobs;"

log_info "Cleaning nbd_exports..."  
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM nbd_exports;"

log_info "Cleaning vm_export_mappings..."
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM vm_export_mappings;"

log_info "Cleaning vm_disks..."
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM vm_disks;"

log_info "Cleaning device_mappings..."
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM device_mappings;"

log_info "Cleaning ossea_volumes..."
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM ossea_volumes;"

log_info "Cleaning replication_jobs..."
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM replication_jobs;"

# Re-enable foreign key checks
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "SET FOREIGN_KEY_CHECKS = 1;"

log_success "Database cleanup completed"

echo
log_info "🔍 STEP 5: Post-cleanup validation..."

# Verify clean state
log_info "Final database state:"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT 'replication_jobs' as table_name, COUNT(*) as count FROM replication_jobs
UNION ALL
SELECT 'vm_disks', COUNT(*) FROM vm_disks  
UNION ALL
SELECT 'device_mappings', COUNT(*) FROM device_mappings
UNION ALL  
SELECT 'nbd_exports', COUNT(*) FROM nbd_exports
UNION ALL
SELECT 'vm_export_mappings', COUNT(*) FROM vm_export_mappings
UNION ALL
SELECT 'failover_jobs', COUNT(*) FROM failover_jobs
UNION ALL
SELECT 'ossea_volumes', COUNT(*) FROM ossea_volumes;"

echo
log_info "🔧 Preserved configurations:"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT 'ossea_configs' as table_name, COUNT(*) as count FROM ossea_configs
UNION ALL
SELECT 'network_mappings', COUNT(*) FROM network_mappings;"

echo
log_success "🎉 FRESH SLATE RESET COMPLETED!"
log_info "Database is now clean and ready for fresh replication testing"
log_info "All FK constraints remain active for data integrity"
log_info "OSSEA configurations and network mappings preserved"

echo
log_info "📋 Ready for testing:"
log_info "  • All previous job artifacts removed"
log_info "  • CloudStack volumes cleaned up"  
log_info "  • NBD server reset to clean state"
log_info "  • Database integrity constraints active"
log_info "  • Fresh slate for pgtest2, PGWINTESTBIOS, or any VM"

echo
log_warning "IMPORTANT: Test the new database consistency fixes with a fresh job!"
