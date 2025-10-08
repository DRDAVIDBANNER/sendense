-- Migration: Backup Context Architecture Refactoring
-- Date: 2025-10-08
-- Purpose: Eliminate time-window hack by implementing proper VM-centric context architecture
-- Job Sheet: /home/oma_admin/sendense/job-sheets/2025-10-08-backup-context-architecture.md

START TRANSACTION;

-- ============================================================================
-- STEP 1: Create vm_backup_contexts table (master context for backup VMs)
-- ============================================================================
CREATE TABLE IF NOT EXISTS vm_backup_contexts (
  context_id VARCHAR(64) PRIMARY KEY,
  vm_name VARCHAR(255) NOT NULL,
  vmware_vm_id VARCHAR(255) NOT NULL DEFAULT '',
  vm_path VARCHAR(500) NOT NULL DEFAULT '',
  vcenter_host VARCHAR(255) NOT NULL DEFAULT '',
  datacenter VARCHAR(255) NOT NULL DEFAULT '',
  repository_id VARCHAR(64) NOT NULL,
  
  -- Backup statistics
  total_backups_run INT DEFAULT 0,
  successful_backups INT DEFAULT 0,
  failed_backups INT DEFAULT 0,
  last_backup_id VARCHAR(64) DEFAULT NULL,
  last_backup_type ENUM('full', 'incremental') DEFAULT NULL,
  last_backup_at TIMESTAMP NULL DEFAULT NULL,
  
  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  
  -- Foreign keys
  FOREIGN KEY fk_backup_context_repository (repository_id) 
    REFERENCES backup_repositories(id) ON DELETE RESTRICT,
  
  -- Constraints
  UNIQUE KEY uk_vm_backup (vm_name, repository_id),
  INDEX idx_vm_name (vm_name),
  INDEX idx_last_backup (last_backup_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
COMMENT='Master context for backup VMs - replaces time-window matching';

-- ============================================================================
-- STEP 2: Create backup_disks table (per-disk backup tracking)
-- ============================================================================
CREATE TABLE IF NOT EXISTS backup_disks (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  vm_backup_context_id VARCHAR(64) NOT NULL,
  backup_job_id VARCHAR(64) NOT NULL,
  
  -- Disk identification
  disk_index INT NOT NULL COMMENT 'Sequential disk index: 0, 1, 2...',
  vmware_disk_key INT NOT NULL DEFAULT 0 COMMENT 'VMware disk key: 2000, 2001...',
  size_gb BIGINT NOT NULL DEFAULT 0,
  unit_number INT DEFAULT NULL COMMENT 'VMware unit number',
  
  -- Backup tracking
  disk_change_id VARCHAR(255) DEFAULT NULL COMMENT 'VMware CBT change ID',
  qcow2_path VARCHAR(512) DEFAULT NULL COMMENT 'Path to QCOW2 file',
  bytes_transferred BIGINT DEFAULT 0,
  status ENUM('pending', 'running', 'completed', 'failed') DEFAULT 'pending',
  error_message TEXT DEFAULT NULL,
  
  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP NULL DEFAULT NULL,
  
  -- Foreign keys
  FOREIGN KEY fk_backup_disk_context (vm_backup_context_id) 
    REFERENCES vm_backup_contexts(context_id) ON DELETE CASCADE,
  FOREIGN KEY fk_backup_disk_job (backup_job_id) 
    REFERENCES backup_jobs(id) ON DELETE CASCADE,
  
  -- Constraints
  UNIQUE KEY uk_backup_disk (backup_job_id, disk_index),
  INDEX idx_change_id_lookup (vm_backup_context_id, disk_index, status),
  INDEX idx_completion (backup_job_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
COMMENT='Per-disk backup tracking - stores individual disk change_ids';

-- ============================================================================
-- STEP 3: Add vm_backup_context_id to backup_jobs table
-- ============================================================================
-- Check if column already exists (for idempotency)
SET @column_exists = (
  SELECT COUNT(*) 
  FROM information_schema.COLUMNS 
  WHERE TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'backup_jobs' 
    AND COLUMN_NAME = 'vm_backup_context_id'
);

SET @alter_sql = IF(@column_exists = 0,
  'ALTER TABLE backup_jobs 
   ADD COLUMN vm_backup_context_id VARCHAR(64) DEFAULT NULL AFTER id,
   ADD INDEX idx_backup_context (vm_backup_context_id)',
  'SELECT "Column vm_backup_context_id already exists" as message'
);

PREPARE stmt FROM @alter_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add foreign key constraint (only if column was just added)
SET @fk_exists = (
  SELECT COUNT(*) 
  FROM information_schema.TABLE_CONSTRAINTS 
  WHERE TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'backup_jobs' 
    AND CONSTRAINT_NAME = 'fk_backup_job_context'
);

SET @fk_sql = IF(@fk_exists = 0 AND @column_exists = 0,
  'ALTER TABLE backup_jobs 
   ADD CONSTRAINT fk_backup_job_context 
   FOREIGN KEY (vm_backup_context_id) 
   REFERENCES vm_backup_contexts(context_id) ON DELETE SET NULL',
  'SELECT "FK constraint already exists or column was pre-existing" as message'
);

PREPARE stmt FROM @fk_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- ============================================================================
-- STEP 4: Migrate existing data
-- ============================================================================
-- Create backup contexts for existing VMs
INSERT IGNORE INTO vm_backup_contexts (
  context_id, 
  vm_name, 
  vmware_vm_id, 
  vm_path, 
  vcenter_host, 
  datacenter, 
  repository_id,
  total_backups_run,
  successful_backups,
  last_backup_id,
  last_backup_type,
  last_backup_at,
  created_at
)
SELECT 
  CONCAT('ctx-backup-', bj.vm_name, '-', DATE_FORMAT(MIN(bj.created_at), '%Y%m%d-%H%i%s')) as context_id,
  bj.vm_name,
  COALESCE(vrc.vmware_vm_id, '') as vmware_vm_id,
  COALESCE(vrc.vm_path, '') as vm_path,
  COALESCE(vrc.vcenter_host, '') as vcenter_host,
  COALESCE(vrc.datacenter, '') as datacenter,
  bj.repository_id,
  COUNT(*) as total_backups_run,
  SUM(CASE WHEN bj.status = 'completed' THEN 1 ELSE 0 END) as successful_backups,
  MAX(CASE WHEN bj.status = 'completed' THEN bj.id ELSE NULL END) as last_backup_id,
  MAX(CASE WHEN bj.status = 'completed' THEN bj.backup_type ELSE NULL END) as last_backup_type,
  MAX(bj.completed_at) as last_backup_at,
  MIN(bj.created_at) as created_at
FROM backup_jobs bj
LEFT JOIN vm_replication_contexts vrc ON bj.vm_name = vrc.vm_name
WHERE bj.vm_name IS NOT NULL 
  AND bj.repository_id IS NOT NULL
GROUP BY bj.vm_name, bj.repository_id;

-- Link existing backup_jobs to contexts
UPDATE backup_jobs bj
JOIN vm_backup_contexts vbc 
  ON bj.vm_name = vbc.vm_name 
  AND bj.repository_id = vbc.repository_id
SET bj.vm_backup_context_id = vbc.context_id
WHERE bj.vm_backup_context_id IS NULL;

-- Migrate per-disk backup records to backup_disks table
-- Only migrate records that have disk_id populated (skip parent jobs with disk_id=0 and no QCOW2 path)
INSERT IGNORE INTO backup_disks (
  vm_backup_context_id,
  backup_job_id,
  disk_index,
  vmware_disk_key,
  size_gb,
  unit_number,
  disk_change_id,
  qcow2_path,
  bytes_transferred,
  status,
  created_at,
  completed_at
)
SELECT 
  bj.vm_backup_context_id,
  bj.id as backup_job_id,
  COALESCE(bj.disk_id, 0) as disk_index,
  COALESCE(bj.disk_id, 0) + 2000 as vmware_disk_key,
  0 as size_gb,
  NULL as unit_number,
  bj.change_id as disk_change_id,
  bj.repository_path as qcow2_path,
  COALESCE(bj.bytes_transferred, 0) as bytes_transferred,
  CASE 
    WHEN bj.status = 'completed' THEN 'completed'
    WHEN bj.status = 'failed' THEN 'failed'
    WHEN bj.status = 'running' THEN 'running'
    ELSE 'pending'
  END as status,
  bj.created_at,
  bj.completed_at
FROM backup_jobs bj
WHERE bj.vm_backup_context_id IS NOT NULL
  AND bj.disk_id IS NOT NULL
  AND bj.repository_path IS NOT NULL
  AND bj.repository_path LIKE '%/disk-%'  -- Only migrate actual per-disk jobs
  AND NOT EXISTS (
    SELECT 1 FROM backup_disks bd 
    WHERE bd.backup_job_id = bj.id AND bd.disk_index = bj.disk_id
  );

COMMIT;

-- ============================================================================
-- VERIFICATION QUERIES (for post-migration validation)
-- ============================================================================
-- Run these manually after migration to verify:

-- SELECT COUNT(*) as backup_contexts FROM vm_backup_contexts;
-- SELECT COUNT(*) as backup_disks FROM backup_disks;
-- SELECT COUNT(*) as jobs_with_context FROM backup_jobs WHERE vm_backup_context_id IS NOT NULL;
-- SELECT COUNT(*) as jobs_without_context FROM backup_jobs WHERE vm_backup_context_id IS NULL;

-- Show sample context:
-- SELECT * FROM vm_backup_contexts LIMIT 1\G

-- Show sample disk records:
-- SELECT * FROM backup_disks LIMIT 3\G

-- Show orphaned jobs (should be minimal):
-- SELECT id, vm_name, status, created_at 
-- FROM backup_jobs 
-- WHERE vm_backup_context_id IS NULL 
-- LIMIT 5;


