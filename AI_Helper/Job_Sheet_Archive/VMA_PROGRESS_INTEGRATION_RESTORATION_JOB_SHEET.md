# 📊 **VMA PROGRESS INTEGRATION RESTORATION JOB SHEET**

**Created**: September 25, 2025  
**Priority**: 🚨 **CRITICAL** - Restore real-time progress tracking  
**Issue ID**: PROGRESS-RESTORATION-001  
**Status**: 📋 **PLANNING PHASE**

---

## 🎯 **EXECUTIVE SUMMARY**

**Problem**: VMA progress tracking was working 7 hours ago but has been completely lost during today's AI session. Real-time progress monitoring is broken - jobs show only "replicating" status with no progress updates.

**Solution**: Systematically rebuild VMA progress integration based on working log patterns and existing infrastructure, while preserving all current functionality (CBT auto-enablement, multi-volume snapshots).

**Success Criteria**: Restore the exact real-time progress tracking that was operational 7 hours ago showing live percentage, bytes transferred, and throughput data.

---

## 🔍 **EVIDENCE OF WORKING SYSTEM (7 HOURS AGO)**

### **Working Log Pattern**
```
time="2025-09-25T12:42:30Z" level=info msg="🎯 VMA progress tracking enabled" job_id=job-20250925-134230.513-8034ee vma_url="http://localhost:8081"
time="2025-09-25T12:42:30Z" level=info msg="🎯 Early progress tracking enabled - monitoring all migration phases" job_id=job-20250925-134230.513-8034ee
time="2025-09-25T12:43:23Z" level=debug msg="Sending progress update to VMA" bytes_transferred=65536 job_id=job-20250925-134230.513-8034ee percent=0.15 stage=Transfer throughput_bps=0
time="2025-09-25T12:43:23Z" level=debug msg="Progress update sent successfully to VMA" job_id=job-20250925-134230.513-8034ee
time="2025-09-25T12:43:23Z" level=debug msg="📊 Progress update sent to VMA" bytes_transferred=65536 disk="[vsanDatastore] 285ea568-64bc-07e9-4bc3-000af7864054/pgtest1-000007.vmdk" percent=0.15 throughput_bps=0 total_bytes=43253760 vm=pgtest1
```

### **Working Data Flow**
```
migratekit (libnbd callbacks) 
    ↓ Real-time progress updates (every 2 seconds)
VMA API (/api/v1/progress/{jobId}/update)
    ↓ Store in VMA Progress Service
VMA Progress Service (in-memory job data)
    ↓ OMA polls every 5 seconds
OMA VMA Progress Poller 
    ↓ Updates database
Database (replication_jobs table)
    ↓ GUI displays live progress
```

---

## 📋 **CURRENT SYSTEM STATUS**

### **✅ WORKING COMPONENTS**
1. **VMA Progress Service**: ✅ Confirmed working (manual test successful)
2. **VMA-OMA Tunnel**: ✅ OMA can reach VMA via localhost:9081
3. **Database Schema**: ✅ All progress fields exist
4. **CBT Auto-Enablement**: ✅ Working (deployed today)
5. **Multi-Volume Snapshots**: ✅ Available for testing

### **❌ BROKEN COMPONENTS**
1. **migratekit VMA Progress Integration**: ❌ Not sending progress updates
2. **OMA VMA Progress Poller**: ❌ Timing issues, stops immediately
3. **Progress Data Flow**: ❌ No data reaching database

### **🎯 CURRENT DEPLOYMENT STATUS**
- **migratekit**: `migratekit-v2.14.0-working-progress-plus-cbt` (has CBT, missing progress)
- **VMA API**: `vma-api-server-v1.10.4-progress-fixed` (working)
- **OMA API**: `oma-api-v2.20.0-working-poller-restored` (restored to fb8768d)
- **Volume Daemon**: `volume-daemon-v1.2.3-multi-volume-snapshots` (enhanced)

---

## 🔧 **IMPLEMENTATION PLAN**

### **📊 PHASE 1: VMA PROGRESS CLIENT INTEGRATION (CRITICAL)**
**Duration**: 60 minutes  
**Risk**: 🟡 **MEDIUM** - Modifying core migratekit functionality  
**Priority**: 🚨 **HIGHEST** - Nothing works without this

#### **Task 1.1: Implement VMA Progress Client Initialization**
**File**: `source/current/migratekit/main.go`  
**Status**: ⏳ **PENDING**

**Current Issue**: VMA progress client not initialized despite environment variable being set

**Required Implementation**:
```go
// In PersistentPreRunE, after setting MIGRATEKIT_PROGRESS_JOB_ID
if jobID != "" {
    os.Setenv("MIGRATEKIT_PROGRESS_JOB_ID", jobID)
    log.WithField("job_id", jobID).Info("Set progress tracking job ID from command line flag")
    
    // 🎯 CRITICAL: Initialize VMA progress client immediately
    vmaProgressClient := progress.NewVMAProgressClient()
    if vmaProgressClient.IsEnabled() {
        log.WithField("job_id", vmaProgressClient.GetJobID()).Info("🎯 VMA progress tracking enabled", "vma_url", "http://localhost:8081")
        log.Info("🎯 Early progress tracking enabled - monitoring all migration phases", "job_id", jobID)
        
        // Add to context for use throughout migration
        ctx = context.WithValue(ctx, "vmaProgressClient", vmaProgressClient)
        
        // Send initial progress update
        vmaProgressClient.SendStageUpdate("Initializing", 5)
    } else {
        log.Warn("❌ VMA progress tracking failed to initialize")
    }
}
```

#### **Task 1.2: Add VMA Progress Client Import**
**File**: `source/current/migratekit/main.go`  
**Status**: ⏳ **PENDING**

**Required Import**:
```go
import (
    // ... existing imports ...
    "github.com/vexxhost/migratekit/internal/progress"
    // ... rest of imports ...
)
```

#### **Task 1.3: Add Stage Progress Updates**
**File**: `source/current/migratekit/main.go`  
**Status**: ⏳ **PENDING**

**Required Updates**:
```go
// At key migration stages
if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
    if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
        vpc.SendStageUpdate("Creating Snapshot", 10)
        vpc.SendStageUpdate("Setting up NBD", 20) 
        vpc.SendStageUpdate("Preparing Migration", 30)
    }
}
```

### **📊 PHASE 2: LIBNBD PROGRESS CALLBACKS (CRITICAL)**
**Duration**: 90 minutes  
**Risk**: 🔴 **HIGH** - Modifying core data transfer logic  
**Priority**: 🚨 **HIGHEST** - Real-time progress depends on this

#### **Task 2.1: Analyze Current libnbd Integration**
**File**: `source/current/migratekit/internal/vmware_nbdkit/vmware_nbdkit.go`  
**Status**: ⏳ **PENDING**

**Investigation Required**:
- Find where libnbd data transfer happens
- Identify progress callback locations  
- Check if VMA progress client can be accessed from libnbd context

#### **Task 2.2: Add VMA Progress Callbacks to libnbd Operations**
**File**: `source/current/migratekit/internal/vmware_nbdkit/vmware_nbdkit.go`  
**Status**: ⏳ **PENDING**

**Target Pattern** (based on working logs):
```go
// In libnbd data transfer loop
totalBytesTransferred += chunkSize
progressPercent := (float64(totalBytesTransferred) / float64(totalBytes)) * 100

// Send VMA progress update every 2 seconds or 1% progress
if time.Since(lastProgressUpdate) >= 2*time.Second || progressPercent >= lastPercent+1.0 {
    if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
        if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
            // Calculate throughput
            throughputBPS := calculateThroughput(totalBytesTransferred, startTime)
            
            vpc.SendUpdate(progress.VMAProgressUpdate{
                Stage:            "Transfer",
                Status:           "in_progress",
                BytesTransferred: totalBytesTransferred,
                TotalBytes:       totalBytes,
                Percent:          progressPercent,
                ThroughputBPS:    throughputBPS,
                DiskID:           diskID,
            })
            
            log.Debug("📊 Progress update sent to VMA", 
                "bytes_transferred", totalBytesTransferred,
                "percent", progressPercent,
                "throughput_bps", throughputBPS,
                "total_bytes", totalBytes,
                "vm", vmName,
                "disk", diskPath)
        }
    }
    lastProgressUpdate = time.Now()
    lastPercent = progressPercent
}
```

#### **Task 2.3: Multi-Disk Progress Support**
**File**: `source/current/migratekit/internal/vmware_nbdkit/vmware_nbdkit.go`  
**Status**: ⏳ **PENDING**

**Required Enhancement**:
- Individual progress tracking per disk (`disk-2000`, `disk-2001`)
- Aggregate progress calculation across all disks
- Proper disk identification in VMA progress updates

### **📊 PHASE 3: OMA VMA PROGRESS POLLER FIXES (TIMING)**
**Duration**: 45 minutes  
**Risk**: 🟡 **MEDIUM** - Modifying polling logic  
**Priority**: 🔥 **HIGH** - Prevents premature polling stop

#### **Task 3.1: Add Startup Delay for VMA Progress Poller**
**File**: `source/current/oma/services/vma_progress_poller.go`  
**Status**: ⏳ **PENDING**

**Issue**: OMA starts polling **before** migratekit sends first progress update  
**Solution**: Add **initial delay** or **retry logic** when job not found initially

**Enhanced StartPolling Method**:
```go
func (vpp *VMAProgressPoller) StartPolling(jobID string) error {
    // ... existing logic ...
    
    // Create polling context with startup grace period
    pollingCtx := &PollingContext{
        JobID:             jobID,
        StartedAt:         time.Now(),
        MaxErrors:         5,
        StopChan:          make(chan struct{}),
        StartupGracePeriod: 30 * time.Second, // Wait 30 seconds before assuming failure
    }
    
    // ... rest of method ...
}
```

#### **Task 3.2: Enhanced Error Handling for Startup Phase**
**File**: `source/current/oma/services/vma_progress_poller.go`  
**Status**: ⏳ **PENDING**

**Enhanced handlePollingError**:
```go
func (vpp *VMAProgressPoller) handlePollingError(jobID string, pollingCtx *PollingContext, err error, logger *log.Entry) {
    pollingCtx.ConsecutiveErrors++

    // Check if it's a "job not found" error
    if vmaErr, ok := err.(*VMAProgressError); ok && vmaErr.StatusCode == 404 {
        jobAge := time.Since(pollingCtx.StartedAt)
        
        // During startup grace period, don't assume completion
        if jobAge < pollingCtx.StartupGracePeriod {
            logger.WithField("job_age", jobAge).Debug("Job not found during startup - waiting for migratekit to send first update")
            return // Continue polling during grace period
        }
        
        // After grace period, assume completion
        logger.Info("📋 Job not found in VMA after grace period - likely completed")
        vpp.StopPolling(jobID)
        return
    }
    
    // ... rest of error handling ...
}
```

### **📊 PHASE 4: INTEGRATION TESTING (VALIDATION)**
**Duration**: 30 minutes  
**Risk**: 🟢 **LOW** - Testing only  
**Priority**: 🔥 **HIGH** - Verify complete functionality

#### **Task 4.1: End-to-End Progress Tracking Test**
**VM**: pgtest2 (ready_for_failover)  
**Status**: ⏳ **PENDING**

**Expected Results**:
```
✅ migratekit: "🎯 VMA progress tracking enabled"
✅ migratekit: "📊 Progress update sent to VMA" (every 2 seconds)
✅ VMA: Progress data stored and retrievable
✅ OMA: Successful polling without premature stops
✅ Database: Live progress updates in replication_jobs
✅ GUI: Real-time progress display
```

#### **Task 4.2: CBT Auto-Enablement Validation**
**VM**: pgtest3 (has CBT disabled)  
**Status**: ⏳ **PENDING**

**Expected Results**:
```
✅ CBT Detection: "Change tracking is not enabled"
✅ CBT Enablement: "✅ CBT enabled successfully"
✅ Progress Tracking: Real-time updates during migration
✅ Migration Success: Complete data transfer
```

#### **Task 4.3: Multi-Volume Snapshot Testing**
**VM**: pgtest2 (2-disk VM, ready for failover)  
**Status**: ⏳ **PENDING**

**Expected Results**:
```
✅ Test Failover: Both disks protected with individual snapshots
✅ Snapshot Storage: device_mappings tracks all volume snapshots
✅ Rollback Capability: All volumes restored from snapshots
✅ Cleanup: Complete snapshot cleanup after testing
```

---

## 🚨 **CRITICAL PROJECT RULES COMPLIANCE**

### **✅ SOURCE CODE AUTHORITY**
- [ ] **ONLY modify code in `/source/current/`**
- [ ] **NO changes to archived or top-level code**
- [ ] **PRESERVE all existing functionality**
- [ ] **NO loss of CBT auto-enablement**
- [ ] **NO loss of multi-volume snapshot capability**

### **✅ TESTING PROTOCOL**
- [ ] **Test each phase incrementally**
- [ ] **Verify no regression in working features**
- [ ] **Document exact code changes made**
- [ ] **Create backup before each modification**

### **✅ DEPLOYMENT SAFETY**
- [ ] **Build and test locally first**
- [ ] **Deploy with explicit version numbers**
- [ ] **Verify service health after each deployment**
- [ ] **No deployment during active replications**

---

## 📊 **DETAILED IMPLEMENTATION TASKS**

### **🔧 TASK 1A: VMA Progress Client Integration**
**File**: `source/current/migratekit/main.go`  
**Lines**: Around 220 (after environment variable setting)  
**Duration**: 20 minutes  
**Status**: ⏳ **PENDING**

**Exact Code to Add**:
```go
// After: os.Setenv("MIGRATEKIT_PROGRESS_JOB_ID", jobID)
// Add: VMA progress client initialization

// 🎯 CRITICAL: Initialize VMA progress client for real-time tracking
vmaProgressClient := progress.NewVMAProgressClient()
if vmaProgressClient.IsEnabled() {
    log.WithFields(log.Fields{
        "job_id":  vmaProgressClient.GetJobID(),
        "vma_url": "http://localhost:8081",
    }).Info("🎯 VMA progress tracking enabled")
    log.WithField("job_id", jobID).Info("🎯 Early progress tracking enabled - monitoring all migration phases")
    
    // Add to context for use throughout migration
    ctx = context.WithValue(ctx, "vmaProgressClient", vmaProgressClient)
    
    // Send initial progress update
    vmaProgressClient.SendStageUpdate("Initializing", 5)
} else {
    log.Warn("❌ VMA progress tracking failed to initialize - check MIGRATEKIT_PROGRESS_JOB_ID")
}
```

**Required Import**:
```go
"github.com/vexxhost/migratekit/internal/progress"
```

**Validation**:
- [ ] Build succeeds
- [ ] migratekit logs show "🎯 VMA progress tracking enabled"
- [ ] Initial progress update sent to VMA

### **🔧 TASK 1B: Stage Progress Updates**
**File**: `source/current/migratekit/main.go`  
**Locations**: Snapshot creation, NBD setup stages  
**Duration**: 15 minutes  
**Status**: ⏳ **PENDING**

**Code Pattern**:
```go
// At each stage, add progress update
if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
    if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
        vpc.SendStageUpdate("Creating Snapshot", 10)
    }
}
```

**Stages to Add**:
- Creating Snapshot: 10%
- Setting up NBD: 20%  
- Preparing Migration: 30%
- Data Transfer: 0-100% (from libnbd callbacks)

### **🔧 TASK 2A: Find libnbd Data Transfer Location**
**File**: `source/current/migratekit/internal/vmware_nbdkit/vmware_nbdkit.go`  
**Duration**: 20 minutes  
**Status**: ⏳ **PENDING**

**Investigation Points**:
- [ ] Find `handle.Pread()` and `handle.Pwrite()` operations
- [ ] Identify data transfer loops with byte counting
- [ ] Check if context with VMA progress client is accessible
- [ ] Locate where throughput calculation should happen

**Expected Findings**:
```go
// Should find libnbd operations like:
handle, err := libnbd.Create()
err = handle.ConnectUri(exportName)
err = handle.Pread(buffer, offset, nil)
err = handle.Pwrite(buffer, offset, nil)
```

### **🔧 TASK 2B: Add VMA Progress Callbacks to libnbd**
**File**: `source/current/migratekit/internal/vmware_nbdkit/vmware_nbdkit.go`  
**Duration**: 45 minutes  
**Status**: ⏳ **PENDING**

**Implementation Pattern** (based on working logs):
```go
// In libnbd data transfer loop
var (
    totalBytesTransferred int64
    lastProgressUpdate = time.Now()
    lastProgressPercent = 0.0
    startTime = time.Now()
)

for {
    // Existing libnbd operations
    err = handle.Pread(buffer, offset, nil)
    err = handle.Pwrite(buffer, offset, nil)
    
    totalBytesTransferred += int64(len(buffer))
    
    // Send progress update every 2 seconds or 1% progress change
    currentPercent := (float64(totalBytesTransferred) / float64(totalBytes)) * 100
    timeSinceUpdate := time.Since(lastProgressUpdate)
    
    if timeSinceUpdate >= 2*time.Second || currentPercent >= lastProgressPercent+1.0 {
        if vmaProgressClient := ctx.Value("vmaProgressClient"); vmaProgressClient != nil {
            if vpc, ok := vmaProgressClient.(*progress.VMAProgressClient); ok && vpc.IsEnabled() {
                // Calculate throughput
                elapsed := time.Since(startTime).Seconds()
                throughputBPS := int64(float64(totalBytesTransferred) / elapsed)
                
                // Send progress update matching working log format
                vpc.SendUpdate(progress.VMAProgressUpdate{
                    Stage:            "Transfer",
                    Status:           "in_progress",
                    BytesTransferred: totalBytesTransferred,
                    TotalBytes:       totalBytes,
                    Percent:          currentPercent,
                    ThroughputBPS:    throughputBPS,
                    DiskID:           diskID,
                })
                
                // Log matching working pattern
                log.WithFields(log.Fields{
                    "bytes_transferred": totalBytesTransferred,
                    "percent":          currentPercent,
                    "throughput_bps":   throughputBPS,
                    "total_bytes":      totalBytes,
                    "vm":               vmName,
                    "disk":             diskPath,
                    "job_id":           jobID,
                }).Debug("📊 Progress update sent to VMA")
                
                lastProgressUpdate = time.Now()
                lastProgressPercent = currentPercent
            }
        }
    }
}
```

### **🔧 TASK 3A: OMA VMA Progress Poller Timing Fix**
**File**: `source/current/oma/services/vma_progress_poller.go`  
**Duration**: 25 minutes  
**Status**: ⏳ **PENDING**

**Issue**: OMA starts polling immediately, gets "job not found", stops polling  
**Solution**: Add startup grace period

**PollingContext Enhancement**:
```go
type PollingContext struct {
    JobID               string
    StartedAt           time.Time
    LastPoll            time.Time
    ConsecutiveErrors   int
    MaxErrors           int
    StopChan            chan struct{}
    StartupGracePeriod  time.Duration  // NEW: Grace period for job initialization
}
```

**Enhanced StartPolling**:
```go
pollingCtx := &PollingContext{
    JobID:               jobID,
    StartedAt:           time.Now(),
    MaxErrors:           5,
    StopChan:            make(chan struct{}),
    StartupGracePeriod:  30 * time.Second, // Wait 30 seconds before assuming failure
}
```

### **🔧 TASK 3B: Smart Error Handling**
**File**: `source/current/oma/services/vma_progress_poller.go`  
**Duration**: 20 minutes  
**Status**: ⏳ **PENDING**

**Enhanced handlePollingError**:
```go
if vmaErr, ok := err.(*VMAProgressError); ok && vmaErr.StatusCode == 404 {
    jobAge := time.Since(pollingCtx.StartedAt)
    
    if jobAge < pollingCtx.StartupGracePeriod {
        logger.WithField("job_age", jobAge).Debug("Job not found during startup grace period - continuing to poll")
        return // Continue polling, don't stop
    }
    
    logger.WithField("job_age", jobAge).Info("📋 Job not found after grace period - likely completed")
    vpp.StopPolling(jobID)
    return
}
```

---

## 🧪 **TESTING STRATEGY**

### **Test 1: VMA Progress Client Initialization**
**Duration**: 5 minutes  
**Status**: ⏳ **PENDING**

```bash
# Start any replication job
# Expected in migratekit logs:
- "Set progress tracking job ID from command line flag"
- "🎯 VMA progress tracking enabled" 
- "🎯 Early progress tracking enabled"
- "Sending progress update to VMA"
- "Progress update sent successfully to VMA"
```

### **Test 2: VMA Progress Service Data Storage**
**Duration**: 5 minutes  
**Status**: ⏳ **PENDING**

```bash
# During active replication, check VMA:
curl "http://localhost:8081/api/v1/progress/job-20250925-XXXXXX"

# Expected: JSON with progress data, NOT "job not found"
```

### **Test 3: OMA VMA Progress Poller**
**Duration**: 5 minutes  
**Status**: ⏳ **PENDING**

```bash
# Check OMA logs during replication:
sudo journalctl -u oma-api -f | grep "VMA.*progress"

# Expected: Continuous polling, NO immediate "Stopped VMA progress polling"
```

### **Test 4: Database Updates**
**Duration**: 5 minutes  
**Status**: ⏳ **PENDING**

```bash
# Check database during replication:
mysql -u oma_user -poma_password migratekit_oma -e "SELECT progress_percent, current_operation, bytes_transferred FROM replication_jobs WHERE id = 'job-20250925-XXXXXX';"

# Expected: Live updates, NOT stuck at 0%
```

---

## 📅 **IMPLEMENTATION TIMELINE**

| **Phase** | **Duration** | **Dependencies** | **Critical Path** |
|-----------|--------------|------------------|-------------------|
| **Phase 1**: VMA Progress Client | 35 min | None | ✅ **CRITICAL** |
| **Phase 2**: libnbd Callbacks | 65 min | Phase 1 complete | ✅ **CRITICAL** |
| **Phase 3**: OMA Poller Fix | 45 min | Phase 1 complete | 🔥 **HIGH** |
| **Phase 4**: Testing | 20 min | All phases complete | 🔥 **HIGH** |
| **Total** | **2.7 hours** | Sequential execution | **CRITICAL PATH** |

---

## 🎯 **SUCCESS METRICS**

### **Technical Metrics**
- [ ] ✅ **migratekit VMA Progress**: `"Progress update sent successfully to VMA"` every 2 seconds
- [ ] ✅ **VMA Progress Service**: Job data stored and retrievable via API
- [ ] ✅ **OMA Polling**: Continuous polling without premature stops
- [ ] ✅ **Database Updates**: Live `progress_percent`, `bytes_transferred`, `current_operation`
- [ ] ✅ **Real-Time Display**: GUI shows live progress (not stuck at "replicating")

### **Functional Metrics**
- [ ] ✅ **CBT Auto-Enablement**: Still working (no regression)
- [ ] ✅ **Multi-Volume Snapshots**: Still available (no regression)
- [ ] ✅ **Migration Success**: Jobs complete successfully
- [ ] ✅ **Progress Accuracy**: Data transfer percentages match actual progress

### **Operational Metrics**
- [ ] ✅ **No Stuck Jobs**: Jobs don't get stuck at "replicating"
- [ ] ✅ **Proper Completion**: Jobs properly marked as "completed"
- [ ] ✅ **Error Handling**: Failures properly detected and marked
- [ ] ✅ **Service Stability**: No service crashes or infinite loops

---

## 🔒 **SAFETY MEASURES**

### **Before Each Phase**
- [ ] **Git Commit**: Save current state before modifications
- [ ] **Service Health**: Verify all services healthy
- [ ] **Backup Binary**: Save current working binaries
- [ ] **Test Environment**: Ensure no active migrations

### **After Each Phase**
- [ ] **Build Test**: Verify builds succeed
- [ ] **Service Restart**: Test service restarts cleanly
- [ ] **Basic Function**: Verify core functionality works
- [ ] **No Regression**: Confirm existing features still work

### **Rollback Plan**
- [ ] **Git Revert**: `git checkout HEAD~1 -- <modified_files>`
- [ ] **Binary Rollback**: Restore previous working binaries
- [ ] **Service Restart**: Restart affected services
- [ ] **Status Verification**: Confirm system returns to working state

---

## 📋 **EXECUTION TRACKING**

### **Current Status**
- **Overall Progress**: 0% ⏳ **READY TO START**
- **Active Phase**: None
- **Blockers**: None
- **Ready for Execution**: ✅

### **Phase Completion Checklist**
- [ ] **Phase 1**: VMA Progress Client Integration
  - [ ] Task 1.1: Client initialization in main.go
  - [ ] Task 1.2: Add progress import
  - [ ] Task 1.3: Stage progress updates
- [ ] **Phase 2**: libnbd Progress Callbacks  
  - [ ] Task 2.1: Analyze current libnbd integration
  - [ ] Task 2.2: Add VMA progress callbacks
  - [ ] Task 2.3: Multi-disk progress support
- [ ] **Phase 3**: OMA Poller Timing Fix
  - [ ] Task 3.1: Add startup delay
  - [ ] Task 3.2: Enhanced error handling
- [ ] **Phase 4**: Integration Testing
  - [ ] Task 4.1: End-to-end progress test
  - [ ] Task 4.2: CBT validation
  - [ ] Task 4.3: Multi-volume snapshot test

---

## 🎉 **EXPECTED FINAL STATE**

**Upon completion, the system will have**:
1. ✅ **Complete Real-Time Progress**: Matching exactly what worked 7 hours ago
2. ✅ **CBT Auto-Enablement**: Functional for VMs without CBT
3. ✅ **Multi-Volume Snapshots**: Ready for failover testing
4. ✅ **Stable Services**: No more stuck jobs or polling issues
5. ✅ **Production Ready**: Enterprise-grade migration platform

**Business Impact**: Restore confidence in real-time monitoring while maintaining all enhancements made today.

---

**Status**: 📋 **READY FOR SYSTEMATIC EXECUTION** - Comprehensive plan to restore VMA progress integration without losing functionality

