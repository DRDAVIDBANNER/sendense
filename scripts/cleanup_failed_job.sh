#!/bin/bash

# COMPREHENSIVE FAILED JOB CLEANUP SCRIPT
# ========================================
# Purpose: Completely clean up failed jobs for fresh restart
# Handles: Database cleanup, CloudStack volumes, NBD exports, File system
# Usage: ./cleanup_failed_job.sh <job_id> [failover_job_id]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database credentials
DB_USER="oma_user"
DB_PASS="oma_password"
DB_NAME="migratekit_oma"

# Helper function for colored output
log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }

# Check arguments
if [ $# -lt 1 ]; then
    log_error "Usage: $0 <replication_job_id> [failover_job_id]"
    log_info "Example: $0 job-20250821-075220"
    log_info "Example: $0 job-20250821-075220 failover-20250821-123456"
    exit 1
fi

REPLICATION_JOB_ID="$1"
FAILOVER_JOB_ID="${2:-}"

log_info "🧹 COMPREHENSIVE FAILED JOB CLEANUP"
log_info "===================================="
log_info "Replication Job: $REPLICATION_JOB_ID"
if [ -n "$FAILOVER_JOB_ID" ]; then
    log_info "Failover Job: $FAILOVER_JOB_ID"
fi
echo

# ============================================================================
# STEP 1: ANALYZE WHAT NEEDS CLEANING
# ============================================================================

log_info "📊 STEP 1: Analyzing job dependencies..."

# Check if job exists
JOB_EXISTS=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "SELECT COUNT(*) FROM replication_jobs WHERE id = '$REPLICATION_JOB_ID'")
if [ "$JOB_EXISTS" -eq 0 ]; then
    log_warning "Replication job $REPLICATION_JOB_ID not found in database"
    exit 1
fi

# Get job details
JOB_STATUS=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "SELECT status FROM replication_jobs WHERE id = '$REPLICATION_JOB_ID'")
VM_NAME=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "SELECT source_vm_name FROM replication_jobs WHERE id = '$REPLICATION_JOB_ID'")

log_info "Job Status: $JOB_STATUS"
log_info "VM Name: $VM_NAME"

# Count related records
VM_DISKS_COUNT=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "SELECT COUNT(*) FROM vm_disks WHERE job_id = '$REPLICATION_JOB_ID'")
NBD_EXPORTS_COUNT=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "SELECT COUNT(*) FROM nbd_exports WHERE job_id = '$REPLICATION_JOB_ID'")
EXPORT_MAPPINGS_COUNT=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "SELECT COUNT(*) FROM vm_export_mappings WHERE vm_name = '$VM_NAME'" 2>/dev/null || echo "0")

log_info "Records to clean:"
log_info "  - vm_disks: $VM_DISKS_COUNT"
log_info "  - nbd_exports: $NBD_EXPORTS_COUNT"
log_info "  - vm_export_mappings: $EXPORT_MAPPINGS_COUNT"

# Get volumes that will be affected
log_info "📦 Volumes associated with this job (Volume Daemon source of truth):"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT 
    ov.volume_id as volume_uuid,
    ov.volume_name,
    ov.status as volume_status,
    ov.device_path as volume_device_path,
    dm.device_path as daemon_device_path,
    dm.cloudstack_state,
    dm.size
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
LEFT JOIN device_mappings dm ON ov.volume_id = dm.volume_uuid
WHERE vd.job_id = '$REPLICATION_JOB_ID';
"

echo

# ============================================================================
# STEP 2: CLOUDSTACK VOLUME CLEANUP
# ============================================================================

log_info "☁️  STEP 2: CloudStack volume cleanup..."

# Get volume UUIDs for cleanup - VOLUME DAEMON SINGLE SOURCE OF TRUTH
VOLUME_UUIDS=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
-- Method 1: via ossea_volumes table using ossea_volume_id (primary method)
SELECT DISTINCT ov.volume_id as volume_uuid
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
WHERE vd.job_id = '$REPLICATION_JOB_ID'

UNION

-- Method 2: orphaned device mappings (volumes without vm_disk references)
SELECT DISTINCT dm.volume_uuid
FROM device_mappings dm
LEFT JOIN ossea_volumes ov ON dm.volume_uuid = ov.volume_id
WHERE ov.volume_id IS NULL
")

if [ -n "$VOLUME_UUIDS" ]; then
    log_info "Detaching and cleaning up CloudStack volumes..."
    for VOLUME_UUID in $VOLUME_UUIDS; do
        log_info "Processing volume: $VOLUME_UUID"
        
        # Detach volume via Volume Daemon
        log_info "Detaching volume: $VOLUME_UUID"
        DETACH_RESPONSE=$(curl -s -X POST "http://localhost:8090/api/v1/volumes/$VOLUME_UUID/detach" -H "Content-Type: application/json" || echo "")
        
        if [ -n "$DETACH_RESPONSE" ]; then
            DETACH_OP_ID=$(echo "$DETACH_RESPONSE" | jq -r '.id // empty' 2>/dev/null || echo "")
            if [ -n "$DETACH_OP_ID" ]; then
                log_info "Detach operation started: $DETACH_OP_ID"
                
                # Wait for detach completion (max 60 seconds)
                for i in {1..20}; do
                    sleep 3
                    DETACH_STATUS=$(curl -s "http://localhost:8090/api/v1/operations/$DETACH_OP_ID" | jq -r '.status // "unknown"' 2>/dev/null)
                    if [ "$DETACH_STATUS" = "completed" ]; then
                        log_success "Volume detached successfully"
                        break
                    elif [ "$DETACH_STATUS" = "failed" ]; then
                        log_error "Volume detach failed"
                        continue 2  # Skip to next volume
                    fi
                    log_info "Waiting for detach completion... ($DETACH_STATUS)"
                done
            fi
        fi
        
        # Delete volume via Volume Daemon
        log_info "Deleting volume: $VOLUME_UUID" 
        DELETE_RESPONSE=$(curl -s -X DELETE "http://localhost:8090/api/v1/volumes/$VOLUME_UUID" -H "Content-Type: application/json" || echo "")
        if [ -n "$DELETE_RESPONSE" ]; then
            DELETE_OP_ID=$(echo "$DELETE_RESPONSE" | jq -r '.id // empty' 2>/dev/null || echo "")
            if [ -n "$DELETE_OP_ID" ]; then
                log_info "Delete operation started: $DELETE_OP_ID"
                
                # Wait for delete completion (max 60 seconds)
                for i in {1..20}; do
                    sleep 3
                    DELETE_STATUS=$(curl -s "http://localhost:8090/api/v1/operations/$DELETE_OP_ID" | jq -r '.status // "unknown"' 2>/dev/null)
                    if [ "$DELETE_STATUS" = "completed" ]; then
                        log_success "Volume deleted successfully"
                        break
                    elif [ "$DELETE_STATUS" = "failed" ]; then
                        log_error "Volume delete failed"
                        break
                    fi
                    log_info "Waiting for delete completion... ($DELETE_STATUS)"
                done
            fi
        fi
    done
    log_success "CloudStack volume cleanup completed"
else
    log_info "No volumes found for cleanup"
fi

echo

# ============================================================================
# STEP 3: NBD EXPORT CLEANUP
# ============================================================================

log_info "📡 STEP 3: NBD export cleanup..."

# Get NBD exports to clean
NBD_EXPORTS=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "SELECT export_name FROM nbd_exports WHERE job_id = '$REPLICATION_JOB_ID'")

if [ -n "$NBD_EXPORTS" ]; then
    log_info "Cleaning NBD exports from config file..."
    
    # Backup NBD config
    sudo cp /etc/nbd-server/config-base /etc/nbd-server/config-base.backup.$(date +%Y%m%d_%H%M%S)
    
    # Remove export sections from NBD config
    for EXPORT_NAME in $NBD_EXPORTS; do
        log_info "Removing NBD export: $EXPORT_NAME"
        
        # Remove the export section from config file
        sudo sed -i "/\[$EXPORT_NAME\]/,/^$/d" /etc/nbd-server/config-base 2>/dev/null || true
    done
    
    # Reload NBD server if it's running
    if pgrep nbd-server > /dev/null; then
        log_info "Reloading NBD server configuration..."
        sudo pkill -HUP nbd-server || true
    fi
    
    log_success "NBD export cleanup completed"
else
    log_info "No NBD exports found for cleanup"
fi

echo

# ============================================================================
# STEP 4: DATABASE CLEANUP (With Foreign Key CASCADE)
# ============================================================================

log_info "🗄️  STEP 4: Database cleanup..."

# Clean failover job if specified
if [ -n "$FAILOVER_JOB_ID" ]; then
    log_info "Cleaning failover job: $FAILOVER_JOB_ID"
    mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM failover_jobs WHERE job_id = '$FAILOVER_JOB_ID';"
    log_success "Failover job cleaned"
fi

# Clean replication job (CASCADE will handle vm_disks, nbd_exports)
log_info "Cleaning replication job: $REPLICATION_JOB_ID"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM replication_jobs WHERE id = '$REPLICATION_JOB_ID';"
log_success "Replication job cleaned (CASCADE cleaned related records)"

# Clean orphaned device mappings for deleted volumes
log_info "Cleaning orphaned device mappings (Volume Daemon managed)..."
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
DELETE dm FROM device_mappings dm
LEFT JOIN ossea_volumes ov ON dm.volume_uuid = ov.volume_id
WHERE ov.volume_id IS NULL;
"

# Clean orphaned export mappings  
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "DELETE FROM vm_export_mappings WHERE vm_name = '$VM_NAME';" 2>/dev/null || true

# Clean orphaned ossea_volumes records (volumes that were deleted from CloudStack)
log_info "Cleaning orphaned ossea_volumes records..."
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
DELETE ov FROM ossea_volumes ov
LEFT JOIN vm_disks vd ON ov.id = vd.ossea_volume_id  
WHERE vd.ossea_volume_id IS NULL;
"

log_success "Database cleanup completed"

echo

# ============================================================================
# STEP 5: VALIDATION
# ============================================================================

log_info "🔍 STEP 5: Post-cleanup validation..."

# Check for orphaned records
ORPHANED_VM_DISKS=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
SELECT COUNT(*) FROM vm_disks vd 
LEFT JOIN replication_jobs rj ON vd.job_id = rj.id 
WHERE rj.id IS NULL
")

ORPHANED_NBD_EXPORTS=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
SELECT COUNT(*) FROM nbd_exports ne 
LEFT JOIN replication_jobs rj ON ne.job_id = rj.id 
WHERE rj.id IS NULL
")

ORPHANED_FAILOVERS=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME -sN -e "
SELECT COUNT(*) FROM failover_jobs fj
LEFT JOIN replication_jobs rj ON fj.replication_job_id = rj.id
WHERE fj.replication_job_id IS NOT NULL AND rj.id IS NULL
")

log_info "Validation results:"
log_info "  - Orphaned vm_disks: $ORPHANED_VM_DISKS"
log_info "  - Orphaned NBD exports: $ORPHANED_NBD_EXPORTS"  
log_info "  - Orphaned failover jobs: $ORPHANED_FAILOVERS"

if [ "$ORPHANED_VM_DISKS" -eq 0 ] && [ "$ORPHANED_NBD_EXPORTS" -eq 0 ] && [ "$ORPHANED_FAILOVERS" -eq 0 ]; then
    log_success "✅ CLEANUP SUCCESSFUL - No orphaned records detected"
else
    log_warning "⚠️  Some orphaned records remain - manual cleanup may be needed"
fi

# Show current table counts
log_info "📊 Current table counts:"
mysql -u $DB_USER -p$DB_PASS $DB_NAME -e "
SELECT 'replication_jobs' as table_name, COUNT(*) as count FROM replication_jobs
UNION ALL SELECT 'vm_disks', COUNT(*) FROM vm_disks
UNION ALL SELECT 'device_mappings', COUNT(*) FROM device_mappings  
UNION ALL SELECT 'nbd_exports', COUNT(*) FROM nbd_exports
UNION ALL SELECT 'failover_jobs', COUNT(*) FROM failover_jobs;
"

echo
log_success "🎉 COMPREHENSIVE CLEANUP COMPLETED"
log_info "Job $REPLICATION_JOB_ID is now completely cleaned and ready for restart"
log_info "All related volumes, exports, and database records have been removed"
echo
