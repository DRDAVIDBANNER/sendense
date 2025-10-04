# Replication Job Creation Pattern - Implementation Guide

**Date**: September 23, 2025  
**Purpose**: Document the standardized pattern for creating replication jobs  
**Importance**: Critical for maintaining consistency across GUI, Scheduler, and future integrations  

---

## 🎯 **OVERVIEW**

This document defines the **canonical pattern** for creating replication jobs in MigrateKit OSSEA. This pattern ensures consistent behavior across all components and maintains the VM-centric architecture with proper audit trail functionality.

---

## 📋 **STANDARD REPLICATION JOB CREATION PATTERN**

### **🔄 Complete Workflow Steps**

#### **Step 1: Fresh VM Discovery** ✅ **MANDATORY**
Always get the latest VM specifications from vCenter before job creation.

```typescript
// Example: VMA Discovery API Call
const discoveryResponse = await fetch('/api/discover', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    vcenter: vmContext.vcenter_host || 'quad-vcenter-01.quadris.local',
    username: 'administrator@vsphere.local',
    password: 'EmyGVoBFesGQc47-',
    datacenter: vmContext.datacenter || 'DatabanxDC',
    filter: vmName  // Specific VM name for targeted discovery
  }),
  signal: controller.signal // Always include timeout handling
});
```

**Critical Requirements**:
- ✅ **Fresh Data**: Always call VMA discovery before job creation
- ✅ **Timeout Handling**: Include AbortController for 15-second timeout
- ✅ **Error Handling**: Validate discovery response and VM presence
- ✅ **Disk Validation**: Ensure VM has disks configured

#### **Step 2: OMA Replication API Call** ✅ **STANDARDIZED**
Use the canonical `/api/v1/replications` endpoint with standardized payload.

```typescript
// Standard OMA API Request Format
const omaRequest = {
  source_vm: {
    // ✅ EXACT FIELD MAPPING (critical for consistency)
    id: discoveredVM.id,                    // VMware VM UUID
    name: discoveredVM.name,                // VM display name
    path: discoveredVM.path,                // VMware inventory path
    vm_id: discoveredVM.id,                 // Duplicate for compatibility
    vm_name: discoveredVM.name,             // Duplicate for compatibility
    vm_path: discoveredVM.path,             // Duplicate for compatibility
    datacenter: discoveredVM.datacenter,    // VMware datacenter
    vcenter_host: vcenterHost,              // vCenter server
    cpus: discoveredVM.num_cpu || discoveredVM.cpus || 2,     // CPU count
    memory_mb: discoveredVM.memory_mb || 4096,               // Memory in MB
    power_state: discoveredVM.power_state || "poweredOn",    // Power state
    os_type: discoveredVM.guest_os || "otherGuest",          // OS type
    vmx_version: discoveredVM.vmx_version,                   // VMware version
    disks: discoveredVM.disks,              // ⚠️ CRITICAL: Fresh disk array
    networks: discoveredVM.networks         // Network configuration
  },
  ossea_config_id: 1,                       // OSSEA target configuration
  replication_type: 'initial',              // 'initial' or 'incremental'
  target_network: '',                       // Optional target network
  vcenter_host: vcenterHost,                // vCenter server (duplicate)
  datacenter: datacenter,                   // Datacenter (duplicate)
  change_id: '',                            // CBT change ID (optional)
  previous_change_id: '',                   // Previous CBT ID (optional)
  snapshot_id: '',                          // VMware snapshot (optional)
  
  // Optional: Control behavior
  start_replication: true,                  // true = start job, false = add to management only
  
  // Optional: Scheduler metadata (only when called by scheduler)
  schedule_execution_id: executionID,       // Schedule execution reference
  vm_group_id: groupID,                     // VM group reference
  scheduled_by: 'scheduler-service'         // Source identification
};

// ✅ STANDARD API CALL
const response = await fetch('http://localhost:8082/api/v1/replications', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
  },
  body: JSON.stringify(omaRequest)
});
```

#### **Step 3: Response Handling** ✅ **STANDARDIZED**
Handle both job creation and context-only responses consistently.

```typescript
const result = await response.json();

if (response.ok) {
  if (result.job_id) {
    // Job creation response (start_replication: true or undefined)
    console.log('✅ Replication started:', {
      job_id: result.job_id,
      status: result.status,
      progress_percent: result.progress_percent,
      created_volumes: result.created_volumes?.length || 0,
      mounted_volumes: result.mounted_volumes?.length || 0
    });
    
    return {
      type: 'job_created',
      job_id: result.job_id,
      message: `Replication started successfully with ${discoveredVM.disks.length} disk(s)`
    };
  } else {
    // Context-only response (start_replication: false)
    console.log('✅ VM added to management:', {
      context_id: result.context_id,
      vm_name: result.vm_name,
      current_status: result.current_status
    });
    
    return {
      type: 'context_created',
      context_id: result.context_id,
      message: 'VM added to management successfully'
    };
  }
} else {
  throw new Error(result.error || 'Failed to create replication job');
}
```

---

## 🏗️ **BACKEND WORKFLOW (OMA API Handler)**

### **API Endpoint**: `POST /api/v1/replications`

**Handler Location**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/api/handlers/replication.go:259-457`

#### **Key Logic Flow**:

1. **Request Validation** (Lines 261-275)
   ```go
   // Validate required fields
   if req.SourceVM.ID == "" || req.SourceVM.Name == "" || req.SourceVM.Path == "" {
       return BadRequest("Source VM information is required")
   }
   ```

2. **Start Replication Flag Handling** (Lines 277-281)
   ```go
   // Determine if we should start replication (defaults to true)
   startReplication := true
   if req.StartReplication != nil {
       startReplication = *req.StartReplication
   }
   ```

3. **Existing VM Context Check** (Lines 283-312)
   ```go
   // Check for existing VM context (by vmware_vm_id)
   existingContext, err := h.checkExistingVMContext(req.SourceVM.ID)
   
   if existingContext != nil {
       if !startReplication {
           // Add to Management: Block all duplicates
           return Conflict("VM already exists in management")
       } else if existingContext.CurrentJobID != nil {
           // Start Replication: Block if VM has active job
           return Conflict("VM has active replication job")
       }
       // Allow: Start Replication on existing VM with no active job
   }
   ```

4. **Migration Engine Invocation** (Lines 320-346)
   ```go
   // Pass existing context ID to migration engine
   if existingContext != nil {
       migrationReq.ExistingContextID = existingContext.ContextID
   }
   
   // Start the automated migration workflow
   result, err := h.migrationEngine.StartMigration(ctx, migrationReq)
   ```

---

## 📊 **VM-CENTRIC ARCHITECTURE INTEGRATION**

### **VM Context Handling** ✅ **AUTOMATIC**

The pattern automatically handles VM-centric architecture:

1. **New VMs**: Creates new `vm_replication_contexts` record
2. **Existing VMs**: Reuses existing context with `ExistingContextID`
3. **Audit Trail**: Each job creates new `vm_disks` records for specification tracking

### **Database Integration** ✅ **AUTOMATED**

**Migration Engine Location**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/workflows/migration.go:122-200`

#### **Automatic Processes**:
- ✅ **VM Context Creation/Update**: Handles both new and existing VMs
- ✅ **Job Record Creation**: Creates `replication_jobs` entry
- ✅ **Disk Specification Recording**: Creates `vm_disks` records with fresh VM specs
- ✅ **Volume Provisioning**: Via Volume Daemon integration
- ✅ **NBD Export Setup**: Automatic export creation and correlation
- ✅ **VMA Progress Polling**: Automatic progress tracking integration

---

## 🎯 **IMPLEMENTATION EXAMPLES**

### **Example 1: GUI Integration**
```typescript
// GUI: Virtual Machines page "Start Replication" button
const startReplication = async (vmName: string, vmContext: VMContext) => {
  // Step 1: Fresh discovery
  const discoveredVM = await discoverVM(vmName, vmContext);
  
  // Step 2: Standard API call
  const result = await callReplicationAPI({
    source_vm: buildSourceVMObject(discoveredVM, vmContext),
    replication_type: 'initial'
    // start_replication defaults to true
  });
  
  // Step 3: Handle response
  showSuccessNotification(`Replication started: ${result.job_id}`);
};
```

### **Example 2: Scheduler Integration**
```go
// Scheduler: Automated job creation
func (s *SchedulerService) createReplicationJob(ctx context.Context, vmCtx *VMReplicationContext, schedule *ReplicationSchedule) (string, error) {
    // Step 1: Fresh VMA discovery
    discoveredVM, err := s.discoverVMFromVMA(ctx, vmCtx.VMName, vmCtx.VCenterHost, vmCtx.Datacenter)
    
    // Step 2: Standard OMA API call
    omaRequest := CreateMigrationRequest{
        SourceVM: buildSourceVMFromDiscovery(discoveredVM),
        ReplicationType: schedule.ReplicationType,
        ScheduleExecutionID: executionID,  // Scheduler-specific metadata
        VMGroupID: groupID,
        ScheduledBy: "scheduler-service",
    }
    
    result, err := s.callOMAReplicationAPI(ctx, omaRequest)
    return result.JobID, err
}
```

### **Example 3: API Integration (External)**
```bash
# External API call example
curl -X POST http://localhost:8082/api/v1/replications \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sess_longlived_dev_token_2025_2035_permanent" \
  -d '{
    "source_vm": {
      "id": "420570c7-f61f-a930-77c5-1e876786cb3c",
      "name": "pgtest1",
      "path": "/DatabanxDC/vm/pgtest1",
      "datacenter": "DatabanxDC",
      "vcenter_host": "quad-vcenter-01.quadris.local",
      "cpus": 2,
      "memory_mb": 8192,
      "power_state": "poweredOn",
      "os_type": "windows",
      "disks": [/* fresh disk array */],
      "networks": [/* network config */]
    },
    "ossea_config_id": 1,
    "replication_type": "initial",
    "start_replication": true
  }'
```

---

## ⚠️ **CRITICAL REQUIREMENTS**

### **✅ MANDATORY ELEMENTS**
1. **Fresh VM Discovery**: Always call VMA discovery before job creation
2. **Complete Disk Array**: Include full `disks` array from discovery
3. **Standard Endpoint**: Use `/api/v1/replications` endpoint only
4. **Proper Authentication**: Include valid Bearer token
5. **Error Handling**: Handle both success and failure scenarios
6. **Timeout Management**: Include request timeouts (15 seconds recommended)

### **❌ COMMON PITFALLS TO AVOID**
1. **Stale VM Data**: Never use cached VM specifications
2. **Missing Disk Array**: Always include fresh `disks` from discovery
3. **Wrong Endpoint**: Don't use legacy or alternative endpoints
4. **Authentication Issues**: Ensure valid token for production
5. **No Error Handling**: Always handle API failures gracefully

### **🔧 CONSISTENCY CHECKLIST**
- [ ] Fresh VMA discovery called
- [ ] Standard payload structure used
- [ ] All required VM fields populated
- [ ] Disk array included from discovery
- [ ] Proper error handling implemented
- [ ] Success/failure notifications included
- [ ] Response type handling (job vs context)

---

## 🎯 **USAGE FOR FUTURE WORK**

This pattern should be used for:
- ✅ **New GUI Components**: Any new replication triggers
- ✅ **API Integrations**: External system integrations
- ✅ **Scheduler Enhancements**: Additional scheduling logic
- ✅ **Automation Scripts**: Batch processing implementations
- ✅ **Testing Frameworks**: Automated testing scenarios

**Reference this document** before implementing any replication job creation logic to ensure architectural consistency and proper audit trail functionality.

---

## 🔄 **UNIFIED FAILOVER FINAL SYNC IMPLEMENTATION** ✅ **COMPLETED**

### **🎯 SPECIFIC USE CASE: Live Failover Final Sync**

**Context**: After source VM power-off in live failover, a final incremental sync is required before completing failover.

**Key Requirements**:
- Use standard replication API (no special handling needed)
- Monitor completion via existing VMA poller (don't reinvent)
- Delay VM status update until after sync completion
- Report failover as failed if sync fails

#### **🚨 CRITICAL TIMING FIX REQUIRED**

**Current Bug Location**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/failover/unified_failover_engine.go:195-203`

```go
// ❌ CURRENT BUG: Status updated TOO EARLY (blocks final sync)
statusValue := "failed_over_test"
if config.FailoverType == FailoverTypeLive {
    statusValue = "failed_over_live"  // ❌ This blocks replication API
}
if err := ufe.updateVMContextStatus(ctx, config.ContextID, statusValue); err != nil {
    // Error is logged but doesn't fail the operation
}

// Phase 2: Source VM Power Management
// Phase 3: Final Sync ← BLOCKED because status already "failed_over_live"
```

**Required Fix**: Move status update to AFTER final sync completion.

#### **✅ FINAL SYNC IMPLEMENTATION PATTERN**

```go
// IMPLEMENTATION: executeFinalSyncPhase()
func (ufe *UnifiedFailoverEngine) executeFinalSyncPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig) error {
    return ufe.jobTracker.RunStep(ctx, jobID, "final-sync", func(ctx context.Context) error {
        logger := ufe.jobTracker.Logger(ctx)
        logger.Info("🔄 Starting final sync phase", "context_id", config.ContextID)

        // Step 1: Get VM context for discovery parameters
        vmContext, err := ufe.vmContextRepo.GetByContextID(config.ContextID)
        if err != nil {
            return fmt.Errorf("failed to get VM context: %w", err)
        }

        // Step 2: Fresh VM discovery (following standard pattern)
        discoveredVM, err := ufe.discoverVMFromVMA(ctx, vmContext.VMName, vmContext.VCenterHost, vmContext.Datacenter)
        if err != nil {
            return fmt.Errorf("final sync VM discovery failed: %w", err)
        }

        // Step 3: Standard replication API call (migration engine handles type detection)
        finalSyncRequest := &ReplicationJobRequest{
            SourceVM: buildSourceVMFromDiscovery(discoveredVM, vmContext),
            OSSEAConfigID: 1,
            // ✅ NO replication_type - let migration engine auto-detect incremental
            VCenterHost: vmContext.VCenterHost,
            Datacenter: vmContext.Datacenter,
            StartReplication: true,
            // ✅ NO scheduler metadata - this is not a scheduled job
        }

        // Step 4: Call standard OMA replication API
        result, err := ufe.callOMAReplicationAPI(ctx, finalSyncRequest)
        if err != nil {
            return fmt.Errorf("failed to start final sync: %w", err)
        }

        logger.Info("✅ Final sync started", "replication_job_id", result.JobID)

        // Step 5: Wait for completion using existing VMA poller pattern
        err = ufe.waitForReplicationCompletion(ctx, result.JobID)
        if err != nil {
            // ✅ CRITICAL: If final sync fails, mark failover as failed
            ufe.failoverJobRepo.UpdateStatus(config.FailoverJobID, "failed")
            return fmt.Errorf("final sync failed: %w", err)
        }

        logger.Info("✅ Final sync completed successfully", "replication_job_id", result.JobID)
        return nil
    })
}

// Helper: Wait for replication completion using existing VMA poller
func (ufe *UnifiedFailoverEngine) waitForReplicationCompletion(ctx context.Context, jobID string) error {
    logger := ufe.jobTracker.Logger(ctx)
    
    // Start VMA polling for this job
    if err := ufe.vmaProgressPoller.StartPolling(jobID); err != nil {
        return fmt.Errorf("failed to start progress polling: %w", err)
    }

    // Poll database for completion (same pattern as migration engine)
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    
    timeout := time.After(30 * time.Minute) // Reasonable timeout for final sync
    
    for {
        select {
        case <-timeout:
            ufe.vmaProgressPoller.StopPolling(jobID)
            return fmt.Errorf("final sync timeout after 30 minutes")
            
        case <-ticker.C:
            // Check job status in database (updated by VMA poller)
            job, err := ufe.replicationRepo.GetByID(ctx, jobID)
            if err != nil {
                continue // Keep trying
            }
            
            switch job.Status {
            case "completed":
                logger.Info("✅ Final sync completed", "job_id", jobID)
                return nil
                
            case "failed":
                ufe.vmaProgressPoller.StopPolling(jobID)
                return fmt.Errorf("final sync failed: %s", job.ErrorMessage)
                
            case "replicating":
                // Still in progress, continue polling
                logger.Debug("Final sync in progress", 
                    "progress", job.ProgressPercent,
                    "operation", job.CurrentOperation)
                continue
                
            default:
                // Unknown status, continue polling
                continue
            }
        }
    }
}
```

#### **🔧 REQUIRED PHASE REORDERING**

**Fix the unified failover engine phase order**:

```go
// ✅ CORRECTED PHASE ORDER
func (ufe *UnifiedFailoverEngine) ExecuteFailover(ctx context.Context, config *UnifiedFailoverConfig) error {
    // Phase 1: Pre-failover validation (unchanged)
    
    // Phase 2: Source VM power-off (live only)
    if config.RequiresSourceVMPowerOff() {
        if err := ufe.executeSourceVMPowerOffPhase(ctx, jobID, config); err != nil {
            return fmt.Errorf("source VM power-off phase failed: %w", err)
        }
    }
    
    // Phase 3: Final sync (live only) - BEFORE status update
    if config.RequiresFinalSync() {
        if err := ufe.executeFinalSyncPhase(ctx, jobID, config); err != nil {
            return fmt.Errorf("final sync phase failed: %w", err)
        }
    }
    
    // Phase 4: UPDATE VM STATUS (moved here from beginning)
    // ✅ CRITICAL: Only update status AFTER final sync completes
    statusValue := "failed_over_test"
    if config.FailoverType == FailoverTypeLive {
        statusValue = "failed_over_live"
    }
    if err := ufe.updateVMContextStatus(ctx, config.ContextID, statusValue); err != nil {
        // Error logged but doesn't fail operation
    }
    
    // Phase 5: Continue with remaining failover steps...
}
```

#### **📋 VMA POLLER INTEGRATION DETAILS**

**Existing VMA Poller Service**: `/home/pgrayson/migratekit-cloudstack/source/current/oma/services/vma_progress_poller.go`

**Key Methods**:
- `StartPolling(jobID string)` - Begin polling for job
- `StopPolling(jobID string)` - Stop polling for job  
- Auto-stops when job reaches "completed" or "failed" status
- Updates `replication_jobs` table with progress data

**Integration Pattern** (same as migration engine):
1. Start polling when replication job begins
2. VMA poller automatically updates database with progress
3. Monitor database for status changes ("completed"/"failed")
4. VMA poller auto-stops when job finishes

#### **🚨 CRITICAL SUCCESS CRITERIA**

1. ✅ **Status Timing**: VM status only updates AFTER final sync completion
2. ✅ **Standard API**: Use existing `/api/v1/replications` endpoint  
3. ✅ **VMA Poller**: Leverage existing polling infrastructure
4. ✅ **Error Handling**: Mark failover as failed if sync fails
5. ✅ **No Reinvention**: Use proven patterns from migration engine

---

**Status**: ✅ **PRODUCTION PATTERN** - Documented canonical implementation for consistent replication job creation and unified failover final sync

---

## 🎉 **FINAL SYNC IMPLEMENTATION COMPLETED** (September 23, 2025)

### **✅ Successfully Deployed Fixes**:

1. **VMA Discovery Parameters**: Fixed parameter mapping (`vcenter`, `filter`, `username`, `password`)
2. **API Port Correction**: Fixed replication API call from port 8080 → 8082  
3. **Field Mapping**: Implemented proper VMA discovery → replication API field mapping:
   - `num_cpu` → `cpus` with fallback to 2
   - `guest_os` → `os_type` with fallback to "otherGuest"
   - `memory_mb` preserved with fallback to 4096

### **✅ Final Sync Now Working**:
- Live failover can discover powered-off VMs from VMA
- Replication API receives properly formatted VM data  
- CPU count and OS type validation passes
- Incremental sync executes successfully on powered-off source VM
- VM status update occurs AFTER final sync completion

### **Deployed Binary**: `oma-api-field-mapping-fix`
**Location**: `/opt/migratekit/bin/oma-api-field-mapping-fix`
