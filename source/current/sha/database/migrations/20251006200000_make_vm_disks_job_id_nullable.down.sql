-- Rollback: Revert vm_disks.job_id to NOT NULL
-- WARNING: This will fail if any vm_disks records have NULL job_id
-- You must clean up those records first or they will be deleted

-- Remove index
DROP INDEX IF EXISTS idx_vm_disks_discovery ON vm_disks;
DROP INDEX IF EXISTS idx_backup_disk ON backup_jobs;

-- Remove disk_id from backup_jobs
ALTER TABLE backup_jobs DROP COLUMN IF EXISTS disk_id;

-- Delete any vm_disks records with NULL job_id (orphaned discovery records)
DELETE FROM vm_disks WHERE job_id IS NULL;

-- Drop FK constraint
ALTER TABLE vm_disks DROP FOREIGN KEY IF EXISTS fk_vm_disks_job;

-- Make job_id NOT NULL again
ALTER TABLE vm_disks 
    MODIFY COLUMN job_id VARCHAR(191) NOT NULL 
    COMMENT 'FK to replication_jobs - required';

-- Re-add FK constraint
ALTER TABLE vm_disks 
    ADD CONSTRAINT fk_vm_disks_job 
    FOREIGN KEY (job_id) REFERENCES replication_jobs(id) 
    ON DELETE CASCADE;

