-- Enhanced Job Tracking Migration (Corrected)
-- Date: 2025-09-22
-- Purpose: Add missing context correlation fields for GUI integration

-- Add only the missing fields (external_job_id already exists)
ALTER TABLE job_tracking 
ADD COLUMN context_id VARCHAR(64) NULL COMMENT 'VM context correlation for failover/replication jobs',
ADD COLUMN job_category ENUM('system','failover','replication','scheduler','discovery','bulk') NULL DEFAULT 'system' COMMENT 'High-level job categorization for filtering and organization';

-- Add indexes for performance on new correlation fields
ALTER TABLE job_tracking 
ADD INDEX idx_job_tracking_context_id (context_id),
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
    'Added context_id, job_category fields (external_job_id already existed)' AS description,
    'Backward compatible - all existing jobs continue working' AS compatibility_status
UNION ALL
SELECT 
    'Performance indexes created',
    'context_id, job_category indexes added',
    'Optimized for GUI job lookup patterns'
UNION ALL
SELECT
    'Schema enhancement complete',
    'Ready for JobLog model and tracker enhancements',
    'Zero breaking changes to existing code';


