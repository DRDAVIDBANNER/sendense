-- Migration Rollback: Remove File-Level Restore Tables
-- Date: 2025-10-08
-- Purpose: Rollback restore_mounts table

START TRANSACTION;

-- Drop restore_mounts table (CASCADE DELETE will clean up any orphaned records)
DROP TABLE IF EXISTS restore_mounts;

COMMIT;

