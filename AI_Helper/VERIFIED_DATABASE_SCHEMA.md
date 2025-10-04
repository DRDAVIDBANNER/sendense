# MigrateKit OSSEA - VERIFIED Database Schema Reference

**Date**: September 9, 2025 - UPDATED WITH VM-CENTRIC ARCHITECTURE  
**Status**: ✅ **VERIFIED FROM SOURCE CODE + VM-CENTRIC IMPLEMENTATION**  
**Purpose**: Complete database schema with VM-Centric Architecture integration

⚠️ **CRITICAL**: This schema was verified directly from source code models. NO ASSUMPTIONS made about field names.

---

## 🔍 **VERIFICATION METHODOLOGY**

### **Sources Examined**
1. **`source/current/oma/database/models.go`** - Primary OMA database models
2. **`source/current/volume-daemon/models/volume.go`** - Volume Daemon models  
3. **`source/current/volume-daemon/repository/`** - Volume Daemon repository structures
4. **`source/current/oma/models/`** - API model structures

### **Key Findings**
- ✅ **Field Names Verified**: All field names extracted from GORM struct tags
- ⚠️ **Critical Differences Found**: Some fields differ from previous assumptions
- 🔧 **Multiple Model Types**: Different models for API vs Database vs Volume Daemon

---

## 📊 **OMA DATABASE MODELS (VERIFIED)**

### **🎯 VM-CENTRIC ARCHITECTURE MASTER TABLE** 

### **vm_replication_contexts** - VM MASTER CONTEXT TABLE
```sql
-- ✅ NEW: VM-Centric Architecture Master Table
-- PURPOSE: Centralized VM context for all migration operations
-- RELATIONSHIPS: Parent table with CASCADE DELETE to all related tables

context_id VARCHAR(64) PRIMARY KEY DEFAULT uuid()  -- Auto-generated UUID
vm_name VARCHAR(255) NOT NULL                       -- VM display name
vmware_vm_id VARCHAR(255) NOT NULL                  -- VMware VM reference  
vm_path VARCHAR(500) NOT NULL                       -- VMware inventory path
vcenter_host VARCHAR(255) NOT NULL                  -- vCenter server
datacenter VARCHAR(255) NOT NULL                    -- VMware datacenter
current_status ENUM('discovered','replicating','ready_for_failover','failed_over_test','failed_over_live','completed','failed','cleanup_required') DEFAULT 'discovered'
current_job_id VARCHAR(191)                         -- Currently active job
total_jobs_run INT DEFAULT 0                        -- Job statistics
successful_jobs INT DEFAULT 0                       -- Success count
failed_jobs INT DEFAULT 0                           -- Failure count
last_successful_job_id VARCHAR(191)                 -- Last success reference
cpu_count INT                                       -- VM specification
memory_mb INT                                       -- VM memory in MB
os_type VARCHAR(255)                                -- Operating system
power_state VARCHAR(50)                             -- VM power state
vm_tools_version VARCHAR(255)                       -- VMware Tools version
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()    -- Context creation
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP() ON UPDATE CURRENT_TIMESTAMP()
first_job_at TIMESTAMP                              -- First job timestamp
last_job_at TIMESTAMP                               -- Last job timestamp
last_status_change TIMESTAMP DEFAULT CURRENT_TIMESTAMP()

-- INDEXES
INDEX idx_vm_name (vm_name)
INDEX idx_vmware_vm_id (vmware_vm_id)  
INDEX idx_vcenter_host (vcenter_host)
INDEX idx_current_status (current_status)
INDEX idx_current_job_id (current_job_id)
INDEX idx_last_successful_job_id (last_successful_job_id)
```

### **replication_jobs** - COMPLETE VERIFIED SCHEMA + VM-CENTRIC
```sql
-- SOURCE: source/current/oma/database/models.go:61-118 ReplicationJob struct
-- TABLE: replication_jobs (GORM auto-pluralizes)
-- ✅ UPDATED: Now includes vm_context_id for VM-Centric Architecture

id VARCHAR(191) PRIMARY KEY                     -- gorm:"primaryKey" - Job ID from API
vm_context_id VARCHAR(64)                       -- ✅ NEW: FK to vm_replication_contexts ON DELETE CASCADE
source_vm_id VARCHAR(191) NOT NULL              -- gorm:"not null" 
source_vm_name VARCHAR(191) NOT NULL            -- gorm:"not null"
source_vm_path VARCHAR(191) NOT NULL            -- gorm:"not null"
vcenter_host VARCHAR(191) NOT NULL              -- gorm:"not null" ⚠️ CRITICAL: "vcenter_host" NOT "v_center_host"
datacenter VARCHAR(191) NOT NULL                -- gorm:"not null"
replication_type VARCHAR(191) NOT NULL          -- gorm:"not null" - Values: "initial", "incremental"
target_network VARCHAR(191)                     -- Optional field
status VARCHAR(191) DEFAULT 'pending'           -- gorm:"default:'pending'" - Values: pending, running, completed, failed, cancelled
progress_percent DECIMAL(5,2) DEFAULT 0.0       -- gorm:"default:0.0"
current_operation VARCHAR(191)                  -- Progress operation description
bytes_transferred BIGINT DEFAULT 0              -- gorm:"default:0"
total_bytes BIGINT DEFAULT 0                    -- gorm:"default:0"
transfer_speed_bps BIGINT DEFAULT 0             -- gorm:"default:0" ⚠️ NOTE: "bps" lowercase, not "Bps"
error_message VARCHAR(191)                      -- Error details
change_id VARCHAR(191)                          -- VMware CBT ChangeID for incremental sync
previous_change_id VARCHAR(191)                 -- For tracking incremental chains
snapshot_id VARCHAR(191)                        -- VMware snapshot reference

-- ✅ VM-CENTRIC FOREIGN KEY
FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
nbd_port INT                                    -- Dynamic NBD port allocation
nbd_export_name VARCHAR(191)                    -- Dynamic NBD export name
target_device VARCHAR(191)                      -- Target device path
ossea_config_id INT                             -- FK reference (constraint temporarily disabled)

-- VMA Progress Integration (v1.5.0) - Lines 98-118
vma_sync_type VARCHAR(191)                      -- gorm:"column:vma_sync_type"
vma_current_phase VARCHAR(191)                  -- gorm:"column:vma_current_phase"
vma_eta_seconds INT DEFAULT 0                   -- gorm:"column:vma_eta_seconds;default:0"
vma_progress_percent DECIMAL(5,2) DEFAULT 0.0   -- gorm:"column:vma_progress_percent;default:0.0"
vma_bytes_transferred BIGINT DEFAULT 0          -- gorm:"column:vma_bytes_transferred;default:0"
vma_total_bytes BIGINT DEFAULT 0                -- gorm:"column:vma_total_bytes;default:0"
vma_transfer_speed_bps BIGINT DEFAULT 0         -- gorm:"column:vma_transfer_speed_bps;default:0"
setup_progress_percent DECIMAL(5,2) DEFAULT 0.0 -- gorm:"column:setup_progress_percent;default:0.0"

-- Standard GORM timestamps
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- GORM automatic
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP -- GORM automatic
completed_at TIMESTAMP NULL                     -- gorm:"column:completed_at"
```

### **vm_disks** - COMPLETE VERIFIED SCHEMA + VM-CENTRIC
```sql
-- SOURCE: source/current/oma/database/models.go:119-151 VMDisk struct
-- TABLE: vm_disks (GORM auto-pluralizes)
-- ✅ UPDATED: Now includes vm_context_id for VM-Centric Architecture

id INT PRIMARY KEY AUTO_INCREMENT               -- gorm:"primaryKey"
job_id VARCHAR(191) NOT NULL                    -- gorm:"type:varchar(191);not null" - FK to replication_jobs.id
vm_context_id VARCHAR(64)                       -- ✅ NEW: FK to vm_replication_contexts ON DELETE CASCADE
disk_id VARCHAR(191) NOT NULL                   -- gorm:"not null" - Source disk identifier
vmdk_path VARCHAR(191) NOT NULL                 -- gorm:"not null" - VMware VMDK path
size_gb INT NOT NULL                            -- gorm:"not null" - Disk size in GB
datastore VARCHAR(191)                          -- VMware datastore name
unit_number INT                                 -- SCSI unit number
label VARCHAR(191)                              -- Disk label
capacity_bytes BIGINT                           -- Disk capacity in bytes
provisioning_type VARCHAR(191)                  -- thick, thin, etc.
ossea_volume_id INT                             -- FK to ossea_volumes.id (constraint temporarily disabled)
disk_change_id VARCHAR(191)                     -- CBT ChangeID for this specific disk
sync_status VARCHAR(191) DEFAULT 'pending'      -- gorm:"default:'pending'" - Values: pending, syncing, completed, failed
sync_progress_percent DECIMAL(5,2) DEFAULT 0.0  -- gorm:"default:0.0"
bytes_synced BIGINT DEFAULT 0                   -- gorm:"default:0"

-- ✅ VM-CENTRIC FOREIGN KEY
FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- GORM automatic
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP -- GORM automatic
```

### **failover_jobs** - COMPLETE VERIFIED SCHEMA + VM-CENTRIC
```sql
-- SOURCE: source/current/oma/database/models.go:228-280 FailoverJob struct
-- TABLE: failover_jobs (GORM auto-pluralizes)
-- ✅ UPDATED: Now includes vm_context_id for VM-Centric Architecture

id INT PRIMARY KEY AUTO_INCREMENT               -- gorm:"primaryKey"
job_id VARCHAR(191) NOT NULL                    -- gorm:"uniqueIndex;not null" - Format: failover-YYYYMMDD-HHMMSS
vm_id VARCHAR(191) NOT NULL                     -- gorm:"not null;index" - Source VM ID from VMware
vm_context_id VARCHAR(64)                       -- ✅ NEW: FK to vm_replication_contexts ON DELETE CASCADE
replication_job_id VARCHAR(191)                 -- gorm:"index" - FK to replication_jobs.id (SET NULL constraint)
job_type VARCHAR(191) NOT NULL                  -- gorm:"not null" - Values: "live", "test"
status VARCHAR(191) DEFAULT 'pending'           -- gorm:"default:'pending'" - Values: pending, validating, snapshotting, creating_vm, switching_volume, powering_on, completed, failed, cleanup, reverting
source_vm_name VARCHAR(191) NOT NULL            -- gorm:"not null" - Original VM name
source_vm_spec TEXT                             -- gorm:"type:TEXT" - JSON-encoded VM specifications
destination_vm_id VARCHAR(191)                  -- Created VM ID in OSSEA
ossea_snapshot_id VARCHAR(191)                  -- Snapshot before failover
linstor_snapshot_name VARCHAR(191)              -- Linstor snapshot name for rollback
error_message TEXT                              -- Error details
progress_percent DECIMAL(5,2) DEFAULT 0.0       -- gorm:"default:0.0"
current_step VARCHAR(191)                       -- Current operation step
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- GORM automatic
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP -- GORM automatic
completed_at TIMESTAMP NULL                     -- Completion timestamp

-- ✅ VM-CENTRIC FOREIGN KEY
FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
```

### **ossea_volumes** - COMPLETE VERIFIED SCHEMA + VM-CENTRIC
```sql
-- SOURCE: source/current/oma/database/models.go OSSEAVolume struct
-- TABLE: ossea_volumes (GORM auto-pluralizes)
-- ✅ UPDATED: Now includes vm_context_id for VM-Centric Architecture

id INT PRIMARY KEY AUTO_INCREMENT               -- gorm:"primaryKey"
volume_id VARCHAR(191) NOT NULL UNIQUE          -- CloudStack volume UUID
vm_context_id VARCHAR(64)                       -- ✅ NEW: FK to vm_replication_contexts ON DELETE CASCADE
name VARCHAR(191) NOT NULL                      -- Volume display name
size_gb INT NOT NULL                            -- Volume size in GB
disk_offering_id VARCHAR(191)                   -- CloudStack disk offering
state VARCHAR(191) DEFAULT 'creating'           -- Volume state
device_path VARCHAR(191)                        -- Linux device path when attached
vm_id VARCHAR(191)                              -- VM currently attached to
zone_id VARCHAR(191)                            -- CloudStack zone
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- Creation timestamp
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

-- ✅ VM-CENTRIC FOREIGN KEY  
FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
```

### **cbt_history** - COMPLETE VERIFIED SCHEMA + VM-CENTRIC
```sql
-- SOURCE: source/current/oma/database/models.go CBTHistory struct
-- TABLE: cbt_history (GORM auto-pluralizes)
-- ✅ UPDATED: Now includes vm_context_id for VM-Centric Architecture

id INT PRIMARY KEY AUTO_INCREMENT               -- gorm:"primaryKey"
job_id VARCHAR(191) NOT NULL                    -- Replication job reference
vm_context_id VARCHAR(64)                       -- ✅ NEW: FK to vm_replication_contexts ON DELETE CASCADE
disk_id VARCHAR(191) NOT NULL                   -- Disk identifier
change_id VARCHAR(191)                          -- VMware CBT change ID
previous_change_id VARCHAR(191)                 -- Previous change ID for incremental chains
sync_type VARCHAR(191) NOT NULL                 -- Type: initial, incremental, full
sync_success BOOLEAN DEFAULT FALSE              -- Sync completion status
blocks_changed BIGINT DEFAULT 0                 -- Number of changed blocks
bytes_transferred BIGINT DEFAULT 0              -- Bytes transferred
sync_duration_seconds INT DEFAULT 0             -- Sync duration
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- Record creation
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

-- ✅ VM-CENTRIC FOREIGN KEY
FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
```

---

## 🔧 **VOLUME DAEMON DATABASE MODELS (VERIFIED)**

### **device_mappings** - COMPLETE VERIFIED SCHEMA + VM-CENTRIC
```sql
-- SOURCE: source/current/volume-daemon/models/volume.go:53-68 DeviceMapping struct
-- TABLE: device_mappings (Volume Daemon managed)
-- ✅ UPDATED: Now includes vm_context_id for VM-Centric Architecture

id VARCHAR(191) PRIMARY KEY                     -- json:"id" db:"id"
vm_context_id VARCHAR(64)                       -- ✅ NEW: FK to vm_replication_contexts ON DELETE CASCADE
volume_uuid VARCHAR(191) NOT NULL UNIQUE        -- json:"volume_uuid" db:"volume_uuid" ⚠️ NOTE: "volume_uuid" NOT "volume_id"
volume_id_numeric BIGINT                        -- json:"volume_id_numeric" db:"volume_id_numeric" - CloudStack numeric ID
vm_id VARCHAR(191) NOT NULL                     -- json:"vm_id" db:"vm_id"
operation_mode VARCHAR(191) NOT NULL            -- json:"operation_mode" db:"operation_mode" - Values: "oma", "failover"
cloudstack_device_id INT                        -- json:"cloudstack_device_id" db:"cloudstack_device_id"
requires_device_correlation BOOLEAN             -- json:"requires_device_correlation" db:"requires_device_correlation"
device_path VARCHAR(191) NOT NULL               -- json:"device_path" db:"device_path" - Linux device path like /dev/vdb
cloudstack_state VARCHAR(191) NOT NULL          -- json:"cloudstack_state" db:"cloudstack_state"
linux_state VARCHAR(191) NOT NULL              -- json:"linux_state" db:"linux_state"
size BIGINT NOT NULL                            -- json:"size" db:"size" - Size in bytes
last_sync TIMESTAMP NOT NULL                   -- json:"last_sync" db:"last_sync"
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- json:"created_at" db:"created_at"
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP -- json:"updated_at" db:"updated_at"

-- ✅ VM-CENTRIC FOREIGN KEY
FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
```

### **nbd_exports** - COMPLETE VERIFIED SCHEMA + VM-CENTRIC
```sql
-- SOURCE: source/current/volume-daemon/models/volume.go:143-153 NBDExportInfo struct
-- TABLE: nbd_exports (Volume Daemon managed)
-- ✅ UPDATED: Now includes vm_context_id for VM-Centric Architecture

id VARCHAR(191) PRIMARY KEY                     -- json:"id"
job_id VARCHAR(191)                             -- Replication job reference
vm_context_id VARCHAR(64)                       -- ✅ NEW: FK to vm_replication_contexts ON DELETE CASCADE
volume_id VARCHAR(191) NOT NULL                 -- json:"volume_id" - References device_mappings.volume_uuid
export_name VARCHAR(191) NOT NULL UNIQUE        -- json:"export_name" - NBD export name
device_path VARCHAR(191) NOT NULL               -- json:"device_path" - Linux device path
port INT NOT NULL DEFAULT 10809                 -- json:"port" - NBD port (usually 10809)
status VARCHAR(191) NOT NULL                    -- json:"status" - Values: "pending", "active", "failed"
config_path VARCHAR(191)                        -- NBD configuration file path
metadata TEXT                                   -- json:"metadata" - JSON metadata
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- json:"created_at"
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP -- json:"updated_at"

-- ✅ VM-CENTRIC FOREIGN KEY
FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
```

---

## 🔗 **FOREIGN KEY RELATIONSHIPS (VERIFIED)**

### **OMA Database FKs**
```sql
-- VERIFIED from source code - some constraints temporarily disabled
vm_disks.job_id → replication_jobs.id (CASCADE DELETE)
vm_disks.ossea_volume_id → ossea_volumes.id (constraint disabled)
failover_jobs.replication_job_id → replication_jobs.id (SET NULL)
```

### **Volume Daemon FKs**
```sql
-- VERIFIED from Volume Daemon models
nbd_exports.volume_id → device_mappings.volume_uuid (logical relationship)
```

---

## ⚠️ **CRITICAL FIELD NAME DIFFERENCES**

### **Database vs Documentation Mismatches**
1. **`vcenter_host`** NOT `v_center_host` - Database uses underscore format
2. **`transfer_speed_bps`** NOT `transfer_speed_Bps` - Lowercase "bps"
3. **`volume_uuid`** in device_mappings NOT `volume_id` - Volume Daemon uses UUID format
4. **Multiple progress fields** - VMA-specific fields prefixed with `vma_`

### **GORM Column Mappings**
- Most fields use default GORM snake_case conversion
- Some fields have explicit `gorm:"column:field_name"` mappings
- VARCHAR(191) is GORM default for MySQL compatibility

---

## 🎯 **REPOSITORY IMPLEMENTATION REQUIREMENTS**

### **Required Methods for ReplicationJobRepository**
```go
// Based on verified schema and current handler usage:
GetByID(ctx context.Context, jobID string) (*database.ReplicationJob, error)
GetJobVolumes(ctx context.Context, jobID string) ([]string, error)
CheckActiveFailover(ctx context.Context, jobID string) (bool, error)
Delete(ctx context.Context, jobID string) error
Create(ctx context.Context, job *database.ReplicationJob) error
Update(ctx context.Context, job *database.ReplicationJob) error
```

### **SQL Queries Verified**
```sql
-- Get job volumes (from current handler implementation)
SELECT DISTINCT ov.volume_id
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
WHERE vd.job_id = ?

-- Check active failover (from current handler implementation)
SELECT COUNT(*) FROM failover_jobs 
WHERE replication_job_id = ? 
AND status IN ('pending', 'executing', 'validating', 'creating_vm', 'switching_volume')
```

## 🗑️ **JOB DELETION IMPLEMENTATION**

### **Complete Job Deletion Workflow**
```sql
-- 1. Get job details and volumes
SELECT id, source_vm_name, status FROM replication_jobs WHERE id = ?;

-- 2. Get job volumes for deletion
SELECT DISTINCT ov.volume_id
FROM vm_disks vd
JOIN ossea_volumes ov ON vd.ossea_volume_id = ov.id
WHERE vd.job_id = ?;

-- 3. Check for active failover operations
SELECT COUNT(*) FROM failover_jobs 
WHERE replication_job_id = ? 
AND status IN ('pending', 'executing', 'validating', 'creating_vm', 'switching_volume');

-- 4. Delete job (leverages CASCADE DELETE)
DELETE FROM replication_jobs WHERE id = ?;
-- Automatically deletes: vm_disks, cbt_history via CASCADE DELETE
-- Sets failover_jobs.replication_job_id to NULL via SET NULL
```

### **Volume Deletion Safety Patterns**
- **CloudStack Protection**: Error 431 `Please specify a volume that is not attached to any VM.`
- **Volume Daemon API**: All volume operations via `localhost:8090/api/v1/volumes/{id}`
- **Expected Behavior**: Attached volume deletion fails safely (not an error condition)
- **Database Connection**: `mysql -u oma_user -poma_password migratekit_oma`

### **Repository Pattern Usage**
```go
// Initialize repository
replicationRepo := database.NewReplicationJobRepository(db)

// Safe deletion workflow
job, err := replicationRepo.GetByID(ctx, jobID)
volumes, err := replicationRepo.GetJobVolumes(ctx, jobID) 
hasActiveFailover, err := replicationRepo.CheckActiveFailover(ctx, jobID)
err = replicationRepo.Delete(ctx, jobID)
```

---

## 🎯 **VM-CENTRIC ARCHITECTURE IMPLEMENTATION**

### **Master Table Architecture**
The `vm_replication_contexts` table serves as the **single source of truth** for all VM-related operations:

- **Centralized VM Metadata**: All VM information consolidated in one location
- **Job Statistics**: Track total jobs, success/failure counts per VM
- **Status Management**: Current VM status across the migration lifecycle  
- **CASCADE DELETE**: All related records automatically cleaned up when VM context is deleted

### **Foreign Key Integration**
ALL migration-related tables now include `vm_context_id` with `CASCADE DELETE`:

```sql
-- All tables link to vm_replication_contexts for automatic cleanup
FOREIGN KEY (vm_context_id) REFERENCES vm_replication_contexts(context_id) ON DELETE CASCADE
```

**Tables with VM Context Integration:**
1. ✅ `replication_jobs` - Migration jobs
2. ✅ `failover_jobs` - Failover operations  
3. ✅ `vm_disks` - Disk tracking
4. ✅ `ossea_volumes` - Volume management
5. ✅ `cbt_history` - Change tracking history
6. ✅ `device_mappings` - Volume Daemon device correlation
7. ✅ `nbd_exports` - Volume Daemon NBD exports

### **Enhanced Deletion System**

#### **VM-Context Aware Deletion API**
```http
DELETE /api/v1/replications/{job_id}
```

**Features:**
- ✅ **JobLog Integration**: Full audit trail with structured logging
- ✅ **Safety Validations**: Prevents deletion of active jobs
- ✅ **Volume Daemon Integration**: Automated volume cleanup
- ✅ **VM Context Updates**: Statistics and status updates
- ✅ **CASCADE DELETE**: Database integrity maintained automatically

#### **Repository Pattern**
```go
// Enhanced repository with VM context awareness
replicationRepo := database.NewReplicationJobRepository(db)

// Creates VM context automatically during job creation
err := replicationRepo.Create(ctx, job)

// VM-context aware deletion with statistics updates  
err := replicationRepo.Delete(ctx, jobID)
```

### **Volume Daemon VM Context Integration**

The Volume Daemon automatically populates `vm_context_id` in managed tables:

#### **Device Mappings & NBD Exports**
- **Lookup Strategy**: Query `ossea_volumes.vm_context_id` by `volume_id`
- **Debug Logging**: Full tracing of VM context lookup process
- **Fallback**: Continues operation if VM context lookup fails
- ✅ **Production Validated**: pgtest2 demonstrates perfect integration

---

## 🏗️ **IMPLEMENTATION STATUS**

### **Phase Completion**
- ✅ **Phase 1**: VM-Centric schema with CASCADE DELETE
- ✅ **Phase 2**: Repository pattern implementation
- ✅ **Phase 3**: Job creation VM context integration
- ✅ **Phase 4**: Enhanced deletion with VM context awareness
- ✅ **Phase 5**: VM context GUI endpoints implementation
- 🔄 **Phase 6**: Backwards compatibility testing (pending)

### **Production Ready**
- ✅ **Database Schema**: All foreign keys and indexes in place
- ✅ **OMA Integration**: Job creation, deletion, and CBT tracking
- ✅ **Volume Daemon**: Device mapping and NBD export integration
- ✅ **GUI Endpoints**: Complete VM context data for frontend integration
- ✅ **Live Testing**: Validated with real migration jobs and API endpoints

---

## 🎯 **VM CONTEXT GUI ENDPOINTS**

### **Phase 5: GUI Integration Complete**

The VM-Centric Architecture provides two minimal, comprehensive endpoints for GUI integration:

#### **1. List All VM Contexts**
```
GET /api/v1/vm-contexts
```

**Response Structure:**
```json
{
  "count": 2,
  "vm_contexts": [
    {
      "context_id": "ctx-pgtest1-20250909-113839",
      "vm_name": "pgtest1",
      "vmware_vm_id": "420570c7-f61f-a930-77c5-1e876786cb3c",
      "vm_path": "/DatabanxDC/vm/pgtest1",
      "vcenter_host": "quad-vcenter-01.quadris.local",
      "datacenter": "DatabanxDC",
      "current_status": "replicating",
      "current_job_id": "job-20250909-114850",
      "total_jobs_run": 2,
      "successful_jobs": 0,
      "failed_jobs": 0,
      "last_successful_job_id": null,
      "created_at": "2025-09-09T11:38:39+01:00",
      "updated_at": "2025-09-09T11:48:50+01:00",
      "first_job_at": "2025-09-09T11:38:39+01:00",
      "last_job_at": "2025-09-09T11:48:50+01:00",
      "last_status_change": "2025-09-09T11:48:50+01:00"
    }
  ]
}
```

#### **2. Detailed VM Context**
```
GET /api/v1/vm-contexts/{vm_name}
```

**Response Structure:**
```json
{
  "context": {
    "context_id": "ctx-pgtest1-20250909-113839",
    "vm_name": "pgtest1",
    "current_status": "replicating",
    "current_job_id": "job-20250909-114850",
    "total_jobs_run": 2,
    "successful_jobs": 0,
    "failed_jobs": 0
  },
  "current_job": {
    "id": "job-20250909-114850",
    "status": "replicating",
    "progress_percent": 49.30555555555556,
    "current_operation": "Transferring Data",
    "bytes_transferred": 19058917376,
    "total_bytes": 38654705664,
    "transfer_speed_bps": 12013439,
    "vma_throughput_mbps": 11.46,
    "vma_eta_seconds": 1631
  },
  "job_history": [
    // Last 10 jobs for this VM
  ],
  "disks": [
    {
      "disk_id": "disk-2000",
      "size_gb": 102,
      "capacity_bytes": 109521666048,
      "cpu_count": 2,
      "memory_mb": 8192,
      "os_type": "windows",
      "network_config": "[{...}]"
    }
  ],
  "cbt_history": [
    // Last 20 CBT records
  ]
}
```

### **Implementation Details**

**Files Modified:**
- `source/current/oma/database/repository.go` - Added `VMReplicationContextRepository`
- `source/current/oma/database/models.go` - Added `VMReplicationContext` model
- `source/current/oma/api/handlers/vm_contexts.go` - New GUI endpoint handler
- `source/current/oma/api/handlers/handlers.go` - Added VM context handler integration
- `source/current/oma/api/server.go` - Added GUI endpoint routes

**Key Features:**
- ✅ **Single Source of Truth**: All VM data centralized via `vm_context_id`
- ✅ **Real-time Progress**: Live job status and transfer metrics
- ✅ **Historical Data**: Job history and CBT change tracking
- ✅ **Complete VM Specs**: CPU, memory, disks, network configuration
- ✅ **Error Handling**: Proper HTTP status codes and error responses
- ✅ **Performance**: Optimized queries with limits (10 jobs, 20 CBT records)

**Production Status:**
- ✅ **Tested**: Both endpoints working with live migration data
- ✅ **Version**: Deployed as `oma-api-v2.18.0-vm-context-endpoints`
- ✅ **Authentication**: Requires authentication via `requireAuth` middleware
- ✅ **Format**: Clean JSON responses following project standards

---

**🚨 CRITICAL**: This schema is the VERIFIED source of truth for all repository implementation. Use EXACT field names as documented here.
