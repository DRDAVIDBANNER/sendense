-- Migration: Make vm_disks.job_id nullable to support disk records from discovery
-- Date: 2025-10-06
-- Purpose: Allow vm_disks to be populated at discovery time without requiring replication job

-- Make job_id nullable
ALTER TABLE vm_disks 
    MODIFY COLUMN job_id VARCHAR(191) NULL 
    COMMENT 'FK to replication_jobs - NULL if disk populated from discovery, populated when replication starts';

-- Drop existing FK constraint
ALTER TABLE vm_disks DROP FOREIGN KEY IF EXISTS fk_vm_disks_job;

-- Re-add FK constraint allowing NULL
ALTER TABLE vm_disks 
    ADD CONSTRAINT fk_vm_disks_job 
    FOREIGN KEY (job_id) REFERENCES replication_jobs(id) 
    ON DELETE CASCADE;

-- Note: Unique index uk_vm_context_disk already exists on (vm_context_id, disk_id)
-- No need to add additional index

-- Add disk_id column to backup_jobs for proper disk tracking
ALTER TABLE backup_jobs 
    ADD COLUMN IF NOT EXISTS disk_id INT NOT NULL DEFAULT 0 
    COMMENT 'Disk number (0, 1, 2...) within VM for multi-disk support';

-- Add index for backup queries by VM and disk
CREATE INDEX IF NOT EXISTS idx_backup_disk 
    ON backup_jobs(vm_context_id, disk_id);

