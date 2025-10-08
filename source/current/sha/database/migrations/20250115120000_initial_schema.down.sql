-- Migration: initial_schema
-- Created: 20250115120000
-- Rollback initial database schema for OMA OSSEA integration (MariaDB)

-- Drop indexes first
DROP INDEX IF EXISTS idx_cbt_history_job_disk ON cbt_history;
DROP INDEX IF EXISTS idx_volume_mounts_job_id ON volume_mounts;
DROP INDEX IF EXISTS idx_ossea_volumes_volume_id ON ossea_volumes;
DROP INDEX IF EXISTS idx_vm_disks_job_id ON vm_disks;
DROP INDEX IF EXISTS idx_replication_jobs_created_at ON replication_jobs;
DROP INDEX IF EXISTS idx_replication_jobs_vcenter ON replication_jobs;
DROP INDEX IF EXISTS idx_replication_jobs_status ON replication_jobs;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS cbt_history;
DROP TABLE IF EXISTS volume_mounts;
DROP TABLE IF EXISTS vm_disks;
DROP TABLE IF EXISTS replication_jobs;
DROP TABLE IF EXISTS ossea_volumes;
DROP TABLE IF EXISTS ossea_configs;