-- Backup Repository Tables (Fixed collation)
-- Date: 2025-10-04
-- Phase: 1 - VMware Backup Implementation
-- Task: 1.1 - Repository Abstraction

-- Repositories
CREATE TABLE IF NOT EXISTS backup_repositories (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    repository_type ENUM('local', 'nfs', 'cifs', 'smb', 's3', 'azure') NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    config JSON NOT NULL,
    is_immutable BOOLEAN DEFAULT FALSE,
    immutable_config JSON NULL,
    min_retention_days INT DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    used_size_bytes BIGINT DEFAULT 0,
    available_size_bytes BIGINT DEFAULT 0,
    last_check_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_name (name),
    INDEX idx_type (repository_type),
    INDEX idx_enabled (enabled),
    INDEX idx_immutable (is_immutable)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Backup policies
CREATE TABLE IF NOT EXISTS backup_policies (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    primary_repository_id VARCHAR(64) NOT NULL,
    retention_days INT DEFAULT 30,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (primary_repository_id) REFERENCES backup_repositories(id),
    UNIQUE KEY unique_name (name),
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Copy rules
CREATE TABLE IF NOT EXISTS backup_copy_rules (
    id VARCHAR(64) PRIMARY KEY,
    policy_id VARCHAR(64) NOT NULL,
    destination_repository_id VARCHAR(64) NOT NULL,
    copy_mode ENUM('immediate', 'scheduled', 'manual') DEFAULT 'immediate',
    priority INT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    verify_after_copy BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (policy_id) REFERENCES backup_policies(id) ON DELETE CASCADE,
    FOREIGN KEY (destination_repository_id) REFERENCES backup_repositories(id),
    INDEX idx_policy (policy_id),
    INDEX idx_priority (priority)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Backup jobs
CREATE TABLE IF NOT EXISTS backup_jobs (
    id VARCHAR(64) PRIMARY KEY,
    vm_context_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    vm_name VARCHAR(255) NOT NULL,
    repository_id VARCHAR(64) NOT NULL,
    policy_id VARCHAR(64) NULL,
    backup_type ENUM('full', 'incremental', 'differential') NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
    repository_path VARCHAR(512) NOT NULL,
    parent_backup_id VARCHAR(64) NULL,
    change_id VARCHAR(191) NULL,
    bytes_transferred BIGINT DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    compression_enabled BOOLEAN DEFAULT TRUE,
    error_message TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE,
    FOREIGN KEY (repository_id) REFERENCES backup_repositories(id) ON DELETE RESTRICT,
    FOREIGN KEY (policy_id) REFERENCES backup_policies(id) ON DELETE SET NULL,
    FOREIGN KEY (parent_backup_id) REFERENCES backup_jobs(id) ON DELETE SET NULL,
    INDEX idx_vm_context (vm_context_id),
    INDEX idx_repository (repository_id),
    INDEX idx_policy (policy_id),
    INDEX idx_status (status),
    INDEX idx_created (created_at),
    INDEX idx_parent (parent_backup_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Backup copies
CREATE TABLE IF NOT EXISTS backup_copies (
    id VARCHAR(64) PRIMARY KEY,
    source_backup_id VARCHAR(64) NOT NULL,
    repository_id VARCHAR(64) NOT NULL,
    copy_rule_id VARCHAR(64) NULL,
    status ENUM('pending', 'copying', 'verifying', 'completed', 'failed') DEFAULT 'pending',
    file_path VARCHAR(512) NOT NULL,
    size_bytes BIGINT DEFAULT 0,
    copy_started_at TIMESTAMP NULL,
    copy_completed_at TIMESTAMP NULL,
    verified_at TIMESTAMP NULL,
    verification_status ENUM('pending', 'passed', 'failed') DEFAULT 'pending',
    error_message TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (repository_id) REFERENCES backup_repositories(id),
    FOREIGN KEY (copy_rule_id) REFERENCES backup_copy_rules(id) ON DELETE SET NULL,
    INDEX idx_source_backup (source_backup_id),
    INDEX idx_repository (repository_id),
    INDEX idx_status (status),
    UNIQUE KEY unique_backup_repo (source_backup_id, repository_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Backup chains
CREATE TABLE IF NOT EXISTS backup_chains (
    id VARCHAR(64) PRIMARY KEY,
    vm_context_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    disk_id INT NOT NULL,
    full_backup_id VARCHAR(64) NOT NULL,
    latest_backup_id VARCHAR(64) NOT NULL,
    total_backups INT DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE,
    FOREIGN KEY (full_backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (latest_backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE,
    UNIQUE KEY unique_vm_disk (vm_context_id, disk_id),
    INDEX idx_vm_context (vm_context_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
