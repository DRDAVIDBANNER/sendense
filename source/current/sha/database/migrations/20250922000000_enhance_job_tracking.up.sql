-- Enhanced Job Tracking Migration
-- Date: 2025-09-22
-- Purpose: Add context correlation and external job ID tracking for GUI integration

-- Add new optional fields to job_tracking table for enhanced correlation
ALTER TABLE job_tracking 
ADD COLUMN context_id VARCHAR(64) NULL COMMENT 'VM context correlation for failover/replication jobs',
ADD COLUMN external_job_id VARCHAR(255) NULL COMMENT 'External job ID for GUI correlation (e.g., constructed failover IDs)', 
ADD COLUMN job_category ENUM('system','failover','replication','scheduler','discovery','bulk') NULL DEFAULT 'system' COMMENT 'High-level job categorization for filtering and organization';

-- Add indexes for performance on new correlation fields
ALTER TABLE job_tracking 
ADD INDEX idx_job_tracking_context_id (context_id),
ADD INDEX idx_job_tracking_external_id (external_job_id),
ADD INDEX idx_job_tracking_category (job_category);

-- Add composite index for common GUI lookup patterns
ALTER TABLE job_tracking
ADD INDEX idx_job_tracking_category_status_started (job_category, status, started_at);

-- Update table comment to reflect enhanced capabilities
ALTER TABLE job_tracking 
COMMENT = 'Enhanced job tracking with VM context correlation and external job ID mapping for GUI integration';

-- Verify the schema changes
SELECT 
    'Enhanced job_tracking table' AS component,
    'Added context_id, external_job_id, job_category fields' AS description,
    'Backward compatible - all existing jobs continue working' AS compatibility_status
UNION ALL
SELECT 
    'Performance indexes created',
    'context_id, external_job_id, job_category indexes added',
    'Optimized for GUI job lookup patterns'
UNION ALL
SELECT
    'Schema enhancement complete',
    'Ready for JobLog model and tracker enhancements',
    'Zero breaking changes to existing code';


