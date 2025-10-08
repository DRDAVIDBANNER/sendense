# Job Sheet: Phase 1 VMware Backup Completion - Multi-Disk Bug Fixes

**Project Goal Reference:** `/sendense/project-goals/phases/phase-1-vmware-backup.md` → Task 7.6 (Integration Testing)  
**Job Sheet Location:** `job-sheets/2025-10-08-phase1-backup-completion.md`  
**Priority:** 🔴 **CRITICAL** - Production Blocker  
**Assigned:** AI Assistant + User Review  
**Started:** October 8, 2025  
**Target Completion:** October 8, 2025 (Same Day)  
**Estimated Effort:** 2.5-3 hours

---

## 📊 PROGRESS TRACKING

**Session Status:** 🟡 E2E TEST RUNNING - Infrastructure Fixed, Test In Progress  
**Time Elapsed:** 3 hours  
**Completion:** 67% (4 of 6 sections complete + API docs, E2E test running)

| Section | Status | Time | Evidence |
|---------|--------|------|----------|
| **1. Pre-Test Cleanup** | ✅ COMPLETE | 30m | Script created, tested, documented |
| **2. Disk Key Mapping** | ✅ COMPLETE | 45m | Binary v2.20.8 deployed, VERIFIED working |
| **3. qemu-nbd Cleanup** | ✅ COMPLETE | 30m | Binary v2.20.9 deployed, documented |
| **4. Error Handling** | ✅ COMPLETE | 25m | Binary v2.21.0 deployed, documented |
| **5. E2E Integration Test** | 🟡 RUNNING | 15m | Test running, 3.2GB transferred, 10 MB/s rate |
| **6. API Documentation** | ✅ COMPLETE | 20m | OMA.md, API_DB_MAPPING.md updated |
| **7. Performance Testing** | ⏳ DEFERRED | - | Will complete after E2E test finishes |

**Current Binary:** `sendense-hub-v2.21.0-error-handling` (deployed, PID 3951363)  
**E2E Test Status (backup-pgtest1-1759901593):**
- ✅ Backup API called successfully
- ✅ Disk keys CORRECT: 2000, 2001 (prevents data corruption)
- ✅ qemu-nbd processes running (PIDs 3956432, 3956438) with --shared 10
- ✅ QCOW2 files created in /backup/repository/
- ✅ SNA API called successfully by SHA
- ✅ Data flowing: 3.2 GiB transferred (disk-2000), 193K (disk-2001)
- ✅ Transfer rate: 10 MB/s sustained
- ✅ Both disks writing to SEPARATE targets (no corruption)
- 🕐 Test Duration: 5 minutes elapsed, ~3 hours estimated (sparse space will be skipped)

**Next Step:** Monitor test completion, then document final results and commit

---

## 🎯 Task Link to Project Goals

**Specific Reference:**
- **Phase:** Phase 1: VMware Backup Implementation
- **Task:** Task 7.6 - Integration Testing (Complete E2E Backup)
- **Current Status:** 85% functional infrastructure, 0% E2E completion
- **Blockers:** 4 critical bugs preventing successful backup completion

**Business Value:**
- Enables production VMware backups (Backup Edition $10/VM revenue)
- Completes Phase 1 deliverables for customer deployment
- Proves multi-disk VM backup reliability (competitive advantage vs Veeam)
- Unblocks Phase 2 (CloudStack Backups) and Phase 3 (GUI Integration)

---

## 🚨 CRITICAL BUGS TO FIX

### **Bug #1: Disk Key Mapping** (HIGH PRIORITY)
**Problem:** Both disks get VMware disk key 2000 (should be 2000, 2001)  
**Impact:** sendense-backup-client writes 102GB disk to wrong 5GB target → data corruption  
**Evidence:** `nbd_targets_string: "2000:nbd://...disk-2000,2000:nbd://...disk-2001"`  
**Expected:** `"2000:nbd://...disk-2000,2001:nbd://...disk-2001"`

### **Bug #2: qemu-nbd Process Lingering** (MEDIUM PRIORITY)
**Problem:** qemu-nbd processes stay alive after failures, lock QCOW2 files  
**Impact:** Prevents cleanup, corrupts test environment  
**Evidence:** Orphaned qemu-nbd PIDs, locked QCOW2 files in /backup/repository/

### **Bug #3: Corrupted QCOW2 Files** (MEDIUM PRIORITY)
**Problem:** Failed tests leave corrupted QCOW2 files that break subsequent tests  
**Impact:** Cannot run clean tests, unpredictable failures  
**Evidence:** Partial QCOW2 files from incomplete backups

### **Bug #4: Missing --shared Flag** (HIGH PRIORITY)
**Problem:** qemu-nbd defaults to --shared=1 (single connection)  
**Impact:** sendense-backup-client hangs waiting for 2nd connection  
**Evidence:** 10+ hours investigation discovered this root cause

---

## 📋 Task Breakdown (Checkboxes Required)

### **SECTION 1: Pre-Test Cleanup System** (30 minutes)

#### **1.1. Create Cleanup Script**
- [ ] Create `/home/oma_admin/sendense/scripts/cleanup-backup-environment.sh`
- [ ] Script kills ALL qemu-nbd processes (pkill -9 -f qemu-nbd)
- [ ] Script deletes ALL QCOW2 files from /backup/repository/
- [ ] Script kills hung sendense-backup-client processes on SNA via SSH
- [ ] Script verifies no file locks remain (lsof check)
- [ ] Script restarts SHA to clear port allocations
- [ ] Make script executable (chmod +x)
- [ ] Add comprehensive logging to script (echo statements)

#### **1.2. Test Cleanup Script**
- [ ] Run script: `./scripts/cleanup-backup-environment.sh`
- [ ] Verify qemu-nbd processes killed (ps aux | grep qemu-nbd = 0)
- [ ] Verify QCOW2 files deleted (ls -lh /backup/repository/ = empty)
- [ ] Verify no file locks (lsof | grep qcow2 = 0)
- [ ] Verify SHA restarted successfully (systemctl status sendense-hub)
- [ ] Test script runs without errors

#### **1.3. Document Cleanup Script**
- [ ] Add README.md to scripts/ directory
- [ ] Document script purpose and usage
- [ ] Add to TROUBLESHOOTING.md guide
- [ ] Update CHANGELOG.md with cleanup script addition

---

### **SECTION 2: Fix Disk Key Mapping Bug** (45 minutes)

#### **2.1. Investigation Phase**
- [ ] Check which binary is running: `ls -lh /usr/local/bin/sendense-hub`
- [ ] Verify v2.20.7-disk-key-fix deployed
- [ ] Check if disk_key_fix in binary: `strings /usr/local/bin/sendense-hub | grep -A 5 -B 5 "diskKey"`
- [ ] Review backup_handlers.go lines 330-345 for disk key calculation
- [ ] Identify if loop index `i` is correct variable

#### **2.2. Add Debug Logging**
- [ ] Add debug logs to backup_handlers.go lines 332-338
- [ ] Log: loop_index, disk_key, result.DiskID for each disk
- [ ] Log: final nbd_targets_string before returning
- [ ] Build new binary: `sendense-hub-v2.20.8-disk-key-debug`
- [ ] Deploy debug binary to SHA

#### **2.3. Test Disk Key Calculation**
- [ ] Start backup via API: `curl -X POST http://localhost:8082/api/v1/backups ...`
- [ ] Capture API response with jq: `| jq '.nbd_targets_string'`
- [ ] Check SHA logs for debug output: `journalctl -u sendense-hub -f`
- [ ] Verify disk keys: Should show "2000:nbd://...disk-2000,2001:nbd://...disk-2001"
- [ ] If wrong: Investigate why loop index `i` not working

#### **2.4. Fix Implementation**
- [ ] If binary deployment issue: Redeploy with verification
- [ ] If loop logic issue: Fix diskKey calculation algorithm
- [ ] Option A: Use loop index `i` (diskKey := i + 2000)
- [ ] Option B: Use DiskID from database (diskKey := vmDisk.DiskID + 2000)
- [ ] Option C: Don't use unit_number (unreliable - both disks = 0)
- [ ] Build fixed binary: `sendense-hub-v2.20.9-disk-key-final`
- [ ] Deploy fixed binary to SHA

#### **2.5. Verify Fix**
- [ ] Run cleanup script
- [ ] Test backup API with pgtest1 (2 disks)
- [ ] Verify nbd_targets_string shows unique keys (2000, 2001)
- [ ] Verify disk-2000 gets 102GB NBD target
- [ ] Verify disk-2001 gets 5GB NBD target
- [ ] Test with curl showing correct response

#### **2.6. Document Fix**
- [ ] Update CHANGELOG.md with bug fix details
- [ ] Document root cause in TROUBLESHOOTING.md
- [ ] Add test case to validation checklist
- [ ] Update API_REFERENCE.md with correct nbd_targets_string format

---

### **SECTION 3: Enhance qemu-nbd Cleanup** (30 minutes)

#### **3.1. Add --shared=10 Flag** (CRITICAL!)
- [ ] Locate qemu-nbd Start() in services/qemu_nbd_manager.go
- [ ] Add `--shared=10` flag to qemu-nbd command (allows 10 concurrent connections)
- [ ] Verify command: `qemu-nbd --shared=10 --format=qcow2 --export-name=... -p ... <qcow2_path>`
- [ ] Test locally: Start qemu-nbd manually with --shared=10
- [ ] Verify multiple connections work (libnbd test)

#### **3.2. Improve Stop() Method**
- [ ] Enhance QemuNBDManager.Stop() with proper cleanup
- [ ] Add SIGTERM with 5-second timeout
- [ ] Add SIGKILL fallback if SIGTERM timeout
- [ ] Add 100ms sleep after kill for kernel to release lock
- [ ] Verify QCOW2 file unlocked after stop (lsof check)
- [ ] Add port release via portAllocator.Release(port)
- [ ] Add comprehensive error logging

#### **3.3. Test qemu-nbd Cleanup**
- [ ] Start qemu-nbd on test port (10150)
- [ ] Call Stop() method programmatically
- [ ] Verify process exits (ps aux | grep qemu-nbd)
- [ ] Verify QCOW2 unlocked (lsof | grep qcow2)
- [ ] Verify port released (check port allocator state)
- [ ] Test force-kill scenario (process hangs)

#### **3.4. Document qemu-nbd Management**
- [ ] Update ARCHITECTURE.md with qemu-nbd lifecycle
- [ ] Document --shared=10 requirement
- [ ] Add to TROUBLESHOOTING.md (orphaned processes)
- [ ] Update CHANGELOG.md with cleanup improvements

---

### **SECTION 4: Improve StartBackup Error Handling** (20 minutes)

#### **4.1. Enhance Cleanup Defer Block**
- [ ] Update backup_handlers.go lines 204-217 with comprehensive cleanup
- [ ] Cleanup ALL qemu-nbd processes on ANY failure
- [ ] Release ALL allocated NBD ports on ANY failure
- [ ] Delete ALL created QCOW2 files on ANY failure
- [ ] Add cleanup verification logging
- [ ] Add cleanup success/failure reporting

#### **4.2. Test Failure Scenarios**
- [ ] Test: Port allocation failure (exhaust port range)
- [ ] Test: QCOW2 creation failure (disk full)
- [ ] Test: qemu-nbd startup failure (invalid path)
- [ ] Test: SNA API call failure (network down)
- [ ] Test: VMware credential failure (invalid creds)
- [ ] Verify cleanup runs in ALL failure cases

#### **4.3. Document Error Handling**
- [ ] Update API_REFERENCE.md with error responses
- [ ] Document cleanup behavior in failure scenarios
- [ ] Add to TROUBLESHOOTING.md (failure recovery)
- [ ] Update CHANGELOG.md with error handling improvements

---

### **SECTION 5: End-to-End Integration Test** (45 minutes)

#### **5.1. Test Preparation**
- [ ] Run cleanup script: `./scripts/cleanup-backup-environment.sh`
- [ ] Verify environment clean (no qemu-nbd, no QCOW2s)
- [ ] Verify pgtest1 in database (2 disks: 102GB + 5GB)
- [ ] Verify VMware credentials in database (ID 35 with plaintext password)
- [ ] Verify SSH tunnel active (101 ports forwarded)
- [ ] Verify SNA accessible via reverse tunnel (port 9081)

#### **5.2. Execute Full Backup Test**
- [ ] Start backup via API:
  ```bash
  curl -X POST http://localhost:8082/api/v1/backups \
    -H "Content-Type: application/json" \
    -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}' \
    | jq '.'
  ```
- [ ] Capture backup_id from response
- [ ] Verify nbd_targets_string shows correct disk keys (2000, 2001)
- [ ] Verify disk_results shows 2 disks with unique ports

#### **5.3. Monitor Backup Execution**
- [ ] Monitor SHA logs: `journalctl -u sendense-hub -f`
- [ ] Monitor SNA logs: `ssh vma@10.0.100.231 tail -f /var/log/sendense/*.log`
- [ ] Verify qemu-nbd processes started (2 processes, PIDs in logs)
- [ ] Verify NBD exports accessible (netstat -tlnp | grep 10104-10105)
- [ ] Verify sendense-backup-client started on SNA
- [ ] Monitor QCOW2 file growth: `watch -n 1 ls -lh /backup/repository/`

#### **5.4. Verify Backup Success**
- [ ] Wait for backup completion (102GB + 5GB transfer)
- [ ] Verify no "request out of bounds" errors
- [ ] Verify QCOW2 files created with correct sizes
- [ ] Verify sendense-backup-client completed successfully
- [ ] Verify backup_jobs status = "completed" in database
- [ ] Verify change_id stored in vm_disks table

#### **5.5. Verify Cleanup**
- [ ] Check qemu-nbd processes stopped (ps aux | grep qemu-nbd)
- [ ] Check NBD ports released (check port allocator)
- [ ] Check QCOW2 files unlocked (lsof | grep qcow2)
- [ ] Check no hung processes remain

#### **5.6. Test Failure Scenario**
- [ ] Run cleanup script
- [ ] Start backup with invalid credential (to trigger failure)
- [ ] Verify comprehensive cleanup runs
- [ ] Verify ALL qemu-nbd stopped
- [ ] Verify ALL ports released
- [ ] Verify ALL QCOW2 files deleted
- [ ] Verify environment clean for next test

---

### **SECTION 6: Performance & Concurrent Testing** (30 minutes)

#### **6.1. Performance Baseline**
- [ ] Run single VM backup (pgtest1)
- [ ] Measure throughput: bytes_transferred / duration
- [ ] Record transfer speed: Expected >100 Mbps via SSH tunnel
- [ ] Compare to baseline: Direct TCP ~130 Mbps
- [ ] Document performance metrics

#### **6.2. Concurrent Backup Test**
- [ ] Start 3 backups simultaneously (different VMs)
- [ ] Verify port allocation works (unique ports)
- [ ] Verify all backups proceed in parallel
- [ ] Verify no resource conflicts
- [ ] Measure aggregate throughput
- [ ] Document concurrent job results

#### **6.3. Document Performance**
- [ ] Update CHANGELOG.md with performance results
- [ ] Add performance section to TROUBLESHOOTING.md
- [ ] Document concurrent job limits
- [ ] Update Phase 1 completion report with metrics

---

## 📝 Documentation Updates Required

### **API Documentation**
- [ ] Update `/source/current/api-documentation/API_REFERENCE.md`
  - Document nbd_targets_string format for multi-disk VMs
  - Add disk_results array structure
  - Document error responses for backup failures
  - Add examples for 2-disk and 3-disk VMs

### **Database Schema**
- [ ] Update `/source/current/api-documentation/DB_SCHEMA.md`
  - Verify backup_jobs table documented
  - Document disk_id column usage
  - Document nbd_targets_string storage (if needed)

### **Changelog**
- [ ] Update `/sendense/start_here/CHANGELOG.md`
  - Add Bug #1: Disk key mapping fix
  - Add Bug #2: qemu-nbd cleanup improvements
  - Add Bug #3: --shared=10 flag requirement
  - Add Bug #4: Comprehensive error handling
  - Document performance results
  - Mark Task 7.6 integration testing COMPLETE

### **Architecture Documentation**
- [ ] Update `/sendense/docs/ARCHITECTURE.md` (if exists)
  - Document qemu-nbd lifecycle management
  - Document port allocation system
  - Document cleanup procedures
  - Document multi-disk backup flow

### **Troubleshooting Guide**
- [ ] Update `/sendense/docs/TROUBLESHOOTING.md` (create if needed)
  - Add "Orphaned qemu-nbd processes" section
  - Add "Corrupted QCOW2 files" section
  - Add "Wrong disk mapping" section
  - Add cleanup script usage
  - Add failure recovery procedures

### **Project Goals**
- [ ] Update `/sendense/project-goals/phases/phase-1-vmware-backup.md`
  - Mark Task 7.6 COMPLETE
  - Update Phase 1 status to 100% (or final %)
  - Document completion evidence
  - Update success metrics

---

## 🧪 Testing Requirements

### **Unit Tests** (Optional - Time Permitting)
- [ ] Test disk key calculation logic
- [ ] Test port allocation/release
- [ ] Test qemu-nbd start/stop
- [ ] Test cleanup script functionality

### **Integration Tests** (MANDATORY)
- [ ] Single-disk VM backup (baseline)
- [ ] Multi-disk VM backup (pgtest1: 2 disks)
- [ ] 3+ disk VM backup (if available)
- [ ] Concurrent backups (3 VMs simultaneously)
- [ ] Failure scenarios (5 test cases)

### **End-to-End Tests** (MANDATORY)
- [ ] Full backup start-to-finish (pgtest1)
- [ ] Incremental backup (after full)
- [ ] File-level restore (mount backup, extract files)
- [ ] Backup chain validation (full + incrementals)

### **Performance Tests** (MANDATORY)
- [ ] Single VM throughput (>100 Mbps)
- [ ] Concurrent job throughput (aggregate)
- [ ] Resource usage (CPU, memory, disk I/O)
- [ ] SSH tunnel overhead measurement

---

## ✅ Acceptance Criteria (Must All Be Met)

### **Functional Criteria**
- [ ] ✅ Multi-disk VM backup completes without errors
- [ ] ✅ Disk key mapping correct: 2000, 2001, 2002... (unique per disk)
- [ ] ✅ sendense-backup-client writes to correct NBD targets
- [ ] ✅ No "request out of bounds" errors
- [ ] ✅ QCOW2 files created with correct data
- [ ] ✅ qemu-nbd processes start with --shared=10
- [ ] ✅ qemu-nbd processes cleanup properly on completion/failure
- [ ] ✅ NBD ports allocated and released correctly
- [ ] ✅ Comprehensive cleanup on ANY failure

### **Performance Criteria**
- [ ] ✅ Throughput >100 Mbps via SSH tunnel
- [ ] ✅ 3+ concurrent backups work without conflicts
- [ ] ✅ No performance regression vs baseline

### **Quality Criteria**
- [ ] ✅ Cleanup script works reliably
- [ ] ✅ No orphaned qemu-nbd processes
- [ ] ✅ No corrupted QCOW2 files remain
- [ ] ✅ Environment clean after failures
- [ ] ✅ All error scenarios handled gracefully

### **Documentation Criteria**
- [ ] ✅ API_REFERENCE.md updated with multi-disk format
- [ ] ✅ CHANGELOG.md updated with all bug fixes
- [ ] ✅ TROUBLESHOOTING.md includes failure recovery
- [ ] ✅ Phase 1 status updated with completion evidence
- [ ] ✅ Performance metrics documented

---

## 📊 Evidence of Completion (Required)

### **Test Results**
- [ ] Link to successful backup completion logs
- [ ] Screenshots of clean API responses (nbd_targets_string)
- [ ] QCOW2 file listings showing correct sizes
- [ ] Performance metrics (throughput, duration)
- [ ] Concurrent job results (3 VMs)

### **Code Changes**
- [ ] Binary version: `sendense-hub-v2.20.X-multi-disk-fix`
- [ ] Git commit hash for fixes
- [ ] Link to backup_handlers.go changes
- [ ] Link to qemu_nbd_manager.go changes

### **Documentation Updates**
- [ ] Link to updated CHANGELOG.md
- [ ] Link to updated API_REFERENCE.md
- [ ] Link to TROUBLESHOOTING.md additions
- [ ] Link to Phase 1 completion update

### **Database Verification**
- [ ] SQL query showing successful backup_jobs
- [ ] SQL query showing correct disk_id values
- [ ] SQL query showing change_id storage

---

## 🚧 Known Limitations / Future Work

### **Current Limitations**
- Discovery sets unit_number = 0 for all disks (workaround: use loop index)
- No automatic retry on transient failures
- No progress tracking during backup (future enhancement)
- No backup validation after completion (future enhancement)

### **Future Enhancements** (Phase 2+)
- [ ] Add progress callbacks during backup
- [ ] Implement backup validation (checksum verification)
- [ ] Add automatic retry logic for transient failures
- [ ] Fix VM discovery to set correct unit_numbers
- [ ] Add backup compression support
- [ ] Add backup deduplication support

---

## 🔗 Dependencies

### **External Dependencies**
- VMware vCenter access (10.245.246.10)
- SSH tunnel active (SNA to SHA on port 443)
- Reverse tunnel active (SHA to SNA on port 9081)
- Database accessible (MariaDB on localhost:3306)
- NBD port range available (10100-10200)

### **Internal Dependencies**
- QemuNBDManager service operational
- NBDPortAllocator service operational
- VMwareCredentialService operational
- VM discovery completed (pgtest1 with 2 disks)
- Repository configured (local repository ID 1)

### **Binary Dependencies**
- sendense-hub binary (SHA)
- sendense-backup-client binary (SNA)
- sna-api binary (SNA)
- qemu-nbd tool installed
- qemu-img tool installed

---

## 🎯 Success Metrics

### **Technical Success**
- ✅ **100% E2E test pass rate** (multi-disk backup completes)
- ✅ **Zero disk mapping errors** (correct NBD targets)
- ✅ **Zero orphaned processes** (clean qemu-nbd management)
- ✅ **>100 Mbps throughput** (SSH tunnel performance)
- ✅ **3+ concurrent backups** (port allocation system)

### **Quality Success**
- ✅ **Zero linter errors** (code quality maintained)
- ✅ **100% documentation currency** (all docs updated)
- ✅ **Complete CHANGELOG** (all changes documented)
- ✅ **Comprehensive troubleshooting** (failure recovery documented)

### **Business Success**
- ✅ **Phase 1 completion** (VMware backups operational)
- ✅ **Production readiness** (per .cursorrules criteria)
- ✅ **Customer demo ready** (reliable multi-disk backups)
- ✅ **Competitive advantage** (multi-disk consistency vs Veeam)

---

## 📞 Escalation

### **Immediate Escalation Required**
- Any show-stopping bugs discovered during testing
- Data corruption in QCOW2 files
- Performance regression >20%
- Architecture violations discovered

### **Standard Escalation**
- Test failures after 2 attempts
- Unexpected disk mapping behavior
- Port allocation exhaustion
- SSH tunnel instability

---

## 🎓 Lessons Learned (Post-Completion)

### **What Worked Well**
- [ ] Document what worked (fill after completion)

### **What Didn't Work**
- [ ] Document failures (fill after completion)

### **Process Improvements**
- [ ] Document improvements needed (fill after completion)

---

## 📋 Session Checklist (For AI Assistant)

### **Before Starting Work**
- [ ] Read FINAL-SESSION-STATUS-2025-10-07.md for current state
- [ ] Review .cursorrules for compliance requirements
- [ ] Check project rules (no simulations, evidence required)
- [ ] Verify source authority (source/current/ only)

### **During Work**
- [ ] Test EVERY change before claiming complete
- [ ] Document EVERY code change
- [ ] Update CHANGELOG.md in real-time
- [ ] No "production ready" claims without evidence

### **After Completion**
- [ ] All tests pass (functional, integration, e2e)
- [ ] All documentation updated
- [ ] All evidence linked
- [ ] Phase 1 status updated
- [ ] Sign-off requested

---

**Job Sheet Owner:** AI Assistant  
**Reviewer:** User  
**Project Goals Link:** `/sendense/project-goals/phases/phase-1-vmware-backup.md` (Task 7.6)  
**Completion Status:** 🔴 **IN PROGRESS**

**Per .cursorrules: No completion claims without evidence. Test, document, verify before marking complete.** ✅
