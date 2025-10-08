-- Database schema for OMA OSSEA integration (MariaDB)
-- Tracks job-to-volume mappings, CBT ChangeIDs, and OSSEA connection details

-- OSSEA connection configuration
CREATE TABLE ossea_configs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    api_url VARCHAR(255) NOT NULL,
    api_key VARCHAR(512) NOT NULL,
    secret_key VARCHAR(512) NOT NULL,
    domain VARCHAR(255),
    zone VARCHAR(255) NOT NULL,
    
    -- Additional OSSEA-specific settings
    template_id VARCHAR(255),
    network_id VARCHAR(255),
    service_offering_id VARCHAR(255),
    disk_offering_id VARCHAR(255),
    
    -- OMA VM identification in OSSEA
    oma_vm_id VARCHAR(255), -- The VM ID of this OMA appliance in OSSEA
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

-- OSSEA volumes tracking
CREATE TABLE ossea_volumes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    volume_id VARCHAR(255) NOT NULL UNIQUE, -- OSSEA volume UUID
    volume_name VARCHAR(255) NOT NULL,
    size_gb INT NOT NULL,
    ossea_config_id INT,
    
    -- Volume metadata
    volume_type VARCHAR(100), -- ROOT, DATADISK, etc.
    device_path VARCHAR(255), -- Mount path on OMA appliance
    mount_point VARCHAR(255), -- Where it's mounted locally
    status VARCHAR(50) DEFAULT 'creating', -- creating, available, attached, detached, error
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Foreign key
    FOREIGN KEY (ossea_config_id) REFERENCES ossea_configs(id)
);

-- Enhanced replication jobs with OSSEA integration
CREATE TABLE replication_jobs (
    id VARCHAR(255) PRIMARY KEY, -- Job ID from API
    
    -- Source VM information
    source_vm_id VARCHAR(255) NOT NULL,
    source_vm_name VARCHAR(255) NOT NULL,
    source_vm_path VARCHAR(255) NOT NULL,
    vcenter_host VARCHAR(255) NOT NULL,
    datacenter VARCHAR(255) NOT NULL,
    
    -- Job configuration
    replication_type VARCHAR(50) NOT NULL, -- initial, incremental
    target_network VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    
    -- Progress tracking
    progress_percent DECIMAL(5,2) DEFAULT 0.0,
    current_operation VARCHAR(255),
    bytes_transferred BIGINT DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    transfer_speed_bps BIGINT DEFAULT 0,
    error_message TEXT,
    
    -- CBT and incremental sync
    change_id VARCHAR(255), -- VMware CBT ChangeID
    previous_change_id VARCHAR(255), -- For incremental sync
    snapshot_id VARCHAR(255), -- VMware snapshot reference
    
    -- Dynamic allocation
    nbd_port INT,
    nbd_export_name VARCHAR(255),
    target_device VARCHAR(255),
    
    -- OSSEA configuration
    ossea_config_id INT,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    
    -- Foreign key
    FOREIGN KEY (ossea_config_id) REFERENCES ossea_configs(id)
);

-- VM disk information
CREATE TABLE vm_disks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    
    -- Source disk info (from VMware)
    disk_id VARCHAR(255) NOT NULL,
    vmdk_path VARCHAR(255) NOT NULL,
    size_gb INT NOT NULL,
    datastore VARCHAR(255),
    unit_number INT,
    label VARCHAR(255),
    capacity_bytes BIGINT,
    provisioning_type VARCHAR(50),
    
    -- Target OSSEA volume mapping
    ossea_volume_id INT,
    
    -- Sync tracking per disk
    disk_change_id VARCHAR(255), -- CBT ChangeID for this specific disk
    sync_status VARCHAR(50) DEFAULT 'pending', -- pending, syncing, completed, failed
    sync_progress_percent DECIMAL(5,2) DEFAULT 0.0,
    bytes_synced BIGINT DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (job_id) REFERENCES replication_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (ossea_volume_id) REFERENCES ossea_volumes(id)
);

-- Volume mount tracking on OMA appliance
CREATE TABLE volume_mounts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    ossea_volume_id INT NOT NULL,
    job_id VARCHAR(255) NOT NULL,
    
    -- Mount details
    device_path VARCHAR(255) NOT NULL, -- e.g., /dev/vdb, /dev/vdc
    mount_point VARCHAR(255), -- e.g., /mnt/migration/job-123-disk-0
    mount_status VARCHAR(50) DEFAULT 'unmounted', -- unmounted, mounting, mounted, unmount_pending, error
    
    -- Mount options and metadata
    filesystem_type VARCHAR(50), -- ext4, xfs, ntfs, etc.
    mount_options VARCHAR(255), -- rw,noatime,etc.
    is_read_only BOOLEAN DEFAULT false,
    
    -- Tracking
    mounted_at TIMESTAMP NULL,
    unmounted_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Foreign keys
    FOREIGN KEY (ossea_volume_id) REFERENCES ossea_volumes(id) ON DELETE CASCADE,
    FOREIGN KEY (job_id) REFERENCES replication_jobs(id) ON DELETE CASCADE
);

-- CBT ChangeID history for incremental sync
CREATE TABLE cbt_history (
    id INT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    disk_id VARCHAR(255) NOT NULL,
    
    -- CBT tracking
    change_id VARCHAR(255) NOT NULL,
    previous_change_id VARCHAR(255),
    sync_type VARCHAR(50) NOT NULL, -- full, incremental
    
    -- Sync results
    blocks_changed INT,
    bytes_transferred BIGINT,
    sync_duration_seconds INT,
    sync_success BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Composite index for efficient lookups
    UNIQUE KEY unique_job_disk_change (job_id, disk_id, change_id)
);

-- Indexes for performance
CREATE INDEX idx_replication_jobs_status ON replication_jobs(status);
CREATE INDEX idx_replication_jobs_vcenter ON replication_jobs(vcenter_host, datacenter);
CREATE INDEX idx_replication_jobs_created_at ON replication_jobs(created_at);
CREATE INDEX idx_vm_disks_job_id ON vm_disks(job_id);
CREATE INDEX idx_ossea_volumes_volume_id ON ossea_volumes(volume_id);
CREATE INDEX idx_volume_mounts_job_id ON volume_mounts(job_id);
CREATE INDEX idx_cbt_history_job_disk ON cbt_history(job_id, disk_id);

-- Note: MariaDB automatically handles updated_at with ON UPDATE CURRENT_TIMESTAMP
-- No triggers needed for timestamp updates in MariaDB