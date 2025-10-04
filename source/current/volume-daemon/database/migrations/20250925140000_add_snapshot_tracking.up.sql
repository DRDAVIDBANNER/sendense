-- Multi-Volume Snapshot Enhancement Migration
-- Adds per-volume snapshot tracking to device_mappings table
-- This enables complete multi-disk VM protection during test failover operations
-- 
-- SAFETY: Additive changes only, no data loss risk
-- Compatible with existing operations

-- Add snapshot tracking fields to device_mappings
ALTER TABLE device_mappings 
ADD COLUMN ossea_snapshot_id VARCHAR(191) NULL 
    COMMENT 'CloudStack volume snapshot ID for rollback protection during test failover',
ADD COLUMN snapshot_created_at TIMESTAMP NULL 
    COMMENT 'Timestamp when snapshot was created during failover operation',
ADD COLUMN snapshot_status VARCHAR(50) DEFAULT 'none' 
    COMMENT 'Snapshot status: none, creating, ready, failed, rollback_complete';

-- Add indexes for efficient snapshot queries
ALTER TABLE device_mappings 
ADD INDEX idx_device_mappings_snapshot_id (ossea_snapshot_id),
ADD INDEX idx_device_mappings_snapshot_status (snapshot_status),
ADD INDEX idx_device_mappings_vm_context_snapshot (vm_context_id, snapshot_status);

-- Verify migration success: all existing records should have default 'none' status
-- SELECT COUNT(*) as existing_records, 
--        SUM(CASE WHEN snapshot_status = 'none' THEN 1 ELSE 0 END) as default_status_count
-- FROM device_mappings;
-- Expected: existing_records = default_status_count (all records have 'none' status)

