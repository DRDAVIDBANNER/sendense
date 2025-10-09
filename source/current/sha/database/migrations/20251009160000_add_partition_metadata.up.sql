-- Migration: Add partition metadata field to restore_mounts table
-- Date: 2025-10-09
-- Purpose: Support multi-partition mounts in file-level restore

ALTER TABLE restore_mounts
ADD COLUMN partition_metadata JSON COMMENT 'Partition details for multi-partition mounts';
