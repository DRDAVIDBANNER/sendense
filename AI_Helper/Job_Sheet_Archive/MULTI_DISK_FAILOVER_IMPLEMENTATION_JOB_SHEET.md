# 🔧 **MULTI-DISK FAILOVER IMPLEMENTATION JOB SHEET**

**Created**: September 25, 2025  
**Priority**: 🚨 **HIGH** - Complete multi-disk VM failover support  
**Bug ID**: MULTIDISK-FAILOVER-001

---

## 🚨 **PROBLEM SUMMARY**

### **Issue Description**
The unified failover system currently only handles the **OS disk** (`disk-2000`) during test failover operations. Multi-disk VMs are **incompletely failed over** with only the root volume attached to the test VM, while data disks are ignored.

### **Evidence from Yesterday's Testing**
- **✅ OS Disk Failover**: VirtIO injection successful on `/dev/vdb` (disk-2000)
- **❌ Data Disk Ignored**: `/dev/vdc` (disk-2001) not attached to test VM
- **❌ Incomplete Test VM**: Test VM created with only root volume, missing data disks
- **✅ Multi-disk Replication**: Proven working - both disks replicate correctly with VMware key correlation

---

## 🎯 **ROOT CAUSE ANALYSIS**

### **Current Unified Failover Architecture**
The unified failover system has **single-disk assumptions** throughout:

#### **Volume Query Logic** (`unified_failover_engine.go:489-518`):
```go
// CURRENT: Only finds OS disk
var osDisk *database.VMDisk
for _, disk := range vmDisks {
    if disk.DiskID == "disk-2000" {  // Only OS disk
        osDisk = &disk
        break  // Stops after finding OS disk
    }
}
// Uses only osDisk for volume attachment
```

#### **Volume Attachment Logic** (`unified_failover_engine.go:550-567`):
```go
// CURRENT: Only attaches OS disk as root
operation, err := ufe.volumeClient.AttachVolumeAsRoot(ctx, volumeID, destinationVMID)
// No logic for data disk attachment
```

#### **VM Creation Logic**:
```go
// CURRENT: Creates VM with only root disk size
"root_disk_size_gb": osseaVolume.SizeGB  // Only OS disk size
// No provision for additional data disks
```

### **Why Data Disks Are Ignored**
1. **Architecture Assumption**: Designed for single-disk VMs
2. **OS-Only Logic**: Explicitly searches for and uses only `disk-2000`
3. **Root Volume Focus**: Only handles root disk attachment
4. **VM Creation Limitation**: No provision for multiple disk attachment

---

## 🔧 **IMPLEMENTATION PLAN**

### **Phase 1: Multi-Disk Volume Query Enhancement** ⚡ **CRITICAL**

#### **Task 1.1: Enhance Volume Info Retrieval**
**File**: `/source/current/oma/failover/unified_failover_engine.go:489-518`
**Status**: ⏳ **PENDING**

**Change from**:
```go
// Current: Only OS disk
var osDisk *database.VMDisk
for _, disk := range vmDisks {
    if disk.DiskID == "disk-2000" {
        osDisk = &disk
        break
    }
}
```

**Change to**:
```go
// New: All disks with OS disk prioritization
type DiskInfo struct {
    VMDisk      database.VMDisk
    OSSEAVolume database.OSSEAVolume
    IsOSDisk    bool
}

var diskInfos []DiskInfo
var osDiskInfo *DiskInfo

for _, disk := range vmDisks {
    // Get OSSEA volume for each disk
    var osseaVolume database.OSSEAVolume
    err := ufe.db.GetGormDB().Where("id = ?", disk.OSSEAVolumeID).First(&osseaVolume).Error
    if err != nil {
        continue // Skip disks without valid volumes
    }
    
    diskInfo := DiskInfo{
        VMDisk:      disk,
        OSSEAVolume: osseaVolume,
        IsOSDisk:    disk.DiskID == "disk-2000",
    }
    
    diskInfos = append(diskInfos, diskInfo)
    
    if diskInfo.IsOSDisk {
        osDiskInfo = &diskInfo
    }
}
```

#### **Task 1.2: Return Multi-Disk Volume Info**
**File**: `/source/current/oma/failover/unified_failover_engine.go:VolumeInfo struct`
**Status**: ⏳ **PENDING**

**Enhance VolumeInfo struct**:
```go
// Current: Single volume
type VolumeInfo struct {
    VolumeID   string
    VolumeName string
    SizeGB     int
}

// New: Multi-volume support
type VolumeInfo struct {
    OSVolume   VolumeDetails      // Root volume (disk-2000)
    DataVolumes []VolumeDetails   // Additional volumes (disk-2001, disk-2002, etc.)
}

type VolumeDetails struct {
    VolumeID   string
    VolumeName string
    SizeGB     int
    DiskID     string  // "disk-2000", "disk-2001", etc.
    VMDiskID   int     // vm_disks.id for correlation
}
```

### **Phase 2: Multi-Disk Volume Attachment** ⚡ **CRITICAL**

#### **Task 2.1: Implement Multi-Disk Attachment Logic**
**File**: `/source/current/oma/failover/unified_failover_engine.go:executeVolumeAttachmentPhase()`
**Status**: ⏳ **PENDING**

**New Multi-Disk Attachment Flow**:
```go
func (ufe *UnifiedFailoverEngine) executeVolumeAttachmentPhase(ctx context.Context, jobID string, config *UnifiedFailoverConfig, destinationVMID string) error {
    return ufe.jobTracker.RunStep(ctx, jobID, "volume-attachment", func(ctx context.Context) error {
        logger := ufe.jobTracker.Logger(ctx)
        
        // Get ALL volume information for multi-disk VM
        volumeInfo, err := ufe.getMultiDiskVolumeInfoForVM(ctx, config.ContextID)
        if err != nil {
            return fmt.Errorf("failed to get multi-disk volume info: %w", err)
        }
        
        // Step 1: Delete destination VM's default root volume
        if err := ufe.deleteDestinationVMRootVolume(ctx, destinationVMID); err != nil {
            return fmt.Errorf("failed to delete destination VM root volume: %w", err)
        }
        
        // Step 2: Attach OS disk as root (device ID 0)
        logger.Info("🔗 Attaching OS disk as root volume", 
            "volume_id", volumeInfo.OSVolume.VolumeID,
            "disk_id", volumeInfo.OSVolume.DiskID)
            
        osOperation, err := ufe.volumeClient.AttachVolumeAsRoot(ctx, volumeInfo.OSVolume.VolumeID, destinationVMID)
        if err != nil {
            return fmt.Errorf("failed to attach OS volume: %w", err)
        }
        
        // Wait for OS disk attachment completion
        if _, err := ufe.volumeClient.WaitForCompletionWithTimeout(ctx, osOperation.ID, 300*time.Second); err != nil {
            return fmt.Errorf("OS volume attachment failed: %w", err)
        }
        
        // Step 3: Attach all data disks as additional volumes
        for i, dataVolume := range volumeInfo.DataVolumes {
            logger.Info("🔗 Attaching data disk as additional volume", 
                "volume_id", dataVolume.VolumeID,
                "disk_id", dataVolume.DiskID,
                "device_index", i+1)
                
            dataOperation, err := ufe.volumeClient.AttachVolume(ctx, dataVolume.VolumeID, destinationVMID)
            if err != nil {
                logger.Error("Failed to attach data volume", "error", err, "disk_id", dataVolume.DiskID)
                return fmt.Errorf("failed to attach data volume %s: %w", dataVolume.DiskID, err)
            }
            
            // Wait for data disk attachment completion
            if _, err := ufe.volumeClient.WaitForCompletionWithTimeout(ctx, dataOperation.ID, 300*time.Second); err != nil {
                logger.Error("Data volume attachment failed", "error", err, "disk_id", dataVolume.DiskID)
                return fmt.Errorf("data volume attachment failed for %s: %w", dataVolume.DiskID, err)
            }
            
            logger.Info("✅ Data disk attached successfully", 
                "volume_id", dataVolume.VolumeID,
                "disk_id", dataVolume.DiskID)
        }
        
        logger.Info("✅ Multi-disk volume attachment completed", 
            "os_volume", volumeInfo.OSVolume.VolumeID,
            "data_volumes", len(volumeInfo.DataVolumes),
            "destination_vm_id", destinationVMID)
            
        return nil
    })
}
```

#### **Task 2.2: Volume Daemon Multi-Disk Support**
**File**: `/source/current/volume-daemon/service/volume_service.go`
**Status**: ⏳ **PENDING**

**Required Method**:
```go
// AttachVolume attaches a volume as additional (non-root) disk
func (vs *VolumeService) AttachVolume(ctx context.Context, volumeID, vmID string) (*models.VolumeOperation, error) {
    // Similar to AttachVolumeAsRoot but without root device ID specification
    // Lets CloudStack assign the next available device ID (1, 2, 3, etc.)
}
```

### **Phase 3: Enhanced VM Validation** 🔧 **ENHANCEMENT**

#### **Task 3.1: Multi-Disk VM Startup Validation**
**File**: `/source/current/oma/failover/unified_failover_engine.go:executeVMStartupPhase()`
**Status**: ⏳ **PENDING**

**Enhanced validation logic**:
```go
// Validate that test VM has ALL expected disks attached
func (ufe *UnifiedFailoverEngine) validateMultiDiskAttachment(ctx context.Context, destinationVMID string, expectedDiskCount int) error {
    // Query CloudStack for VM's attached volumes
    // Verify count matches expected disk count
    // Validate device ordering (root at device ID 0, data at 1, 2, etc.)
}
```

### **Phase 4: Enhanced Cleanup Logic** 🧹 **MAINTENANCE**

#### **Task 4.1: Multi-Disk Cleanup Enhancement**
**File**: Cleanup services
**Status**: ⏳ **PENDING**

**Enhanced cleanup to handle**:
- **Multiple volume detachments** from test VM
- **All data disk restoration** to OMA
- **Complete volume reattachment** with proper device correlation

---

## 🧪 **TESTING STRATEGY**

### **Test 1: Multi-Disk Test Failover Validation**
```bash
# After implementation:
# 1. Start pgtest1 test failover
# 2. Verify test VM creation with correct CPU/memory specs
# 3. Verify OS disk attachment as /dev/vda (root)
# 4. Verify data disk attachment as /dev/vdb (additional)
# 5. Verify complete VM functionality

# Expected CloudStack VM:
VM: pgtest1-test-TIMESTAMP
├── Root Volume: f7462ed3... (107GB, device ID 0) ✅
└── Data Volume: b16589be... (10GB, device ID 1) ✅
```

### **Test 2: Multi-Disk Cleanup Validation**
```bash
# After test failover completion:
# 1. Execute cleanup/rollback
# 2. Verify both volumes detached from test VM
# 3. Verify both volumes reattached to OMA with correct device paths
# 4. Verify OMA device state: /dev/vdb (OS), /dev/vdc (data)
# 5. Verify VM context restored to ready_for_failover
```

### **Test 3: Multi-Disk Live Failover Validation**
```bash
# Final validation:
# 1. Execute live failover after successful test failover
# 2. Verify production VM creation with both disks
# 3. Verify VM boots and operates with complete disk set
# 4. Verify data integrity on both OS and data volumes
```

---

## 🚀 **IMPLEMENTATION SEQUENCE**

### **Step 1: Volume Info Enhancement (60 minutes)**
- [ ] **1.1**: Create enhanced VolumeInfo struct for multi-disk support
- [ ] **1.2**: Implement `getMultiDiskVolumeInfoForVM()` method
- [ ] **1.3**: Update volume query logic to find ALL disks, not just OS disk
- [ ] **1.4**: Test volume info retrieval with pgtest1 (2 disks expected)

### **Step 2: Volume Daemon Enhancement (45 minutes)**
- [ ] **2.1**: Implement `AttachVolume()` method for data disk attachment
- [ ] **2.2**: Test volume attachment API with CloudStack
- [ ] **2.3**: Validate device ID assignment for multiple disks
- [ ] **2.4**: Deploy enhanced Volume Daemon

### **Step 3: Multi-Disk Attachment Logic (90 minutes)**
- [ ] **3.1**: Implement multi-disk attachment in `executeVolumeAttachmentPhase()`
- [ ] **3.2**: Add proper error handling for partial attachment failures
- [ ] **3.3**: Add device correlation tracking for all attached volumes
- [ ] **3.4**: Test complete attachment workflow

### **Step 4: VM Validation Enhancement (30 minutes)**
- [ ] **4.1**: Enhance VM startup validation for multi-disk verification
- [ ] **4.2**: Add CloudStack API queries to verify attached volume count
- [ ] **4.3**: Validate device ordering (root=0, data=1,2,3...)
- [ ] **4.4**: Test validation with complete multi-disk test VM

### **Step 5: Cleanup Enhancement (45 minutes)**
- [ ] **5.1**: Enhance cleanup logic for multiple volume detachment
- [ ] **5.2**: Update volume reattachment for all disks
- [ ] **5.3**: Ensure proper device path restoration
- [ ] **5.4**: Test complete cleanup workflow

### **Step 6: Integration Testing (60 minutes)**
- [ ] **6.1**: Deploy all multi-disk failover enhancements
- [ ] **6.2**: Test complete pgtest1 multi-disk test failover
- [ ] **6.3**: Verify test VM has both OS and data disks
- [ ] **6.4**: Test cleanup/rollback with multi-disk handling
- [ ] **6.5**: Validate system ready for production use

---

## 📚 **COMPLIANCE CHECKLIST**

### **🚨 Absolute Project Rules Compliance**
- [ ] **Source Code Authority**: All changes in `/source/current/` only
- [ ] **Volume Operations**: Use Volume Daemon for all volume operations  
- [ ] **Database Schema**: Validate field names against existing schema
- [ ] **Logging**: Use `internal/joblog` for all business logic operations

### **🔒 Operational Safety**
- [ ] **NO Failover Operations**: No live/test failover execution during development
- [ ] **NO VM State Changes**: No operations that affect production VM state
- [ ] **User Approval**: Ask permission before any operational testing

### **📊 Architecture Standards**
- [ ] **No Monster Code**: Keep functions focused and manageable
- [ ] **Modular Design**: Clean interfaces and separation of concerns
- [ ] **Volume Daemon Compliance**: All volume operations via Volume Daemon API

---

## 🎯 **SUCCESS CRITERIA**

### **Technical Goals**
- [ ] ✅ **Complete Multi-Disk Test Failover**: Test VM created with OS + all data disks
- [ ] ✅ **Proper Disk Attachment**: Root disk as device ID 0, data disks as 1, 2, 3...
- [ ] ✅ **Volume Correlation**: All volumes properly tracked and correlated
- [ ] ✅ **Complete VM Functionality**: Test VM boots and operates with all disks
- [ ] ✅ **Enhanced Cleanup**: All volumes properly detached and restored to OMA

### **Validation Tests**
- [ ] ✅ **pgtest1 Multi-Disk Failover**: Both OS and data disks in test VM
- [ ] ✅ **CloudStack VM Verification**: Correct volume attachment in CloudStack
- [ ] ✅ **Device Path Validation**: Proper device ordering within test VM
- [ ] ✅ **Cleanup Verification**: Complete volume restoration after rollback
- [ ] ✅ **Regression Testing**: Single-disk VMs continue working (backward compatibility)

---

## 📋 **ARCHITECTURAL CONSIDERATIONS**

### **🔗 Volume Daemon API Extensions**
The Volume Daemon currently provides:
- **✅ AttachVolumeAsRoot**: For root disk (device ID 0)
- **❌ AttachVolume**: For data disks (device ID 1+) - **MISSING**

**Required Addition**: Standard volume attachment without root device specification.

### **🏗️ CloudStack VM Multi-Disk Support** 
CloudStack supports multiple disk attachment:
- **Root Volume**: Must be attached with `root` device type
- **Data Volumes**: Can be attached as additional volumes with auto device ID assignment
- **Device Ordering**: CloudStack maintains proper device ID sequence

### **🔄 Rollback Considerations**
Multi-disk rollback requires:
- **All volume detachment** from test VM (not just OS disk)
- **Proper volume restoration** to OMA with original device paths
- **Device path correlation** maintained throughout the process

---

## 🌟 **EXPECTED IMPACT**

### **🚀 Production Benefits**
- **Complete VM Failover**: Test and live failover for multi-disk VMs
- **Data Integrity**: All VM data (OS + application data) available in failed-over VM
- **Enterprise Readiness**: Support for complex VM configurations with multiple disks
- **Operational Confidence**: Comprehensive testing includes all VM components

### **🔧 Technical Improvements**
- **Architectural Completeness**: Unified failover supports full VM complexity
- **Volume Management**: Complete multi-disk volume lifecycle management
- **Error Recovery**: Enhanced rollback handles all VM components
- **Monitoring**: Complete visibility into multi-disk operations

---

## 📋 **CURRENT STATUS**

**Overall Progress**: 0% ⏳ **READY TO START**

**Prerequisites**: ✅ **COMPLETE**
- Multi-disk replication working with VMware key correlation
- Stable vm_disks architecture operational
- Unified failover system compatible with database changes
- Clean system state for implementation and testing

**Next Action**: Begin with **Step 1** - Volume Info Enhancement

---

**🚨 CRITICAL**: This implementation will complete the multi-disk enterprise migration platform, enabling comprehensive failover testing and production deployment for complex multi-disk VM configurations.








