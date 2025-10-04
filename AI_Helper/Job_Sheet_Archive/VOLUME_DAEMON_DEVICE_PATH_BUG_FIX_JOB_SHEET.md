# Volume Daemon Device Path Duplicate Bug Fix - Job Sheet

**Created**: September 28, 2025  
**Priority**: 🚨 **CRITICAL** - Blocks multi-disk VM replication  
**Status**: 📋 **READY FOR EXECUTION**  
**Affected System**: Volume Daemon device mapping creation  
**Bug ID**: VD-2025-0928-001

---

## 🎯 **PROBLEM STATEMENT**

**Issue**: Multi-disk VM replication fails during volume provisioning due to duplicate device path constraint violation in Volume Daemon.

**Root Cause**: Device path generation logic in `volume_service.go:467` only uses VM ID, causing all volumes for the same VM to generate identical paths, violating the database's `UNIQUE KEY unique_device_path` constraint.

**Impact**: 
- ❌ Multi-disk VMs cannot complete replication
- ❌ Second and subsequent disks fail to create device mappings
- ❌ Replication jobs stuck in "provisioning" status
- ✅ Single-disk VMs work normally

---

## 📋 **JOB EXECUTION CHECKLIST**

### **Phase 1: Pre-Fix Assessment** ⏱️ *Est: 10 minutes*
- [ ] **Task 1.1**: Document current failed state
  - [ ] Check pgtest1 replication job status
  - [ ] List failed volume operations
  - [ ] Count device mappings with duplicate path pattern
  - [ ] Verify CloudStack volumes are properly attached

- [ ] **Task 1.2**: Backup current Volume Daemon state
  - [ ] Create database backup of affected tables
  - [ ] Backup current `volume_service.go` file
  - [ ] Document current service version/commit

### **Phase 2: Code Fix Implementation** ⏱️ *Est: 15 minutes*
- [ ] **Task 2.1**: Fix device path generation logic
  - [ ] Update `volume_service.go:467` to include volume UUID
  - [ ] Update `volume_service.go:605` (root attachment path) if needed
  - [ ] Ensure consistent path format across all attachment methods
  - [ ] Add validation to prevent future duplicates

- [ ] **Task 2.2**: Code review and testing
  - [ ] Review all `remote-vm-` path generation instances
  - [ ] Verify path uniqueness logic
  - [ ] Check for any hardcoded path assumptions elsewhere

### **Phase 3: Database Cleanup** ⏱️ *Est: 10 minutes*
- [ ] **Task 3.1**: Clean failed operations
  - [ ] Delete failed volume operation `eb5032e1-8461-49ea-8756-3e4fa4000399`
  - [ ] Reset any stuck replication job status
  - [ ] Verify database consistency

- [ ] **Task 3.2**: Prepare for retry
  - [ ] Ensure no orphaned device mappings
  - [ ] Verify CloudStack volumes are still attached
  - [ ] Clear any cached state

### **Phase 4: Deployment & Testing** ⏱️ *Est: 20 minutes*
- [ ] **Task 4.1**: Deploy fix to remote server
  - [ ] Build updated Volume Daemon binary
  - [ ] Stop Volume Daemon service
  - [ ] Deploy new binary
  - [ ] Start Volume Daemon service
  - [ ] Verify service health

- [ ] **Task 4.2**: Test fix with pgtest1
  - [ ] Retry pgtest1 replication job
  - [ ] Monitor device mapping creation
  - [ ] Verify both volumes get unique device paths
  - [ ] Confirm replication progresses beyond provisioning

### **Phase 5: Validation & Documentation** ⏱️ *Est: 10 minutes*
- [ ] **Task 5.1**: Comprehensive testing
  - [ ] Test with another multi-disk VM
  - [ ] Verify single-disk VMs still work
  - [ ] Check failover operations work with new paths

- [ ] **Task 5.2**: Update documentation
  - [ ] Document new device path format
  - [ ] Update troubleshooting guides
  - [ ] Record fix in project status

---

## 🔧 **TECHNICAL IMPLEMENTATION DETAILS**

### **Current Problematic Code:**
```go
// File: volume_service.go:467
devicePath = fmt.Sprintf("remote-vm-%s", vmID)
```

### **Fixed Code:**
```go
// File: volume_service.go:467
devicePath = fmt.Sprintf("remote-vm-%s-%s", vmID, volumeID)
```

### **Additional Locations to Check:**
- `volume_service.go:605` - Root attachment path generation
- Any NBD export name generation that depends on device paths
- Cleanup services that reference device path patterns

### **Database Impact:**
- **Table**: `device_mappings`
- **Constraint**: `UNIQUE KEY unique_device_path (device_path)`
- **New Path Format**: `remote-vm-{vm_id}-{volume_uuid}`
- **Backward Compatibility**: Existing single-disk mappings will continue to work

---

## 🧪 **TEST SCENARIOS**

### **Test Case 1: Multi-Disk VM Replication**
- **VM**: pgtest1 (2 disks: 107GB + 10GB)
- **Expected**: Both volumes get unique device paths
- **Validation**: Check `device_mappings` table for two entries with different paths

### **Test Case 2: Single-Disk VM Replication**
- **VM**: Any single-disk VM
- **Expected**: Normal replication flow continues to work
- **Validation**: Replication completes successfully

### **Test Case 3: Failover Operations**
- **VM**: Multi-disk VM with existing device mappings
- **Expected**: Failover operations work with new path format
- **Validation**: Test and live failover complete successfully

---

## 🚨 **ROLLBACK PLAN**

If the fix causes issues:

1. **Immediate Rollback**:
   ```bash
   sudo systemctl stop volume-daemon
   sudo cp /path/to/backup/volume_service.go /source/current/volume-daemon/service/
   sudo systemctl start volume-daemon
   ```

2. **Database Rollback**:
   ```sql
   -- Restore from backup if needed
   -- Remove any new device mappings created during testing
   ```

3. **Verification**:
   - Confirm original functionality restored
   - Check existing replication jobs still work

---

## 📊 **SUCCESS CRITERIA**

- [ ] ✅ pgtest1 replication job completes successfully
- [ ] ✅ Both pgtest1 disks have unique device mappings in database
- [ ] ✅ Device paths follow format: `remote-vm-{vm_id}-{volume_uuid}`
- [ ] ✅ No duplicate device path errors in Volume Daemon logs
- [ ] ✅ Single-disk VM replication still works normally
- [ ] ✅ Failover operations work with new device paths
- [ ] ✅ No regression in existing functionality

---

## 🔍 **MONITORING & VERIFICATION**

### **Key Log Patterns to Watch:**
```bash
# Success patterns
"✅ Device mapping created successfully"
"device_path=remote-vm-{vm_id}-{volume_uuid}"

# Failure patterns (should not appear)
"Duplicate entry 'remote-vm-" 
"mapping creation failed"
```

### **Database Queries for Validation:**
```sql
-- Check for duplicate device paths (should return 0)
SELECT device_path, COUNT(*) FROM device_mappings 
GROUP BY device_path HAVING COUNT(*) > 1;

-- Verify pgtest1 device mappings
SELECT device_path, volume_uuid FROM device_mappings 
WHERE vm_id='1c266316-503d-451d-9392-9585a6fcba41';
```

---

## 📝 **EXECUTION LOG**

| Task | Status | Timestamp | Notes |
|------|--------|-----------|-------|
| Pre-Fix Assessment | ⏳ Pending | | |
| Code Fix Implementation | ⏳ Pending | | |
| Database Cleanup | ⏳ Pending | | |
| Deployment & Testing | ⏳ Pending | | |
| Validation & Documentation | ⏳ Pending | | |

---

**🎯 This job sheet provides a systematic approach to fixing the critical Volume Daemon device path duplication bug that's blocking multi-disk VM replication on the remote server.**






