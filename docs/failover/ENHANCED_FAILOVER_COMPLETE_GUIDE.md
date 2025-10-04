# Enhanced Test Failover Complete Implementation Guide

## 🎯 **OVERVIEW**

The Enhanced Test Failover system is the **ONLY** supported failover implementation in MigrateKit OSSEA as of September 2025. This system provides comprehensive VM failover capabilities with Linstor snapshot protection, VirtIO driver injection, and complete audit trails.

> **⚠️ IMPORTANT**: The original test_failover.go and live_failover.go are **DEPRECATED** and should not be used. Only the enhanced failover system is supported.

## 🏗️ **ARCHITECTURE**

### **Core Components**

1. **Enhanced Test Failover Engine** (`enhanced_test_failover.go`)
   - Primary orchestration engine
   - Manages 6-step enhanced test failover workflow
   - Integrates with all subsystems (Linstor, VirtIO, Volume Daemon)

2. **Enhanced Live Failover Engine** (`enhanced_live_failover.go`)
   - Permanent failover operations
   - Same architecture as test failover but permanent

3. **Enhanced Failover Wrapper** (`enhanced_failover_wrapper.go`)
   - API endpoint handlers
   - Centralized logging integration
   - Request/response management

4. **Pre-Failover Validator** (`validator.go`)
   - Comprehensive validation before failover
   - VM existence, sync status, network mappings

5. **Enhanced Cleanup Service** (`enhanced_cleanup_service.go`)
   - Rollback and cleanup operations
   - Linstor snapshot rollback capabilities

## 🔄 **ENHANCED TEST FAILOVER WORKFLOW**

### **6-Step Enhanced Process**

```
1. Pre-Failover Validation
   ├── VM existence check
   ├── Active jobs validation  
   ├── Sync status verification
   └── Network mapping validation

2. CloudStack Volume Snapshot Creation ⭐ CRITICAL
   ├── Get volume UUID from database (replication_jobs → vm_disks → ossea_volumes)
   ├── Create snapshot via CloudStack API (osseaClient.CreateVolumeSnapshot)
   ├── Wait for snapshot completion (WaitForSnapshotState)
   ├── Record snapshot ID in failover_jobs.ossea_snapshot_id
   └── Enable emergency rollback capability

3. VirtIO Driver Injection
   ├── Get device path from Volume Daemon
   ├── Execute virt-v2v-in-place script
   ├── Inject Windows drivers for KVM compatibility
   └── Log injection results

4. Test VM Creation
   ├── Get CloudStack configuration (zone, template, service offering)
   ├── Resolve zone names to UUIDs if needed
   ├── Create VM with identical specs to source
   └── Record destination VM ID

5. Volume Attachment (via Volume Daemon)
   ├── Detach volume from OMA
   ├── Attach volume as root to test VM
   └── Verify attachment success

6. Test VM Startup & Validation
   ├── Power on test VM
   ├── Verify boot success
   └── Validate network connectivity
```

## 💾 **CRITICAL DATABASE INTEGRATION**

### **failover_jobs Table Integration**

**⚠️ CRITICAL FIX (September 2025)**: The enhanced failover system now properly integrates with the `failover_jobs` table to enable emergency recovery.

#### **Key Methods:**

1. **createTestFailoverJob()** - Creates record at start
2. **updateFailoverJobWithSnapshot()** - Records Linstor snapshot name
3. **Enhanced Constructor** - Initializes `failoverJobRepo`

#### **Database Schema:**
```sql
CREATE TABLE failover_jobs (
    id INT PRIMARY KEY,
    job_id VARCHAR(191) UNIQUE NOT NULL,
    vm_id VARCHAR(191) NOT NULL,
    job_type VARCHAR(50) NOT NULL,        -- 'test' or 'live'
    status VARCHAR(50) DEFAULT 'pending',
    source_vm_name VARCHAR(191) NOT NULL,
    source_vm_spec TEXT,                  -- JSON VM specifications
    destination_vm_id VARCHAR(191),       -- Created VM ID
    ossea_snapshot_id VARCHAR(191),       -- CloudStack snapshot
    linstor_snapshot_name VARCHAR(191),   -- ⭐ CRITICAL: Linstor snapshot for rollback
    network_mappings TEXT,               -- JSON network mappings
    error_message TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);
```

### **Data Flow Pattern**

```
replication_jobs (source_vm_id) 
    → vm_disks (job_id, ossea_volume_id) 
    → ossea_volumes (id, volume_id) 
    → CloudStack volume UUID for Linstor operations
```

## 🔧 **TECHNICAL IMPLEMENTATION**

### **Enhanced Test Failover Engine Structure**

```go
type EnhancedTestFailoverEngine struct {
    db                    database.Connection
    osseaClient           *ossea.Client
    networkClient         *ossea.NetworkClient
    vmInfoService         services.VMInfoProvider
    networkMappingService *services.NetworkMappingService
    validator             *PreFailoverValidator
    jobTrackingService    *services.GenericJobTrackingService
    linstorConfigManager  *config.LinstorConfigManager
    vmDiskRepo            *database.VMDiskRepository
    failoverJobRepo       *database.FailoverJobRepository  // ⭐ CRITICAL
}
```

### **Key Constructor Fix**

```go
func NewEnhancedTestFailoverEngine(...) *EnhancedTestFailoverEngine {
    return &EnhancedTestFailoverEngine{
        // ... other fields ...
        failoverJobRepo: database.NewFailoverJobRepository(db), // ⭐ CRITICAL FIX
    }
}
```

### **Volume Discovery Logic**

```go
func (etfe *EnhancedTestFailoverEngine) getVolumeUUIDForVM(vmID string) (string, error) {
    // Step 1: Get replication job by source_vm_id
    var replicationJob database.ReplicationJob
    err := etfe.db.GetGormDB().Where("source_vm_id = ?", vmID).
        Order("created_at DESC").First(&replicationJob).Error
    
    // Step 2: Get vm_disks by job_id
    vmDisks, err := etfe.vmDiskRepo.GetByJobID(replicationJob.ID)
    
    // Step 3: Get CloudStack volume UUID from ossea_volumes
    var osseaVolume database.OSSEAVolume
    err = etfe.db.GetGormDB().Where("id = ?", rootDisk.OSSEAVolumeID).First(&osseaVolume).Error
    
    // Step 4: Return real CloudStack UUID
    return osseaVolume.VolumeID, nil // cs-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
}
```

### **CloudStack VM Creation**

```go
func (etfe *EnhancedTestFailoverEngine) createTestCloudStackVM(...) (string, error) {
    // Get OSSEA configuration
    osseaConfig, err := etfe.getOSSEAConfig()
    
    // Resolve zone ID if needed
    zoneID := osseaConfig.Zone
    if len(zoneID) < 36 { // Not UUID format
        zoneID, err = etfe.resolveZoneID(osseaConfig.Zone)
    }
    
    // Create VM with COMPLETE CloudStack parameters
    createRequest := &ossea.CreateVMRequest{
        Name:              vmSpec.Name,
        DisplayName:       vmSpec.DisplayName,
        ServiceOfferingID: osseaConfig.ServiceOfferingID,  // ⭐ REQUIRED
        TemplateID:        osseaConfig.TemplateID,         // ⭐ REQUIRED
        ZoneID:            zoneID,                         // ⭐ REQUIRED
        NetworkID:         osseaConfig.NetworkID,          // ⭐ REQUIRED
        DiskOfferingID:    osseaConfig.DiskOfferingID,     // ⭐ REQUIRED
        RootDiskSize:      etfe.calculateDiskSize(vmSpec), // ⭐ REQUIRED
        StartVM:           false,
        CPUNumber:         vmSpec.CPUs,
        Memory:            vmSpec.MemoryMB,
    }
    
    return etfe.osseaClient.CreateVM(createRequest)
}
```

## 📋 **JOBLOG INTEGRATION** ⭐ **NEW (September 2025)**

All operations now use the **JobLog system** (`internal/joblog/`) for structured logging, job tracking, and step management:

### **JobLog Architecture**

```go
func (etfe *EnhancedTestFailoverEngine) ExecuteEnhancedTestFailover(ctx context.Context, request *EnhancedTestFailoverRequest) error {
    // Start job with tracking and correlation
    ctx, jobID, err := etfe.jobTracker.StartJob(ctx, joblog.JobStart{
        JobType:   "test-failover",
        Operation: "enhanced-test-failover",
        Owner:     "system",
        Metadata: map[string]interface{}{
            "vm_id":   request.VMID,
            "vm_name": request.VMName,
        },
    })
    if err != nil {
        return fmt.Errorf("failed to start job: %w", err)
    }
    defer etfe.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

    // Execute steps with automatic tracking
    err = etfe.jobTracker.RunStep(ctx, jobID, "pre-failover-validation", func(ctx context.Context) error {
        logger := etfe.jobTracker.Logger(ctx)
        logger.Info("Starting pre-failover validation", "vm_id", request.VMID)
        return etfe.executePreFailoverValidation(ctx, request)
    })
    
    return nil
}
```

### **Database Integration**

JobLog system uses three tables for complete audit trail:

```sql
-- job_tracking: Master job records
CREATE TABLE job_tracking (
    id VARCHAR(255) PRIMARY KEY,
    job_type ENUM('test-failover', 'live-failover', 'cleanup', 'migration'),
    operation VARCHAR(255) NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed') DEFAULT 'pending',
    percent_complete TINYINT DEFAULT 0,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    metadata JSON
);

-- job_steps: Individual step tracking
CREATE TABLE job_steps (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    step_order INT NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed') DEFAULT 'pending',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    error_message TEXT NULL,
    FOREIGN KEY (job_id) REFERENCES job_tracking(id) ON DELETE CASCADE
);

-- log_events: Structured log entries
CREATE TABLE log_events (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    step_id BIGINT NULL,
    correlation_id VARCHAR(255) NOT NULL,
    level ENUM('DEBUG', 'INFO', 'WARN', 'ERROR') NOT NULL,
    message TEXT NOT NULL,
    fields JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (job_id) REFERENCES job_tracking(id) ON DELETE CASCADE,
    FOREIGN KEY (step_id) REFERENCES job_steps(id) ON DELETE CASCADE
);
```

### **Key Features**

- ✅ **Automatic step tracking** with database persistence
- ✅ **Correlation IDs** linking all related operations  
- ✅ **Progress percentage** tracking for UI integration
- ✅ **Panic recovery** with automatic error handling
- ✅ **Structured logging** with `log/slog` integration
- ✅ **Real-time monitoring** via database queries

## 📸 **CLOUDSTACK VOLUME SNAPSHOT INTEGRATION** ⭐ **NEW (September 2025)**

Enhanced failover now uses **native CloudStack volume snapshots** instead of Linstor snapshots for simplified architecture and better reliability.

### **Snapshot Creation**
```go
// Create CloudStack volume snapshot
snapshotReq := &ossea.CreateSnapshotRequest{
    VolumeID: volumeUUID,
    Name:     "test-failover-4205a841-1756905671",
}
snapshot, err := osseaClient.CreateVolumeSnapshot(snapshotReq)

// Wait for completion
err = osseaClient.WaitForSnapshotState(snapshot.ID, "BackedUp", 300*time.Second)
```

### **Snapshot Naming Convention**
- **Test Failover**: `test-failover-{shortVMID}-{timestamp}`
- **Live Failover**: `live-failover-{shortVMID}-{timestamp}`
- **No Length Restrictions**: CloudStack allows longer names than Linstor

### **Emergency Rollback**
```go
// Rollback to CloudStack volume snapshot
err := osseaClient.RevertVolumeSnapshot(snapshotID)
```

### **Database Integration**
- **Field**: `failover_jobs.ossea_snapshot_id` stores CloudStack snapshot ID
- **Method**: `UpdateSnapshot(jobID, snapshotID)` for database tracking
- **Deprecated**: `linstor_snapshot_name` field (legacy only)

### **Benefits Over Linstor**
- ✅ **Native Integration**: No Python script dependencies
- ✅ **Better Error Handling**: Go SDK exceptions vs script parsing
- ✅ **Real State Management**: `WaitForSnapshotState()` for completion
- ✅ **Simplified Architecture**: Removes external Linstor dependency
- ✅ **JobLog Compatible**: Full structured logging integration

## 🛡️ **VIRTIO INTEGRATION**

### **Driver Injection Process**
1. **Script Location**: `/opt/migratekit/bin/inject-virtio-drivers.sh`
2. **Tool Used**: `virt-v2v-in-place` with `/usr/share/virtio-win/virtio-win.iso`
3. **Execution**: Passwordless sudo configured for `oma` user
4. **Target**: Windows VMs migrating from VMware to KVM

### **Injection Command**
```bash
sudo inject-virtio-drivers.sh <device-path> <job-id>
```

## 🏷️ **CLOUDSTACK VM NAMING** ⭐ **NEW (September 2025)**

Enhanced failover now includes **automatic VM name sanitization** for CloudStack compatibility:

### **Naming Rules Compliance**
CloudStack VM names must follow strict rules:
- Only ASCII letters 'a' through 'z', digits '0' through '9', and hyphen '-'
- Must be between 1 and 63 characters long  
- Can't start or end with "-"
- Can't start with digit

### **Automatic Sanitization**
```go
func (etfe *EnhancedTestFailoverEngine) sanitizeVMName(originalName string) string {
    // "PhilB Test machine" → "philb-test-machine"
    // "123 Server" → "vm-123-server"  
    // "My_VM@2025" → "my-vm-2025"
}
```

### **Example Transformations**
- **"PhilB Test machine"** → **"philb-test-machine"**
- **"pgtest2"** → **"pgtest2"** (already compliant)
- **"VM-123_Test@2025"** → **"vm-123-test-2025"**
- **"123StartWithDigit"** → **"vm-123startwithdigit"**

## 🔄 **VOLUME DAEMON INTEGRATION**

All volume operations **MUST** use the Volume Daemon API (localhost:8090):

```go
volumeClient := common.NewVolumeClient("http://localhost:8090")

// Detach from OMA
detachOp, err := volumeClient.DetachVolume(volumeID)

// Attach to test VM
attachOp, err := volumeClient.AttachVolume(volumeID, testVMID)
```

## 🚨 **EMERGENCY RECOVERY**

### **Rollback Capability**

The enhanced failover system now provides complete emergency recovery through:

1. **Snapshot ID Tracking**: All CloudStack snapshots recorded in `failover_jobs.ossea_snapshot_id`
2. **Database Audit Trail**: Complete operation history in `job_tracking` tables
3. **Correlation IDs**: All operations linked for traceability
4. **Native Rollback**: CloudStack `RevertVolumeSnapshot` API for instant recovery

### **Emergency Recovery Process**

```sql
-- 1. Find the CloudStack snapshot ID
SELECT ossea_snapshot_id FROM failover_jobs WHERE vm_id = '<vm-id>' ORDER BY created_at DESC LIMIT 1;

-- 2. Detach volume (if attached)
curl -X POST http://localhost:8090/api/v1/volumes/<volume-id>/detach

-- 3. Rollback CloudStack volume snapshot (via Go API)
// osseaClient.RevertVolumeSnapshot(snapshotID)

-- 4. Reattach to OMA
curl -X POST http://localhost:8090/api/v1/volumes/<volume-id>/attach -d '{"vm_id": "<oma-vm-id>"}'
```

## 📊 **VALIDATION & TESTING**

### **Successful Test Criteria**

1. ✅ **failover_jobs record created** with all metadata
2. ✅ **CloudStack volume snapshot created** and ID recorded in `ossea_snapshot_id`
3. ✅ **VirtIO injection completed** (for Windows VMs)
4. ✅ **Test VM created** with proper CloudStack configuration
5. ✅ **Volume attachment successful** via Volume Daemon
6. ✅ **Test VM boots** and is accessible

### **Monitoring Queries**

```sql
-- Check failover progress
SELECT job_id, vm_id, status, linstor_snapshot_name, created_at 
FROM failover_jobs 
WHERE vm_id = '<vm-id>' 
ORDER BY created_at DESC LIMIT 1;

-- Check job tracking
SELECT operation, status, error_message 
FROM job_tracking 
WHERE metadata LIKE '%<vm-id>%' 
ORDER BY created_at DESC LIMIT 10;
```

## 🏆 **VERSION HISTORY**

- **v2.4.9**: Fixed critical failover_jobs integration
- **v2.4.10**: Fixed CloudStack VM creation parameters 
- **v1.2.0**: Initial JobLog integration (simulation code)
- **v1.2.1**: Real implementation replacing simulation
- **v1.2.2**: Fixed VirtIO injection path and Volume Daemon integration
- **v1.2.3**: Added CloudStack VM name sanitization
- **v1.3.0**: CloudStack volume snapshot integration (replaces Linstor)
- **v1.3.1**: Clean architecture - removed all deprecated code ⭐ **CURRENT**

## ⚠️ **CRITICAL RULES** ⭐ **UPDATED SEPTEMBER 2025**

1. **ONLY Enhanced Failover**: Never use original test_failover.go/live_failover.go (DEPRECATED)
2. **JobLog Mandatory**: ALL operations MUST use `internal/joblog` system for tracking
3. **CloudStack Snapshots Only**: Use CloudStack volume snapshots, NOT Linstor (migrated)
4. **Volume Daemon Mandatory**: All volume operations via localhost:8090 
5. **Database Integration**: Both `failover_jobs` AND `job_tracking` tables MUST be updated
6. **Snapshot Field**: Use `ossea_snapshot_id` field, NOT `linstor_snapshot_name` (deprecated)
7. **VM Naming**: CloudStack naming rules automatically enforced via sanitization
8. **No Simulation Code**: Only real implementation allowed (project rule compliance)
9. **Clean Architecture**: No deprecated stub code or engine references allowed

---

**This documentation represents the definitive guide for the Enhanced Failover system as of September 2025.**
