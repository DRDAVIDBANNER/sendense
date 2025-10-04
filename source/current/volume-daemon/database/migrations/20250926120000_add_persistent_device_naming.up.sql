-- Add persistent device naming support to device_mappings table
-- This provides stable device names that persist throughout volume lifecycle operations

ALTER TABLE device_mappings 
ADD COLUMN persistent_device_name VARCHAR(255) NULL 
    COMMENT 'Stable device name for NBD export consistency (e.g., pgtest3disk0)',
ADD COLUMN symlink_path VARCHAR(255) NULL 
    COMMENT 'Device mapper symlink path for persistent access (e.g., /dev/mapper/pgtest3disk0)',
ADD INDEX idx_device_mappings_persistent_name (persistent_device_name),
ADD INDEX idx_device_mappings_symlink_path (symlink_path);

-- Verify migration success: all existing records should have NULL persistent names
SELECT COUNT(*) as total_records,
       SUM(CASE WHEN persistent_device_name IS NULL THEN 1 ELSE 0 END) as null_persistent_names,
       SUM(CASE WHEN symlink_path IS NULL THEN 1 ELSE 0 END) as null_symlink_paths
FROM device_mappings;
-- Expected: total_records = null_persistent_names = null_symlink_paths (all new fields NULL)

