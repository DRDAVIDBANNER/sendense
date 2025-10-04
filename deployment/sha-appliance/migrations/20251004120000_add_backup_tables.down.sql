-- Rollback: Remove Backup Repository Tables
-- Date: 2025-10-04
-- Purpose: Remove backup repository system tables

-- Drop tables in reverse order (respect foreign keys)
DROP TABLE IF EXISTS backup_chains;
DROP TABLE IF EXISTS backup_copies;
DROP TABLE IF EXISTS backup_jobs;
DROP TABLE IF EXISTS backup_copy_rules;
DROP TABLE IF EXISTS backup_policies;
DROP TABLE IF EXISTS backup_repositories;
