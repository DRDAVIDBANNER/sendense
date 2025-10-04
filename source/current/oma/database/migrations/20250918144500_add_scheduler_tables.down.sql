-- Migration: add_scheduler_tables (DOWN)
-- Created: 20250918144500
-- Purpose: Rollback scheduler system tables

-- Drop views first (they depend on tables)
DROP VIEW IF EXISTS schedule_execution_summary;
DROP VIEW IF EXISTS vm_schedule_status;
DROP VIEW IF EXISTS active_schedules;

-- Remove foreign key constraints before dropping columns
ALTER TABLE replication_jobs DROP FOREIGN KEY IF EXISTS fk_replication_jobs_schedule_execution;
ALTER TABLE replication_jobs DROP FOREIGN KEY IF EXISTS fk_replication_jobs_vm_group;

-- Drop indexes on replication_jobs scheduler fields
DROP INDEX IF EXISTS idx_replication_jobs_schedule_execution ON replication_jobs;
DROP INDEX IF EXISTS idx_replication_jobs_scheduled_by ON replication_jobs;
DROP INDEX IF EXISTS idx_replication_jobs_vm_group ON replication_jobs;

-- Remove scheduler columns from replication_jobs
ALTER TABLE replication_jobs 
DROP COLUMN IF EXISTS schedule_execution_id,
DROP COLUMN IF EXISTS scheduled_by,
DROP COLUMN IF EXISTS vm_group_id;

-- Drop indexes on vm_replication_contexts scheduler fields
DROP INDEX IF EXISTS idx_vm_contexts_auto_added ON vm_replication_contexts;
DROP INDEX IF EXISTS idx_vm_contexts_next_scheduled ON vm_replication_contexts;
DROP INDEX IF EXISTS idx_vm_contexts_scheduler_enabled ON vm_replication_contexts;

-- Remove scheduler columns from vm_replication_contexts
ALTER TABLE vm_replication_contexts
DROP COLUMN IF EXISTS auto_added,
DROP COLUMN IF EXISTS last_scheduled_job_id,
DROP COLUMN IF EXISTS next_scheduled_at,
DROP COLUMN IF EXISTS scheduler_enabled;

-- Drop scheduler tables (in reverse dependency order)
DROP TABLE IF EXISTS schedule_executions;
DROP TABLE IF EXISTS vm_group_memberships;
DROP TABLE IF EXISTS vm_machine_groups;
DROP TABLE IF EXISTS replication_schedules;
