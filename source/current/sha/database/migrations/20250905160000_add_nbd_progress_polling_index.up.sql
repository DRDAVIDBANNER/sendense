-- Migration: Add NBD Progress Polling Index
-- Created: 2025-09-05 16:00:00
-- Purpose: Optimize VMA progress polling queries using nbd_export_name

-- Add index for efficient VMA progress poller discovery of active jobs
-- Query pattern: SELECT id, nbd_export_name FROM replication_jobs WHERE status = 'running' AND nbd_export_name IS NOT NULL
-- Use prefix index for nbd_export_name to avoid key length limit
CREATE INDEX idx_replication_jobs_nbd_polling 
ON replication_jobs(status, nbd_export_name(50));

-- Add index for progress timeout detection
-- Query pattern: SELECT id FROM replication_jobs WHERE status = 'running' AND updated_at < NOW() - INTERVAL 5 MINUTE
CREATE INDEX idx_replication_jobs_progress_timeout
ON replication_jobs(status, updated_at);

-- Add index for job completion tracking  
-- Query pattern: SELECT id FROM replication_jobs WHERE status IN ('completed', 'failed') AND completed_at > ?
CREATE INDEX idx_replication_jobs_completion_tracking
ON replication_jobs(status, completed_at);
