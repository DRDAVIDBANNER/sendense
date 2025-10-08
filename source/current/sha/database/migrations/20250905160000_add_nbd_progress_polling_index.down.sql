-- Migration: Remove NBD Progress Polling Index
-- Created: 2025-09-05 16:00:00
-- Purpose: Rollback optimization indexes for VMA progress polling

-- Remove indexes added for VMA progress polling optimization
DROP INDEX IF EXISTS idx_replication_jobs_nbd_polling ON replication_jobs;
DROP INDEX IF EXISTS idx_replication_jobs_progress_timeout ON replication_jobs;
DROP INDEX IF EXISTS idx_replication_jobs_completion_tracking ON replication_jobs;
