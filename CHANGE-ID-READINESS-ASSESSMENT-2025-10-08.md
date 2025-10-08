# Change ID Recording Readiness Assessment
**Date:** October 8, 2025  
**Context:** Phase 1 VMware Backup - Incremental Backup Preparation  
**Status:** ⚠️ **NOT READY** - Missing completion webhook/polling

---

## 🎯 **USER REQUEST**
Need to record VMware CBT change IDs after full backups complete to enable incremental/differential backups.

---

## ✅ **INFRASTRUCTURE READY**

### **1. Database Schema** ✅
**Table:** `backup_jobs`
```sql
change_id VARCHAR(191) NULL
```
- Field exists in production schema
- Indexed for quick lookups
- Nullable (correct for full backups that don't have previous change_id)

### **2. Code Infrastructure** ✅
**BackupEngine** (`sha/workflows/backup.go`):
```go
// Line 432: CompleteBackup method EXISTS
func (be *BackupEngine) CompleteBackup(
    ctx context.Context, 
    backupID string, 
    changeID string,  // ✅ Accepts change_id
    bytesTransferred int64
) error
```

**What it does:**
- Updates `backup_jobs.change_id` field in database
- Updates `backup_jobs.status = 'completed'`
- Updates `backup_jobs.bytes_transferred`
- Updates backup chain tracking

**BackupResponse** (`sha/api/handlers/backup_handlers.go`):
```go
type BackupResponse struct {
    ...
    ChangeID string `json:"change_id,omitempty"` // Line 98
    ...
}
```

### **3. Change ID Extraction** ✅
**sendense-backup-client** has full CBT support:
- `internal/vmware/change_id.go` - Parse and extract change IDs
- `GetChangeID(disk *types.VirtualDisk)` - Get current change ID from VMware
- Used for incremental backups via `QueryChangedDiskAreas()`

### **4. Replication Reference Implementation** ✅
**Replications already record change_id:**
```go
// sha/api/handlers/replication.go
// Lines 811-880: GetPreviousChangeID(), StoreChangeID()
// Table: vm_disks.disk_change_id
```

Workflow:
1. Replication completes
2. SNA/VMA returns change_id to SHA
3. SHA calls `StoreChangeID()` API
4. Stored in `vm_disks.disk_change_id` table
5. Next incremental queries `GetPreviousChangeID()`

---

## ❌ **WHAT'S MISSING**

### **CRITICAL GAP: No Completion Callback**

**Current Backup Flow:**
```
1. SHA StartBackup() → Creates backup_jobs entry (status='started')
2. SHA calls SNA /api/v1/backup/start with NBD targets
3. SHA returns HTTP 200 immediately ✅
4. SNA sendense-backup-client runs (takes hours) 📡
5. ❌ NO MECHANISM TO NOTIFY SHA WHEN COMPLETE ❌
6. ❌ change_id NEVER RECORDED IN DATABASE ❌
```

**Problem:**
- `CompleteBackup()` method exists but **is never called**
- No webhook endpoint for SNA to call back
- No polling mechanism to check SNA backup status
- `backup_jobs` entry stuck in 'started' status forever
- `change_id` field remains NULL

### **What Sendense-Backup-Client Returns:**
The client extracts change_id from VMware disk after snapshot:
```go
// internal/vmware/change_id.go
currentChangeID, err := vmware.GetChangeID(disk)
// Returns: "52d0eb97-27ad-...../52" (UUID/sequence)
```

**But:** This is only available AFTER the backup completes (hours later), and there's no mechanism to send it back to SHA.

---

## 🔧 **IMPLEMENTATION NEEDED**

### **Option 1: Polling (Recommended for MVP)**
**Rationale:** Simpler, works with existing SNA API

**Implementation:**
1. SHA starts backup, stores `backup_job_id` mapping to SNA job
2. SHA background worker polls SNA `/api/v1/backups/{job_id}/status` every 30s
3. When status='completed', SNA returns `change_id` in response
4. SHA calls `BackupEngine.CompleteBackup(backupID, changeID, bytes)`
5. Done!

**Files to Modify:**
- `sha/services/backup_completion_poller.go` (NEW - 150 lines)
- `sna/api/handlers/backup_status.go` (MODIFY - add change_id to response)
- Start poller in `sha/cmd/main.go`

**Estimated Effort:** 2-3 hours

### **Option 2: Webhook (Better Long-Term)**
**Rationale:** More scalable, real-time updates

**Implementation:**
1. SHA provides webhook URL in backup start request
2. SNA calls webhook when backup completes: `POST /api/v1/backups/{id}/complete`
3. Webhook payload includes `change_id`, `bytes_transferred`, `status`
4. SHA calls `BackupEngine.CompleteBackup()`

**Files to Modify:**
- `sha/api/handlers/backup_handlers.go` (ADD webhook endpoint)
- `sna` sendense-backup-client (ADD webhook call after completion)
- Requires SNA to have outbound HTTPS access to SHA

**Estimated Effort:** 4-5 hours

### **Option 3: Hybrid (Production-Ready)**
- Webhook as primary mechanism
- Polling as fallback if webhook fails
- Best reliability

**Estimated Effort:** 6-7 hours

---

## 📊 **CURRENT TEST STATUS**

**Running Backup:**
- Job ID: `backup-pgtest1-1759901593`
- Status in DB: `started`
- Actual Status: Running (8.5GB/102GB transferred)
- **Problem:** When this completes, `change_id` will be lost!

**What Will Happen:**
1. sendense-backup-client extracts change_id from VMware ✅
2. sendense-backup-client writes all data to QCOW2 ✅
3. sendense-backup-client finishes successfully ✅
4. ❌ **change_id discarded, never sent to SHA** ❌
5. Database shows `backup_jobs.change_id = NULL` forever ❌

**Impact:**
- Next backup request for `pgtest1` will be forced to do FULL backup
- Cannot do incremental because no previous `change_id` on record
- Wastes hours of time and bandwidth

---

## 🎯 **RECOMMENDATIONS**

### **Immediate (This Session):**
1. ✅ Document the gap (this file)
2. ⏳ Create job sheet for polling implementation
3. ⏳ Add TODO to current E2E test: "Verify change_id NULL after completion"

### **Next Session (Before Incremental Testing):**
1. Implement Option 1 (Polling)
2. Test full backup with change_id recording
3. Verify `backup_jobs.change_id` populated
4. Test incremental backup using previous change_id

### **Production (Phase 1 Complete):**
- Implement Option 3 (Hybrid webhook + polling)
- Full resilience for intermittent network issues

---

## 📋 **ACCEPTANCE CRITERIA FOR "READY"**

- [ ] Full backup completes
- [ ] SHA detects completion (polling or webhook)
- [ ] `backup_jobs.change_id` field populated with VMware CBT change ID
- [ ] `backup_jobs.status = 'completed'`
- [ ] `backup_jobs.bytes_transferred` accurate
- [ ] Second backup request queries previous `change_id`
- [ ] Incremental backup uses `previous_change_id` parameter
- [ ] Only changed blocks transferred

---

## 🔗 **REFERENCES**

**Documentation:**
- Phase 1 Goals: `project-goals/phases/phase-1-vmware-backup.md` (mentions CBT)
- Database Schema: `api-documentation/DB_SCHEMA.md` (line 122: change_id field)

**Code:**
- BackupEngine: `sha/workflows/backup.go` (line 432: CompleteBackup)
- BackupHandler: `sha/api/handlers/backup_handlers.go` (line 133: StartBackup)
- ReplicationHandler: `sha/api/handlers/replication.go` (line 811-880: change_id reference impl)
- Change ID Utils: `sendense-backup-client/internal/vmware/change_id.go`

**Related Issues:**
- Multi-disk backup infrastructure (FIXED October 8, 2025)
- Disk key mapping bug (FIXED October 8, 2025)
- qemu-nbd cleanup (FIXED October 8, 2025)

---

## 💡 **SUMMARY**

**Status:** ⚠️ **70% Ready**

**What's Working:**
- ✅ Database schema
- ✅ Code methods exist
- ✅ Change ID extraction working
- ✅ Multi-disk infrastructure operational

**What's Missing:**
- ❌ Completion detection mechanism (polling or webhook)
- ❌ No way to capture change_id from SNA after backup finishes
- ❌ `backup_jobs.change_id` stays NULL forever

**Impact:**
- Full backups work perfectly ✅
- Incremental backups will fail (no previous change_id) ❌
- Every backup forced to be full backup ❌
- Phase 1 "Incremental backup using VMware CBT" criterion NOT MET ❌

**Time to Fix:** 2-3 hours for polling (MVP), 6-7 hours for production-ready

**Recommendation:** Implement polling solution before marking Phase 1 complete.

---

**Report Generated:** October 8, 2025 06:50 UTC  
**Current Test:** pgtest1 backup running (8.5GB/102GB), will lose change_id on completion

