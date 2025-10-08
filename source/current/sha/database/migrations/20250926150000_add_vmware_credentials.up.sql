-- Add VMware credentials management table for centralized credential storage
-- This eliminates hardcoded credentials throughout the codebase

CREATE TABLE vmware_credentials (
    id INT PRIMARY KEY AUTO_INCREMENT,
    credential_name VARCHAR(255) NOT NULL UNIQUE 
        COMMENT 'Human-readable name (e.g., Production-vCenter, Dev-vCenter)',
    vcenter_host VARCHAR(255) NOT NULL 
        COMMENT 'vCenter hostname or IP address',
    username VARCHAR(255) NOT NULL 
        COMMENT 'vCenter username (e.g., administrator@vsphere.local)',
    password_encrypted TEXT NOT NULL 
        COMMENT 'AES-256 encrypted password',
    datacenter VARCHAR(255) NOT NULL 
        COMMENT 'Default datacenter name for this vCenter',
    is_active BOOLEAN DEFAULT TRUE 
        COMMENT 'Enable/disable this credential set',
    is_default BOOLEAN DEFAULT FALSE 
        COMMENT 'Default credential set for operations',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NULL 
        COMMENT 'User who created this credential set',
    last_used TIMESTAMP NULL 
        COMMENT 'Last time these credentials were used in operations',
    usage_count INT DEFAULT 0 
        COMMENT 'Number of times credentials have been used',
    
    INDEX idx_vmware_creds_active (is_active),
    INDEX idx_vmware_creds_default (is_default),
    INDEX idx_vmware_creds_host (vcenter_host),
    INDEX idx_vmware_creds_last_used (last_used)
) COMMENT 'Centralized VMware vCenter credential management';

-- Add foreign key to vm_replication_contexts for credential association
ALTER TABLE vm_replication_contexts 
ADD COLUMN vmware_credential_id INT NULL 
    COMMENT 'FK to vmware_credentials - which credential set was used for this VM',
ADD FOREIGN KEY fk_vm_context_vmware_creds (vmware_credential_id) 
    REFERENCES vmware_credentials(id) ON DELETE SET NULL;

-- Insert default credential set from current hardcoded values
-- Note: Password will be encrypted when encryption service is implemented
INSERT INTO vmware_credentials (
    credential_name, 
    vcenter_host, 
    username, 
    password_encrypted, 
    datacenter, 
    is_active, 
    is_default,
    created_by
) VALUES (
    'Production-vCenter',
    'quad-vcenter-01.quadris.local',
    'administrator@vsphere.local',
    'TEMP_PLAINTEXT_EmyGVoBFesGQc47-', -- Will be encrypted in Phase 2
    'DatabanxDC',
    TRUE,
    TRUE,
    'system_migration'
);

-- Verify migration success
SELECT 
    id, 
    credential_name, 
    vcenter_host, 
    username, 
    SUBSTRING(password_encrypted, 1, 10) as password_preview,
    datacenter,
    is_default
FROM vmware_credentials;

-- Verify vm_replication_contexts FK
SELECT COUNT(*) as total_contexts,
       SUM(CASE WHEN vmware_credential_id IS NULL THEN 1 ELSE 0 END) as null_credential_refs
FROM vm_replication_contexts;
-- Expected: All contexts have NULL credential_id (will be populated when service is deployed)

