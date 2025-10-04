-- Migration: Create vm_export_mappings table for persistent VM-to-export mapping
-- Purpose: Enable export reuse to prevent unnecessary SIGHUP operations
-- Date: 2025-08-12

CREATE TABLE IF NOT EXISTS vm_export_mappings (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    vm_id VARCHAR(36) NOT NULL COMMENT 'VMware VM UUID',
    disk_unit_number INT NOT NULL COMMENT 'SCSI unit number (0,1,2...)',
    vm_name VARCHAR(255) NOT NULL COMMENT 'VMware VM name for reference',
    export_name VARCHAR(255) NOT NULL UNIQUE COMMENT 'NBD export name (migration-vm-{id}-disk{unit})',
    device_path VARCHAR(255) NOT NULL COMMENT 'Linux device path (/dev/vdb, /dev/vdc, etc.)',
    status ENUM('active', 'inactive') DEFAULT 'active' COMMENT 'Export availability status',
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    
    -- Constraints
    UNIQUE KEY unique_vm_disk (vm_id, disk_unit_number),
    UNIQUE KEY unique_device_path (device_path),
    
    -- Indexes for performance
    INDEX idx_vm_id (vm_id),
    INDEX idx_export_name (export_name),
    INDEX idx_device_path (device_path),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci 
COMMENT='Persistent mapping between VMware VMs and NBD exports to enable export reuse';
