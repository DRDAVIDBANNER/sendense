-- Add snapshot tracking fields to ossea_volumes table
-- This provides stable snapshot tracking that survives volume detach/attach operations

ALTER TABLE ossea_volumes 
ADD COLUMN snapshot_id VARCHAR(191) NULL 
    COMMENT 'CloudStack volume snapshot ID for test failover protection',
ADD COLUMN snapshot_created_at TIMESTAMP NULL 
    COMMENT 'Timestamp when snapshot was created during test failover',
ADD COLUMN snapshot_status VARCHAR(50) DEFAULT 'none' 
    COMMENT 'Snapshot status: none, creating, ready, failed, rollback_complete';

-- Add indexes for efficient snapshot queries
ALTER TABLE ossea_volumes 
ADD INDEX idx_ossea_volumes_snapshot_id (snapshot_id),
ADD INDEX idx_ossea_volumes_snapshot_status (snapshot_status);

-- Verify migration success: all existing records should have default 'none' status
SELECT COUNT(*) as existing_records, 
       SUM(CASE WHEN snapshot_status = 'none' THEN 1 ELSE 0 END) as default_status_count
FROM ossea_volumes;
-- Expected: existing_records = default_status_count (all records have 'none' status)

