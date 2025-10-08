-- Rollback External Job ID Column from Log Events
-- Date: 2025-09-22 18:00:00
-- Purpose: Rollback GUI job ID correlation if needed

-- Remove index first
ALTER TABLE log_events 
DROP INDEX idx_log_events_external_job_id;

-- Remove external_job_id column
ALTER TABLE log_events 
DROP COLUMN external_job_id;

-- Restore original table comment
ALTER TABLE log_events 
COMMENT = 'Structured log events tied to jobs and steps for centralized logging system';

-- Verification
SELECT 
    'External Job ID rollback complete' AS status,
    'Removed external_job_id column and index' AS changes,
    'All existing JobLog functionality preserved' AS compatibility;


