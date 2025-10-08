# Project Overseer: Task 2.4 Ready for Worker

**Date:** October 7, 2025  
**Task:** Multi-Disk VM Backup Support (CRITICAL)  
**Status:** 📋 **DOCUMENTED & READY FOR WORKER**  
**Priority:** 🚨 **CRITICAL** - Data corruption risk

---

## ✅ PREPARATION COMPLETE

All documentation has been prepared for worker to implement Task 2.4:

### **1. Job Sheet Updated** ✅
**File:** `job-sheets/2025-10-07-unified-nbd-architecture.md`  
**Location:** Task 2.4 (after Task 2.3, before Phase 3)  
**Content:** 
- Problem statement
- Implementation requirements
- Code patterns
- Success criteria
- Action items checklist

### **2. Worker Prompt Created** ✅
**File:** `TASK-2.4-WORKER-PROMPT.md`  
**Size:** Comprehensive 7+ page implementation guide  
**Content:**
- Step-by-step implementation (Steps 1-7)
- Complete code template for StartBackup()
- Testing checklist
- Success criteria
- Common mistakes to avoid
- Reporting format

### **3. Technical Analysis** ✅
**File:** `CRITICAL-MULTI-DISK-BACKUP-PLAN.md`  
**Size:** Detailed 11-page technical document  
**Content:**
- Problem statement with examples
- Proof that system supports this
- Evidence from SendenseBackupClient
- Evidence from replication workflow
- Before/after comparison
- Implementation plan
- Impact analysis

### **4. CHANGELOG Updated** ✅
**File:** `start_here/CHANGELOG.md`  
**Section:** `[Unreleased] -> Critical`  
**Entry:** Full documentation of Task 2.4 issue and fix

---

## 🎯 WORKER PROMPT

**Give to worker:**

```
You are working on the Sendense backup platform project.

CRITICAL TASK: Task 2.4 - Multi-Disk VM Backup Support

⚠️ SEVERITY: CRITICAL - Current implementation causes data corruption for multi-disk VMs

START BY READING (in order):
1. /home/oma_admin/sendense/TASK-2.4-WORKER-PROMPT.md (step-by-step guide)
2. /home/oma_admin/sendense/CRITICAL-MULTI-DISK-BACKUP-PLAN.md (technical analysis)
3. /home/oma_admin/sendense/job-sheets/2025-10-07-unified-nbd-architecture.md (Task 2.4 section)

OBJECTIVE:
Change backup API from disk-level to VM-level operations to maintain VMware snapshot consistency.

PROBLEM:
Current API requires 3 separate calls for 3-disk VM, creating 3 separate VMware snapshots
at different times (T0, T1, T2). This causes data corruption for database/application workloads.

SOLUTION:
- Remove disk_id from request (VM-level backups)
- Query ALL disks for VM
- Allocate NBD port for EACH disk
- Start qemu-nbd for EACH disk
- Build multi-disk NBD targets string
- Call SNA API ONCE with ALL disk targets
- Return results for ALL disks

EVIDENCE SYSTEM SUPPORTS THIS:
✅ SendenseBackupClient has --nbd-targets flag (main.go:426)
✅ Replication already handles multi-disk correctly (migration.go:337)
✅ Your job: Make backup work like replication does!

FILE TO MODIFY:
/home/oma_admin/sendense/source/current/sha/api/handlers/backup_handlers.go

ESTIMATED TIME: 3-4 hours

CRITICAL SUCCESS CRITERIA:
- Remove disk_id from BackupStartRequest
- Loop through ALL disks for VM
- Allocate port and start qemu-nbd for each disk
- Build NBD targets string (format: "key:url,key:url")
- Call SNA API once with nbd_targets
- SHA compiles cleanly
- Zero linter errors

Follow the worker prompt for detailed step-by-step instructions.

Report back after each major step (1-7).

GO! 🚀
```

---

## 📊 TASK 2.4 SUMMARY

### **Problem:**
- Current backup API accepts `disk_id` field
- Requires separate API call per disk
- Creates separate VMware snapshot per call
- **Result:** Data corruption for multi-disk VMs

### **Example of Broken Behavior:**
```
POST /api/v1/backups {"vm_name": "db", "disk_id": 0}  → Snapshot at 10:00am
POST /api/v1/backups {"vm_name": "db", "disk_id": 1}  → Snapshot at 10:05am ❌
POST /api/v1/backups {"vm_name": "db", "disk_id": 2}  → Snapshot at 10:10am ❌

Result: Disk 0 has data from 10:00, disk 1 from 10:05, disk 2 from 10:10
        → DATABASE SEES INCONSISTENT STATE → CORRUPTION!
```

### **Solution:**
- Remove `disk_id` from request
- Accept VM name only
- Loop through ALL disks in single operation
- Create ONE VMware snapshot
- Backup ALL disks from SAME snapshot
- **Result:** Consistent, safe backups

### **Example of Correct Behavior:**
```
POST /api/v1/backups {"vm_name": "db"}  → ONE Snapshot at 10:00am
  ├── Disk 0 backed up from 10:00am snapshot ✅
  ├── Disk 1 backed up from 10:00am snapshot ✅
  └── Disk 2 backed up from 10:00am snapshot ✅

Result: All disks consistent at SAME instant → SAFE TO RESTORE!
```

---

## 🔧 IMPLEMENTATION CHANGES

**File:** `sha/api/handlers/backup_handlers.go`

**Changes Required:**
1. **Request struct:** Remove `disk_id` field
2. **Response struct:** Add `disk_results` array and `nbd_targets_string`
3. **StartBackup():** Rewrite to process ALL disks in single call
4. **Cleanup:** Update defer logic to handle multiple disks

**Lines Changed:** ~250 lines  
**Complexity:** Medium  
**Time:** 3-4 hours

---

## ✅ SUCCESS CRITERIA

**Before Approval:**
- [ ] Code compiles cleanly (`go build`)
- [ ] Zero linter errors
- [ ] Request has NO `disk_id` field
- [ ] Response has `disk_results` array
- [ ] Response has `nbd_targets_string` field
- [ ] Code queries ALL disks for VM
- [ ] Code allocates port for EACH disk
- [ ] Code starts qemu-nbd for EACH disk
- [ ] NBD targets string built correctly
- [ ] SNA API called ONCE (not per-disk)
- [ ] Cleanup releases ALL ports on failure
- [ ] Cleanup stops ALL qemu-nbd on failure

**VMware Consistency:**
- [ ] ONE VM snapshot (not per-disk)
- [ ] ALL disks from SAME snapshot instant
- [ ] Application consistency maintained
- [ ] Zero data corruption risk

---

## 📋 PROJECT OVERSEER AUDIT PLAN

**When Worker Reports "Complete":**

1. **Read Completion Report**
   - Verify all success criteria met
   - Check compilation evidence

2. **Code Review**
   - Verify `disk_id` removed from request
   - Verify disk loop implementation
   - Verify NBD targets string format
   - Verify cleanup logic

3. **Compilation Test**
   ```bash
   cd /home/oma_admin/sendense/source/current/sha
   go build ./cmd/main.go
   # Verify exit code 0, binary size ~34MB
   ```

4. **Grep Verification**
   ```bash
   # Should NOT find disk_id in BackupStartRequest
   grep -A10 "type BackupStartRequest" backup_handlers.go
   
   # Should find disk_results
   grep "disk_results" backup_handlers.go
   
   # Should find nbd_targets
   grep "nbd_targets" backup_handlers.go
   ```

5. **Approval Decision**
   - If all criteria met: APPROVE
   - If issues found: Document and request fixes
   - Create completion report

---

## 🚨 BLOCKING CONDITIONS

**Task 2.4 BLOCKS:**
- Phase 2 approval
- Task 2.3 approval (current implementation incomplete)
- Any production backup deployments

**Task 2.4 is CRITICAL PATH** - Must be fixed before proceeding!

---

## 📁 DOCUMENTATION FILES

All files created and ready:

1. **`job-sheets/2025-10-07-unified-nbd-architecture.md`**
   - Task 2.4 section added (lines 608-754)
   - Full implementation requirements
   - Success criteria checklist

2. **`TASK-2.4-WORKER-PROMPT.md`**
   - 7-page comprehensive guide
   - Step-by-step implementation (1-7)
   - Complete code template
   - Testing checklist
   - Common mistakes

3. **`CRITICAL-MULTI-DISK-BACKUP-PLAN.md`**
   - 11-page technical analysis
   - Problem statement with evidence
   - Proof system supports this
   - Implementation plan
   - Before/after comparison

4. **`start_here/CHANGELOG.md`**
   - Critical section added
   - Full documentation of issue and fix
   - Estimated time: 3-4 hours

---

## ✅ READY TO ASSIGN

**Status:** 🟢 **READY FOR WORKER**  
**Priority:** 🚨 **CRITICAL**  
**Blocking:** Phase 2 approval  
**Documentation:** ✅ Complete  
**Worker Guidance:** ✅ Comprehensive  
**Overseer Audit Plan:** ✅ Defined

---

**ASSIGN TASK NOW!** 🚀

---

*Project Overseer ready to audit upon worker completion*  
*All documentation complete and compliance requirements defined*  
*Task 2.4 ready for immediate execution*
