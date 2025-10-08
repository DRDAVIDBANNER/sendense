-- Migration: add_scheduler_tables
-- Created: 20250918144500
-- Purpose: Add replication job scheduler system tables for automated replication scheduling
-- Adheres to RULES_AND_CONSTRAINTS.md: DATABASE SCHEMA SAFETY - validated against existing schema

-- Replication schedules table - defines when and how to run scheduled replications
CREATE TABLE replication_schedules (
    id VARCHAR(64) PRIMARY KEY DEFAULT (UUID()),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    
    -- Schedule configuration
    cron_expression VARCHAR(100) NOT NULL COMMENT 'Cron expression for schedule timing (e.g., "0 2 * * *" for daily 2 AM)',
    schedule_type ENUM('cron', 'chain') NOT NULL DEFAULT 'cron',
    timezone VARCHAR(50) DEFAULT 'UTC' COMMENT 'Timezone for cron expression',
    
    -- Chain scheduling (for dependency chains)
    chain_parent_schedule_id VARCHAR(64) NULL COMMENT 'Parent schedule for chain dependency',
    chain_delay_minutes INT DEFAULT 0 COMMENT 'Minutes to wait after parent completion',
    
    -- Job configuration
    replication_type ENUM('full', 'incremental', 'auto') DEFAULT 'auto' COMMENT 'Type of replication for scheduled jobs',
    max_concurrent_jobs INT DEFAULT 1 COMMENT 'Maximum number of concurrent jobs from this schedule',
    retry_attempts INT DEFAULT 3 COMMENT 'Number of retry attempts for failed jobs',
    retry_delay_minutes INT DEFAULT 30 COMMENT 'Minutes between retry attempts',
    
    -- Control flags
    enabled BOOLEAN DEFAULT true COMMENT 'Whether this schedule is active',
    skip_if_running BOOLEAN DEFAULT true COMMENT 'Skip execution if jobs from this schedule are still running',
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system',
    
    -- Foreign key for chain scheduling
    FOREIGN KEY (chain_parent_schedule_id) REFERENCES replication_schedules(id) ON DELETE SET NULL,
    
    -- Indexes for performance
    INDEX idx_replication_schedules_enabled (enabled),
    INDEX idx_replication_schedules_name (name),
    INDEX idx_replication_schedules_type (schedule_type),
    INDEX idx_replication_schedules_parent (chain_parent_schedule_id)
);

-- Machine groups for organizing VMs by schedule
CREATE TABLE vm_machine_groups (
    id VARCHAR(64) PRIMARY KEY DEFAULT (UUID()),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    schedule_id VARCHAR(64) NULL COMMENT 'Default schedule for this group',
    
    -- Group settings
    max_concurrent_vms INT DEFAULT 5 COMMENT 'Maximum VMs to process concurrently in this group',
    priority INT DEFAULT 0 COMMENT 'Group priority (lower number = higher priority)',
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system',
    
    -- Foreign key
    FOREIGN KEY (schedule_id) REFERENCES replication_schedules(id) ON DELETE SET NULL,
    
    -- Indexes
    INDEX idx_vm_machine_groups_name (name),
    INDEX idx_vm_machine_groups_schedule (schedule_id),
    INDEX idx_vm_machine_groups_priority (priority)
);

-- VM membership in machine groups
CREATE TABLE vm_group_memberships (
    id VARCHAR(64) PRIMARY KEY DEFAULT (UUID()),
    group_id VARCHAR(64) NOT NULL,
    vm_context_id VARCHAR(64) NOT NULL COMMENT 'References vm_replication_contexts.context_id',
    
    -- Per-VM settings
    enabled BOOLEAN DEFAULT true COMMENT 'Whether this VM participates in scheduled replications',
    priority INT DEFAULT 0 COMMENT 'VM priority within group (lower number = higher priority)',
    schedule_override_id VARCHAR(64) NULL COMMENT 'Optional override schedule for this specific VM',
    
    -- Metadata
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    added_by VARCHAR(255) DEFAULT 'system',
    
    -- Foreign keys
    FOREIGN KEY (group_id) REFERENCES vm_machine_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE,
    FOREIGN KEY (schedule_override_id) REFERENCES replication_schedules(id) ON DELETE SET NULL,
    
    -- Ensure each VM is only in each group once
    UNIQUE KEY unique_vm_group (group_id, vm_context_id),
    
    -- Indexes
    INDEX idx_vm_group_memberships_group (group_id),
    INDEX idx_vm_group_memberships_vm (vm_context_id),
    INDEX idx_vm_group_memberships_enabled (enabled),
    INDEX idx_vm_group_memberships_priority (priority)
);

-- Schedule execution tracking
CREATE TABLE schedule_executions (
    id VARCHAR(64) PRIMARY KEY DEFAULT (UUID()),
    schedule_id VARCHAR(64) NOT NULL,
    group_id VARCHAR(64) NULL COMMENT 'Group being processed (null for VM-specific schedules)',
    
    -- Execution timing
    scheduled_at TIMESTAMP NOT NULL COMMENT 'When this execution was scheduled to run',
    started_at TIMESTAMP NULL COMMENT 'When execution actually started',
    completed_at TIMESTAMP NULL COMMENT 'When execution finished (success or failure)',
    
    -- Execution status
    status ENUM('scheduled', 'running', 'completed', 'failed', 'skipped', 'cancelled') DEFAULT 'scheduled',
    
    -- Job statistics
    vms_eligible INT DEFAULT 0 COMMENT 'Number of VMs eligible for replication',
    jobs_created INT DEFAULT 0 COMMENT 'Number of replication jobs created',
    jobs_completed INT DEFAULT 0 COMMENT 'Number of jobs completed successfully',
    jobs_failed INT DEFAULT 0 COMMENT 'Number of jobs that failed',
    jobs_skipped INT DEFAULT 0 COMMENT 'Number of VMs skipped (already running)',
    
    -- Execution details and error tracking
    execution_details JSON NULL COMMENT 'Detailed execution information (job IDs, VM states, etc.)',
    error_message TEXT NULL COMMENT 'Error message if execution failed',
    error_details JSON NULL COMMENT 'Detailed error information',
    
    -- Performance metrics
    execution_duration_seconds INT NULL COMMENT 'Total execution time',
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    triggered_by VARCHAR(255) DEFAULT 'scheduler' COMMENT 'What triggered this execution (scheduler, manual, chain)',
    
    -- Foreign keys
    FOREIGN KEY (schedule_id) REFERENCES replication_schedules(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES vm_machine_groups(id) ON DELETE SET NULL,
    
    -- Indexes for performance and monitoring
    INDEX idx_schedule_executions_schedule (schedule_id),
    INDEX idx_schedule_executions_status (status),
    INDEX idx_schedule_executions_scheduled_at (scheduled_at),
    INDEX idx_schedule_executions_started_at (started_at),
    INDEX idx_schedule_executions_group (group_id)
);

-- Enhance vm_replication_contexts table for scheduler support
ALTER TABLE vm_replication_contexts 
ADD COLUMN auto_added BOOLEAN DEFAULT false COMMENT 'VM was added via discovery without immediate replication job',
ADD COLUMN last_scheduled_job_id VARCHAR(255) NULL COMMENT 'Most recent job created by scheduler',
ADD COLUMN next_scheduled_at TIMESTAMP NULL COMMENT 'When this VM is next scheduled for replication',
ADD COLUMN scheduler_enabled BOOLEAN DEFAULT true COMMENT 'Whether this VM participates in scheduled replications';

-- Add indexes for scheduler queries on vm_replication_contexts
CREATE INDEX idx_vm_contexts_auto_added ON vm_replication_contexts(auto_added);
CREATE INDEX idx_vm_contexts_next_scheduled ON vm_replication_contexts(next_scheduled_at);
CREATE INDEX idx_vm_contexts_scheduler_enabled ON vm_replication_contexts(scheduler_enabled);

-- Enhance replication_jobs table to track scheduler origin
ALTER TABLE replication_jobs
ADD COLUMN schedule_execution_id VARCHAR(64) NULL COMMENT 'Links job to schedule execution that created it',
ADD COLUMN scheduled_by VARCHAR(255) NULL COMMENT 'Which scheduler component created this job',
ADD COLUMN vm_group_id VARCHAR(64) NULL COMMENT 'Machine group this job belongs to';

-- Add foreign key and indexes for replication_jobs scheduler fields
ALTER TABLE replication_jobs
ADD FOREIGN KEY fk_replication_jobs_schedule_execution (schedule_execution_id) REFERENCES schedule_executions(id) ON DELETE SET NULL,
ADD FOREIGN KEY fk_replication_jobs_vm_group (vm_group_id) REFERENCES vm_machine_groups(id) ON DELETE SET NULL;

CREATE INDEX idx_replication_jobs_schedule_execution ON replication_jobs(schedule_execution_id);
CREATE INDEX idx_replication_jobs_scheduled_by ON replication_jobs(scheduled_by);
CREATE INDEX idx_replication_jobs_vm_group ON replication_jobs(vm_group_id);

-- Create views for scheduler monitoring and management

-- View: Active schedules with next execution times
CREATE VIEW active_schedules AS
SELECT 
    s.id,
    s.name,
    s.description,
    s.cron_expression,
    s.schedule_type,
    s.enabled,
    s.max_concurrent_jobs,
    COUNT(DISTINCT g.id) as group_count,
    COUNT(DISTINCT m.vm_context_id) as vm_count,
    -- Calculate next execution time (simplified - actual implementation will use cron parser)
    CASE 
        WHEN s.schedule_type = 'cron' AND s.enabled = true 
        THEN DATE_ADD(NOW(), INTERVAL 1 HOUR) -- Placeholder - real implementation uses cron parser
        ELSE NULL 
    END as next_execution,
    (SELECT MAX(se.started_at) 
     FROM schedule_executions se 
     WHERE se.schedule_id = s.id) as last_execution
FROM replication_schedules s
LEFT JOIN vm_machine_groups g ON s.id = g.schedule_id
LEFT JOIN vm_group_memberships m ON g.id = m.group_id AND m.enabled = true
WHERE s.enabled = true
GROUP BY s.id, s.name, s.description, s.cron_expression, s.schedule_type, s.enabled, s.max_concurrent_jobs;

-- View: VM schedule status for monitoring
CREATE VIEW vm_schedule_status AS
SELECT 
    vrc.context_id,
    vrc.vm_name,
    vrc.current_status as vm_status,
    vrc.next_scheduled_at,
    vrc.scheduler_enabled,
    vmg.name as group_name,
    vmg.priority as group_priority,
    vgm.priority as vm_priority,
    vgm.enabled as membership_enabled,
    rs.name as schedule_name,
    rs.cron_expression,
    rs.enabled as schedule_enabled,
    -- Check if VM has active replication job
    CASE WHEN rj.id IS NOT NULL THEN true ELSE false END as has_active_job,
    rj.id as active_job_id,
    rj.status as job_status,
    rj.progress_percent
FROM vm_replication_contexts vrc
LEFT JOIN vm_group_memberships vgm ON vrc.context_id = vgm.vm_context_id
LEFT JOIN vm_machine_groups vmg ON vgm.group_id = vmg.id
LEFT JOIN replication_schedules rs ON vmg.schedule_id = rs.id OR vgm.schedule_override_id = rs.id
LEFT JOIN replication_jobs rj ON vrc.current_job_id = rj.id AND rj.status IN ('replicating', 'provisioning')
WHERE vrc.scheduler_enabled = true;

-- View: Schedule execution summary for dashboard
CREATE VIEW schedule_execution_summary AS
SELECT 
    se.id,
    se.schedule_id,
    rs.name as schedule_name,
    se.status,
    se.scheduled_at,
    se.started_at,
    se.completed_at,
    se.execution_duration_seconds,
    se.vms_eligible,
    se.jobs_created,
    se.jobs_completed,
    se.jobs_failed,
    se.jobs_skipped,
    CASE 
        WHEN se.jobs_created > 0 
        THEN ROUND((se.jobs_completed / se.jobs_created) * 100, 2)
        ELSE 0 
    END as success_rate_percent,
    se.error_message,
    se.triggered_by
FROM schedule_executions se
JOIN replication_schedules rs ON se.schedule_id = rs.id
ORDER BY se.scheduled_at DESC;
