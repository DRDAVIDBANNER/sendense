# Normalized Database Schema Design
**Created**: 2025-08-21  
**Status**: DESIGN PHASE  
**Priority**: CRITICAL

## 🎯 **Objectives**

1. **Eliminate volume ID inconsistencies** across all tables
2. **Establish proper foreign key relationships** 
3. **Prevent data corruption** through constraints
4. **Single source of truth** for volume-to-device mappings
5. **Automated cleanup** of stale exports and jobs

## 📊 **Current Schema Problems**

### **❌ Volume ID Data Type Chaos**
- `vm_disks.ossea_volume_id`: `bigint(20)` (CloudStack numeric IDs)
- `nbd_exports.volume_id`: `varchar(191)` (Mixed strings)  
- `device_mappings.volume_id`: `varchar(64)` (UUID strings)

### **❌ Missing Foreign Key Relationships**
- No link between `replication_jobs` → `vm_disks`
- No link between `vm_disks` → `device_mappings`
- No link between `device_mappings` → `nbd_exports`

### **❌ Orphaned Record Issues**
- Stale NBD exports pointing to wrong devices
- No automatic cleanup when jobs fail
- Multiple records claiming same device paths

## 🔧 **Normalized Schema Design**

### **1. Core Replication Job Table** (No changes needed)
```sql
CREATE TABLE replication_jobs (
    id varchar(191) PRIMARY KEY,
    source_vm_id longtext NOT NULL,
    source_vm_name longtext NOT NULL,
    -- ... existing fields remain ...
    status varchar(191) DEFAULT 'pending',
    created_at datetime(3),
    updated_at datetime(3)
);
```

### **2. VM Disks Table** ✨ **ENHANCED**
```sql
CREATE TABLE vm_disks (
    id bigint(20) PRIMARY KEY AUTO_INCREMENT,
    
    -- Foreign key to replication job
    job_id varchar(191) NOT NULL,
    
    -- VMware disk information
    disk_id longtext NOT NULL,
    vm_dk_path longtext NOT NULL,
    size_gb bigint(20) NOT NULL,
    unit_number bigint(20),
    
    -- CloudStack volume link (NORMALIZED to UUID)
    cloudstack_volume_uuid varchar(64) NULL,  -- NEW: UUID format
    cloudstack_volume_id bigint(20) NULL,     -- Keep for CloudStack API
    
    -- Sync tracking
    sync_status varchar(191) DEFAULT 'pending',
    sync_progress_percent double DEFAULT 0,
    bytes_synced bigint(20) DEFAULT 0,
    
    -- Timestamps
    created_at datetime(3),
    updated_at datetime(3),
    
    -- Foreign key constraints
    FOREIGN KEY (job_id) REFERENCES replication_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (cloudstack_volume_uuid) REFERENCES device_mappings(volume_id) ON DELETE SET NULL,
    
    -- Unique constraints
    UNIQUE KEY unique_job_disk (job_id, disk_id)
);
```

### **3. Device Mappings Table** ✨ **ENHANCED** 
```sql
CREATE TABLE device_mappings (
    id varchar(64) PRIMARY KEY,
    
    -- Volume identification (STANDARDIZED)
    volume_uuid varchar(64) NOT NULL UNIQUE,     -- CloudStack UUID
    volume_id bigint(20) NOT NULL,               -- CloudStack numeric ID
    
    -- VM and device information
    vm_id varchar(64) NOT NULL,
    device_path varchar(255) NOT NULL UNIQUE,
    
    -- Operation mode
    operation_mode enum('oma','failover') DEFAULT 'oma',
    cloudstack_device_id int(11),
    requires_device_correlation tinyint(1) DEFAULT 1,
    
    -- State tracking
    cloudstack_state varchar(32) NOT NULL,
    linux_state varchar(32) NOT NULL,
    size bigint(20) NOT NULL,
    
    -- Timestamps
    last_sync timestamp DEFAULT CURRENT_TIMESTAMP,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Indexes for performance
    KEY idx_vm_id (vm_id),
    KEY idx_volume_id (volume_id),
    KEY idx_last_sync (last_sync)
);
```

### **4. NBD Exports Table** ✨ **COMPLETELY REDESIGNED**
```sql
CREATE TABLE nbd_exports (
    id bigint(20) unsigned PRIMARY KEY AUTO_INCREMENT,
    
    -- Foreign key relationships
    job_id varchar(191) NOT NULL,
    vm_disk_id bigint(20) NOT NULL,               -- NEW: Link to vm_disks
    device_mapping_id varchar(64) NOT NULL,      -- NEW: Link to device_mappings
    
    -- Export configuration
    export_name varchar(191) NOT NULL UNIQUE,
    port bigint(20) NOT NULL,
    device_path varchar(255) NOT NULL,           -- Synced from device_mappings
    config_path longtext NOT NULL,
    
    -- Status and lifecycle
    status varchar(191) NOT NULL DEFAULT 'pending',
    
    -- Timestamps
    created_at datetime(3),
    updated_at datetime(3),
    
    -- Foreign key constraints
    FOREIGN KEY (job_id) REFERENCES replication_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (vm_disk_id) REFERENCES vm_disks(id) ON DELETE CASCADE,
    FOREIGN KEY (device_mapping_id) REFERENCES device_mappings(volume_uuid) ON DELETE CASCADE,
    
    -- Indexes
    KEY idx_job_id (job_id),
    KEY idx_status (status)
);
```

### **5. Volume Operations Table** (Volume Daemon - Keep as-is)
```sql
-- This table is managed by Volume Daemon and should remain unchanged
-- It tracks individual operations but doesn't need direct links to replication jobs
```

## 🔄 **Data Flow with Normalized Schema**

### **Phase 1: Job Creation**
```sql
INSERT INTO replication_jobs (id, source_vm_name, ...)
```

### **Phase 2: VM Disk Discovery**
```sql
INSERT INTO vm_disks (job_id, disk_id, vm_dk_path, ...)
-- cloudstack_volume_uuid = NULL (to be populated during provisioning)
```

### **Phase 3: Volume Provisioning** 
```sql
-- Volume Daemon creates volume and device mapping
INSERT INTO device_mappings (volume_uuid, volume_id, vm_id, device_path, ...)

-- Update vm_disks with volume link
UPDATE vm_disks SET 
    cloudstack_volume_uuid = ?, 
    cloudstack_volume_id = ?
WHERE job_id = ? AND disk_id = ?
```

### **Phase 4: NBD Export Creation**
```sql
INSERT INTO nbd_exports (
    job_id, 
    vm_disk_id, 
    device_mapping_id,
    export_name, 
    device_path  -- GUARANTEED to match device_mappings
)
```

### **Phase 5: Cleanup (Automatic)**
```sql
-- When job fails/completes, CASCADE DELETE removes:
DELETE FROM replication_jobs WHERE id = ?
-- Automatically cascades to:
-- - vm_disks (via FK)
-- - nbd_exports (via FK)
-- - device_mappings remain for Volume Daemon tracking
```

## 🛡️ **Data Integrity Guarantees**

### **1. Referential Integrity**
- All NBD exports MUST have valid vm_disks and device_mappings
- All vm_disks MUST have valid replication_jobs
- Orphaned records automatically cleaned up via CASCADE DELETE

### **2. Volume ID Consistency** 
- `device_mappings.volume_uuid` is the single source of truth
- `vm_disks.cloudstack_volume_uuid` links directly to it
- `vm_disks.cloudstack_volume_id` kept for CloudStack API compatibility

### **3. Device Path Accuracy**
- `nbd_exports.device_path` synced from `device_mappings.device_path`
- Unique constraints prevent duplicate device assignments
- Volume Daemon maintains real-time device correlation

### **4. Automatic Cleanup**
- Failed jobs automatically remove all related records
- No stale NBD exports pointing to wrong devices
- No orphaned vm_disks without jobs

## 📈 **Benefits of Normalized Schema**

### **✅ Data Consistency**
- Single source of truth for volume-device relationships
- Impossible to have orphaned or inconsistent records
- Automatic cleanup prevents data corruption

### **✅ Performance**
- Proper indexes on foreign keys and frequently queried columns
- Efficient joins between related tables
- Optimized for common query patterns

### **✅ Maintainability**
- Clear relationships between all entities
- Easy to trace data flow from job → disk → device → export
- Simplified debugging and auditing

### **✅ Reliability**
- Foreign key constraints prevent invalid data
- Cascade deletes ensure complete cleanup
- Unique constraints prevent conflicts

## 🚀 **Migration Strategy**

### **Phase 1: Schema Updates** (IMMEDIATE)
1. Add new columns to existing tables
2. Create foreign key relationships
3. Add unique constraints and indexes

### **Phase 2: Data Migration** (IMMEDIATE) 
1. Populate new volume UUID fields
2. Create proper foreign key links
3. Validate all relationships

### **Phase 3: Code Updates** (NEXT)
1. Update replication workflow to use new schema
2. Modify NBD export creation logic
3. Implement automatic cleanup triggers

### **Phase 4: Validation** (FINAL)
1. Test complete data flow with new schema
2. Verify automatic cleanup works
3. Performance testing and optimization

This design eliminates all identified data consistency issues and provides a solid foundation for reliable replication operations.
