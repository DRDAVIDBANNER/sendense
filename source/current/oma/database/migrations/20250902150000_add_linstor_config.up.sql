-- Add Linstor configuration table
-- Following OSSEA configuration pattern for consistency

CREATE TABLE linstor_configs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    
    -- Linstor API configuration
    api_url VARCHAR(255) NOT NULL,
    api_port INT DEFAULT 3370,
    api_protocol VARCHAR(10) DEFAULT 'http',
    
    -- Optional authentication (if Linstor API requires it)
    api_key VARCHAR(512),
    api_secret VARCHAR(512),
    
    -- Optional connection settings
    connection_timeout_seconds INT DEFAULT 30,
    retry_attempts INT DEFAULT 3,
    
    -- Metadata
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    
    -- Indexes for performance
    INDEX idx_linstor_configs_active (is_active),
    INDEX idx_linstor_configs_name (name)
);

-- Add linstor_snapshot_name to failover_jobs table
ALTER TABLE failover_jobs 
ADD COLUMN linstor_snapshot_name VARCHAR(255) NULL AFTER ossea_snapshot_id,
ADD COLUMN linstor_config_id INT NULL AFTER linstor_snapshot_name,
ADD INDEX idx_failover_jobs_linstor_snapshot (linstor_snapshot_name);

-- Add foreign key constraint (optional, for data integrity)
-- ALTER TABLE failover_jobs 
-- ADD CONSTRAINT fk_failover_linstor_config 
-- FOREIGN KEY (linstor_config_id) REFERENCES linstor_configs(id) ON DELETE SET NULL;

-- Insert default Linstor configuration
INSERT INTO linstor_configs (
    name, api_url, api_port, description, is_active
) VALUES (
    'default-linstor', 
    'http://10.245.241.101', 
    3370, 
    'Default Linstor configuration for volume snapshots',
    true
);

