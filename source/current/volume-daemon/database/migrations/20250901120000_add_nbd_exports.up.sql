-- Migration: add_nbd_exports
-- Created: 20250901120000
-- Adds NBD export tracking table to Volume Daemon database

-- ================================================================
-- NBD Exports Table for Volume Daemon Integration
-- ================================================================

-- NBD export tracking for volume-to-export mapping
CREATE TABLE IF NOT EXISTS nbd_exports (
    id VARCHAR(64) PRIMARY KEY,                          -- Export record ID
    volume_id VARCHAR(64) NOT NULL,                      -- CloudStack volume UUID (matches device_mappings.volume_uuid)
    export_name VARCHAR(255) NOT NULL UNIQUE,            -- NBD export name (e.g., migration-vm-{uuid}-disk0)
    device_path VARCHAR(255) NOT NULL,                   -- Linux device path (/dev/vdb, /dev/vdc, etc.)
    port INT NOT NULL DEFAULT 10809,                     -- NBD server port
    status ENUM('pending', 'active', 'failed') NOT NULL DEFAULT 'pending',
    metadata JSON NULL,                                  -- Additional metadata (VM info, creation details)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Indexes for performance
    INDEX idx_nbd_exports_volume_id (volume_id),
    INDEX idx_nbd_exports_export_name (export_name),
    INDEX idx_nbd_exports_device_path (device_path),
    INDEX idx_nbd_exports_status (status),
    INDEX idx_nbd_exports_port (port),
    INDEX idx_nbd_exports_created_at (created_at),
    
    -- Foreign key to device_mappings (volume correlation)
    CONSTRAINT fk_nbd_exports_volume 
        FOREIGN KEY (volume_id) 
        REFERENCES device_mappings(volume_uuid) 
        ON DELETE CASCADE ON UPDATE CASCADE
);

-- ================================================================
-- NBD Export Cleanup Procedures
-- ================================================================

-- Procedure to cleanup orphaned NBD exports
DELIMITER //

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

-- Procedure to sync NBD export device paths with current mappings
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
-- Triggers for Automatic NBD Export Management
-- ================================================================

-- Trigger to auto-cleanup NBD exports when device mappings are deleted
DELIMITER //

CREATE TRIGGER cleanup_nbd_exports_on_device_delete
    AFTER DELETE ON device_mappings
    FOR EACH ROW
BEGIN
    -- Remove any NBD exports for the deleted device mapping
    DELETE FROM nbd_exports 
    WHERE volume_id = OLD.volume_uuid;
END //

-- Trigger to update NBD export device paths when device mappings change
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
-- Enhanced Views for NBD Export Monitoring
-- ================================================================

-- Update the health view to include NBD export issues
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
    ne.volume_id as identifier,
    1 as count,
    CONCAT('export:', ne.export_name, ' device:', ne.device_path) as details
FROM nbd_exports ne
LEFT JOIN device_mappings dm ON ne.volume_id = dm.volume_uuid
WHERE dm.volume_uuid IS NULL

UNION ALL

SELECT 
    'mismatched_export_paths' as issue_type,
    ne.volume_id as identifier,
    1 as count,
    CONCAT('export_path:', ne.device_path, ' mapping_path:', dm.device_path) as details
FROM nbd_exports ne
INNER JOIN device_mappings dm ON ne.volume_id = dm.volume_uuid
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

-- Enhanced statistics view including NBD exports
DROP VIEW IF EXISTS volume_daemon_statistics;

CREATE VIEW volume_daemon_statistics AS
SELECT 
    (SELECT COUNT(*) FROM device_mappings) as total_mappings,
    (SELECT COUNT(*) FROM device_mappings WHERE operation_mode = 'oma') as oma_mappings,
    (SELECT COUNT(*) FROM device_mappings WHERE operation_mode = 'failover') as failover_mappings,
    (SELECT COUNT(*) FROM nbd_exports) as total_nbd_exports,
    (SELECT COUNT(*) FROM nbd_exports WHERE status = 'active') as active_nbd_exports,
    (SELECT COUNT(*) FROM nbd_exports WHERE status = 'failed') as failed_nbd_exports,
    (SELECT COUNT(*) FROM volume_operations WHERE status = 'pending') as pending_operations,
    (SELECT COUNT(*) FROM volume_operations WHERE status = 'executing') as executing_operations,
    (SELECT COUNT(*) FROM volume_operations WHERE status = 'failed') as failed_operations,
    (SELECT COUNT(*) FROM device_mapping_health) as health_issues,
    NOW() as last_updated;

-- ================================================================
-- Initial Cleanup
-- ================================================================

-- Run initial NBD export cleanup
CALL CleanupOrphanedNBDExports();

-- Log migration completion
INSERT INTO volume_operation_history (operation_id, previous_status, new_status, changed_at, details)
VALUES (
    'migration-20250901120000',
    'pending',
    'completed',
    NOW(),
    JSON_OBJECT(
        'migration_type', 'add_nbd_exports_table',
        'nbd_exports_created', (SELECT COUNT(*) FROM nbd_exports),
        'device_mappings_total', (SELECT COUNT(*) FROM device_mappings),
        'triggers_created', 3,
        'procedures_created', 2
    )
);
