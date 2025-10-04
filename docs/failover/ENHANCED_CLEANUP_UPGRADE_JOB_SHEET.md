# Enhanced Cleanup Service Upgrade Job Sheet

**Date**: 2025-09-03  
**Version**: v1.4.0 (CloudStack Snapshot Cleanup Integration)  
**Priority**: HIGH - Critical for complete failover lifecycle  
**Status**: PLANNING PHASE - DO NOT START IMPLEMENTATION YET

---

## 🎯 **OBJECTIVE**

Upgrade the Enhanced Cleanup Service to use:
1. **JobLog Integration** (consistency with enhanced failover)
2. **CloudStack Volume Snapshot Rollback** (reverse of snapshot creation in failover)
3. **Complete Database Tracking** (audit trail and status management)
4. **Zero Impact on Working Failover System** ⚠️ **CRITICAL CONSTRAINT**

---

## 🔒 **CRITICAL CONSTRAINTS & SAFETY RULES**

### ⚠️ **FAILOVER SYSTEM PROTECTION**
- **NO CHANGES** to `enhanced_test_failover.go` or `enhanced_live_failover.go`
- **NO CHANGES** to database schema or existing table fields
- **NO CHANGES** to Volume Daemon integration patterns
- **NO CHANGES** to API handler failover methods (only cleanup method)
- **NO CHANGES** to JobLog implementation or `internal/joblog` package

### 🛡️ **TESTING SAFETY**
- **BACKUP** existing cleanup service before changes
- **VERSION** all modified files with proper semantic versioning
- **TEST** with non-critical VMs first (NOT production failover VMs)
- **ROLLBACK PLAN** ready before starting any modifications

---

## 📊 **CURRENT STATE ANALYSIS**

### ✅ **What Works (DO NOT TOUCH)**
```go
// File: internal/oma/failover/enhanced_cleanup_service.go
- Volume Daemon integration (lines 27, 174-229, 315-382)
- CloudStack VM power management (lines 231-278)  
- CloudStack VM deletion (lines 280-313)
- Database VM discovery (lines 139-161)
- API handler integration (lines 684-717 in failover.go)
```

### ❌ **What Needs Upgrade**
```go
// 1. Constructor missing JobLog tracker
func NewEnhancedCleanupService(db *gorm.DB) // CURRENT
func NewEnhancedCleanupService(db *gorm.DB, jobTracker *joblog.Tracker) // REQUIRED

// 2. Old logging pattern (lines 35-48)
logger := logging.NewOperationLogger("enhanced-cleanup") // DEPRECATED
logger := ecs.jobTracker.Logger(ctx) // REQUIRED

// 3. Missing CloudStack snapshot rollback
// NO SNAPSHOT LOGIC EXISTS - need to add between VM shutdown and volume detach

// 4. No failover job status updates
// Need: status transitions during cleanup phases
```

---

## 🏗️ **IMPLEMENTATION PHASES**

### **PHASE 1: SAFE BACKUP & PREPARATION** ⏱️ 15 min
```bash
# 1.1 Archive current working version
cp internal/oma/failover/enhanced_cleanup_service.go \
   internal/oma/failover/enhanced_cleanup_service.go.backup-$(date +%Y%m%d)

# 1.2 Create versioned backup
cp cmd/oma/oma-api-clean-v1.3.1 cmd/oma/oma-api-cleanup-backup-v1.3.1

# 1.3 Verify current API is working
curl -X POST http://localhost:8082/api/v1/failover/cleanup/test-vm | jq .
```

**CRITICAL CHECKPOINTS:**
- [ ] Backup files created successfully
- [ ] Current cleanup API responds (even if fails, must respond)
- [ ] No active failover operations running

---

### **PHASE 2: JOBLOG INTEGRATION** ⏱️ 30 min

#### **2.1 Constructor Update** (NON-BREAKING)
```go
// File: internal/oma/failover/enhanced_cleanup_service.go

// BEFORE (line 18-22)
type EnhancedCleanupService struct {
	volumeClient *common.VolumeClient
	osseaClient  *ossea.Client
	db           *gorm.DB
}

// AFTER (maintaining backward compatibility)
type EnhancedCleanupService struct {
	jobTracker   *joblog.Tracker       // NEW: JobLog integration
	volumeClient *common.VolumeClient  // UNCHANGED
	osseaClient  *ossea.Client         // UNCHANGED  
	db           *gorm.DB              // UNCHANGED
}

// Constructor update (line 25-30)
func NewEnhancedCleanupService(db *gorm.DB, jobTracker *joblog.Tracker) *EnhancedCleanupService {
	return &EnhancedCleanupService{
		jobTracker:   jobTracker,         // NEW
		volumeClient: common.NewVolumeClient("http://localhost:8090"), // UNCHANGED
		db:           db,                 // UNCHANGED
	}
}
```

#### **2.2 API Handler Update** (SINGLE LINE CHANGE)
```go
// File: internal/oma/api/handlers/failover.go (line 43)

// BEFORE
enhancedCleanupService: failover.NewEnhancedCleanupService(db.GetGormDB()),

// AFTER  
enhancedCleanupService: failover.NewEnhancedCleanupService(db.GetGormDB(), jobTracker),
```

**SAFETY VERIFICATION:**
- [ ] Constructor signature updated
- [ ] API handler updated with JobLog tracker
- [ ] Compilation successful
- [ ] NO changes to failover engine constructors

---

### **PHASE 3: LOGGING SYSTEM UPGRADE** ⏱️ 45 min

#### **3.1 Main Method Transformation**
```go
// File: internal/oma/failover/enhanced_cleanup_service.go
// Method: ExecuteTestFailoverCleanupWithTracking (lines 33-136)

// BEFORE: Old centralized logging pattern
func (ecs *EnhancedCleanupService) ExecuteTestFailoverCleanupWithTracking(ctx context.Context, vmID string) (err error) {
	logger := logging.NewOperationLogger("enhanced-cleanup")
	opCtx := logger.StartOperation("test-failover-cleanup", vmID)
	defer func() { /* old pattern */ }()

// AFTER: JobLog pattern (EXACT COPY from enhanced_test_failover.go lines 159-177)
func (ecs *EnhancedCleanupService) ExecuteTestFailoverCleanupWithTracking(ctx context.Context, vmID string) (err error) {
	// START: Job creation with joblog (COPY PATTERN)
	ctx, jobID, err := ecs.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "cleanup",
		Operation: "enhanced-test-failover-cleanup",
		Owner:     stringPtr("system"),
		Metadata: map[string]interface{}{
			"vm_id":     vmID,
			"operation": "test-failover-cleanup",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start cleanup job: %w", err)
	}
	defer ecs.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)

	// Get logger with job context
	logger := ecs.jobTracker.Logger(ctx)
```

#### **3.2 Step-by-Step Method Conversion**
```go
// Convert each helper method to use JobLog RunStep pattern

// BEFORE: Old opCtx.LogStep pattern
err = ecs.ensureVMPoweredOff(ctx, opCtx, testVMID)

// AFTER: JobLog RunStep pattern
if err := ecs.jobTracker.RunStep(ctx, jobID, "test-vm-shutdown", func(ctx context.Context) error {
	return ecs.ensureVMPoweredOff(ctx, testVMID)
}); err != nil {
	return fmt.Errorf("test VM shutdown failed: %w", err)
}
```

**LOGGING CONVERSION MAP:**
- [ ] `ensureVMPoweredOff` → JobLog pattern (remove opCtx parameter)
- [ ] `detachVolumesFromTestVM` → JobLog pattern  
- [ ] `deleteTestVM` → JobLog pattern
- [ ] `reattachVolumesToOMA` → JobLog pattern
- [ ] All internal `opCtx.LogStep` → `logger.Info` calls

---

### **PHASE 4: CLOUDSTACK SNAPSHOT ROLLBACK** ⏱️ 60 min

#### **4.1 Database Snapshot Retrieval**
```go
// NEW METHOD: Get snapshot ID from failover job
func (ecs *EnhancedCleanupService) getFailoverJobSnapshot(ctx context.Context, vmID string) (string, string, error) {
	logger := ecs.jobTracker.Logger(ctx)
	
	var failoverJob database.FailoverJob
	err := ecs.db.Where("vm_id = ? AND job_type = 'test' AND status IN ('executing', 'completed')", vmID).
		Order("created_at DESC").First(&failoverJob).Error
	if err != nil {
		return "", "", fmt.Errorf("no test failover job found for VM %s: %w", vmID, err)
	}
	
	logger.Info("Retrieved failover job for cleanup", 
		"job_id", failoverJob.JobID,
		"snapshot_id", failoverJob.OSSEASnapshotID,
		"destination_vm_id", failoverJob.DestinationVMID)
	
	return failoverJob.JobID, failoverJob.OSSEASnapshotID, nil
}
```

#### **4.2 CloudStack Snapshot Rollback Method** 
```go
// NEW METHOD: CloudStack volume snapshot rollback (COPY PATTERN from enhanced_test_failover.go lines 970-1000)
func (ecs *EnhancedCleanupService) rollbackCloudStackVolumeSnapshot(ctx context.Context, vmID, snapshotID string) error {
	logger := ecs.jobTracker.Logger(ctx)
	
	if snapshotID == "" {
		logger.Info("No CloudStack snapshot to rollback", "vm_id", vmID)
		return nil // Not an error - older jobs may not have snapshots
	}
	
	logger.Info("Performing CloudStack volume snapshot rollback",
		"vm_id", vmID,
		"snapshot_id", snapshotID)
	
	// Get snapshot details first
	snapshot, err := ecs.osseaClient.GetVolumeSnapshot(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to get snapshot details: %w", err)
	}
	
	logger.Info("Retrieved snapshot details",
		"snapshot_id", snapshotID,
		"volume_id", snapshot.VolumeID,
		"state", snapshot.State)
	
	// Perform rollback
	err = ecs.osseaClient.RevertVolumeSnapshot(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to revert volume snapshot: %w", err)
	}
	
	logger.Info("CloudStack volume snapshot rollback completed",
		"vm_id", vmID,
		"snapshot_id", snapshotID)
	
	return nil
}
```

#### **4.3 Integration into Cleanup Flow**
```go
// MODIFY: Main cleanup method to add snapshot rollback step
// INSERT AFTER: Volume detachment, BEFORE: Volume reattachment

// STEP 4: CloudStack Volume Snapshot Rollback (NEW)
if err := ecs.jobTracker.RunStep(ctx, jobID, "cloudstack-snapshot-rollback", func(ctx context.Context) error {
	return ecs.rollbackCloudStackVolumeSnapshot(ctx, vmID, snapshotID)
}); err != nil {
	return fmt.Errorf("CloudStack snapshot rollback failed: %w", err)
}
```

---

### **PHASE 5: DATABASE STATUS TRACKING** ⏱️ 30 min

#### **5.1 Cleanup Status Updates**
```go
// NEW METHOD: Update failover job status during cleanup
func (ecs *EnhancedCleanupService) updateFailoverJobStatus(ctx context.Context, jobID, status string) error {
	logger := ecs.jobTracker.Logger(ctx)
	
	// Use existing repository pattern from enhanced_test_failover.go (line 292)
	var repo database.FailoverJobRepository
	repo.DB = ecs.db
	
	err := repo.UpdateStatus(jobID, status)
	if err != nil {
		logger.Error("Failed to update failover job status", "error", err, "job_id", jobID, "status", status)
		return fmt.Errorf("failed to update failover job status: %w", err)
	}
	
	logger.Info("Updated failover job status", "job_id", jobID, "status", status)
	return nil
}
```

#### **5.2 Status Transition Integration**
```go
// ADD STATUS UPDATES throughout cleanup flow:

// Start of cleanup
ecs.updateFailoverJobStatus(ctx, failoverJobID, "cleaning_up")

// After VM shutdown
ecs.updateFailoverJobStatus(ctx, failoverJobID, "vm_shutdown_complete")

// After snapshot rollback  
ecs.updateFailoverJobStatus(ctx, failoverJobID, "snapshot_rollback_complete")

// End of cleanup
ecs.updateFailoverJobStatus(ctx, failoverJobID, "cleanup_completed")
```

---

### **PHASE 6: INTEGRATION & TESTING** ⏱️ 45 min

#### **6.1 Compilation & Deployment**
```bash
# Build new version
cd cmd/oma
go build -o oma-api-cleanup-v1.4.0

# Stop current API
sudo systemctl stop oma-api

# Deploy new version  
sudo cp oma-api-cleanup-v1.4.0 /usr/local/bin/oma-api

# Start with monitoring
sudo systemctl start oma-api
journalctl -u oma-api -f
```

#### **6.2 Safety Testing Protocol**
```bash
# Test 1: API Health Check
curl -s http://localhost:8082/api/v1/health | jq .

# Test 2: Failover API Still Works (CRITICAL)
curl -s -X POST http://localhost:8082/api/v1/failover/test \
  -H "Content-Type: application/json" \
  -d '{"vm_id":"test","vm_name":"test"}' | jq .

# Test 3: Cleanup with Non-Critical VM
curl -s -X POST http://localhost:8082/api/v1/failover/cleanup/test-vm-name | jq .

# Test 4: JobLog Database Verification
mysql -e "SELECT * FROM job_tracking WHERE operation = 'enhanced-test-failover-cleanup' ORDER BY created_at DESC LIMIT 5;"
```

#### **6.3 Rollback Procedure** (If Any Issues)
```bash
# Emergency rollback
sudo systemctl stop oma-api
sudo cp cmd/oma/oma-api-cleanup-backup-v1.3.1 /usr/local/bin/oma-api
sudo systemctl start oma-api

# Verify rollback
curl -s http://localhost:8082/api/v1/health | jq .
```

---

### **PHASE 7: LEGACY CODE CLEANUP** ⏱️ 30 min

#### **7.1 Remove Old Cleanup Logic** (AFTER SUCCESSFUL TESTING)
```bash
# Archive old cleanup service implementation
mkdir -p archived_deprecated_code/cleanup_v1.3.x/
mv internal/oma/failover/enhanced_cleanup_service.go.backup-* \
   archived_deprecated_code/cleanup_v1.3.x/

# Archive old binaries
mkdir -p cmd/oma/archived_versions/cleanup_legacy/
mv cmd/oma/oma-api-cleanup-backup-v1.3.1 \
   cmd/oma/archived_versions/cleanup_legacy/
mv cmd/oma/oma-api-cleanup-test \
   cmd/oma/archived_versions/cleanup_legacy/
```

#### **7.2 Verify No Legacy References**
```bash
# Search for any remaining old cleanup patterns
grep -r "logging.NewOperationLogger.*cleanup" internal/
grep -r "opCtx.*cleanup" internal/
grep -r "centralized.*cleanup" internal/

# Search for old binary references
find . -name "*cleanup*backup*" -o -name "*cleanup*old*"
```

#### **7.3 Documentation Update**
```bash
# Update version history and remove deprecated patterns
# Clean commit with proper version increment
git add -A
git commit -m "feat: Fresh Enhanced Cleanup Service v1.4.0 - Remove Legacy Code

COMPLETE REWRITE BENEFITS:
✅ Clean JobLog integration following enhanced failover patterns
✅ CloudStack volume snapshot rollback functionality  
✅ Reduced complexity: ~200 lines vs 417 lines legacy code
✅ Zero legacy logging system dependencies
✅ Complete audit trail and correlation with failover jobs

LEGACY CLEANUP:
✅ Removed all old centralized logging patterns
✅ Archived deprecated implementation and binaries
✅ Clean codebase with no confusing legacy references

Version: v1.4.0 - Production ready cleanup service"
```

---

## 📋 **ACCEPTANCE CRITERIA**

### ✅ **Functional Requirements**
- [ ] Enhanced cleanup uses JobLog integration (same pattern as failover)
- [ ] CloudStack volume snapshot rollback functional
- [ ] Volume Daemon integration maintained (no changes)
- [ ] Database status tracking operational
- [ ] All cleanup steps tracked in job_tracking and log_events tables

### ✅ **Safety Requirements** ⚠️ **CRITICAL**
- [ ] Enhanced failover system unchanged and functional
- [ ] Existing test failover VMs can still be cleaned up
- [ ] API backward compatibility maintained
- [ ] No database schema changes
- [ ] No Volume Daemon changes

### ✅ **Quality Requirements**
- [ ] JobLog integration follows exact pattern from enhanced_test_failover.go
- [ ] Error handling comprehensive with proper context
- [ ] CloudStack snapshot rollback has timeout and retry logic
- [ ] All operations logged with correlation IDs
- [ ] Performance comparable to current cleanup (no regression)

---

## 🎯 **SUCCESS METRICS**

### **Technical Validation**
- **JobLog Integration**: Job tracking entries created for cleanup operations
- **Snapshot Rollback**: CloudStack snapshots successfully reverted before volume operations
- **Database Tracking**: Failover job status properly updated through cleanup lifecycle
- **Volume Operations**: All volume detach/attach operations via Volume Daemon (unchanged)

### **Operational Validation**  
- **Complete Flow**: VM shutdown → volume detach → snapshot rollback → volume reattach → VM deletion → status update
- **Error Recovery**: Failed cleanup operations properly logged and recoverable
- **Audit Trail**: Complete operation history in database for troubleshooting

---

## 🚨 **CRITICAL DEPENDENCIES**

### **DO NOT START UNTIL:**
- [ ] Current enhanced failover system is stable and tested
- [ ] No active production failover operations
- [ ] Backup plan verified and tested
- [ ] All team members aware of upgrade window

### **REQUIRED RESOURCES:**
- [ ] Access to database for job tracking verification
- [ ] Test VM available for cleanup testing
- [ ] Monitoring tools active for real-time system health
- [ ] Emergency contact available for rollback if needed

---

## 📝 **VERSION CONTROL**

**Files to be Modified:**
1. `internal/oma/failover/enhanced_cleanup_service.go` (PRIMARY)
2. `internal/oma/api/handlers/failover.go` (SINGLE LINE)
3. `cmd/oma/main.go` (VERSION BUMP)

**Files to be Created:**
1. `internal/oma/failover/enhanced_cleanup_service.go.backup-YYYYMMDD`
2. `cmd/oma/oma-api-cleanup-v1.4.0`

**Version History:**
- **v1.3.1**: CloudStack snapshot failover system
- **v1.4.0**: CloudStack snapshot cleanup integration ⭐ **TARGET**

---

**⚠️ REMEMBER: This upgrade maintains 100% compatibility with the working enhanced failover system while adding critical cleanup capabilities.**
