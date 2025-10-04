-- Database Validation and Audit Script
-- Created: 2025-08-21
-- Purpose: Validate normalized database schema and relationships

-- ============================================================================
-- SCHEMA VALIDATION CHECKS
-- ============================================================================

-- Check 1: Verify all foreign key constraints exist
SELECT 
    'Foreign Key Constraints' as check_type,
    CONSTRAINT_NAME,
    TABLE_NAME,
    COLUMN_NAME,
    REFERENCED_TABLE_NAME,
    REFERENCED_COLUMN_NAME
FROM information_schema.KEY_COLUMN_USAGE 
WHERE REFERENCED_TABLE_NAME IS NOT NULL 
AND TABLE_SCHEMA = 'migratekit_oma'
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- Check 2: Verify unique constraints
SELECT 
    'Unique Constraints' as check_type,
    TABLE_NAME,
    CONSTRAINT_NAME,
    CONSTRAINT_TYPE
FROM information_schema.TABLE_CONSTRAINTS 
WHERE CONSTRAINT_TYPE = 'UNIQUE' 
AND TABLE_SCHEMA = 'migratekit_oma'
ORDER BY TABLE_NAME;

-- Check 3: Verify indexes exist
SELECT 
    'Indexes' as check_type,
    TABLE_NAME,
    INDEX_NAME,
    COLUMN_NAME,
    NON_UNIQUE
FROM information_schema.STATISTICS 
WHERE TABLE_SCHEMA = 'migratekit_oma'
AND INDEX_NAME != 'PRIMARY'
ORDER BY TABLE_NAME, INDEX_NAME;

-- ============================================================================
-- DATA INTEGRITY CHECKS
-- ============================================================================

-- Check 4: Orphaned records (should be 0 with FK constraints)
SELECT 'Orphaned vm_disks (no replication job)' as check_type, COUNT(*) as count
FROM vm_disks vd 
LEFT JOIN replication_jobs rj ON vd.job_id = rj.id 
WHERE rj.id IS NULL;

-- Check 5: NBD exports without proper relationships
SELECT 'Orphaned NBD exports (no job)' as check_type, COUNT(*) as count
FROM nbd_exports ne 
LEFT JOIN replication_jobs rj ON ne.job_id = rj.id 
WHERE rj.id IS NULL;

SELECT 'NBD exports without vm_disk' as check_type, COUNT(*) as count
FROM nbd_exports ne 
LEFT JOIN vm_disks vd ON ne.vm_disk_id = vd.id 
WHERE ne.vm_disk_id IS NOT NULL AND vd.id IS NULL;

-- Check 6: Device mappings consistency
SELECT 'Device mappings with duplicate paths' as check_type, COUNT(*) as violations
FROM (
    SELECT device_path, COUNT(*) as count 
    FROM device_mappings 
    GROUP BY device_path 
    HAVING COUNT(*) > 1
) duplicates;

-- Check 7: Volume UUID consistency
SELECT 'vm_disks with invalid volume_uuid refs' as check_type, COUNT(*) as count
FROM vm_disks vd 
LEFT JOIN device_mappings dm ON vd.cloudstack_volume_uuid = dm.volume_uuid 
WHERE vd.cloudstack_volume_uuid IS NOT NULL AND dm.volume_uuid IS NULL;

-- ============================================================================
-- CURRENT STATE SUMMARY
-- ============================================================================

-- Summary of all table counts
SELECT 'replication_jobs' as table_name, COUNT(*) as count FROM replication_jobs
UNION ALL
SELECT 'vm_disks', COUNT(*) FROM vm_disks  
UNION ALL
SELECT 'device_mappings', COUNT(*) FROM device_mappings
UNION ALL  
SELECT 'nbd_exports', COUNT(*) FROM nbd_exports
UNION ALL
SELECT 'volume_operations', COUNT(*) FROM volume_operations
ORDER BY table_name;

-- Active vs completed jobs
SELECT 
    status,
    COUNT(*) as count
FROM replication_jobs 
GROUP BY status
ORDER BY status;

-- Device mapping modes
SELECT 
    operation_mode,
    COUNT(*) as count  
FROM device_mappings
GROUP BY operation_mode;

-- NBD export status
SELECT 
    status,
    COUNT(*) as count
FROM nbd_exports  
GROUP BY status;
