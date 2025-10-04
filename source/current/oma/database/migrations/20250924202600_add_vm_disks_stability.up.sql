-- Migration: Add vm_disks stability for multi-disk corruption fix
-- This migration enables stable vm_disks records across job lifecycles
-- by adding a unique constraint on vm_context_id + disk_id combination

-- STEP 1: Clean up duplicate vm_disks records first
-- Keep the most recent record for each (vm_context_id, disk_id) combination
-- and remove older duplicates

-- Identify duplicates and keep only the latest record for each vm_context_id + disk_id
DELETE t1 FROM vm_disks t1
INNER JOIN vm_disks t2 
WHERE t1.vm_context_id = t2.vm_context_id 
  AND t1.disk_id = t2.disk_id 
  AND t1.id < t2.id;

-- STEP 2: Remove the existing unique constraint that prevents our fix (if it exists)
-- The current constraint is per-job (job_id, disk_id) but we need per-context (vm_context_id, disk_id)
-- Use conditional DROP to handle cases where index may not exist
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.statistics 
     WHERE table_schema = DATABASE() AND table_name = 'vm_disks' AND index_name = 'unique_job_disk') > 0,
    'ALTER TABLE vm_disks DROP INDEX unique_job_disk',
    'SELECT "Index unique_job_disk does not exist, skipping"'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- STEP 3: Add unique constraint to ensure vm_context_id + disk_id combination is unique
-- This prevents duplicate disk records for the same VM context
-- and enables stable vm_disks.id across multiple replication jobs
-- Note: disk_id is longtext, so we need to specify a prefix length for the constraint
ALTER TABLE vm_disks 
ADD UNIQUE KEY uk_vm_context_disk (vm_context_id, disk_id(255));

-- STEP 4: Add indexes for performance on lookups
-- Note: disk_id is longtext, so we need to specify a prefix length for the index
CREATE INDEX idx_vm_disks_context_disk_lookup ON vm_disks (vm_context_id, disk_id(255));

-- Add index for ossea_volume_id correlation (needed for NBD export mapping)
CREATE INDEX idx_vm_disks_ossea_volume ON vm_disks (ossea_volume_id);
