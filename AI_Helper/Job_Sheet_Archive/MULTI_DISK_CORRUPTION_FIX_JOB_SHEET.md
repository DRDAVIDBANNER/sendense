# 🔧 **MULTI-DISK CORRUPTION FIX JOB SHEET**

**Created**: September 24, 2025  
**Priority**: 🚨 **CRITICAL** - Multi-disk VMs corrupting data due to disk mapping issues  
**Bug ID**: MULTIDISK-CORRUPTION-001

---

## 🚨 **PROBLEM SUMMARY**

### **Issue Description**
Multi-disk VMs (like pgtest1) experience data corruption where:
- **OS disk** shows wrong partition layout (5GB partitions instead of 100GB Windows)
- **Data disk** shows no partitions (should have NTFS data partitions)
- **Root Cause**: Multiple VMware disks writing to same OMA target volume due to broken disk-to-target mapping

### **Evidence**
- **pgtest1-disk-0** (`/dev/vdc`): Shows wrong partition layout (5GB partition instead of 100GB Windows)
- **pgtest1-disk-1** (`/dev/vdg`): Shows no partitions (should have NTFS data)
- **Database Evidence**: `nbd_exports.vm_disk_id` is **NULL** for both exports
- **OMA→VMA Communication**: Sends array indices (0, 1) instead of meaningful disk correlation

---

## 🎯 **ROOT CAUSE ANALYSIS**

### **Problem 1: vm_disks Auto-Increment Instability**
- **Issue**: Every replication job creates NEW `vm_disks` records with NEW auto-increment IDs
- **Impact**: No stable identifier for disk-to-export correlation across job lifecycles
- **Evidence**: `vm_disks.id` changes from 760→764, 761→765 between jobs

### **Problem 2: NBD Export Correlation Broken**
- **Issue**: `nbd_exports.vm_disk_id` is **NULL** - no correlation to `vm_disks` records
- **Impact**: No way to determine which export corresponds to which VMware disk
- **Evidence**: Query shows `vm_disk_id=NULL` for both pgtest1 exports

### **Problem 3: OMA→VMA Wrong Disk IDs**
- **File**: `/source/current/oma/workflows/migration.go:929`
- **Issue**: `"vm_disk_id": i, // Use index as disk ID for now` sends array indices
- **Impact**: migratekit can't map VMware disks to correct NBD targets
- **Evidence**: VMA receives `vm_disk_id: 0, 1` instead of meaningful correlation

---

## 🔧 **IMPLEMENTATION PLAN**

### **Phase 1: Stable vm_disks Architecture** ⚡ **CRITICAL**

#### **Task 1.1: Implement vm_disks Upsert Logic**
**File**: `/source/current/oma/workflows/migration.go:306-380`
**Status**: ⏳ **PENDING**

**Change analyzeAndRecordVMDisks() to UPDATE existing records instead of CREATE new ones**:
```go
// Current (BROKEN): Always creates new records
vmDisk := &database.VMDisk{
    JobID:       req.JobID,           // NEW job ID every time
    VMContextID: vmContextID,
    DiskID:      disk.ID,             // "disk-2000", "disk-2001" (stable)
    // ...
}
m.vmDiskRepo.Create(vmDisk)  // Always creates NEW record

// New (FIXED): Upsert logic to maintain stable IDs
existingDisk, err := m.vmDiskRepo.FindByContextAndDiskID(vmContextID, disk.ID)
if err == nil && existingDisk != nil {
    // UPDATE existing record, preserve stable ID
    existingDisk.JobID = req.JobID
    existingDisk.UpdatedAt = time.Now()
    // Update other fields as needed
    err = m.vmDiskRepo.Update(existingDisk)
} else {
    // CREATE new record only if doesn't exist
    err = m.vmDiskRepo.Create(vmDisk)
}
```

#### **Task 1.2: Add Database Schema Changes**
**File**: Database migration
**Status**: ⏳ **PENDING**

**Add unique constraint to enforce vm_disks stability**:
```sql
-- Ensure vm_context_id + disk_id combination is unique
ALTER TABLE vm_disks 
ADD UNIQUE KEY uk_vm_context_disk (vm_context_id, disk_id);

-- Add index for performance
CREATE INDEX idx_vm_disks_context_disk ON vm_disks (vm_context_id, disk_id);
```

#### **Task 1.3: Implement Repository Methods**
**File**: `/source/current/oma/database/repository.go`
**Status**: ⏳ **PENDING**

**Add missing repository methods**:
```go
// FindByContextAndDiskID finds existing vm_disk record
func (r *VMDiskRepository) FindByContextAndDiskID(vmContextID, diskID string) (*VMDisk, error) {
    var vmDisk VMDisk
    err := r.db.Where("vm_context_id = ? AND disk_id = ?", vmContextID, diskID).First(&vmDisk).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil // Not found is not an error
        }
        return nil, fmt.Errorf("failed to find vm_disk: %w", err)
    }
    return &vmDisk, nil
}

// Update updates existing vm_disk record
func (r *VMDiskRepository) Update(disk *VMDisk) error {
    return r.db.Save(disk).Error
}
```

### **Phase 2: Fix NBD Export Correlation** ⚡ **CRITICAL**

#### **Task 2.1: Fix NBD Export vm_disk_id Population**
**File**: `/source/current/oma/nbd/server.go` or Volume Daemon
**Status**: ⏳ **PENDING**

**Correlate NBD exports to stable vm_disks.id**:
```go
// When creating NBD export, lookup corresponding vm_disk.id
func AddDynamicExportWithVolume(jobID, vmName, vmID, volumeID string, diskUnitNumber int, repo *database.VMExportMappingRepository) (*ExportInfo, bool, error) {
    // ... existing logic ...
    
    // NEW: Find corresponding vm_disk.id for correlation
    vmDiskID, err := findVMDiskIDByVolumeID(volumeID, diskUnitNumber)
    if err != nil {
        log.WithError(err).Warn("Failed to find vm_disk correlation, proceeding without")
        vmDiskID = nil // Allow NBD export creation but log warning
    }
    
    // Create export with proper vm_disk_id correlation
    exportMapping := &database.VMExportMapping{
        // ... existing fields ...
        VMDiskID: vmDiskID, // Link to stable vm_disks.id
    }
}

// Helper function to find vm_disk.id from volume correlation
func findVMDiskIDByVolumeID(volumeID string, diskUnitNumber int) (*int, error) {
    // Query: volume_id → vm_disks.ossea_volume_id → vm_disks.id
    var vmDisk database.VMDisk
    err := db.Where("ossea_volume_id = ?", volumeID).First(&vmDisk).Error
    if err != nil {
        return nil, err
    }
    return &vmDisk.ID, nil
}
```

#### **Task 2.2: Fix OMA→VMA Communication**
**File**: `/source/current/oma/workflows/migration.go:927-931`
**Status**: ⏳ **PENDING**

**Send stable vm_disks.id instead of array indices**:
```go
// Current (BROKEN):
nbd_targets = append(nbd_targets, map[string]interface{}{
    "device_path": devicePath,
    "vm_disk_id":  i, // Use index as disk ID for now  ❌ WRONG!
})

// New (FIXED):
// Get corresponding vm_disk.id from NBD export correlation
vmDiskID, err := getVMDiskIDFromExport(export)
if err != nil {
    log.WithError(err).Warn("Failed to get vm_disk correlation, using fallback")
    vmDiskID = i // Fallback to index for backward compatibility
}

nbd_targets = append(nbd_targets, map[string]interface{}{
    "device_path": devicePath,
    "vm_disk_id":  vmDiskID, // Stable vm_disks.id  ✅ CORRECT!
})
```

### **Phase 3: Enhanced Multi-Disk Validation** 🔧 **ENHANCEMENT**

#### **Task 3.1: Add Multi-Disk Corruption Detection**
**File**: `/source/current/oma/workflows/migration.go`
**Status**: ⏳ **PENDING**

**Pre-flight validation to detect mapping issues**:
```go
func (m *MigrationEngine) validateMultiDiskMapping(req *MigrationRequest) error {
    if len(req.SourceVM.Disks) <= 1 {
        return nil // Single disk - no correlation issues
    }
    
    // Validate NBD export correlations exist and are unique
    vmContextID, _ := m.getVMContextIDForJob(req.JobID)
    for _, disk := range req.SourceVM.Disks {
        vmDisk, err := m.vmDiskRepo.FindByContextAndDiskID(vmContextID, disk.ID)
        if err != nil || vmDisk == nil {
            return fmt.Errorf("missing vm_disk record for disk %s - corruption risk", disk.ID)
        }
        
        // Check NBD export correlation exists
        export, err := m.findNBDExportByVMDiskID(vmDisk.ID)
        if err != nil || export == nil {
            return fmt.Errorf("missing NBD export correlation for vm_disk.id %d - corruption risk", vmDisk.ID)
        }
    }
    
    log.Info("✅ Multi-disk mapping validation passed", "disk_count", len(req.SourceVM.Disks))
    return nil
}
```

#### **Task 3.2: Add Source Code Authority Violation Fix**
**File**: Copy VMA multi-disk source to authoritative location
**Status**: ⏳ **PENDING**

**Resolve source code authority violation**:
- Copy working VMA multi-disk source from VMA appliance to `/source/current/vma/`
- Ensure deployed `vma-api-server-multi-disk-debug` matches authoritative source
- Document multi-disk VMA logic that's currently missing from source

---

## 🧪 **TESTING STRATEGY**

### **Test 1: Stable vm_disks Validation**
```sql
-- Before fix: New IDs every job
SELECT id, disk_id, job_id FROM vm_disks WHERE vm_context_id = 'ctx-pgtest1...' ORDER BY created_at;
-- Should show: id=760,764 for disk-2000 (different IDs)

-- After fix: Same IDs across jobs  
SELECT id, disk_id, job_id FROM vm_disks WHERE vm_context_id = 'ctx-pgtest1...' ORDER BY created_at;
-- Should show: id=760,760 for disk-2000 (same ID)
```

### **Test 2: NBD Export Correlation**
```sql
-- Before fix: NULL vm_disk_id
SELECT vm_disk_id, volume_id, export_name FROM nbd_exports WHERE vm_context_id = 'ctx-pgtest1...';
-- Should show: vm_disk_id=NULL

-- After fix: Proper correlation
SELECT vm_disk_id, volume_id, export_name FROM nbd_exports WHERE vm_context_id = 'ctx-pgtest1...';
-- Should show: vm_disk_id=760,761 (stable IDs)
```

### **Test 3: Disk Corruption Prevention**
```bash
# After fix: Test pgtest1 replication
# OS disk (/dev/vdc) should show: 100GB Windows partitions
# Data disk (/dev/vdg) should show: NTFS data partitions
lsblk
fdisk -l /dev/vdc /dev/vdg
```

---

## 🚀 **IMPLEMENTATION SEQUENCE**

### **Step 1: Database Schema (30 minutes)** ✅ **COMPLETED**
- [x] **1.1**: Create migration for unique constraint ✅
- [x] **1.2**: Apply migration to development database ✅
- [x] **1.3**: Validate constraint works as expected ✅

### **Step 2: Repository Methods (30 minutes)** ✅ **COMPLETED**
- [x] **2.1**: Add `FindByContextAndDiskID()` method ✅
- [x] **2.2**: Add `Update()` method ✅
- [x] **2.3**: Test repository methods with sample data ✅

### **Step 3: vm_disks Upsert Logic (60 minutes)** ✅ **COMPLETED**
- [x] **3.1**: Implement upsert logic in `analyzeAndRecordVMDisks()` ✅
- [x] **3.2**: Add error handling and rollback logic ✅
- [x] **3.3**: Test with pgtest1 to verify stable IDs ✅

### **Step 4: NBD Export Correlation (45 minutes)** ✅ **COMPLETED**
- [x] **4.1**: Implement `correlateNBDExportsWithVMDisks()` helper ✅
- [x] **4.2**: Fix NBD export creation to populate `vm_disk_id` ✅
- [x] **4.3**: Test NBD export correlation works ✅

### **Step 5: OMA→VMA Communication Fix (30 minutes)** ✅ **COMPLETED**
- [x] **5.1**: Implement `getVMDiskIDFromNBDExport()` helper ✅
- [x] **5.2**: Fix NBD targets to send stable `vm_disk_id` ✅
- [x] **5.3**: Test VMA receives correct disk correlation ✅

### **Step 6: Integration Testing (45 minutes)** 🔄 **IN PROGRESS**
- [x] **6.1**: Deploy all changes to development ✅ **DEPLOYED: oma-api-v2.13.2-multidisk-corruption-fix**
- [ ] **6.2**: Test pgtest1 complete replication workflow ⏳ **READY FOR USER TESTING**
- [ ] **6.3**: Validate no disk corruption occurs ⏳ **AWAITING TEST RESULTS**
- [ ] **6.4**: Test QCDev-Jump05 single-disk (regression test) ⏳ **PENDING**

### **Step 7: Production Deployment (30 minutes)**
- [ ] **7.1**: Build and deploy OMA API with fixes
- [ ] **7.2**: Apply database migration
- [ ] **7.3**: Monitor system for issues
- [ ] **7.4**: Test production VM replication

---

## 📚 **COMPLIANCE CHECKLIST**

### **🚨 Absolute Project Rules Compliance**
- [ ] **Source Code Authority**: All changes in `/source/current/` only
- [ ] **Volume Operations**: No direct OSSEA SDK calls - Volume Daemon only
- [ ] **Database Schema**: Validate all field names against migrations
- [ ] **Logging**: Use `internal/joblog` for all business logic operations

### **🔒 Operational Safety**
- [ ] **NO Failover Operations**: No live/test failover execution during fix
- [ ] **NO VM State Changes**: No operations that affect VM state
- [ ] **User Approval**: Ask permission before any operational changes

### **📊 Architecture Standards**
- [ ] **No Monster Code**: Keep functions focused and manageable
- [ ] **Modular Design**: Clean interfaces and separation of concerns
- [ ] **Documentation**: Document major logic changes

---

## 🎯 **SUCCESS CRITERIA**

### **Technical Goals**
- [ ] ✅ **Stable vm_disks**: Same `vm_disks.id` across multiple jobs for same VM disk
- [ ] ✅ **Proper NBD Correlation**: `nbd_exports.vm_disk_id` links to stable `vm_disks.id`
- [ ] ✅ **Correct Disk Mapping**: Each VMware disk writes to its intended OMA volume
- [ ] ✅ **No Corruption**: pgtest1-style disk corruption eliminated

### **Validation Tests**
- [ ] ✅ **pgtest1 Replication**: OS disk shows correct Windows partitions
- [ ] ✅ **Multi-disk Integrity**: Data disk shows proper NTFS partitions  
- [ ] ✅ **Single-disk Regression**: QCDev-Jump05 continues working
- [ ] ✅ **Database Consistency**: All correlations properly maintained

---

## 📋 **CURRENT STATUS**

**Overall Progress**: 0% ⏳ **READY TO START**

**Phase 1**: ⏳ Pending - Database schema and vm_disks stability
**Phase 2**: ⏳ Pending - NBD export correlation fixes
**Phase 3**: ⏳ Pending - Multi-disk validation enhancements

**Next Action**: Begin with Step 1 - Database schema changes

---

**🚨 CRITICAL**: This fix addresses a fundamental data corruption issue affecting all multi-disk VMs. Proper implementation will prevent data loss and ensure reliable multi-disk replication.
