-- Migration: fix_nbd_exports_schema_conflict (ROLLBACK)
-- Created: 20250904120000
-- Rolls back nbd_exports schema fix: volume_uuid → volume_id

-- ================================================================
-- ROLLBACK: SCHEMA CONFLICT RESOLUTION
-- ================================================================
-- Reverses: nbd_exports.volume_uuid → nbd_exports.volume_id

-- ================================================================
-- Step 1: Drop existing foreign key constraint
-- ================================================================
ALTER TABLE nbd_exports DROP FOREIGN KEY fk_nbd_exports_volume;

-- ================================================================
-- Step 2: Rename column volume_uuid back to volume_id
-- ================================================================
ALTER TABLE nbd_exports CHANGE COLUMN volume_uuid volume_id VARCHAR(64) NOT NULL;

-- ================================================================
-- Step 3: Recreate foreign key constraint with original column name
-- ================================================================
ALTER TABLE nbd_exports 
ADD CONSTRAINT fk_nbd_exports_volume 
    FOREIGN KEY (volume_id) 
    REFERENCES device_mappings(volume_uuid) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- ================================================================
-- Step 4: Update index to use original column name
-- ================================================================
DROP INDEX idx_nbd_exports_volume_uuid ON nbd_exports;
CREATE INDEX idx_nbd_exports_volume_id ON nbd_exports(volume_id);

-- ================================================================
-- Step 5: Rollback stored procedures to original state
-- ================================================================

DELIMITER //

-- Rollback CleanupOrphanedNBDExports procedure
DROP PROCEDURE IF EXISTS CleanupOrphanedNBDExports //

CREATE PROCEDURE CleanupOrphanedNBDExports()
BEGIN
    -- Remove NBD exports that no longer have corresponding device mappings
    DELETE ne FROM nbd_exports ne
    LEFT JOIN device_mappings dm ON ne.volume_id = dm.volume_uuid
    WHERE dm.volume_uuid IS NULL;
    
    -- Update status of exports with missing device paths
    UPDATE nbd_exports ne
    LEFT JOIN device_mappings dm ON ne.volume_id = dm.volume_uuid
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

-- Rollback SyncNBDExportPaths procedure
DROP PROCEDURE IF EXISTS SyncNBDExportPaths //

CREATE PROCEDURE SyncNBDExportPaths()
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE export_volume_id VARCHAR(64);
    DECLARE current_device_path VARCHAR(255);
    DECLARE new_device_path VARCHAR(255);
    
    -- Cursor to find exports with outdated device paths
    DECLARE sync_cursor CURSOR FOR 
        SELECT ne.volume_id, ne.device_path, dm.device_path
        FROM nbd_exports ne
        INNER JOIN device_mappings dm ON ne.volume_id = dm.volume_uuid
        WHERE ne.device_path != dm.device_path
          AND ne.status = 'active';
          
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;
    
    OPEN sync_cursor;
    
    sync_loop: LOOP
        FETCH sync_cursor INTO export_volume_id, current_device_path, new_device_path;
        IF done THEN
            LEAVE sync_loop;
        END IF;
        
        -- Update the NBD export with the current device path
        UPDATE nbd_exports 
        SET device_path = new_device_path,
            updated_at = NOW()
        WHERE volume_id = export_volume_id
          AND device_path = current_device_path;
          
    END LOOP;
    
    CLOSE sync_cursor;
END //

DELIMITER ;

-- ================================================================
-- Step 6: Rollback triggers to original state
-- ================================================================

-- Drop current triggers
DROP TRIGGER IF EXISTS cleanup_nbd_exports_on_device_delete;
DROP TRIGGER IF EXISTS sync_nbd_export_paths_on_device_update;

-- Recreate triggers with original column reference
DELIMITER //

CREATE TRIGGER cleanup_nbd_exports_on_device_delete
    AFTER DELETE ON device_mappings
    FOR EACH ROW
BEGIN
    -- Remove any NBD exports for the deleted device mapping
    DELETE FROM nbd_exports 
    WHERE volume_id = OLD.volume_uuid;
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
        WHERE volume_id = NEW.volume_uuid;
    END IF;
END //

DELIMITER ;

-- ================================================================
-- Migration Rollback Logging
-- ================================================================

-- Log migration rollback
INSERT INTO volume_operation_history (operation_id, previous_status, new_status, changed_at, details)
VALUES (
    'rollback-20250904120000',
    'completed',
    'rolled_back',
    NOW(),
    JSON_OBJECT(
        'migration_type', 'rollback_nbd_exports_schema_conflict',
        'column_reverted', 'volume_uuid -> volume_id',
        'foreign_key_reverted', 'fk_nbd_exports_volume',
        'procedures_reverted', 2,
        'triggers_reverted', 2
    )
);
