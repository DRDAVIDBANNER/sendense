-- Migration: Add VMA Progress Tracking Columns
-- Created: 2025-08-22 15:00:00
-- Purpose: Integrate VMA v1.5.0 progress API with replication_jobs table

-- Add VMA progress tracking columns to replication_jobs
ALTER TABLE replication_jobs 
ADD COLUMN vma_sync_type VARCHAR(50) DEFAULT NULL COMMENT 'VMA detected sync type (initial/incremental)',
ADD COLUMN vma_current_phase VARCHAR(100) DEFAULT NULL COMMENT 'VMA current phase (Initializing/Snapshot Creation/Copying Data/Cleanup)',
ADD COLUMN vma_throughput_mbps DECIMAL(10,2) DEFAULT 0.0 COMMENT 'VMA throughput in MB/s',
ADD COLUMN vma_eta_seconds INT DEFAULT NULL COMMENT 'VMA estimated time to completion in seconds',
ADD COLUMN vma_last_poll_at TIMESTAMP NULL COMMENT 'Last time VMA progress was polled',
ADD COLUMN vma_error_classification VARCHAR(50) DEFAULT NULL COMMENT 'VMA error classification (connection/authentication/permission/disk/network/system)',
ADD COLUMN vma_error_details TEXT DEFAULT NULL COMMENT 'VMA detailed error information';

-- Create index for active polling operations
CREATE INDEX idx_replication_jobs_vma_polling 
ON replication_jobs(status, vma_last_poll_at);

-- Create index for error tracking
CREATE INDEX idx_replication_jobs_vma_errors
ON replication_jobs(vma_error_classification, status);

-- Create index for throughput analysis
CREATE INDEX idx_replication_jobs_vma_throughput
ON replication_jobs(vma_throughput_mbps, status, created_at);
