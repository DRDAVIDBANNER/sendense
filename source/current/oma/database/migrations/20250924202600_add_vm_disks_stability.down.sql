-- Rollback: Remove vm_disks stability constraints
-- This rollback migration removes the stability enhancements
-- and restores the original job-based unique constraint

-- Remove the indexes we added
DROP INDEX IF EXISTS idx_vm_disks_ossea_volume ON vm_disks;
DROP INDEX IF EXISTS idx_vm_disks_context_disk_lookup ON vm_disks;

-- Remove the unique constraint we added
ALTER TABLE vm_disks DROP INDEX uk_vm_context_disk;

-- Restore the original unique constraint (job_id, disk_id)
ALTER TABLE vm_disks 
ADD UNIQUE KEY unique_job_disk (job_id, disk_id) USING HASH;








