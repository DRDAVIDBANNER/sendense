-- Remove persistent device naming fields from device_mappings table

ALTER TABLE device_mappings 
DROP INDEX idx_device_mappings_persistent_name,
DROP INDEX idx_device_mappings_symlink_path,
DROP COLUMN persistent_device_name,
DROP COLUMN symlink_path;

