-- Migration: fix_nbd_exports_schema_conflict
-- Created: 20250904120000
-- Fixes schema conflict: nbd_exports.volume_id → nbd_exports.volume_uuid to match device_mappings

-- ================================================================
-- CRITICAL SCHEMA CONFLICT RESOLUTION
-- ================================================================
-- Problem: nbd_exports table uses 'volume_id' but references device_mappings.volume_uuid
-- Solution: Rename nbd_exports.volume_id → nbd_exports.volume_uuid for consistency

-- ================================================================
-- Step 1: Drop existing foreign key constraint
-- ================================================================
ALTER TABLE nbd_exports DROP FOREIGN KEY fk_nbd_exports_volume;

-- ================================================================
-- Step 2: Rename column volume_id to volume_uuid
-- ================================================================
ALTER TABLE nbd_exports CHANGE COLUMN volume_id volume_uuid VARCHAR(64) NOT NULL;

-- ================================================================
-- Step 3: Recreate foreign key constraint with correct column name
-- ================================================================
ALTER TABLE nbd_exports 
ADD CONSTRAINT fk_nbd_exports_volume 
    FOREIGN KEY (volume_uuid) 
    REFERENCES device_mappings(volume_uuid) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- ================================================================
-- Step 4: Update index to use new column name
-- ================================================================
DROP INDEX idx_nbd_exports_volume_id ON nbd_exports;
CREATE INDEX idx_nbd_exports_volume_uuid ON nbd_exports(volume_uuid);

-- ================================================================
-- Step 5: Update stored procedures to use correct column name
-- ================================================================

DELIMITER //

-- Update CleanupOrphanedNBDExports procedure
DROP PROCEDURE IF EXISTS CleanupOrphanedNBDExports //

CREATE PROCEDURE CleanupOrphanedNBDExports()
BEGIN
    -- Remove NBD exports that no longer have corresponding device mappings
    DELETE ne FROM nbd_exports ne
    LEFT JOIN device_mappings dm ON ne.volume_uuid = dm.volume_uuid
    WHERE dm.volume_uuid IS NULL;
    
    -- Update status of exports with missing device paths
    UPDATE nbd_exports ne
    LEFT JOIN device_mappings dm ON ne.volume_uuid = dm.volume_uuid
    SET ne.status = 'failed',
        ne.updated_at = NOW()
    WHERE ne.device_path != dm.device_path
      AND ne.status = 'active';
      
    -- Log cleanup activity
    INSERT INTO volume_operation_history (operation_id, previous_status, new_status, changed_at, details)
    VALUES (
        CONCAT('nbd-cleanup-', UNIX_TIMESTAMP()),
        'pending',
        'completed', 
        NOW(),
        JSON_OBJECT(
            'cleanup_type', 'orphaned_nbd_exports',
            'exports_cleaned', ROW_COUNT()
        )
    );
END //

-- Update SyncNBDExportPaths procedure
DROP PROCEDURE IF EXISTS SyncNBDExportPaths //

CREATE PROCEDURE SyncNBDExportPaths()
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE export_volume_uuid VARCHAR(64);
    DECLARE current_device_path VARCHAR(255);
    DECLARE new_device_path VARCHAR(255);
    
    -- Cursor to find exports with outdated device paths
    DECLARE sync_cursor CURSOR FOR 
        SELECT ne.volume_uuid, ne.device_path, dm.device_path
        FROM nbd_exports ne
        INNER JOIN device_mappings dm ON ne.volume_uuid = dm.volume_uuid
        WHERE ne.device_path != dm.device_path
          AND ne.status = 'active';
          
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;
    
    OPEN sync_cursor;
    
    sync_loop: LOOP
        FETCH sync_cursor INTO export_volume_uuid, current_device_path, new_device_path;
        IF done THEN
            LEAVE sync_loop;
        END IF;
        
        -- Update the NBD export with the current device path
        UPDATE nbd_exports 
        SET device_path = new_device_path,
            updated_at = NOW()
        WHERE volume_uuid = export_volume_uuid
          AND device_path = current_device_path;
          
    END LOOP;
    
    CLOSE sync_cursor;
END //

DELIMITER ;

-- ================================================================
-- Step 6: Update triggers to use correct column name
-- ================================================================

-- Drop existing triggers
DROP TRIGGER IF EXISTS cleanup_nbd_exports_on_device_delete;
DROP TRIGGER IF EXISTS sync_nbd_export_paths_on_device_update;

-- Recreate triggers with correct column reference
DELIMITER //

CREATE TRIGGER cleanup_nbd_exports_on_device_delete
    AFTER DELETE ON device_mappings
    FOR EACH ROW
BEGIN
    -- Remove any NBD exports for the deleted device mapping
    DELETE FROM nbd_exports 
    WHERE volume_uuid = OLD.volume_uuid;
END //

CREATE TRIGGER sync_nbd_export_paths_on_device_update
    AFTER UPDATE ON device_mappings
    FOR EACH ROW
BEGIN
    -- Update NBD export device path if the device mapping changed
    IF OLD.device_path != NEW.device_path THEN
        UPDATE nbd_exports 
        SET device_path = NEW.device_path,
            updated_at = NOW()
        WHERE volume_uuid = NEW.volume_uuid;
    END IF;
END //

DELIMITER ;

-- ================================================================
-- Step 7: Update views to use correct column name
-- ================================================================

-- Recreate the health view with correct column reference
DROP VIEW IF EXISTS device_mapping_health;

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
    'orphaned_nbd_exports' as issue_type,
    ne.volume_uuid as identifier,
    1 as count,
    CONCAT('export:', ne.export_name, ' device:', ne.device_path) as details
FROM nbd_exports ne
LEFT JOIN device_mappings dm ON ne.volume_uuid = dm.volume_uuid
WHERE dm.volume_uuid IS NULL

UNION ALL

SELECT 
    'mismatched_export_paths' as issue_type,
    ne.volume_uuid as identifier,
    1 as count,
    CONCAT('export_path:', ne.device_path, ' mapping_path:', dm.device_path) as details
FROM nbd_exports ne
INNER JOIN device_mappings dm ON ne.volume_uuid = dm.volume_uuid
WHERE ne.device_path != dm.device_path

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

-- ================================================================
-- Step 8: Run cleanup to ensure consistency
-- ================================================================

-- Run the updated cleanup procedures
CALL CleanupOrphanedNBDExports();

-- ================================================================
-- Migration Verification and Logging
-- ================================================================

-- Log migration completion
INSERT INTO volume_operation_history (operation_id, previous_status, new_status, changed_at, details)
VALUES (
    'migration-20250904120000',
    'pending',
    'completed',
    NOW(),
    JSON_OBJECT(
        'migration_type', 'fix_nbd_exports_schema_conflict',
        'column_renamed', 'volume_id -> volume_uuid',
        'foreign_key_fixed', 'fk_nbd_exports_volume',
        'procedures_updated', 2,
        'triggers_updated', 2,
        'views_updated', 1,
        'total_nbd_exports', (SELECT COUNT(*) FROM nbd_exports),
        'orphaned_exports_cleaned', 0
    )
);
