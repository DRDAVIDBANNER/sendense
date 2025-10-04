# Failover Database Schema Documentation

**Version**: 1.0  
**Date**: 2025-08-18  
**Database**: MariaDB (migratekit_oma)

## 📋 **Overview**

The failover database schema extends the existing migration database with tables and fields specifically designed for VM failover operations. The schema supports both live and test failover scenarios with comprehensive tracking and validation capabilities.

## 🗂️ **Failover-Specific Tables**

### **1. failover_jobs**
Central table for tracking all failover operations.

```sql
CREATE TABLE failover_jobs (
    id               INT PRIMARY KEY AUTO_INCREMENT,
    job_id           VARCHAR(255) UNIQUE NOT NULL,     -- Unique job identifier
    vm_id            VARCHAR(255) NOT NULL,            -- VMware VM ID
    replication_job_id VARCHAR(255),                   -- FK to replication_jobs.id
    job_type         VARCHAR(50) NOT NULL,             -- 'live' or 'test'
    status           VARCHAR(50) DEFAULT 'pending',    -- Current job status
    source_vm_name   VARCHAR(255) NOT NULL,            -- Original VM name
    source_vm_spec   TEXT,                             -- JSON VM specifications
    destination_vm_id VARCHAR(255),                    -- Created OSSEA VM ID
    ossea_snapshot_id VARCHAR(255),                    -- Snapshot ID for rollback
    network_mappings TEXT,                             -- JSON network mappings
    error_message    TEXT,                             -- Error details if failed
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    started_at       TIMESTAMP NULL,                   -- When execution started
    completed_at     TIMESTAMP NULL,                   -- When execution finished
    
    -- Indexes for performance
    INDEX idx_failover_jobs_vm_id (vm_id),
    INDEX idx_failover_jobs_replication (replication_job_id),
    INDEX idx_failover_jobs_status (status),
    INDEX idx_failover_jobs_type (job_type),
    INDEX idx_failover_jobs_created (created_at)
);
```

**Field Details:**

- **job_id**: Format `{type}-failover-{vm_id}-{timestamp}`
  - Example: `test-failover-4205a841-0265-f4bd-39a6-39fd92196f53-1755530108`
  
- **status**: Tracks job progression through failover pipeline
  - `pending` - Job created, queued for execution
  - `validating` - Running pre-failover validation checks
  - `snapshotting` - Creating backup snapshot for rollback
  - `creating_vm` - Creating OSSEA VM instance
  - `switching_volume` - Detaching/attaching volumes
  - `powering_on` - Starting the OSSEA VM
  - `completed` - Failover successful
  - `failed` - Failover failed at some stage
  - `cleanup` - Cleaning up test resources
  - `reverting` - Reverting test failover changes

- **source_vm_spec**: JSON-encoded VM specifications for recreation
  ```json
  {
    "name": "PGWINTESTBIOS",
    "display_name": "PGWINTESTBIOS (Test)",
    "cpus": 2,
    "memory_mb": 4096,
    "os_type": "windows2019srv_64",
    "power_state": "poweredOn",
    "networks": [
      {
        "network_name": "VM Network",
        "adapter_type": "vmxnet3",
        "connected": true
      }
    ],
    "disks": [
      {
        "disk_id": "e915ef05-ddf5-48d5-8352-a01300609717",
        "size_gb": 54,
        "provisioning_type": "thin"
      }
    ]
  }
  ```

- **network_mappings**: JSON-encoded network mapping configuration
  ```json
  {
    "VM Network": "test-network-default",
    "Legacy Network": "production-network-123"
  }
  ```

**Example Records:**
```sql
-- Test Failover Job
INSERT INTO failover_jobs (
    job_id, vm_id, replication_job_id, job_type, status, 
    source_vm_name, source_vm_spec, network_mappings
) VALUES (
    'test-failover-4205a841-0265-f4bd-39a6-39fd92196f53-1755530108',
    '4205a841-0265-f4bd-39a6-39fd92196f53',
    'job-20250818-153521',
    'test',
    'pending',
    'PGWINTESTBIOS',
    '{"cpus": 2, "memory_mb": 4096, "os_type": "windows2019srv_64"}',
    '{"VM Network": "test-network-default"}'
);

-- Live Failover Job  
INSERT INTO failover_jobs (
    job_id, vm_id, replication_job_id, job_type, status,
    source_vm_name, destination_vm_id, ossea_snapshot_id
) VALUES (
    'live-failover-1234-5678-abcd-1755530200',
    '1234-5678-abcd-efgh',
    'job-20250818-160000',
    'live', 
    'completed',
    'ProductionVM',
    'ossea-vm-prod-12345',
    'snapshot-backup-67890'
);
```

---

### **2. network_mappings**
Defines source to destination network mappings for each VM.

```sql
CREATE TABLE network_mappings (
    id                      INT PRIMARY KEY AUTO_INCREMENT,
    vm_id                   VARCHAR(255) NOT NULL,            -- VMware VM ID
    source_network_name     VARCHAR(255) NOT NULL,            -- Source network name from VMware
    destination_network_id  VARCHAR(255) NOT NULL,            -- OSSEA network ID
    destination_network_name VARCHAR(255) NOT NULL,           -- OSSEA network name (display)
    is_test_network         BOOLEAN DEFAULT FALSE,            -- Test Layer 2 network flag
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Constraints and indexes
    UNIQUE INDEX idx_network_mappings_unique (vm_id, source_network_name),
    INDEX idx_network_mappings_vm_id (vm_id),
    INDEX idx_network_mappings_destination (destination_network_id),
    INDEX idx_network_mappings_test (is_test_network)
);
```

**Field Details:**

- **is_test_network**: Differentiates between production and test network mappings
  - `true` - Test Layer 2 network for isolated test failovers
  - `false` - Production network for live failovers

**Example Records:**
```sql
-- Production Network Mapping
INSERT INTO network_mappings (
    vm_id, source_network_name, destination_network_id, 
    destination_network_name, is_test_network
) VALUES (
    '4205a841-0265-f4bd-39a6-39fd92196f53',
    'VM Network',
    'network-production-123',
    'Production Network',
    FALSE
);

-- Test Network Mapping  
INSERT INTO network_mappings (
    vm_id, source_network_name, destination_network_id,
    destination_network_name, is_test_network
) VALUES (
    '4205a841-0265-f4bd-39a6-39fd92196f53',
    'VM Network', 
    'test-network-default',
    'Test Network',
    TRUE
);
```

---

## 🔗 **Extended Existing Tables**

### **3. vm_disks (Extended)**
The existing `vm_disks` table has been enhanced with VM specification fields.

```sql
-- Additional columns added to existing vm_disks table
ALTER TABLE vm_disks ADD COLUMN cpu_count INT DEFAULT 0;
ALTER TABLE vm_disks ADD COLUMN memory_mb INT DEFAULT 0;
ALTER TABLE vm_disks ADD COLUMN os_type VARCHAR(255) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN vm_tools_version VARCHAR(255) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN network_config TEXT;
ALTER TABLE vm_disks ADD COLUMN display_name VARCHAR(255) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN annotation TEXT;
ALTER TABLE vm_disks ADD COLUMN power_state VARCHAR(50) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN vmware_uuid VARCHAR(255) DEFAULT '';
ALTER TABLE vm_disks ADD COLUMN bios_setup TEXT;
```

**Extended Schema:**
```sql
-- Complete vm_disks table with failover extensions
CREATE TABLE vm_disks (
    id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
    job_id                LONGTEXT NOT NULL,                 -- FK to replication_jobs.id
    disk_id               LONGTEXT NOT NULL,                 -- VMware disk identifier
    vm_dk_path            LONGTEXT NOT NULL,                 -- VMDK file path
    size_gb               BIGINT NOT NULL,                   -- Disk size in GB
    datastore             LONGTEXT,                          -- VMware datastore
    unit_number           BIGINT,                            -- SCSI unit number
    label                 LONGTEXT,                          -- Disk label
    capacity_bytes        BIGINT,                            -- Capacity in bytes
    provisioning_type     LONGTEXT,                          -- thin, thick, etc.
    ossea_volume_id       BIGINT,                            -- FK to ossea_volumes.id
    disk_change_id        LONGTEXT,                          -- CBT change ID
    sync_status           VARCHAR(191) DEFAULT 'pending',   -- Sync status
    sync_progress_percent DOUBLE DEFAULT 0,                 -- Sync completion %
    bytes_synced          BIGINT DEFAULT 0,                 -- Bytes synchronized
    created_at            DATETIME(3),
    updated_at            DATETIME(3),
    
    -- VM Specification fields (populated for first disk of VM)
    cpu_count             INT DEFAULT 0,                     -- vCPU count
    memory_mb             INT DEFAULT 0,                     -- Memory in MB
    os_type               VARCHAR(255) DEFAULT '',           -- Guest OS type
    vm_tools_version      VARCHAR(255) DEFAULT '',           -- VMware Tools version
    network_config        TEXT,                              -- JSON network configuration
    display_name          VARCHAR(255) DEFAULT '',           -- VM display name
    annotation            TEXT,                              -- VM notes/description
    power_state           VARCHAR(50) DEFAULT '',            -- poweredOn, poweredOff, suspended
    vmware_uuid           VARCHAR(255) DEFAULT '',           -- VMware VM UUID
    bios_setup            TEXT,                              -- BIOS configuration (JSON)
    
    -- Indexes
    INDEX idx_vm_disks_job_id (job_id(255)),
    INDEX idx_vm_disks_ossea_volume (ossea_volume_id),
    INDEX idx_vm_disks_sync_status (sync_status)
);
```

**VM Specification Storage Strategy:**
- VM specifications are stored in the **first disk record** of each VM
- Subsequent disk records for the same VM have these fields empty/default
- This approach avoids creating a separate `vm_specifications` table
- Failover operations query for the first disk to get VM specs

**Example Extended Record:**
```sql
-- First disk record with VM specifications
INSERT INTO vm_disks (
    job_id, disk_id, vm_dk_path, size_gb, ossea_volume_id,
    cpu_count, memory_mb, os_type, display_name, power_state,
    network_config
) VALUES (
    'job-20250818-153521',
    'disk-2000',
    '[DatabanxDatastore] PGWINTESTBIOS/PGWINTESTBIOS.vmdk',
    54,
    79,
    2,                    -- 2 vCPUs
    4096,                 -- 4GB RAM
    'windows2019srv_64',  -- Windows Server 2019
    'PGWINTESTBIOS',
    'poweredOn',
    '{"adapters": [{"network": "VM Network", "type": "vmxnet3"}]}'
);
```

---

## 🔍 **Related Tables**

### **4. ossea_volumes** 
Tracks CloudStack volumes created during migration.

```sql
CREATE TABLE ossea_volumes (
    id                INT PRIMARY KEY AUTO_INCREMENT,
    volume_id         VARCHAR(255) UNIQUE NOT NULL,      -- CloudStack volume UUID
    volume_name       VARCHAR(255) NOT NULL,             -- Volume name
    size_gb           INT NOT NULL,                      -- Volume size
    ossea_config_id   INT,                               -- FK to ossea_configs.id
    volume_type       VARCHAR(50),                       -- ROOT, DATADISK
    device_path       VARCHAR(255),                      -- Device path on OMA
    mount_point       VARCHAR(255),                      -- Mount point path
    status            VARCHAR(50) DEFAULT 'creating',    -- Volume status
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_ossea_volumes_config (ossea_config_id),
    INDEX idx_ossea_volumes_status (status)
);
```

**Key for Failover:**
- `volume_id` contains the real CloudStack UUID (e.g., `e915ef05-ddf5-48d5-8352-a01300609717`)
- Referenced by `vm_disks.ossea_volume_id` (foreign key relationship)
- Essential for volume snapshot and detach/attach operations

### **5. cbt_history**
Tracks Change Block Tracking data for sync validation.

```sql
CREATE TABLE cbt_history (
    id            BIGINT PRIMARY KEY AUTO_INCREMENT,
    job_id        VARCHAR(255) NOT NULL,               -- FK to replication_jobs.id
    change_id     TEXT,                                -- CBT change ID
    sync_success  BOOLEAN DEFAULT FALSE,               -- Sync operation success
    bytes_synced  BIGINT DEFAULT 0,                    -- Bytes synchronized
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_cbt_history_job_id (job_id),
    INDEX idx_cbt_history_sync_success (sync_success)
);
```

**Failover Validation Use:**
- Used by pre-failover validator to check VM sync status
- Ensures VM has valid ChangeID from successful sync operation
- Required for failover readiness assessment

### **6. replication_jobs**
Core migration jobs table (existing).

```sql
CREATE TABLE replication_jobs (
    id              VARCHAR(255) PRIMARY KEY,          -- Job ID 
    source_vm_id    VARCHAR(255) NOT NULL,             -- VMware VM ID
    source_vm_name  VARCHAR(255) NOT NULL,             -- VM name
    source_vm_path  VARCHAR(255) NOT NULL,             -- VM path in vCenter
    vcenter_host    VARCHAR(255) NOT NULL,             -- vCenter hostname
    datacenter      VARCHAR(255) NOT NULL,             -- Datacenter name
    status          VARCHAR(50) DEFAULT 'pending',     -- Job status
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_replication_jobs_vm_id (source_vm_id),
    INDEX idx_replication_jobs_status (status)
);
```

**Failover Relationship:**
- `failover_jobs.vm_id` = `replication_jobs.source_vm_id`
- `failover_jobs.replication_job_id` = `replication_jobs.id`
- Ensures failover only operates on successfully migrated VMs

---

## 🔍 **Critical Queries for Failover Operations**

### **VM Readiness Validation**
```sql
-- Check if VM has completed migration and valid ChangeID
SELECT 
    rj.id as job_id,
    rj.source_vm_name,
    rj.status as job_status,
    vd.cpu_count,
    vd.memory_mb,
    vd.os_type,
    ov.volume_id as cloudstack_volume_id,
    ov.status as volume_status,
    cb.change_id,
    cb.sync_success
FROM replication_jobs rj
JOIN vm_disks vd ON rj.id = vd.job_id
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id  
LEFT JOIN cbt_history cb ON rj.id = cb.job_id AND cb.sync_success = TRUE
WHERE rj.source_vm_id = '4205a841-0265-f4bd-39a6-39fd92196f53'
  AND vd.cpu_count > 0  -- First disk with VM specs
ORDER BY rj.created_at DESC, cb.created_at DESC
LIMIT 1;
```

### **Network Mapping Validation**
```sql
-- Check network mappings for VM
SELECT 
    nm.source_network_name,
    nm.destination_network_id,
    nm.destination_network_name,
    nm.is_test_network,
    CASE 
        WHEN nm.destination_network_id IS NOT NULL THEN 'mapped'
        ELSE 'unmapped'
    END as mapping_status
FROM network_mappings nm
WHERE nm.vm_id = '4205a841-0265-f4bd-39a6-39fd92196f53';
```

### **Active Jobs Check**
```sql
-- Check for active jobs that would prevent failover
SELECT 
    'migration' as job_type,
    rj.id as job_id,
    rj.status,
    rj.created_at
FROM replication_jobs rj
WHERE rj.source_vm_id = '4205a841-0265-f4bd-39a6-39fd92196f53'
  AND rj.status IN ('pending', 'running', 'replicating')

UNION ALL

SELECT 
    'failover' as job_type,
    fj.job_id,
    fj.status,
    fj.created_at
FROM failover_jobs fj  
WHERE fj.vm_id = '4205a841-0265-f4bd-39a6-39fd92196f53'
  AND fj.status IN ('pending', 'validating', 'executing', 'snapshotting', 'creating_vm', 'switching_volume');
```

### **Volume Mapping Resolution**
```sql
-- Get real CloudStack volume ID for VM
SELECT 
    vd.ossea_volume_id as db_volume_id,
    ov.volume_id as cloudstack_volume_id,
    ov.volume_name,
    ov.status,
    ov.size_gb
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
JOIN replication_jobs rj ON vd.job_id = rj.id
WHERE rj.source_vm_id = '4205a841-0265-f4bd-39a6-39fd92196f53'
ORDER BY vd.created_at DESC
LIMIT 1;
```

---

## 📊 **Performance Considerations**

### **Indexing Strategy**
- **vm_id indexes** on all failover tables for fast VM lookups
- **Status indexes** for filtering active/completed jobs
- **Timestamp indexes** for chronological queries
- **Unique constraints** prevent duplicate network mappings

### **Query Optimization**
- Use `LIMIT 1` for latest record queries
- Join on indexed foreign key relationships
- Filter on indexed status fields before other conditions
- Consider query execution plans for complex validation queries

### **Data Retention**
- **Completed failover jobs**: Retain for audit/compliance (configurable)
- **Failed jobs**: Retain for troubleshooting and analysis
- **Test failover jobs**: Can be cleaned up after successful completion
- **Network mappings**: Retain indefinitely (small data footprint)

---

## 🔄 **Database Migrations**

### **Applied Migrations**
```sql
-- Migration 001: Create failover_jobs table
CREATE TABLE failover_jobs (...);

-- Migration 002: Create network_mappings table  
CREATE TABLE network_mappings (...);

-- Migration 003: Extend vm_disks with VM specifications
ALTER TABLE vm_disks ADD COLUMN cpu_count INT DEFAULT 0;
ALTER TABLE vm_disks ADD COLUMN memory_mb INT DEFAULT 0;
-- ... additional columns

-- Migration 004: Add failover-specific indexes
CREATE INDEX idx_failover_jobs_vm_id ON failover_jobs(vm_id);
CREATE INDEX idx_network_mappings_vm_id ON network_mappings(vm_id);
-- ... additional indexes
```

### **Pending Migrations**
No pending database migrations for failover system.

---

## 📚 **Related Documentation**
- [VM Failover System](../features/VM_FAILOVER_SYSTEM.md)
- [Failover API Documentation](../api/FAILOVER_API.md)
- [Main Database Schema](../database-schema.md)
- [OSSEA Integration](../ossea-integration.md)

