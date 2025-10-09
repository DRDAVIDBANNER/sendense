-- Migration: Create Protection Flows Tables
-- Date: 2025-10-09
-- Purpose: Add tables for protection flows engine (Phase 1 - VMware Backups)
--         Enables unified orchestration of scheduled backup and replication operations

-- Protection flows table (flow definitions)
CREATE TABLE IF NOT EXISTS protection_flows (
    id VARCHAR(64) PRIMARY KEY DEFAULT (UUID()),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,

    -- Flow configuration
    flow_type ENUM('backup', 'replication') NOT NULL,
    target_type ENUM('vm', 'group') NOT NULL,
    target_id VARCHAR(64) NOT NULL,

    -- Backup configuration
    repository_id VARCHAR(64),
    policy_id VARCHAR(64),

    -- Replication configuration (Phase 5)
    destination_type ENUM('ossea', 'vmware', 'hyperv') DEFAULT NULL,
    destination_config JSON DEFAULT NULL,

    -- Scheduling
    schedule_id VARCHAR(64),

    -- Control
    enabled BOOLEAN DEFAULT true,

    -- Statistics (denormalized for performance)
    last_execution_id VARCHAR(64),
    last_execution_status ENUM('success', 'warning', 'error', 'running', 'pending') DEFAULT 'pending',
    last_execution_time TIMESTAMP,
    next_execution_time TIMESTAMP,
    total_executions INT DEFAULT 0,
    successful_executions INT DEFAULT 0,
    failed_executions INT DEFAULT 0,

    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system',

    -- Indexes for common queries
    INDEX idx_flow_type (flow_type),
    INDEX idx_target (target_type, target_id),
    INDEX idx_enabled (enabled),
    INDEX idx_schedule (schedule_id),
    INDEX idx_last_execution (last_execution_time),

    -- Constraints
    CHECK (
        (flow_type = 'backup' AND repository_id IS NOT NULL) OR
        (flow_type = 'replication' AND destination_type IS NOT NULL)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Note: Foreign key constraints will be added in a separate migration
-- after verifying all referenced tables exist in production

-- Protection flow executions table (execution history)
CREATE TABLE IF NOT EXISTS protection_flow_executions (
    id VARCHAR(64) PRIMARY KEY DEFAULT (UUID()),
    flow_id VARCHAR(64) NOT NULL,

    -- Execution details
    status ENUM('pending', 'running', 'success', 'warning', 'error', 'cancelled') NOT NULL DEFAULT 'pending',
    execution_type ENUM('scheduled', 'manual', 'api') NOT NULL,

    -- Job tracking
    jobs_created INT DEFAULT 0,
    jobs_completed INT DEFAULT 0,
    jobs_failed INT DEFAULT 0,
    jobs_skipped INT DEFAULT 0,

    -- Timing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    execution_time_seconds INT,

    -- Results
    vms_processed INT DEFAULT 0,
    bytes_transferred BIGINT DEFAULT 0,
    error_message TEXT,
    execution_metadata JSON,

    -- Links to actual work
    created_job_ids JSON,
    schedule_execution_id VARCHAR(64),

    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    triggered_by VARCHAR(255) DEFAULT 'system',

    -- Indexes
    INDEX idx_flow_executions (flow_id, started_at DESC),
    INDEX idx_status (status),
    INDEX idx_execution_time (started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
