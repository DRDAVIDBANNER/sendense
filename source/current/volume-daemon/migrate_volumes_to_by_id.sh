#!/bin/bash

# Volume Migration Script: Convert old device paths to by-id paths
# This updates existing volumes to use stable by-id device paths

set -e

echo "🔄 Volume Migration: Converting to by-id paths"
echo "================================================"

# Function to construct by-id path from volume UUID
construct_by_id_path() {
    local volume_uuid="$1"
    local clean_uuid=$(echo "$volume_uuid" | tr -d '-')
    local short_id=${clean_uuid:0:20}
    echo "/dev/disk/by-id/virtio-$short_id"
}

# Function to validate by-id path exists
validate_by_id_path() {
    local by_id_path="$1"
    if [ -L "$by_id_path" ]; then
        local actual_device=$(readlink -f "$by_id_path")
        if [ -b "$actual_device" ]; then
            echo "$actual_device"
            return 0
        fi
    fi
    return 1
}

# Get volumes that need migration
volumes=$(mysql -u oma_user -poma_password migratekit_oma -se "
SELECT volume_uuid 
FROM device_mappings 
WHERE device_path NOT LIKE '/dev/disk/by-id/%' 
  AND operation_mode = 'oma'
  AND device_path NOT LIKE 'remote-vm-%'
ORDER BY device_path
" 2>/dev/null)

if [ -z "$volumes" ]; then
    echo "✅ No volumes need migration - all already using by-id paths"
    exit 0
fi

echo "Found $(echo "$volumes" | wc -l) volumes to migrate:"
echo "$volumes" | while read volume_uuid; do
    echo "  - $volume_uuid"
done
echo ""

# Process each volume
success_count=0
error_count=0

echo "$volumes" | while read volume_uuid; do
    echo "🔄 Processing volume: $volume_uuid"
    
    # Get current device path
    current_path=$(mysql -u oma_user -poma_password migratekit_oma -se "
        SELECT device_path FROM device_mappings WHERE volume_uuid = '$volume_uuid'
    " 2>/dev/null)
    
    echo "  Current path: $current_path"
    
    # Construct by-id path
    by_id_path=$(construct_by_id_path "$volume_uuid")
    echo "  by-id path:   $by_id_path"
    
    # Validate by-id path exists
    if actual_device=$(validate_by_id_path "$by_id_path"); then
        echo "  ✅ by-id path exists → $actual_device"
        
        # Update device_mappings
        mysql -u oma_user -poma_password migratekit_oma -e "
            UPDATE device_mappings 
            SET device_path = '$by_id_path',
                persistent_device_name = NULL,
                symlink_path = NULL,
                updated_at = NOW()
            WHERE volume_uuid = '$volume_uuid'
        " 2>/dev/null
        
        if [ $? -eq 0 ]; then
            echo "  ✅ Database updated"
            
            # Update NBD export if exists
            nbd_exists=$(mysql -u oma_user -poma_password migratekit_oma -se "
                SELECT COUNT(*) FROM nbd_exports WHERE volume_id = '$volume_uuid'
            " 2>/dev/null)
            
            if [ "$nbd_exists" -gt 0 ]; then
                mysql -u oma_user -poma_password migratekit_oma -e "
                    UPDATE nbd_exports 
                    SET device_path = '$by_id_path',
                        updated_at = NOW()
                    WHERE volume_id = '$volume_uuid'
                " 2>/dev/null
                
                # Update NBD config file
                export_name="migration-vol-$volume_uuid"
                config_file="/etc/nbd-server/conf.d/$export_name.conf"
                
                if [ -f "$config_file" ]; then
                    sudo tee "$config_file" > /dev/null <<EOF
[$export_name]
exportname = $by_id_path
readonly = false
multifile = false
copyonwrite = false
EOF
                    echo "  ✅ NBD config updated"
                else
                    echo "  ⚠️  NBD config file not found: $config_file"
                fi
            else
                echo "  ℹ️  No NBD export (normal for non-exported volumes)"
            fi
            
            echo "  🎉 Volume migration COMPLETE"
            ((success_count++))
        else
            echo "  ❌ Database update failed"
            ((error_count++))
        fi
    else
        echo "  ❌ by-id path not found - volume may be detached"
        echo "  ℹ️  Skipping (will be fixed when volume reattached)"
        ((error_count++))
    fi
    
    echo ""
done

echo "================================================"
echo "🎯 Migration Summary:"
echo "  ✅ Successful: $success_count volumes"
echo "  ❌ Errors:     $error_count volumes"
echo ""

if [ $success_count -gt 0 ]; then
    echo "🔄 Reloading NBD server to pick up config changes..."
    sudo kill -HUP $(pgrep nbd-server) 2>/dev/null || echo "⚠️  NBD server not running"
    echo "✅ NBD server reloaded"
fi

echo "🎉 Volume migration to by-id paths complete!"

