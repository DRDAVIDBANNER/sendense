-- Migration rollback: add_job_tracking
-- Created: 20250121130000

-- Drop views first
DROP VIEW IF EXISTS active_polling_queue;
DROP VIEW IF EXISTS job_tracking_hierarchy;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS cloudstack_job_metrics;
DROP TABLE IF EXISTS cloudstack_job_poll_queue;
DROP TABLE IF EXISTS cloudstack_job_execution_log;
DROP TABLE IF EXISTS cloudstack_job_tracking;



