-- Rollback migration for multi-volume snapshot enhancement
-- Removes snapshot tracking fields from device_mappings table
-- 
-- WARNING: This will remove all snapshot tracking data
-- Only use if rolling back the multi-volume snapshot feature

-- Remove indexes first
ALTER TABLE device_mappings 
DROP INDEX IF EXISTS idx_device_mappings_vm_context_snapshot,
DROP INDEX IF EXISTS idx_device_mappings_snapshot_status,
DROP INDEX IF EXISTS idx_device_mappings_snapshot_id;

-- Remove snapshot tracking columns
ALTER TABLE device_mappings 
DROP COLUMN IF EXISTS snapshot_status,
DROP COLUMN IF EXISTS snapshot_created_at,
DROP COLUMN IF EXISTS ossea_snapshot_id;

