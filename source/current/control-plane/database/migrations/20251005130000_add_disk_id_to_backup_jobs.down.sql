-- Migration Rollback: Remove disk_id column from backup_jobs table
-- Task 4: File-Level Restore - Schema fix rollback
-- Date: 2025-10-05

-- Drop the index
DROP INDEX idx_backup_vm_disk ON backup_jobs;

-- Remove disk_id column
ALTER TABLE backup_jobs DROP COLUMN disk_id;
