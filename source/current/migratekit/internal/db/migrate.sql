-- Enhanced Job Tracking and Logging Database Migration
-- Created: 2025-01-22
-- Purpose: Augment existing job_tracking table and add step/log tables for unified logging system

-- First, enhance the existing job_tracking table with new fields
ALTER TABLE job_tracking 
ADD COLUMN percent_complete TINYINT UNSIGNED NULL DEFAULT NULL,
ADD COLUMN canceled_at DATETIME(6) NULL DEFAULT NULL,
ADD COLUMN owner VARCHAR(100) NULL DEFAULT NULL;

-- Add index for better query performance on status and timing
ALTER TABLE job_tracking 
ADD INDEX idx_job_tracking_status_started (status, started_at);

-- Create job_steps table for detailed step tracking
CREATE TABLE job_steps (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(64) NOT NULL,
    name VARCHAR(200) NOT NULL,
    seq INT NOT NULL,
    status ENUM('running','completed','failed','skipped') NOT NULL DEFAULT 'running',
    started_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    completed_at DATETIME(6) NULL DEFAULT NULL,
    error_message LONGTEXT NULL,
    metadata JSON NULL,
    
    -- Foreign key to job_tracking with cascade delete
    CONSTRAINT fk_job_steps_job_id 
        FOREIGN KEY (job_id) REFERENCES job_tracking(id) 
        ON DELETE CASCADE,
    
    -- Unique constraint to prevent duplicate step sequences per job
    CONSTRAINT uk_job_steps_job_seq UNIQUE (job_id, seq),
    
    -- Indexes for performance
    INDEX idx_job_steps_job_id_seq (job_id, seq),
    INDEX idx_job_steps_status (status),
    INDEX idx_job_steps_started_at (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create log_events table for structured logging
CREATE TABLE log_events (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(64) NULL,
    step_id BIGINT NULL,
    level ENUM('DEBUG','INFO','WARN','ERROR') NOT NULL,
    message TEXT NOT NULL,
    attrs JSON NULL,
    ts DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    
    -- Foreign keys with cascade rules
    CONSTRAINT fk_log_events_job_id 
        FOREIGN KEY (job_id) REFERENCES job_tracking(id) 
        ON DELETE SET NULL,
    
    CONSTRAINT fk_log_events_step_id 
        FOREIGN KEY (step_id) REFERENCES job_steps(id) 
        ON DELETE SET NULL,
    
    -- Indexes for efficient log querying
    INDEX idx_log_events_job_step_ts (job_id, step_id, ts),
    INDEX idx_log_events_level_ts (level, ts),
    INDEX idx_log_events_ts (ts),
    INDEX idx_log_events_job_id (job_id),
    INDEX idx_log_events_step_id (step_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create a view for easy job progress monitoring
CREATE VIEW job_progress AS
SELECT 
    jt.id AS job_id,
    jt.job_type,
    jt.operation,
    jt.status AS job_status,
    jt.percent_complete,
    jt.started_at,
    jt.completed_at,
    jt.owner,
    COUNT(js.id) AS total_steps,
    COUNT(CASE WHEN js.status = 'completed' THEN 1 END) AS completed_steps,
    COUNT(CASE WHEN js.status = 'failed' THEN 1 END) AS failed_steps,
    COUNT(CASE WHEN js.status = 'running' THEN 1 END) AS running_steps,
    CASE 
        WHEN COUNT(js.id) > 0 THEN 
            ROUND((COUNT(CASE WHEN js.status IN ('completed', 'skipped') THEN 1 END) / COUNT(js.id)) * 100, 2)
        ELSE 0 
    END AS step_completion_percentage
FROM job_tracking jt
LEFT JOIN job_steps js ON jt.id = js.job_id
GROUP BY jt.id, jt.job_type, jt.operation, jt.status, jt.percent_complete, jt.started_at, jt.completed_at, jt.owner;

-- Create a view for active jobs monitoring
CREATE VIEW active_jobs AS
SELECT 
    jt.id AS job_id,
    jt.parent_job_id,
    jt.job_type,
    jt.operation,
    jt.status,
    jt.percent_complete,
    jt.owner,
    jt.started_at,
    TIMESTAMPDIFF(SECOND, jt.started_at, NOW()) AS runtime_seconds,
    COUNT(js.id) AS total_steps,
    COUNT(CASE WHEN js.status = 'running' THEN 1 END) AS active_steps,
    MAX(js.started_at) AS last_step_start
FROM job_tracking jt
LEFT JOIN job_steps js ON jt.id = js.job_id
WHERE jt.status IN ('pending', 'running')
GROUP BY jt.id, jt.parent_job_id, jt.job_type, jt.operation, jt.status, jt.percent_complete, jt.owner, jt.started_at
ORDER BY jt.started_at DESC;

-- Create optimized indexes for common query patterns
ALTER TABLE job_tracking 
ADD INDEX idx_job_tracking_owner (owner),
ADD INDEX idx_job_tracking_parent (parent_job_id),
ADD INDEX idx_job_tracking_type_status (job_type, status);

-- Create indexes for log analysis and cleanup
ALTER TABLE log_events 
ADD INDEX idx_log_events_ts_level (ts, level);

-- Add comments for documentation
ALTER TABLE job_steps 
COMMENT = 'Individual steps within jobs for detailed progress tracking and logging';

ALTER TABLE log_events 
COMMENT = 'Structured log events tied to jobs and steps for centralized logging system';

-- Show the enhanced schema summary
SELECT 
    'Enhanced job_tracking table with progress tracking' AS component,
    'job_tracking' AS table_name,
    'Added percent_complete, canceled_at, owner fields' AS description
UNION ALL
SELECT 
    'New job_steps table for step tracking',
    'job_steps',
    'Tracks individual steps within jobs with status and timing'
UNION ALL
SELECT 
    'New log_events table for structured logging',
    'log_events', 
    'Centralized structured logging with job/step correlation'
UNION ALL
SELECT 
    'Performance views for monitoring',
    'job_progress, active_jobs',
    'Materialized views for easy job status and progress monitoring';
