-- Migration: Add disk_id column to backup_jobs table
-- Task 4: File-Level Restore - Schema fix discovered during testing
-- Date: 2025-10-05
-- Purpose: Support multi-disk VM backups

-- Add disk_id column after vm_name
ALTER TABLE backup_jobs 
ADD COLUMN disk_id INT NOT NULL DEFAULT 0 AFTER vm_name;

-- Add index for better query performance (backup chain queries)
CREATE INDEX idx_backup_vm_disk ON backup_jobs(vm_context_id, disk_id, backup_type);

-- Add comment for documentation
ALTER TABLE backup_jobs 
MODIFY COLUMN disk_id INT NOT NULL DEFAULT 0 COMMENT 'Disk identifier for multi-disk VMs (0 for first disk, 1 for second, etc.)';
