-- Migration: add_nbd_exports (DOWN)
-- Created: 20250901120000
-- Removes NBD export tracking from Volume Daemon database

-- ================================================================
-- Remove NBD Export Management Components
-- ================================================================

-- Drop triggers first (dependent on tables)
DROP TRIGGER IF EXISTS cleanup_nbd_exports_on_device_delete;
DROP TRIGGER IF EXISTS sync_nbd_export_paths_on_device_update;

-- Drop procedures
DROP PROCEDURE IF EXISTS CleanupOrphanedNBDExports;
DROP PROCEDURE IF EXISTS SyncNBDExportPaths;

-- Drop views (recreate original versions)
DROP VIEW IF EXISTS device_mapping_health;
DROP VIEW IF EXISTS volume_daemon_statistics;

-- Drop NBD exports table
DROP TABLE IF EXISTS nbd_exports;

-- ================================================================
-- Restore Original Views
-- ================================================================

-- Restore original device_mapping_health view
CREATE VIEW device_mapping_health AS
SELECT 
    'duplicate_volumes' as issue_type,
    volume_uuid as identifier,
    COUNT(*) as count,
    GROUP_CONCAT(device_path) as details
FROM device_mappings 
GROUP BY volume_uuid 
HAVING COUNT(*) > 1

UNION ALL

SELECT 
    'duplicate_device_paths' as issue_type,
    CONCAT(vm_id, ':', device_path) as identifier,
    COUNT(*) as count,
    GROUP_CONCAT(volume_uuid) as details
FROM device_mappings 
GROUP BY vm_id, device_path 
HAVING COUNT(*) > 1

UNION ALL

SELECT 
    'long_device_paths' as issue_type,
    volume_uuid as identifier,
    LENGTH(device_path) as count,
    device_path as details
FROM device_mappings 
WHERE LENGTH(device_path) > 200

UNION ALL

SELECT 
    'stale_operations' as issue_type,
    id as identifier,
    TIMESTAMPDIFF(MINUTE, created_at, NOW()) as count,
    CONCAT(type, ':', status) as details
FROM volume_operations 
WHERE status IN ('pending', 'executing') 
  AND created_at < DATE_SUB(NOW(), INTERVAL 30 MINUTE);

-- Restore original volume_daemon_statistics view
CREATE VIEW volume_daemon_statistics AS
SELECT 
    (SELECT COUNT(*) FROM device_mappings) as total_mappings,
    (SELECT COUNT(*) FROM device_mappings WHERE operation_mode = 'oma') as oma_mappings,
    (SELECT COUNT(*) FROM device_mappings WHERE operation_mode = 'failover') as failover_mappings,
    (SELECT COUNT(*) FROM volume_operations WHERE status = 'pending') as pending_operations,
    (SELECT COUNT(*) FROM volume_operations WHERE status = 'executing') as executing_operations,
    (SELECT COUNT(*) FROM volume_operations WHERE status = 'failed') as failed_operations,
    (SELECT COUNT(*) FROM device_mapping_health) as health_issues,
    NOW() as last_updated;

-- Log migration rollback
INSERT INTO volume_operation_history (operation_id, previous_status, new_status, changed_at, details)
VALUES (
    'rollback-20250901120000',
    'completed',
    'rolled_back',
    NOW(),
    JSON_OBJECT(
        'migration_type', 'remove_nbd_exports_table',
        'rollback_reason', 'manual_rollback'
    )
);
