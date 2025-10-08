-- Rollback: Remove setup_progress_percent field
ALTER TABLE replication_jobs DROP COLUMN setup_progress_percent;
