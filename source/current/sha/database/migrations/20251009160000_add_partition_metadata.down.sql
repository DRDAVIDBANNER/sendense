-- Migration: Remove partition metadata field from restore_mounts table
-- Date: 2025-10-09
-- Purpose: Rollback multi-partition mounts support

ALTER TABLE restore_mounts
DROP COLUMN partition_metadata;

