-- Rollback Migration: Remove restore_mounts table
-- Task 4: File-Level Restore (Phase 1 - QCOW2 Mount Management)
-- Date: 2025-10-05

-- Drop the restore_mounts table
DROP TABLE IF EXISTS restore_mounts;

