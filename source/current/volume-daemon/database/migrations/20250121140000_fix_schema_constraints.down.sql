-- Migration rollback: fix_schema_constraints
-- Created: 20250121140000

-- ================================================================
-- Rollback Step 1: Drop views
-- ================================================================

DROP VIEW IF EXISTS volume_daemon_statistics;
DROP VIEW IF EXISTS device_mapping_health;

-- ================================================================
-- Rollback Step 2: Drop triggers
-- ================================================================

DROP TRIGGER IF EXISTS prevent_duplicate_volumes;
DROP TRIGGER IF EXISTS prevent_duplicate_device_paths;

-- ================================================================
-- Rollback Step 3: Drop procedures
-- ================================================================

DROP PROCEDURE IF EXISTS CleanupOrphanedOperations;
DROP PROCEDURE IF EXISTS CleanupDuplicateDeviceMappings;

-- ================================================================
-- Rollback Step 4: Restore original device_mappings table structure
-- ================================================================

-- Create original table structure
CREATE TABLE IF NOT EXISTS device_mappings_original (
    id VARCHAR(64) PRIMARY KEY,
    volume_id VARCHAR(64) NOT NULL UNIQUE,
    vm_id VARCHAR(64) NOT NULL,
    device_path VARCHAR(32) NOT NULL,
    cloudstack_state VARCHAR(32) NOT NULL,
    linux_state VARCHAR(32) NOT NULL,
    size BIGINT NOT NULL,
    last_sync TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_volume_id (volume_id),
    UNIQUE KEY unique_device_path (device_path),
    INDEX idx_device_mappings_vm_id (vm_id),
    INDEX idx_device_mappings_device_path (device_path),
    INDEX idx_device_mappings_last_sync (last_sync)
);

-- Migrate data back (with truncation for device_path)
INSERT IGNORE INTO device_mappings_original (
    id, volume_id, vm_id, device_path, cloudstack_state, 
    linux_state, size, last_sync, created_at, updated_at
)
SELECT 
    id,
    volume_uuid as volume_id,
    vm_id,
    CASE 
        WHEN LENGTH(device_path) > 32 THEN SUBSTRING(device_path, 1, 32)
        ELSE device_path
    END as device_path,
    cloudstack_state,
    linux_state,
    size,
    last_sync,
    created_at,
    updated_at
FROM device_mappings;

-- Replace new table with original structure
DROP TABLE device_mappings;
ALTER TABLE device_mappings_original RENAME TO device_mappings;

-- ================================================================
-- Rollback Step 5: Remove added indexes from volume_operations
-- ================================================================

ALTER TABLE volume_operations 
DROP INDEX IF EXISTS idx_volume_operations_volume_vm,
DROP INDEX IF EXISTS idx_volume_operations_type_status;

-- ================================================================
-- Rollback verification
-- ================================================================

-- Log rollback completion
INSERT INTO volume_operation_history (operation_id, previous_status, new_status, changed_at, details)
VALUES (
    'rollback-20250121140000',
    'completed',
    'rolled_back',
    NOW(),
    JSON_OBJECT(
        'rollback_type', 'schema_constraints_fix',
        'device_mappings_restored', (SELECT COUNT(*) FROM device_mappings),
        'data_truncated', 'device_path column restored to VARCHAR(32)'
    )
);



