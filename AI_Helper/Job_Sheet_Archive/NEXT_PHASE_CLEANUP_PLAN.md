# Next Phase Work - Cleanup Plan

**Date**: September 7, 2025  
**Status**: ✅ **COMPLETED**  
**Purpose**: Comprehensive plan for logging fixes and duplicate code cleanup - EXECUTION COMPLETE

---

## 🎯 **WORK OVERVIEW**

The next phase of work involves **two main components**:
1. **Fix logging violations** in Enhanced Failover System (9 instances)
2. **Clean up duplicate code** from old `/internal/` locations

## 📊 **DUPLICATE CODE LOCATIONS IDENTIFIED**

### **🔄 FAILOVER SYSTEM DUPLICATES**
```
CONSOLIDATED (USE THIS):
/source/current/oma/failover/
├── enhanced_test_failover.go      ✅ AUTHORITATIVE
├── enhanced_live_failover.go      ✅ AUTHORITATIVE  
├── enhanced_cleanup_service.go    ✅ AUTHORITATIVE
└── validator.go                   ✅ AUTHORITATIVE

DUPLICATES (REMOVE):
/source/current/migratekit/internal/oma/failover/
├── enhanced_test_failover.go      ❌ DUPLICATE
├── enhanced_live_failover.go      ❌ DUPLICATE
└── enhanced_cleanup_service.go    ❌ DUPLICATE

/source/current/migratekit/internal/oma/api/handlers/
├── enhanced_failover_wrapper.go   ❌ DUPLICATE
└── failover.go                    ❌ DUPLICATE
```

### **🔄 LOGGING SYSTEM DUPLICATES**
```
CONSOLIDATED (USE THIS):
/source/current/oma/joblog/        ✅ AUTHORITATIVE (9 files)
/source/current/oma/common/logging/ ✅ AUTHORITATIVE (1 file)

DUPLICATES (REMOVE):
/internal/joblog/                  ❌ DUPLICATE (9 files)
/internal/common/logging/          ❌ DUPLICATE (1 file)
```

---

## 🔧 **PHASE 1: LOGGING COMPLIANCE FIXES**

### **🚨 PRIORITY 1: Fix Enhanced Failover Logging**

#### **Target File**: `/source/current/oma/failover/enhanced_test_failover.go`
#### **Violations**: 9 instances of direct logrus usage (lines 680-766)

#### **Conversion Pattern**:
```go
// BEFORE (NON-COMPLIANT):
log.WithFields(log.Fields{
    "vm_id": vmID,
    "snapshot_name": snapshotName,
}).Info("Creating Linstor snapshot")

// AFTER (COMPLIANT):
err = etfe.jobTracker.RunStep(ctx, jobID, "create-linstor-snapshot", func(ctx context.Context) error {
    log := etfe.jobTracker.Logger(ctx)
    log.Info("Creating Linstor snapshot", "vm_id", vmID, "snapshot_name", snapshotName)
    // ... operation logic
    return nil
})
```

#### **Specific Lines to Fix**:
1. Line 680: `log.WithFields(log.Fields{...}).Info("🔍 Starting Linstor snapshot creation")`
2. Line 697: `log.WithFields(log.Fields{...}).Info("📋 Snapshot request prepared")`
3. Line 710: `log.WithFields(log.Fields{...}).Error("❌ Linstor snapshot creation failed")`
4. Line 722: `log.WithFields(log.Fields{...}).Info("✅ Linstor snapshot created successfully")`
5. Line 730: `log.WithField("snapshot_name", snapshotName).Info("🔍 Verifying snapshot creation")`
6. Line 735: `log.WithFields(log.Fields{...}).Error("❌ Snapshot verification failed")`
7. Line 750: `log.WithFields(log.Fields{...}).Error("❌ Snapshot not found in list")`
8. Line 760: `log.WithFields(log.Fields{...}).Info("✅ Snapshot verification completed")`
9. Line 766: `log.WithField("snapshot_name", snapshotName).Info("💾 Storing snapshot metadata in database")`

---

## 🧹 **PHASE 2: DUPLICATE CODE CLEANUP**

### **🔄 CLEANUP STRATEGY**

#### **Step 1: Verify Dependencies**
```bash
# Check for any remaining imports to old locations
grep -r "github.com/vexxhost/migratekit/internal/oma/failover" --include="*.go" .
grep -r "github.com/vexxhost/migratekit/internal/joblog" --include="*.go" .
grep -r "github.com/vexxhost/migratekit/internal/common/logging" --include="*.go" .
```

#### **Step 2: Archive Duplicate Failover Code**
```bash
# Archive duplicate failover files
mkdir -p source/archive/internal-oma-failover-duplicates-$(date +%Y%m%d-%H%M%S)
mv source/current/migratekit/internal/oma/failover source/archive/internal-oma-failover-duplicates-$(date +%Y%m%d-%H%M%S)/
mv source/current/migratekit/internal/oma/api/handlers/enhanced_failover_wrapper.go source/archive/internal-oma-failover-duplicates-$(date +%Y%m%d-%H%M%S)/
mv source/current/migratekit/internal/oma/api/handlers/failover.go source/archive/internal-oma-failover-duplicates-$(date +%Y%m%d-%H%M%S)/
```

#### **Step 3: Archive Duplicate Logging Code**
```bash
# Archive duplicate logging files
mkdir -p source/archive/internal-joblog-$(date +%Y%m%d-%H%M%S)
mv internal/joblog source/archive/internal-joblog-$(date +%Y%m%d-%H%M%S)/

mkdir -p source/archive/internal-common-logging-$(date +%Y%m%d-%H%M%S)
mv internal/common/logging source/archive/internal-common-logging-$(date +%Y%m%d-%H%M%S)/
```

#### **Step 4: Verify Clean State**
```bash
# Verify no remaining duplicates
find . -path "./source/archive" -prune -o -name "*failover*" -type f -print
find . -path "./source/archive" -prune -o -path "./source/current/oma" -prune -o -name "*joblog*" -type f -print
find . -path "./source/archive" -prune -o -path "./source/current/oma" -prune -o -path "*/logging/*" -type f -print
```

---

## 📋 **DETAILED WORK PLAN**

### **🚨 TASK 1: Fix Logging Violations**
- **File**: `/source/current/oma/failover/enhanced_test_failover.go`
- **Lines**: 680-766 (9 specific instances)
- **Method**: Convert direct logrus to JobLog patterns
- **Testing**: Verify JobLog integration works correctly
- **Estimated Time**: 1-2 hours

### **🧹 TASK 2: Clean Duplicate Failover Code**
- **Source**: `/source/current/migratekit/internal/oma/failover/`
- **Action**: Archive to `/source/archive/internal-oma-failover-duplicates-TIMESTAMP/`
- **Verification**: Ensure no remaining imports to old locations
- **Estimated Time**: 30 minutes

### **🧹 TASK 3: Clean Duplicate Logging Code**
- **Sources**: `/internal/joblog/` and `/internal/common/logging/`
- **Action**: Archive to `/source/archive/internal-*-TIMESTAMP/`
- **Verification**: Ensure consolidated versions work correctly
- **Estimated Time**: 30 minutes

### **✅ TASK 4: Verification & Testing**
- **Build Tests**: Ensure all systems build correctly
- **Functionality Tests**: Verify failover and logging work
- **Import Verification**: No references to old locations
- **Estimated Time**: 1 hour

---

## 🎯 **SUCCESS CRITERIA**

### **✅ LOGGING COMPLIANCE**
- [ ] Zero direct logrus calls in enhanced failover system
- [ ] All business logic uses JobLog tracker
- [ ] Proper correlation IDs in all log entries
- [ ] No logging rule violations

### **✅ CODE CLEANUP**
- [ ] No duplicate failover code in `/internal/` locations
- [ ] No duplicate logging code in `/internal/` locations
- [ ] All old code safely archived with timestamps
- [ ] Zero references to old internal locations

### **✅ FUNCTIONALITY**
- [ ] Enhanced failover system works correctly
- [ ] JobLog integration functional
- [ ] All services build and deploy successfully
- [ ] No broken imports or dependencies

---

## ⚠️ **RISKS & MITIGATION**

### **🚨 POTENTIAL RISKS**
1. **Import Dependencies**: Some code might still reference old locations
2. **Build Failures**: Cleanup might break builds
3. **Functionality Loss**: Logging changes might affect behavior

### **🛡️ MITIGATION STRATEGIES**
1. **Comprehensive Search**: Scan entire codebase for old imports before cleanup
2. **Incremental Testing**: Test after each major change
3. **Safe Archiving**: Archive (don't delete) old code for rollback
4. **Git Commits**: Commit each phase separately for easy rollback

---

## 📝 **EXECUTION ORDER**

### **RECOMMENDED SEQUENCE**:
1. **🔧 Fix Logging First**: Convert 9 logrus calls to JobLog
2. **✅ Test Logging**: Verify JobLog integration works
3. **🧹 Clean Failover Duplicates**: Archive duplicate failover files
4. **🧹 Clean Logging Duplicates**: Archive duplicate logging files
5. **✅ Final Verification**: Comprehensive testing and validation
6. **📚 Update Documentation**: Reflect changes in docs

### **SAFETY APPROACH**:
- Commit after each major step
- Test functionality after each cleanup
- Keep archives for rollback capability
- Verify no broken dependencies

---

## 🎉 **EXPECTED OUTCOME**

After completion:
- **✅ Full Project Rule Compliance**: No logging violations
- **✅ Clean Architecture**: Single source of truth in `/source/current/`
- **✅ Maintainable Codebase**: No duplicate code confusion
- **✅ Production Ready**: All systems functional and compliant

**Total Estimated Time**: 3-4 hours
**Risk Level**: Low (safe archiving approach)
**Impact**: High (full architectural compliance)

---

**Status**: Ready for execution - comprehensive plan with safety measures in place
