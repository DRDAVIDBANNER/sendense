# Job Sheet: NBD File Export for Backup Operations

**Date Created:** 2025-10-05  
**Status:** 🔴 **READY TO START**  
**Project Goal Link:** [project-goals/phases/phase-1-vmware-backup.md → Task 2: NBD File Export]  
**Duration:** 1-2 weeks  
**Priority:** Critical (Foundation for backup workflows)

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`  
**Task Section:** **Task 2: Modify NBD Server for File Export** (Lines 118-227)  
**Sub-Tasks:** **2.1 config.d Pattern, 2.2 File Export Support, 2.3 QCOW2 Features**  
**Business Value:** Foundation for VMware backup workflows using QCOW2 files  
**Success Criteria:** NBD server can export both block devices and QCOW2 files with no service restarts

**Task Description (From Project Goals):**
```
Goal: Extend NBD server to export files (not just block devices)
Current State: NBD server exports /dev/vdX block devices
New State: NBD server can also export QCOW2 files

Architecture Decision: Follow Volume Daemon pattern with config.d + SIGHUP
Benefits:
- No service restarts for backup export management
- Proven architecture from Volume Daemon
- Atomic operations with individual export files
- Clean separation between backup and migration exports
```

**Acceptance Criteria (From Project Goals):**
- [ ] NBD server migrated to config.d pattern with SIGHUP reload
- [ ] NBD server can export QCOW2 files alongside block devices
- [ ] Backup exports use unique naming scheme (no collisions with migrations)
- [ ] Export names support multi-disk VMs and multiple backup types
- [ ] Export names remain under 64 character NBD limit
- [ ] Capture Agent can connect to file exports (same as block device exports)
- [ ] Data writes to QCOW2 file correctly with proper file locking
- [ ] No regression on existing block device exports
- [ ] Export add/remove operations use SIGHUP (no service restart)

---

## 🔗 DEPENDENCY STATUS

### **Required Before Starting:**
- ✅ Task 1 completed (Repository infrastructure operational)
- ✅ Volume Daemon NBD pattern analyzed (proven architecture)
- ✅ Export naming strategy defined (collision prevention)
- ✅ Architecture decisions documented

### **Blocks These Tasks:**
- ⏸️ Task 3: Backup Workflow Implementation (needs NBD file export)
- ⏸️ Task 4: File-Level Restore (needs QCOW2 NBD exports)

---

## 📋 JOB BREAKDOWN (Detailed Implementation)

### **Phase 1: Migrate to config.d Pattern (Days 1-3)**

- [ ] **Analyze Volume Daemon NBD Implementation**
  - **File:** `source/current/volume-daemon/nbd/config_manager.go`
  - **Study:** SIGHUP reload mechanism, config.d management, export lifecycle
  - **Evidence:** Understanding of proven architecture patterns

- [ ] **Create Base Configuration Structure**
  - **File:** `source/current/oma/nbd/config_base.go`
  - **Implementation:** Base NBD config with `includedir` directive
  - **Config Path:** `/opt/migratekit/nbd-configs/nbd-server.conf`
  - **Evidence:** Base config file created with proper includedir

- [ ] **Implement Individual Export File Management**
  - **Directory:** `/opt/migratekit/nbd-configs/conf.d/`
  - **Pattern:** One `.conf` file per export
  - **Naming:** Match export names for easy correlation
  - **Evidence:** Export files created/deleted atomically

- [ ] **Add SIGHUP Reload Functionality**
  - **Method:** `reloadNBDServer()` following Volume Daemon pattern
  - **Implementation:** Send SIGHUP signal to NBD process
  - **Error Handling:** Graceful failure without breaking existing exports
  - **Evidence:** Config changes applied without service restart

- [ ] **Migration Testing**
  - **Validation:** Existing migration exports continue working
  - **Test:** Add/remove exports via SIGHUP
  - **Evidence:** No regression in existing functionality

### **Phase 2: File Export Support (Days 4-6)**

- [ ] **Add FileExport Model**
  - **File:** `source/current/oma/nbd/models.go`
  - **Type:** FileExport struct with file-specific fields
  - **Fields:** FilePath, FileSize, ReadWrite, BackupType
  - **Evidence:** Clear separation between block and file exports

- [ ] **Implement CreateFileExport Method**
  - **File:** `source/current/oma/nbd/server.go`
  - **Method:** `CreateFileExport(filePath, exportName string) error`
  - **Integration:** Use config.d pattern for file exports
  - **Evidence:** QCOW2 files exported via NBD successfully

- [ ] **QCOW2 File Size Detection**
  - **Command:** `qemu-img info --output=json`
  - **Parsing:** Extract virtual-size field for NBD export
  - **Validation:** Ensure accurate file size reporting
  - **Evidence:** NBD exports report correct QCOW2 sizes

- [ ] **Export Name Generation**
  - **Function:** `BuildBackupExportName()` implementation
  - **Format:** `backup-{vmContextID}-disk{diskID}-{backupType}-{timestamp}`
  - **Length Limit:** Handle 64-character NBD limit with truncation
  - **Evidence:** Unique names prevent collisions with migrations

### **Phase 3: QCOW2-Specific Features (Days 7-10)**

- [ ] **Read-Write File Export Support**
  - **Mode:** Support both read-only and read-write QCOW2 exports
  - **Use Case:** Read-write needed for incremental backup writes
  - **Validation:** File locking prevents corruption
  - **Evidence:** Incremental backups can write to QCOW2 via NBD

- [ ] **File Locking and Safety**
  - **Implementation:** Proper file locking for concurrent access
  - **Error Handling:** Detect and prevent file conflicts
  - **Cleanup:** Release locks when exports removed
  - **Evidence:** No file corruption under concurrent access

- [ ] **Integration with Existing NBD Server**
  - **Coexistence:** Block device and file exports in same server
  - **Port Management:** Both export types use same port (10809)
  - **Config Separation:** Clear distinction in configuration
  - **Evidence:** Mixed export types work simultaneously

### **Phase 4: Testing & Validation (Days 8-10)**

- [ ] **Capture Agent Connectivity Testing**
  - **Test:** VMA can connect to file exports
  - **Validation:** Same connectivity as block device exports
  - **Protocol:** Ensure NBD protocol compatibility
  - **Evidence:** Successful NBD connections to QCOW2 files

- [ ] **Performance Validation**
  - **Baseline:** Maintain existing block device performance
  - **File Performance:** Measure QCOW2 export performance
  - **Comparison:** Document any performance differences
  - **Evidence:** Performance metrics within acceptable ranges

- [ ] **Stress Testing**
  - **Multiple Exports:** Many backup exports simultaneously
  - **SIGHUP Frequency:** Rapid add/remove operations
  - **Long Running:** Extended operation stability
  - **Evidence:** System remains stable under load

- [ ] **Integration Testing**
  - **Mixed Operations:** Block and file exports together
  - **Migration Compatibility:** Existing migrations unaffected
  - **Export Lifecycle:** Full create/use/delete cycle
  - **Evidence:** Complete workflow operates correctly

---

## 🏗️ TECHNICAL ARCHITECTURE

### **Export Naming Strategy**
```go
// BuildBackupExportName generates collision-proof export names
func BuildBackupExportName(vmContextID string, diskID int, backupType string, timestamp time.Time) string {
    timestampStr := timestamp.Format("20060102T150405")
    exportName := fmt.Sprintf("backup-%s-disk%d-%s-%s", 
        vmContextID, diskID, backupType, timestampStr)
    
    // Handle NBD 64-character limit
    if len(exportName) > 63 {
        maxContextLen := 63 - len(fmt.Sprintf("backup--disk%d-%s-%s", diskID, backupType, timestampStr))
        if maxContextLen > 0 {
            truncatedContext := vmContextID[:maxContextLen]
            exportName = fmt.Sprintf("backup-%s-disk%d-%s-%s", 
                truncatedContext, diskID, backupType, timestampStr)
        }
    }
    
    return exportName
}
```

### **Directory Structure**
```
/opt/migratekit/nbd-configs/
├── nbd-server.conf          # Base config with includedir
└── conf.d/                  # Individual export files
    ├── backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000.conf
    ├── backup-ctx-pgtest2-20251005-120000-disk0-incr-20251005T130000.conf
    └── migration-vm-a1b2c3d4-e5f6-7890-abcd-ef1234567890-disk0.conf
```

### **File Export Configuration Template**
```ini
[backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000]
exportname = /opt/backups/pgtest2/disk0/backup-full-20251005T120000.qcow2
readonly = false
multifile = false
copyonwrite = false
```

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **Architecture Migration:** Config.d pattern operational with SIGHUP
- [ ] **File Export Capability:** QCOW2 files exported successfully via NBD
- [ ] **Naming Compliance:** Unique export names prevent all collisions
- [ ] **Performance Maintenance:** No degradation in existing functionality
- [ ] **Integration Success:** Capture Agent connects to file exports
- [ ] **Concurrent Operations:** Block device and file exports coexist
- [ ] **Service Reliability:** SIGHUP reload without service interruption

### **Testing Evidence Required**
- [ ] VMA successfully connects to backup QCOW2 file via NBD
- [ ] Multiple export types (migration + backup) operate simultaneously
- [ ] SIGHUP reload adds/removes exports without disconnecting clients
- [ ] Export names comply with length limits and uniqueness requirements
- [ ] File locking prevents corruption during concurrent access
- [ ] Performance metrics show acceptable QCOW2 export performance

---

## 🚨 PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All code in `source/current/` only
- ✅ **Volume Daemon Pattern:** Follow proven config.d + SIGHUP architecture
- ✅ **No Service Restarts:** SIGHUP only for backup export management
- ✅ **Collision Prevention:** Implement comprehensive naming strategy
- ✅ **Backward Compatibility:** No regression in migration functionality
- ✅ **Modular Design:** Clean separation between block and file exports
- ✅ **Error Handling:** Graceful failures with comprehensive logging
- ✅ **No Simulations:** Real QCOW2 file operations only

### **Architecture Constraints:**
- **NBD Port:** Continue using single port 10809 for all exports
- **Export Limit:** NBD export names must be <64 characters
- **File Access:** Proper locking for read-write QCOW2 exports
- **Performance:** Maintain 3.2 GiB/s baseline for block device exports

---

## 📊 DELIVERABLES

### **Code Deliverables**
- `source/current/oma/nbd/server.go` - Enhanced with file export support
- `source/current/oma/nbd/config.go` - Migrated to config.d pattern
- `source/current/oma/nbd/models.go` - FileExport type and naming functions
- `source/current/oma/nbd/config_base.go` - Base configuration management

### **Configuration Deliverables**
- `/opt/migratekit/nbd-configs/nbd-server.conf` - Base NBD configuration
- `/opt/migratekit/nbd-configs/conf.d/` - Individual export file management
- SIGHUP reload mechanism for dynamic export management

### **Documentation Deliverables**
- Updated API documentation if new public interfaces added
- Architecture documentation for file export patterns
- Performance benchmarks for QCOW2 vs block device exports

---

## 🔗 INTEGRATION POINTS

### **Upstream Dependencies**
- **Task 1:** Repository infrastructure provides QCOW2 backup files
- **Volume Daemon:** NBD config.d pattern reference implementation
- **Existing NBD:** Migration exports must continue working unchanged

### **Downstream Consumers**
- **Task 3:** Backup Workflow will create file exports for backup operations
- **Task 4:** File-Level Restore will mount QCOW2 backups via NBD
- **Capture Agent:** Will connect to file exports same as block exports

---

## 🎯 PHASE 1 CONTEXT

**Current Phase 1 Progress:**
```
Task 1: Repository Abstraction     [██████████] 100% ✅ COMPLETE (2,098 lines)
Task 2: NBD File Export            [▱▱▱▱▱▱▱▱▱▱]   0% 🔴 THIS JOB
Task 3: Backup Workflow            [▱▱▱▱▱▱▱▱▱▱]   0% ⏸️ Waiting
Task 4: File-Level Restore         [▱▱▱▱▱▱▱▱▱▱]   0% ⏸️ Waiting
```

**This job is critical** - it enables the entire backup workflow architecture. Get the NBD file export foundation right, and the rest of Phase 1 becomes straightforward.

---

**THIS JOB ENABLES VMWARE BACKUP OPERATIONS**

**FILE-BASED BACKUP INFRASTRUCTURE FOUNDATION**

---

**Job Owner:** Backend Engineering Team  
**Reviewer:** Architecture Lead  
**Status:** 🔴 Ready to Start  
**Last Updated:** 2025-10-05
