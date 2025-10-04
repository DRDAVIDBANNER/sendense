-- Volume Management Daemon Database Schema
-- Centralized schema for all volume operations and device mappings

-- Volume operations tracking table
CREATE TABLE IF NOT EXISTS volume_operations (
    id VARCHAR(64) PRIMARY KEY,
    type ENUM('create', 'attach', 'detach', 'delete') NOT NULL,
    status ENUM('pending', 'executing', 'completed', 'failed', 'cancelled') NOT NULL DEFAULT 'pending',
    volume_id VARCHAR(64) NOT NULL,
    vm_id VARCHAR(64) NULL,
    request JSON NOT NULL,
    response JSON NULL,
    error TEXT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    
    INDEX idx_volume_operations_volume_id (volume_id),
    INDEX idx_volume_operations_vm_id (vm_id),
    INDEX idx_volume_operations_status (status),
    INDEX idx_volume_operations_type (type),
    INDEX idx_volume_operations_created_at (created_at)
);

-- Real-time device mappings table
CREATE TABLE IF NOT EXISTS device_mappings (
    id VARCHAR(64) PRIMARY KEY,
    volume_id VARCHAR(64) NOT NULL UNIQUE,
    vm_id VARCHAR(64) NOT NULL,
    device_path VARCHAR(32) NOT NULL,
    cloudstack_state VARCHAR(32) NOT NULL,
    linux_state VARCHAR(32) NOT NULL,
    size BIGINT NOT NULL,
    last_sync TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_volume_id (volume_id),
    UNIQUE KEY unique_device_path (device_path),
    INDEX idx_device_mappings_vm_id (vm_id),
    INDEX idx_device_mappings_device_path (device_path),
    INDEX idx_device_mappings_last_sync (last_sync)
);

-- Operation history for auditing (optional - for future use)
CREATE TABLE IF NOT EXISTS volume_operation_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    operation_id VARCHAR(64) NOT NULL,
    previous_status ENUM('pending', 'executing', 'completed', 'failed', 'cancelled'),
    new_status ENUM('pending', 'executing', 'completed', 'failed', 'cancelled'),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    details JSON NULL,
    
    INDEX idx_operation_history_operation_id (operation_id),
    INDEX idx_operation_history_changed_at (changed_at)
);

-- Service metrics (for monitoring and alerting)
CREATE TABLE IF NOT EXISTS volume_daemon_metrics (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_operations BIGINT NOT NULL DEFAULT 0,
    pending_operations BIGINT NOT NULL DEFAULT 0,
    active_mappings BIGINT NOT NULL DEFAULT 0,
    operations_by_type JSON NULL,
    operations_by_status JSON NULL,
    average_response_time_ms DECIMAL(10,2) NOT NULL DEFAULT 0,
    error_rate_percent DECIMAL(5,2) NOT NULL DEFAULT 0,
    details JSON NULL,
    
    INDEX idx_daemon_metrics_timestamp (timestamp)
);
