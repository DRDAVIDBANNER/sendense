-- COMPREHENSIVE JOB CLEANUP SYSTEM
-- ===================================
-- Purpose: Clean up failed/restart jobs completely across all systems
-- Created: 2025-08-21
-- Handles: Replication jobs, Failover jobs, Volumes, NBD exports, Database consistency

-- ============================================================================
-- STEP 1: CLEANUP SPECIFIC JOB (Replace JOB_ID_TO_CLEAN)
-- ============================================================================

-- Example usage:
-- SET @job_to_clean = 'job-20250821-075220';
-- SET @failover_job_to_clean = 'failover-20250821-123456';

-- ============================================================================
-- REPLICATION JOB CLEANUP PROCEDURE
-- ============================================================================

-- 1. Find all related records for replication job
SELECT 'Replication Job Cleanup Analysis' as step;

SELECT 
    'vm_disks' as table_name,
    COUNT(*) as records
FROM vm_disks 
WHERE job_id = @job_to_clean

UNION ALL

SELECT 
    'nbd_exports' as table_name,
    COUNT(*) as records  
FROM nbd_exports
WHERE job_id = @job_to_clean

UNION ALL

SELECT 
    'vm_export_mappings' as table_name,
    COUNT(*) as records
FROM vm_export_mappings
WHERE job_id = @job_to_clean

UNION ALL

SELECT 
    'failover_jobs' as table_name,
    COUNT(*) as records
FROM failover_jobs  
WHERE replication_job_id = @job_to_clean;

-- 2. Get volume information before cleanup
SELECT 
    'Volumes to be cleaned' as step,
    vd.cloudstack_volume_uuid,
    vd.ossea_volume_id,
    dm.device_path,
    dm.cloudstack_state
FROM vm_disks vd
LEFT JOIN device_mappings dm ON vd.cloudstack_volume_uuid = dm.volume_uuid
WHERE vd.job_id = @job_to_clean;

-- 3. Get NBD exports to be removed
SELECT 
    'NBD exports to be cleaned' as step,
    export_name,
    device_path,
    status
FROM nbd_exports
WHERE job_id = @job_to_clean;

-- ============================================================================
-- FAILOVER JOB CLEANUP PROCEDURE  
-- ============================================================================

-- 1. Find failover job details
SELECT 'Failover Job Cleanup Analysis' as step;

SELECT 
    job_id,
    vm_id,
    replication_job_id,
    job_type,
    status,
    destination_vm_id,
    ossea_snapshot_id
FROM failover_jobs
WHERE job_id = @failover_job_to_clean;

-- ============================================================================
-- SAFE CLEANUP EXECUTION (Manual verification required)
-- ============================================================================

-- WARNING: Review output above before executing cleanup!
-- Replace @job_to_clean and @failover_job_to_clean with actual values

-- STEP A: Clean up replication job (CASCADE will handle related records)
-- DELETE FROM replication_jobs WHERE id = @job_to_clean;

-- STEP B: Clean up specific failover job  
-- DELETE FROM failover_jobs WHERE job_id = @failover_job_to_clean;

-- STEP C: Clean up orphaned device mappings (if volumes deleted from CloudStack)
-- DELETE FROM device_mappings WHERE volume_uuid IN (
--     SELECT volume_uuid FROM device_mappings dm
--     LEFT JOIN vm_disks vd ON dm.volume_uuid = vd.cloudstack_volume_uuid
--     WHERE vd.cloudstack_volume_uuid IS NULL
-- );

-- ============================================================================
-- VALIDATION AFTER CLEANUP
-- ============================================================================

-- Check for orphaned records
SELECT 'Post-cleanup validation' as step;

SELECT 'Orphaned vm_disks' as check_type, COUNT(*) as count
FROM vm_disks vd 
LEFT JOIN replication_jobs rj ON vd.job_id = rj.id 
WHERE rj.id IS NULL

UNION ALL

SELECT 'Orphaned NBD exports', COUNT(*)
FROM nbd_exports ne 
LEFT JOIN replication_jobs rj ON ne.job_id = rj.id 
WHERE rj.id IS NULL

UNION ALL

SELECT 'Orphaned failover jobs', COUNT(*)
FROM failover_jobs fj
LEFT JOIN replication_jobs rj ON fj.replication_job_id = rj.id
WHERE fj.replication_job_id IS NOT NULL AND rj.id IS NULL

UNION ALL

SELECT 'Unused device mappings', COUNT(*)
FROM device_mappings dm
LEFT JOIN vm_disks vd ON dm.volume_uuid = vd.cloudstack_volume_uuid
WHERE vd.cloudstack_volume_uuid IS NULL;

-- Summary of current state
SELECT 'Current table counts' as summary;

SELECT 'replication_jobs' as table_name, COUNT(*) as count FROM replication_jobs
UNION ALL SELECT 'vm_disks', COUNT(*) FROM vm_disks
UNION ALL SELECT 'device_mappings', COUNT(*) FROM device_mappings  
UNION ALL SELECT 'nbd_exports', COUNT(*) FROM nbd_exports
UNION ALL SELECT 'failover_jobs', COUNT(*) FROM failover_jobs;
