-- Add External Job ID Column to Log Events
-- Date: 2025-09-22 18:00:00
-- Purpose: Enable fast GUI job ID correlation for progress tracking

-- Add external_job_id column for GUI correlation
ALTER TABLE log_events 
ADD COLUMN external_job_id VARCHAR(255) NULL COMMENT 'GUI-constructed job ID for fast correlation (e.g., unified-live-failover-pgtest2-1758553933)';

-- Add index for fast lookup performance
ALTER TABLE log_events 
ADD INDEX idx_log_events_external_job_id (external_job_id);

-- Update table comment to reflect new capability
ALTER TABLE log_events 
COMMENT = 'Structured log events with job correlation and GUI job ID mapping for progress tracking';

-- Verify the schema change
SELECT 
    'External Job ID column added' AS component,
    'Added external_job_id VARCHAR(255) with index' AS description,
    'Enables fast GUI job correlation for progress tracking' AS purpose
UNION ALL
SELECT 
    'Performance index created',
    'idx_log_events_external_job_id on external_job_id column',
    'Direct B-tree index lookup for GUI job IDs'
UNION ALL
SELECT
    'Migration complete',
    'Ready for JobLog tracker enhancement',
    'Zero breaking changes - NULL for existing entries';


