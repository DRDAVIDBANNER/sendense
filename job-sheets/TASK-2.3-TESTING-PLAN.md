# Task 2.3: NBD File Export Testing & Validation

**Status:** 🟡 PAUSED - Waiting for NBD server deployment on dev server  
**Date:** 2025-10-05  
**Prerequisites:** NBD server must be installed and configured on dev server

---

## 🎯 OBJECTIVE

Complete Task 2 to 100% by implementing comprehensive testing and validation of the NBD file export system.

**Current Task 2 Progress:** 67% (Phase 2.1 and 2.2 complete)  
**Target:** 100% production-ready

---

## ✅ COMPLETED (Phase 2.1 & 2.2)

- ✅ NBD Config Manager (`nbd_config_manager.go` - 512 lines)
- ✅ Backup Export Helpers (`backup_export_helpers.go` - 232 lines)
- ✅ FileExport model in `models.go`
- ✅ File export methods in `server.go` (CreateFileExport, RemoveFileExport, ListFileExports)
- ✅ Initial unit test suite (`backup_export_helpers_test.go` - 285 lines)

**Commit:** 2cf590d

---

## 🧪 TESTING PLAN (Phase 2.3)

### **1. Unit Tests** (In Progress)

**File:** `source/current/oma/nbd/backup_export_helpers_test.go`

**Coverage:**
- ✅ BuildBackupExportName() - collision-proof naming
- ✅ IsBackupExport() - export type detection
- ✅ ParseBackupExportName() - name parsing
- ✅ GetQCOW2FileSize() - qemu-img integration
- ✅ ValidateQCOW2File() - file validation

**Still Needed:**
- [ ] NBD config manager tests
- [ ] Export lifecycle tests
- [ ] Error handling tests

### **2. Integration Tests** (Pending NBD Server)

**What to Test:**
- [ ] Create QCOW2 file and export via NBD
- [ ] Verify export appears in NBD server config.d
- [ ] Test SIGHUP reload without service restart
- [ ] Remove export and verify cleanup
- [ ] Multiple concurrent exports (migration + backup)
- [ ] Export name uniqueness under load

**Test Scenarios:**
```bash
# Create test QCOW2 files
qemu-img create -f qcow2 /tmp/test-full-backup.qcow2 1G
qemu-img create -f qcow2 -b /tmp/test-full-backup.qcow2 /tmp/test-incr-backup.qcow2

# Export via NBD (once server deployed)
# Call CreateFileExport() and verify NBD export created
# Use nbd-client or qemu-nbd to connect and verify

# Performance test
# Write data to QCOW2 export and measure throughput
```

### **3. File Locking & Concurrent Access** (Critical)

**Safety Tests:**
- [ ] Multiple writers to same QCOW2 file (should fail gracefully)
- [ ] Read-write vs read-only export modes
- [ ] Proper file locking during backup operations
- [ ] Cleanup on connection failure

**Implementation Needed:**
```go
// File locking wrapper for QCOW2 exports
func acquireFileLock(qcow2Path string) (*os.File, error) {
    // flock() implementation
}

func releaseFileLock(lockFile *os.File) error {
    // Cleanup implementation
}
```

### **4. Performance Validation** (Production Baseline)

**Requirements:**
- [ ] Maintain 3.2 GiB/s baseline for block device exports
- [ ] Measure QCOW2 file export performance (target: >1 GiB/s)
- [ ] Test with various QCOW2 file sizes (10GB, 100GB, 500GB)
- [ ] Concurrent export performance (multiple backups simultaneously)

**Benchmark Test:**
```bash
# Write 10GB to QCOW2 export via NBD
dd if=/dev/zero of=/dev/nbd0 bs=1M count=10240 oflag=direct

# Measure throughput
# Target: >1 GiB/s for QCOW2, maintain 3.2 GiB/s for block devices
```

### **5. SIGHUP Reload Validation** (No Restarts)

**Critical Test:**
- [ ] Add export via SIGHUP while clients connected
- [ ] Remove export via SIGHUP while clients connected
- [ ] Verify existing connections unaffected
- [ ] Confirm new export immediately available

**Test Script:**
```bash
# Connect client to export A
nbd-client 127.0.0.1 10809 export-a /dev/nbd0

# Add export B via SIGHUP (should not disconnect A)
# Verify export B available
# Verify export A still connected
```

### **6. Capture Agent Integration** (End-to-End)

**Full Workflow Test:**
- [ ] Create backup QCOW2 file via BackupEngine
- [ ] Generate NBD export with unique name
- [ ] Trigger VMA (Capture Agent) to connect
- [ ] Transfer data via NBD
- [ ] Verify data integrity (checksum)
- [ ] Clean up export after completion

---

## 📋 NBD SERVER DEPLOYMENT REQUIREMENTS

**Before testing can continue, dev server needs:**

1. **NBD Server Installation:**
   ```bash
   sudo apt-get install nbd-server nbd-client
   ```

2. **Directory Structure:**
   ```
   /opt/migratekit/nbd-configs/
   ├── nbd-server.conf          # Base config with includedir
   └── conf.d/                  # Individual export files
   ```

3. **Base Configuration:**
   ```ini
   [generic]
   port = 10809
   user = nbd
   group = nbd
   includedir = /opt/migratekit/nbd-configs/conf.d
   ```

4. **Systemd Service:**
   - Start: `systemctl start nbd-server`
   - Enable: `systemctl enable nbd-server`
   - Reload: `kill -SIGHUP $(cat /var/run/nbd-server.pid)`

5. **Permissions:**
   ```bash
   sudo mkdir -p /opt/migratekit/nbd-configs/conf.d
   sudo chown -R nbd:nbd /opt/migratekit/nbd-configs
   sudo chmod 755 /opt/migratekit/nbd-configs
   sudo chmod 755 /opt/migratekit/nbd-configs/conf.d
   ```

---

## 🎯 SUCCESS CRITERIA (Task 2.3)

All must pass before marking Task 2 as 100% complete:

- [ ] Unit tests: 100% passing (go test -v ./nbd)
- [ ] Integration tests: All scenarios passing
- [ ] File locking: Prevents corruption under concurrent access
- [ ] Performance: Maintains 3.2 GiB/s for block devices, >1 GiB/s for QCOW2
- [ ] SIGHUP reload: Works without disconnecting clients
- [ ] Capture Agent: Can connect and transfer data successfully
- [ ] Export naming: No collisions under stress testing
- [ ] Documentation: All acceptance criteria met

---

## 📝 WHEN READY TO CONTINUE

**Tell me when NBD server is deployed, and I'll:**

1. ✅ Complete the unit test suite
2. ✅ Implement integration tests
3. ✅ Add file locking safety mechanisms
4. ✅ Create performance benchmarks
5. ✅ Run comprehensive validation
6. ✅ Update documentation with results
7. ✅ Mark Task 2 as 100% complete

**Estimated Time:** 2-3 hours once NBD server is ready

---

## 🔗 REFERENCES

- **Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-05-nbd-file-export.md`
- **Project Goals:** `/home/oma_admin/sendense/project-goals/phases/phase-1-vmware-backup.md` (lines 118-227)
- **NBD Config Manager:** `/home/oma_admin/sendense/source/current/oma/nbd/nbd_config_manager.go`
- **Backup Helpers:** `/home/oma_admin/sendense/source/current/oma/nbd/backup_export_helpers.go`
- **Current Tests:** `/home/oma_admin/sendense/source/current/oma/nbd/backup_export_helpers_test.go`

---

**Last Updated:** 2025-10-05  
**Status:** Waiting for NBD server deployment  
**Next Action:** Continue testing when environment ready
