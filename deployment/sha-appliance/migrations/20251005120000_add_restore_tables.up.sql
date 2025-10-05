-- Migration: Add restore_mounts table for file-level restore functionality
-- Task 4: File-Level Restore (Phase 1 - QCOW2 Mount Management)
-- Date: 2025-10-05
-- Purpose: Track active QCOW2 backup mounts for file browsing and recovery

-- restore_mounts table: Tracks active QCOW2 backup mounts
CREATE TABLE IF NOT EXISTS restore_mounts (
    -- Identity
    id VARCHAR(64) PRIMARY KEY COMMENT 'Unique mount identifier (UUID)',
    
    -- Backup reference
    backup_id VARCHAR(64) NOT NULL COMMENT 'Reference to backup_jobs.id',
    
    -- Mount details
    mount_path VARCHAR(512) NOT NULL COMMENT 'Filesystem mount point (/mnt/sendense/restore/...)',
    nbd_device VARCHAR(32) NOT NULL COMMENT 'NBD device path (/dev/nbd0-7)',
    filesystem_type VARCHAR(32) COMMENT 'Detected filesystem type (ext4, xfs, ntfs, etc.)',
    
    -- Mount configuration
    mount_mode ENUM('read-only') DEFAULT 'read-only' COMMENT 'Mount access mode',
    
    -- Lifecycle tracking
    status ENUM('mounting', 'mounted', 'unmounting', 'failed') DEFAULT 'mounting' COMMENT 'Mount lifecycle status',
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Mount creation time',
    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last access time for idle detection',
    expires_at TIMESTAMP NULL COMMENT 'Automatic cleanup time (created_at + 1 hour)',
    
    -- Indexes for performance
    INDEX idx_restore_mounts_backup_id (backup_id),
    INDEX idx_restore_mounts_expires_at (expires_at),
    INDEX idx_restore_mounts_status (status),
    INDEX idx_restore_mounts_nbd_device (nbd_device),
    
    -- Foreign key relationships
    CONSTRAINT fk_restore_mounts_backup_id 
        FOREIGN KEY (backup_id) 
        REFERENCES backup_jobs(id) 
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Track active QCOW2 backup mounts for file-level restore';

-- Create unique constraint to prevent duplicate NBD device allocations
CREATE UNIQUE INDEX idx_restore_mounts_nbd_device_unique 
    ON restore_mounts(nbd_device) 
    WHERE status IN ('mounting', 'mounted');

-- Create unique constraint to prevent duplicate mount paths
CREATE UNIQUE INDEX idx_restore_mounts_mount_path_unique 
    ON restore_mounts(mount_path) 
    WHERE status IN ('mounting', 'mounted');
