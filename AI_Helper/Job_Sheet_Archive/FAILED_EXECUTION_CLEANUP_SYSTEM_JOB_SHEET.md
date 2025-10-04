# 🧹 **FAILED EXECUTION CLEANUP SYSTEM JOB SHEET**

**Created**: September 27, 2025  
**Priority**: 🔥 **CRITICAL** - Essential for production reliability  
**Issue ID**: FAILED-CLEANUP-001  
**Status**: 📋 **DESIGN PHASE** - Comprehensive failure recovery system

---

## 🎯 **EXECUTIVE SUMMARY**

**Problem**: When failover/rollback operations fail mid-process (like current VirtIO injection hang), the system is left in an inconsistent state with orphaned snapshots, detached volumes, and stuck database records.

**Solution**: Comprehensive "Failed Execution Cleanup" system that provides one-click recovery from any failed failover/rollback operation, returning the VM to clean "ready_for_failover" state.

**Business Value**: 
- ✅ **Operational Reliability**: Clean recovery from any failure scenario
- ✅ **User Experience**: Clear "Cleanup Failed Job" button for stuck operations
- ✅ **Resource Management**: Eliminates orphaned snapshots and volume inconsistencies
- ✅ **Production Ready**: Professional failure handling for enterprise environments

---

## 🏗️ **COMPREHENSIVE CLEANUP WORKFLOW**

### **🔧 Failed Execution Cleanup Process**

#### **Phase 1: Volume Operations (Via Volume Daemon)**
```go
// 1. Detach volumes from OMA (if attached)
for _, volume := range failedJob.Volumes {
    err := volumeClient.DetachVolume(ctx, volume.VolumeID)
    if err != nil {
        logger.Warn("Volume already detached or detach failed", "volume_id", volume.VolumeID)
    }
}
```

#### **Phase 2: Snapshot Rollback and Cleanup**
```go
// 2. Rollback snapshots to original state
for _, volume := range failedJob.Volumes {
    if volume.SnapshotID != "" {
        // Rollback to snapshot
        err := osseaClient.RevertVolumeToSnapshot(ctx, volume.VolumeID, volume.SnapshotID)
        if err != nil {
            logger.Error("Failed to rollback snapshot", "volume_id", volume.VolumeID, "snapshot_id", volume.SnapshotID)
        }
        
        // Delete snapshot after rollback
        err = osseaClient.DeleteSnapshot(ctx, volume.SnapshotID)
        if err != nil {
            logger.Warn("Failed to delete snapshot", "snapshot_id", volume.SnapshotID)
        }
    }
}
```

#### **Phase 3: Volume Reattachment (Via Volume Daemon)**
```go
// 3. Reattach volumes to OMA
const omaVMID = "8a4400e5-c92a-4bc4-8bff-4b6b0b6a018c"
for _, volume := range failedJob.Volumes {
    err := volumeClient.AttachVolume(ctx, volume.VolumeID, omaVMID)
    if err != nil {
        return fmt.Errorf("failed to reattach volume %s to OMA: %w", volume.VolumeID, err)
    }
}
```

#### **Phase 4: Test VM Cleanup**
```go
// 4. Delete test VM if it was created
if failedJob.TestVMID != "" {
    err := osseaClient.DeleteVM(ctx, failedJob.TestVMID)
    if err != nil {
        logger.Warn("Failed to delete test VM", "vm_id", failedJob.TestVMID)
    }
}
```

#### **Phase 5: Database Cleanup**
```go
// 5. Clean up database records
// Mark failover job as failed
err := db.UpdateFailoverJobStatus(ctx, failedJob.JobID, "failed")

// Clean up job tracking records
err = db.CleanupJobTrackingRecords(ctx, failedJob.JobID)

// Reset VM context to ready_for_failover
err = db.UpdateVMContextStatus(ctx, failedJob.VMContextID, "ready_for_failover")
```

---

## 🎨 **USER INTERFACE INTEGRATION**

### **🔴 Failed Job Detection**

#### **GUI Enhancement:**
```typescript
// Add to VM cards when failover/rollback jobs are stuck
{vmHasFailedJob && (
  <div className="mt-2 p-2 bg-red-50 dark:bg-red-900/20 rounded border border-red-200 dark:border-red-800">
    <div className="flex items-center justify-between">
      <span className="text-sm text-red-700 dark:text-red-300">
        ⚠️ Failed Operation Detected
      </span>
      <Button 
        size="xs" 
        color="red" 
        onClick={() => handleFailedJobCleanup(vm.vm_name)}
      >
        Cleanup
      </Button>
    </div>
    <p className="text-xs text-red-600 dark:text-red-400 mt-1">
      {failedJob.operation} - Stuck since {failedJob.stuckTime}
    </p>
  </div>
)}
```

#### **Cleanup Confirmation Modal:**
```typescript
<Modal show={showCleanupModal} onClose={() => setShowCleanupModal(false)}>
  <Modal.Header>Failed Job Cleanup</Modal.Header>
  <Modal.Body>
    <div className="space-y-4">
      <Alert color="warning">
        <span>This will clean up the failed {failedJob.operation} operation for {vmName}.</span>
      </Alert>
      
      <div className="space-y-2 text-sm">
        <h4 className="font-medium">Cleanup will perform:</h4>
        <ul className="list-disc list-inside space-y-1 text-gray-600">
          <li>Detach volumes from OMA</li>
          <li>Rollback and delete snapshots</li>
          <li>Reattach volumes to OMA</li>
          <li>Delete test VM (if created)</li>
          <li>Reset VM to ready_for_failover state</li>
        </ul>
      </div>
    </div>
  </Modal.Body>
  <Modal.Footer>
    <Button onClick={executeCleanup} color="red">
      Execute Cleanup
    </Button>
    <Button onClick={() => setShowCleanupModal(false)} color="gray">
      Cancel
    </Button>
  </Modal.Footer>
</Modal>
```

---

## 🔧 **TECHNICAL IMPLEMENTATION**

### **📋 Backend Service: FailedExecutionCleanupService**

#### **File: `source/current/oma/services/failed_execution_cleanup_service.go`**
```go
type FailedExecutionCleanupService struct {
    db           *database.Connection
    volumeClient *common.VolumeClient
    osseaClient  *ossea.Client
    jobTracker   *joblog.Tracker
}

func (fecs *FailedExecutionCleanupService) CleanupFailedExecution(ctx context.Context, vmName string) error {
    // 1. Identify failed job
    failedJob, err := fecs.identifyFailedJob(ctx, vmName)
    if err != nil {
        return fmt.Errorf("failed to identify failed job: %w", err)
    }
    
    // 2. Execute cleanup workflow
    return fecs.executeCleanupWorkflow(ctx, failedJob)
}

func (fecs *FailedExecutionCleanupService) executeCleanupWorkflow(ctx context.Context, job *FailedJob) error {
    jobID, err := fecs.jobTracker.StartJob(ctx, joblog.JobStart{
        JobType:   "cleanup",
        Operation: "failed-execution-cleanup",
        Owner:     "system",
    })
    if err != nil {
        return fmt.Errorf("failed to start cleanup job: %w", err)
    }
    
    // Phase 1: Volume detachment
    err = fecs.jobTracker.RunStep(ctx, jobID, "detach-volumes", func(ctx context.Context) error {
        return fecs.detachVolumesFromOMA(ctx, job)
    })
    if err != nil {
        return fmt.Errorf("volume detachment failed: %w", err)
    }
    
    // Phase 2: Snapshot rollback and cleanup
    err = fecs.jobTracker.RunStep(ctx, jobID, "cleanup-snapshots", func(ctx context.Context) error {
        return fecs.cleanupSnapshots(ctx, job)
    })
    if err != nil {
        return fmt.Errorf("snapshot cleanup failed: %w", err)
    }
    
    // Phase 3: Volume reattachment
    err = fecs.jobTracker.RunStep(ctx, jobID, "reattach-volumes", func(ctx context.Context) error {
        return fecs.reattachVolumesToOMA(ctx, job)
    })
    if err != nil {
        return fmt.Errorf("volume reattachment failed: %w", err)
    }
    
    // Phase 4: Test VM cleanup
    err = fecs.jobTracker.RunStep(ctx, jobID, "cleanup-test-vm", func(ctx context.Context) error {
        return fecs.cleanupTestVM(ctx, job)
    })
    if err != nil {
        return fmt.Errorf("test VM cleanup failed: %w", err)
    }
    
    // Phase 5: Database cleanup
    err = fecs.jobTracker.RunStep(ctx, jobID, "database-cleanup", func(ctx context.Context) error {
        return fecs.cleanupDatabaseRecords(ctx, job)
    })
    if err != nil {
        return fmt.Errorf("database cleanup failed: %w", err)
    }
    
    fecs.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
    return nil
}
```

### **📋 API Endpoint**
```go
// POST /api/v1/failover/{vm_name}/cleanup-failed
func (fh *FailoverHandler) CleanupFailedExecution(w http.ResponseWriter, r *http.Request) {
    vmName := mux.Vars(r)["vm_name"]
    
    err := fh.cleanupService.CleanupFailedExecution(r.Context(), vmName)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "message": fmt.Sprintf("Failed execution cleanup completed for %s", vmName),
    })
}
```

---

## 🎯 **IMMEDIATE APPLICATION**

### **For Current Stuck Jobs:**
1. **Implement cleanup service** (30 minutes)
2. **Add GUI cleanup buttons** (15 minutes)  
3. **Execute cleanup** on all 5 stuck test failovers
4. **Verify all VMs** return to ready_for_failover state

### **Expected Result:**
- ✅ **All 5 VMs**: Clean ready_for_failover state
- ✅ **No orphaned snapshots**: Proper cleanup completed
- ✅ **Volume consistency**: All volumes properly attached to OMA
- ✅ **Future protection**: System ready for any failure scenario

**This creates a production-ready failure recovery system that handles any stuck operation scenario!**

**Should I implement the Failed Execution Cleanup system?** 🧹🚀






