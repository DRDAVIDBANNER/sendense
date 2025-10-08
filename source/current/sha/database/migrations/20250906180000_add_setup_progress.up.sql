-- Add setup_progress_percent field to separate OMA setup progress from VMA replication progress
-- This prevents mixing of setup phases (0-85%) with actual replication progress (0-100%)

ALTER TABLE replication_jobs ADD COLUMN setup_progress_percent DECIMAL(5,2) DEFAULT 0.0 COMMENT 'OMA setup progress (0-85%): job creation, volume provisioning, NBD setup';

-- Update existing jobs to have setup_progress_percent = 85.0 if they are in replicating status
-- This handles jobs that already completed setup
UPDATE replication_jobs 
SET setup_progress_percent = 85.0 
WHERE status IN ('replicating', 'completed', 'failed') 
  AND setup_progress_percent = 0.0;

-- Reset progress_percent to 0 for replicating jobs so VMA can start fresh
UPDATE replication_jobs 
SET progress_percent = 0.0 
WHERE status = 'replicating' 
  AND progress_percent > 0.0;
