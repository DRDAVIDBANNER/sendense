-- Rollback Enhanced Job Tracking Migration
-- Date: 2025-09-22
-- Purpose: Rollback job tracking enhancements if needed

-- Remove composite index
ALTER TABLE job_tracking
DROP INDEX idx_job_tracking_category_status_started;

-- Remove individual indexes
ALTER TABLE job_tracking
DROP INDEX idx_job_tracking_context_id,
DROP INDEX idx_job_tracking_external_id, 
DROP INDEX idx_job_tracking_category;

-- Remove enhanced columns (data will be lost)
ALTER TABLE job_tracking
DROP COLUMN context_id,
DROP COLUMN external_job_id,
DROP COLUMN job_category;

-- Restore original table comment
ALTER TABLE job_tracking 
COMMENT = 'Job tracking and step management with integrated logging';

-- Verification
SELECT 
    'Enhanced job tracking rollback complete' AS status,
    'Removed context_id, external_job_id, job_category fields' AS changes,
    'All existing JobLog code remains functional' AS compatibility;


