-- Migration: Add telemetry tracking fields
-- Date: 2025-10-10
-- Purpose: Add real-time progress tracking fields for backup job telemetry

-- Add telemetry tracking fields to backup_jobs
ALTER TABLE backup_jobs 
    ADD COLUMN current_phase VARCHAR(50) DEFAULT 'pending' COMMENT 'Current phase of backup: snapshot, transferring, finalizing',
    ADD COLUMN transfer_speed_bps BIGINT DEFAULT 0 COMMENT 'Current transfer speed in bytes per second',
    ADD COLUMN eta_seconds INT DEFAULT 0 COMMENT 'Estimated time to completion in seconds',
    ADD COLUMN progress_percent DECIMAL(5,2) DEFAULT 0.0 COMMENT 'Overall progress percentage (0.00-100.00)',
    ADD COLUMN last_telemetry_at DATETIME NULL COMMENT 'Last telemetry update timestamp for stale detection';

-- Add progress tracking to backup_disks
ALTER TABLE backup_disks
    ADD COLUMN progress_percent DECIMAL(5,2) DEFAULT 0.0 COMMENT 'Per-disk progress percentage (0.00-100.00)';

-- Index for stale job detection (composite index for efficient queries)
CREATE INDEX idx_last_telemetry ON backup_jobs(status, last_telemetry_at);

-- Note: This migration supports push-based telemetry from SBC to SHA
-- Replaces polling-based progress tracking with real-time updates

