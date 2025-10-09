-- Migration: Add File-Level Restore Tables
-- Date: 2025-10-08
-- Purpose: Add restore_mounts table for QCOW2 backup mounting and file-level restore
-- Task: File-Level Restore (Task 4 - Phase 1 VMware Backups)
-- Job Sheet: TBD

START TRANSACTION;

-- ============================================================================
-- Restore Mounts Table
-- ============================================================================
-- Tracks active QCOW2 backup mounts for file-level restore operations
-- Uses qemu-nbd + NBD devices (/dev/nbd0-7) for mounting backups
-- 
-- CASCADE DELETE Chain:
--   vm_backup_contexts → backup_jobs → backup_disks → restore_mounts
--   When a backup disk is deleted, its mounts are automatically cleaned up
CREATE TABLE IF NOT EXISTS restore_mounts (
    id VARCHAR(64) PRIMARY KEY COMMENT 'Mount UUID',
    backup_disk_id BIGINT NOT NULL COMMENT 'FK to backup_disks.id',
    
    -- Mount configuration
    mount_path VARCHAR(512) NOT NULL COMMENT '/mnt/sendense/restore/{uuid}',
    nbd_device VARCHAR(32) NOT NULL COMMENT '/dev/nbd0-7 allocation',
    filesystem_type VARCHAR(32) DEFAULT NULL COMMENT 'ext4, xfs, ntfs, etc.',
    mount_mode ENUM('read-only', 'read-write') DEFAULT 'read-only',
    
    -- Mount status
    status ENUM('mounting', 'mounted', 'unmounting', 'failed', 'unmounted') NOT NULL DEFAULT 'mounting',
    error_message TEXT DEFAULT NULL,
    
    -- Access tracking (for idle cleanup)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL DEFAULT NULL COMMENT 'created_at + idle_timeout',
    unmounted_at TIMESTAMP NULL DEFAULT NULL,
    
    -- Foreign keys (CASCADE DELETE integration with backup_disks)
    FOREIGN KEY fk_restore_mount_disk (backup_disk_id) 
        REFERENCES backup_disks(id) ON DELETE CASCADE,
    
    -- Indexes for performance
    INDEX idx_backup_disk (backup_disk_id),
    INDEX idx_status (status),
    INDEX idx_nbd_device (nbd_device),
    INDEX idx_expires (expires_at),
    INDEX idx_last_accessed (last_accessed_at),
    
    -- Ensure one mount per NBD device
    UNIQUE KEY uk_nbd_device (nbd_device),
    
    -- Ensure one mount per backup disk
    UNIQUE KEY uk_backup_disk (backup_disk_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Tracks QCOW2 backup mounts for file-level restore';

-- ============================================================================
-- Create mount directory if it doesn't exist (requires shell access)
-- ============================================================================
-- This is a reference only - actual directory creation happens in code
-- mkdir -p /mnt/sendense/restore

COMMIT;

-- ============================================================================
-- VERIFICATION QUERIES (for post-migration validation)
-- ============================================================================
-- Run these manually after migration to verify:

-- Verify table exists:
-- SHOW CREATE TABLE restore_mounts\G

-- Check for foreign key constraints:
-- SELECT 
--   CONSTRAINT_NAME, 
--   TABLE_NAME, 
--   COLUMN_NAME, 
--   REFERENCED_TABLE_NAME, 
--   REFERENCED_COLUMN_NAME
-- FROM information_schema.KEY_COLUMN_USAGE
-- WHERE TABLE_SCHEMA = DATABASE()
--   AND TABLE_NAME = 'restore_mounts'
--   AND REFERENCED_TABLE_NAME IS NOT NULL;

-- Check indexes:
-- SHOW INDEX FROM restore_mounts;

