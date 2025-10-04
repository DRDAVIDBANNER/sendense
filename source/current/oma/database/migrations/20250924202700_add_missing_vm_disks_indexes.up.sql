-- Migration: Add missing indexes for vm_disks optimization
-- The unique constraint uk_vm_context_disk already exists
-- We just need to add the missing performance indexes

-- Add index for ossea_volume_id correlation (needed for NBD export mapping)
CREATE INDEX IF NOT EXISTS idx_vm_disks_ossea_volume ON vm_disks (ossea_volume_id);








