-- Migration: fix_schema_constraints
-- Created: 20250121140000
-- Fixes database schema constraints for device_path length and duplicate handling

-- ================================================================
-- Step 1: Update device_mappings table to match Volume Daemon models
-- ================================================================

-- First, create new improved device_mappings table with correct structure
CREATE TABLE IF NOT EXISTS device_mappings_new (
    id VARCHAR(64) PRIMARY KEY,
    volume_uuid VARCHAR(64) NOT NULL,                    -- Updated column name to match models
    volume_id_numeric BIGINT NULL,                       -- Added numeric ID for CloudStack compatibility
    vm_id VARCHAR(64) NOT NULL,
    operation_mode ENUM('oma', 'failover') NOT NULL DEFAULT 'oma',  -- Added operation mode
    cloudstack_device_id INT NULL,                       -- CloudStack device ID
    requires_device_correlation BOOLEAN NOT NULL DEFAULT false,     -- Correlation flag
    device_path VARCHAR(255) NOT NULL,                   -- INCREASED from 32 to 255 characters
    cloudstack_state VARCHAR(32) NOT NULL,
    linux_state VARCHAR(32) NOT NULL,
    size BIGINT NOT NULL,
    last_sync TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Enhanced constraints with better duplicate handling
    UNIQUE KEY unique_volume_uuid (volume_uuid),                     -- Primary volume constraint
    UNIQUE KEY unique_vm_device_path (vm_id, device_path),          -- Prevent same device path per VM
    INDEX idx_device_mappings_vm_id (vm_id),
    INDEX idx_device_mappings_device_path (device_path),
    INDEX idx_device_mappings_operation_mode (operation_mode),
    INDEX idx_device_mappings_cloudstack_device_id (cloudstack_device_id),
    INDEX idx_device_mappings_last_sync (last_sync),
    INDEX idx_device_mappings_volume_numeric (volume_id_numeric)
);

-- ================================================================
-- Step 2: Data Migration with Duplicate Resolution
-- ================================================================

-- Migrate existing data, handling duplicates by keeping the newest record
INSERT INTO device_mappings_new (
    id, volume_uuid, volume_id_numeric, vm_id, operation_mode,
    cloudstack_device_id, requires_device_correlation, device_path,
    cloudstack_state, linux_state, size, last_sync, created_at, updated_at
)
SELECT 
    CONCAT('migrated-', SUBSTRING(MD5(CONCAT(volume_id, vm_id, device_path)), 1, 16)) as id,
    volume_id as volume_uuid,                            -- Map old volume_id to volume_uuid
    NULL as volume_id_numeric,                           -- Will be populated by recovery service
    vm_id,
    'oma' as operation_mode,                             -- Default to OMA mode
    NULL as cloudstack_device_id,                        -- Will be populated by recovery service
    true as requires_device_correlation,                 -- Mark for correlation
    CASE 
        WHEN LENGTH(device_path) > 255 THEN SUBSTRING(device_path, 1, 255)
        ELSE device_path
    END as device_path,                                  -- Truncate if too long
    cloudstack_state,
    linux_state,
    size,
    last_sync,
    created_at,
    updated_at
FROM device_mappings dm1
WHERE dm1.created_at = (
    -- Only keep the newest record for each volume_id to resolve duplicates
    SELECT MAX(dm2.created_at) 
    FROM device_mappings dm2 
    WHERE dm2.volume_id = dm1.volume_id
);

-- ================================================================
-- Step 3: Replace Old Table
-- ================================================================

-- Drop the old table and rename the new one
DROP TABLE device_mappings;
ALTER TABLE device_mappings_new RENAME TO device_mappings;

-- ================================================================
-- Step 4: Update volume_operations table for better constraint handling
-- ================================================================

-- Add additional indexes and constraints to volume_operations
ALTER TABLE volume_operations 
ADD INDEX idx_volume_operations_volume_vm (volume_id, vm_id),
ADD INDEX idx_volume_operations_type_status (type, status);

-- ================================================================
-- Step 5: Create duplicate cleanup procedures
-- ================================================================

-- Create a stored procedure for ongoing duplicate cleanup
DELIMITER //

CREATE PROCEDURE CleanupDuplicateDeviceMappings()
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE duplicate_volume_uuid VARCHAR(64);
    
    -- Cursor to find duplicate volume UUIDs
    DECLARE duplicate_cursor CURSOR FOR 
        SELECT volume_uuid 
        FROM device_mappings 
        GROUP BY volume_uuid 
        HAVING COUNT(*) > 1;
        
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;
    
    OPEN duplicate_cursor;
    
    duplicate_loop: LOOP
        FETCH duplicate_cursor INTO duplicate_volume_uuid;
        IF done THEN
            LEAVE duplicate_loop;
        END IF;
        
        -- Keep only the newest record for each duplicate volume_uuid
        DELETE dm1 FROM device_mappings dm1
        INNER JOIN device_mappings dm2 
        WHERE dm1.volume_uuid = dm2.volume_uuid 
          AND dm1.volume_uuid = duplicate_volume_uuid
          AND dm1.created_at < dm2.created_at;
          
    END LOOP;
    
    CLOSE duplicate_cursor;
END //

CREATE PROCEDURE CleanupOrphanedOperations()
BEGIN
    -- Clean up volume operations that have been stuck for more than 1 hour
    UPDATE volume_operations 
    SET status = 'failed', 
        error = 'Operation timed out during migration',
        updated_at = NOW()
    WHERE status IN ('pending', 'executing') 
      AND created_at < DATE_SUB(NOW(), INTERVAL 1 HOUR);
      
    -- Clean up operations with no corresponding device mappings (for completed attach operations)
    UPDATE volume_operations vo
    LEFT JOIN device_mappings dm ON vo.volume_id = dm.volume_uuid
    SET vo.status = 'completed',
        vo.updated_at = NOW(),
        vo.completed_at = NOW()
    WHERE vo.type = 'attach' 
      AND vo.status = 'executing'
      AND dm.volume_uuid IS NOT NULL;
END //

DELIMITER ;

-- ================================================================
-- Step 6: Enhanced duplicate prevention triggers
-- ================================================================

-- Trigger to prevent duplicate device paths per VM
DELIMITER //

CREATE TRIGGER prevent_duplicate_device_paths
    BEFORE INSERT ON device_mappings
    FOR EACH ROW
BEGIN
    DECLARE existing_count INT DEFAULT 0;
    
    -- Check if device path already exists for this VM
    SELECT COUNT(*) INTO existing_count
    FROM device_mappings 
    WHERE vm_id = NEW.vm_id 
      AND device_path = NEW.device_path 
      AND volume_uuid != NEW.volume_uuid;
    
    IF existing_count > 0 THEN
        SIGNAL SQLSTATE '45000' 
        SET MESSAGE_TEXT = 'Device path already mapped to another volume for this VM';
    END IF;
END //

CREATE TRIGGER prevent_duplicate_volumes
    BEFORE INSERT ON device_mappings
    FOR EACH ROW
BEGIN
    DECLARE existing_count INT DEFAULT 0;
    
    -- Check if volume already has a mapping
    SELECT COUNT(*) INTO existing_count
    FROM device_mappings 
    WHERE volume_uuid = NEW.volume_uuid;
    
    IF existing_count > 0 THEN
        SIGNAL SQLSTATE '45000' 
        SET MESSAGE_TEXT = 'Volume already has an existing device mapping';
    END IF;
END //

DELIMITER ;

-- ================================================================
-- Step 7: Create views for monitoring and debugging
-- ================================================================

-- View to identify potential issues
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

-- View for operational statistics
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

-- ================================================================
-- Step 8: Run initial cleanup
-- ================================================================

-- Run the cleanup procedures
CALL CleanupDuplicateDeviceMappings();
CALL CleanupOrphanedOperations();

-- ================================================================
-- Migration Verification
-- ================================================================

-- Log migration completion
INSERT INTO volume_operation_history (operation_id, previous_status, new_status, changed_at, details)
VALUES (
    'migration-20250121140000',
    'pending',
    'completed',
    NOW(),
    JSON_OBJECT(
        'migration_type', 'schema_constraints_fix',
        'device_mappings_migrated', (SELECT COUNT(*) FROM device_mappings),
        'operations_cleaned', (SELECT COUNT(*) FROM volume_operations WHERE error LIKE '%timed out during migration%'),
        'health_issues', (SELECT COUNT(*) FROM device_mapping_health)
    )
);



