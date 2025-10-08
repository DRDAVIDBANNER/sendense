-- Migration: add_job_tracking
-- Created: 20250121130000
-- CloudStack async job tracking infrastructure with correlation IDs and parent/child hierarchy

-- CloudStack job tracking table with parent/child hierarchy
CREATE TABLE cloudstack_job_tracking (
    id VARCHAR(64) PRIMARY KEY,
    
    -- CloudStack async job details
    cloudstack_job_id VARCHAR(255) NULL, -- CloudStack async job ID (set when received)
    cloudstack_command VARCHAR(255) NOT NULL, -- CloudStack API command (deployVirtualMachine, attachVolume, etc.)
    cloudstack_status VARCHAR(50) DEFAULT 'pending', -- pending, in-progress, success, failure
    cloudstack_result_code INT NULL, -- CloudStack result code
    cloudstack_response JSON NULL, -- Full CloudStack response
    
    -- Operation correlation
    operation_type VARCHAR(100) NOT NULL, -- test-failover-cleanup, volume-attach, vm-delete, etc.
    correlation_id VARCHAR(64) NOT NULL, -- Links related operations together
    parent_job_id VARCHAR(64) NULL, -- For hierarchical operations
    
    -- Request/Response tracking
    request_data JSON NOT NULL, -- Original request parameters
    local_operation_id VARCHAR(64) NULL, -- Volume Daemon operation ID if applicable
    
    -- Execution tracking
    status VARCHAR(50) DEFAULT 'initiated', -- initiated, submitted, polling, completed, failed, cancelled
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    next_poll_at TIMESTAMP NULL, -- When to next poll CloudStack
    
    -- Error handling
    error_message TEXT NULL,
    error_details JSON NULL,
    
    -- Audit trail
    initiated_by VARCHAR(255) NOT NULL, -- Service/user that initiated the operation
    initiated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    submitted_at TIMESTAMP NULL, -- When submitted to CloudStack
    completed_at TIMESTAMP NULL,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Foreign key for parent/child relationships
    FOREIGN KEY (parent_job_id) REFERENCES cloudstack_job_tracking(id) ON DELETE CASCADE,
    
    -- Indexes for performance
    INDEX idx_job_tracking_cloudstack_job_id (cloudstack_job_id),
    INDEX idx_job_tracking_correlation_id (correlation_id),
    INDEX idx_job_tracking_parent_job_id (parent_job_id),
    INDEX idx_job_tracking_status (status),
    INDEX idx_job_tracking_operation_type (operation_type),
    INDEX idx_job_tracking_next_poll_at (next_poll_at),
    INDEX idx_job_tracking_created_at (created_at)
);

-- CloudStack job execution log for detailed audit trail
CREATE TABLE cloudstack_job_execution_log (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_tracking_id VARCHAR(64) NOT NULL,
    
    -- Log entry details
    log_level VARCHAR(20) NOT NULL, -- INFO, WARN, ERROR, DEBUG
    message TEXT NOT NULL,
    details JSON NULL,
    
    -- Context
    operation_phase VARCHAR(100) NULL, -- initiation, submission, polling, completion, error-handling
    cloudstack_job_status VARCHAR(50) NULL, -- CloudStack job status at time of log
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key
    FOREIGN KEY (job_tracking_id) REFERENCES cloudstack_job_tracking(id) ON DELETE CASCADE,
    
    -- Indexes
    INDEX idx_job_log_job_tracking_id (job_tracking_id),
    INDEX idx_job_log_created_at (created_at),
    INDEX idx_job_log_level (log_level),
    INDEX idx_job_log_operation_phase (operation_phase)
);

-- Job polling queue for active CloudStack jobs
CREATE TABLE cloudstack_job_poll_queue (
    id VARCHAR(64) PRIMARY KEY,
    job_tracking_id VARCHAR(64) NOT NULL UNIQUE,
    
    -- Polling configuration
    poll_interval_seconds INT NOT NULL DEFAULT 2, -- Default 2-second polling
    next_poll_at TIMESTAMP NOT NULL,
    consecutive_failures INT DEFAULT 0,
    max_consecutive_failures INT DEFAULT 5,
    
    -- Queue management
    is_active BOOLEAN DEFAULT true,
    priority INT DEFAULT 0, -- For future prioritization
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Foreign key
    FOREIGN KEY (job_tracking_id) REFERENCES cloudstack_job_tracking(id) ON DELETE CASCADE,
    
    -- Indexes
    INDEX idx_poll_queue_next_poll_at (next_poll_at),
    INDEX idx_poll_queue_is_active (is_active),
    INDEX idx_poll_queue_priority (priority)
);

-- Operational metrics for job tracking performance
CREATE TABLE cloudstack_job_metrics (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    
    -- Time window
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    window_start TIMESTAMP NOT NULL,
    window_end TIMESTAMP NOT NULL,
    
    -- Job statistics
    total_jobs_initiated INT DEFAULT 0,
    total_jobs_completed INT DEFAULT 0,
    total_jobs_failed INT DEFAULT 0,
    total_jobs_cancelled INT DEFAULT 0,
    
    -- Performance metrics
    average_completion_time_seconds DECIMAL(10,2) DEFAULT 0,
    average_poll_cycles DECIMAL(5,1) DEFAULT 0,
    longest_completion_time_seconds INT DEFAULT 0,
    
    -- Operation type breakdown
    operations_by_type JSON NULL, -- {"vm-delete": 5, "volume-attach": 12, ...}
    
    -- Error analysis
    common_errors JSON NULL, -- Top error patterns and counts
    
    INDEX idx_job_metrics_timestamp (timestamp),
    INDEX idx_job_metrics_window (window_start, window_end)
);

-- View for hierarchical job tracking (parent/child relationships)
CREATE VIEW job_tracking_hierarchy AS
SELECT 
    j.id,
    j.operation_type,
    j.correlation_id,
    j.status,
    j.cloudstack_status,
    j.parent_job_id,
    j.initiated_at,
    j.completed_at,
    CASE 
        WHEN j.parent_job_id IS NULL THEN 'root'
        ELSE 'child'
    END as job_level,
    (SELECT COUNT(*) FROM cloudstack_job_tracking WHERE parent_job_id = j.id) as child_count
FROM cloudstack_job_tracking j;

-- View for active polling queue
CREATE VIEW active_polling_queue AS
SELECT 
    q.id,
    q.job_tracking_id,
    q.next_poll_at,
    q.poll_interval_seconds,
    q.consecutive_failures,
    j.cloudstack_job_id,
    j.operation_type,
    j.status as tracking_status,
    j.cloudstack_status
FROM cloudstack_job_poll_queue q
JOIN cloudstack_job_tracking j ON q.job_tracking_id = j.id
WHERE q.is_active = true 
  AND q.next_poll_at <= NOW()
ORDER BY q.next_poll_at ASC, q.priority DESC;



