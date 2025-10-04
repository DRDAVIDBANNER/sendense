-- Remove snapshot tracking fields from ossea_volumes table

ALTER TABLE ossea_volumes 
DROP INDEX idx_ossea_volumes_snapshot_id,
DROP INDEX idx_ossea_volumes_snapshot_status,
DROP COLUMN snapshot_id,
DROP COLUMN snapshot_created_at,
DROP COLUMN snapshot_status;

