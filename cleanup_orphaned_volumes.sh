#!/bin/bash

# Script to clean up orphaned device mappings via Volume Daemon
# This ensures proper NBD export cleanup and CloudStack volume deletion

set -e  # Exit on any error

echo "=== MigrateKit Orphaned Volume Cleanup Script ==="
echo "Cleaning up device mappings from failed tests using Volume Daemon API..."
echo

# Check Volume Daemon health first
echo "Checking Volume Daemon health..."
if ! curl -s http://localhost:8090/health > /dev/null; then
    echo "❌ Volume Daemon is not responding. Please start it first."
    exit 1
fi

echo "✅ Volume Daemon is healthy"
echo

# Get all orphaned device mappings (those with NULL volume_id_numeric)
echo "Finding orphaned device mappings..."
ORPHANED_VOLUMES=$(mysql -u oma_user -poma_password -D migratekit_oma -se "SELECT volume_uuid FROM device_mappings WHERE volume_id_numeric IS NULL ORDER BY created_at ASC;")

if [ -z "$ORPHANED_VOLUMES" ]; then
    echo "No orphaned volumes found. Database is clean."
    exit 0
fi

# Count total orphaned volumes
TOTAL_VOLUMES=$(echo "$ORPHANED_VOLUMES" | wc -l)
echo "Found $TOTAL_VOLUMES orphaned volumes to clean up"
echo

# Clean up each volume via Volume Daemon
COUNT=0
DELETED=0
FAILED=0

for VOLUME_UUID in $ORPHANED_VOLUMES; do
    COUNT=$((COUNT + 1))
    echo "[$COUNT/$TOTAL_VOLUMES] Cleaning up volume: $VOLUME_UUID"
    
    # Get device path for logging
    DEVICE_PATH=$(mysql -u oma_user -poma_password -D migratekit_oma -se "SELECT device_path FROM device_mappings WHERE volume_uuid='$VOLUME_UUID';")
    echo "  Device path: $DEVICE_PATH"
    
    # Call Volume Daemon DELETE API (will detach first, then delete)
    RESPONSE=$(curl -s -w "%{http_code}" -X DELETE "http://localhost:8090/api/v1/volumes/$VOLUME_UUID")
    HTTP_CODE="${RESPONSE: -3}"
    BODY="${RESPONSE%???}"
    
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ]; then
        if [ "$HTTP_CODE" = "200" ]; then
            echo "  ✅ Successfully cleaned up volume $VOLUME_UUID"
        else
            echo "  ✅ Volume $VOLUME_UUID was already deleted (404 - cleaning database)"
        fi
        
        # Clean up database records manually if Volume Daemon didn't (e.g., 404 cases)
        mysql -u oma_user -poma_password -D migratekit_oma -e "DELETE FROM device_mappings WHERE volume_uuid='$VOLUME_UUID';" 2>/dev/null || true
        mysql -u oma_user -poma_password -D migratekit_oma -e "DELETE FROM nbd_exports WHERE volume_uuid='$VOLUME_UUID';" 2>/dev/null || true
        
        DELETED=$((DELETED + 1))
    else
        echo "  ❌ Failed to clean up volume $VOLUME_UUID (HTTP $HTTP_CODE)"
        if [ -n "$BODY" ]; then
            echo "     Response: $BODY"
        fi
        FAILED=$((FAILED + 1))
    fi
    
    # Small delay to avoid overwhelming the API
    sleep 1
done

echo
echo "=== Cleanup Summary ==="
echo "Total volumes processed: $TOTAL_VOLUMES"
echo "Successfully cleaned: $DELETED"
echo "Failed cleanups: $FAILED"
echo

# Verify cleanup
echo "=== Verification ==="
REMAINING_DEVICE_MAPPINGS=$(mysql -u oma_user -poma_password -D migratekit_oma -se "SELECT COUNT(*) FROM device_mappings WHERE volume_id_numeric IS NULL;")
REMAINING_NBD_EXPORTS=$(mysql -u oma_user -poma_password -D migratekit_oma -se "SELECT COUNT(*) FROM nbd_exports;")

echo "Remaining orphaned device mappings: $REMAINING_DEVICE_MAPPINGS"
echo "Remaining NBD exports: $REMAINING_NBD_EXPORTS"

if [ "$REMAINING_DEVICE_MAPPINGS" = "0" ]; then
    echo "✅ Device mappings cleanup completed successfully!"
else
    echo "⚠️  Warning: $REMAINING_DEVICE_MAPPINGS orphaned device mappings remain"
    echo "Listing remaining orphaned mappings:"
    mysql -u oma_user -poma_password -D migratekit_oma -e "SELECT volume_uuid, device_path, created_at FROM device_mappings WHERE volume_id_numeric IS NULL;"
fi

echo
echo "Orphaned volume cleanup script completed."
