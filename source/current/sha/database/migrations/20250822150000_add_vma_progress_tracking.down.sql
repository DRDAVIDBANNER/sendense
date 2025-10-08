-- Migration Rollback: Remove VMA Progress Tracking Columns
-- Created: 2025-08-22 15:00:00
-- Purpose: Rollback VMA v1.5.0 progress API integration

-- Drop indexes first
DROP INDEX IF EXISTS idx_replication_jobs_vma_throughput ON replication_jobs;
DROP INDEX IF EXISTS idx_replication_jobs_vma_errors ON replication_jobs;
DROP INDEX IF EXISTS idx_replication_jobs_vma_polling ON replication_jobs;

-- Remove VMA progress tracking columns from replication_jobs
ALTER TABLE replication_jobs 
DROP COLUMN IF EXISTS vma_error_details,
DROP COLUMN IF EXISTS vma_error_classification,
DROP COLUMN IF EXISTS vma_last_poll_at,
DROP COLUMN IF EXISTS vma_eta_seconds,
DROP COLUMN IF EXISTS vma_throughput_mbps,
DROP COLUMN IF EXISTS vma_current_phase,
DROP COLUMN IF EXISTS vma_sync_type;

