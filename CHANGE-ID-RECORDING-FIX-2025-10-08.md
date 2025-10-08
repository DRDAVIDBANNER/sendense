# Change ID Recording Fix - Root Cause & Solution
**Date:** October 8, 2025 07:00 UTC  
**Context:** Full backup completed, change_id NOT recorded  
**Status:** ✅ **ROOT CAUSE IDENTIFIED** - Simple fix required

---

## 🔍 **ROOT CAUSE ANALYSIS**

### **Evidence from Backup Client Log:**
```
time="2025-10-08T06:03:07Z" level=info msg="✅ Parallel full copy completed successfully"
⚠️ No MIGRATEKIT_JOB_ID environment variable set, cannot store ChangeID
```

**What Happened:**
1. Full backup completed successfully ✅
2. sendense-backup-client extracted change_id from VMware ✅
3. Tried to store change_id via SHA API ✅
4. **Missing:** `MIGRATEKIT_JOB_ID` environment variable ❌
5. **Result:** Change_id discarded, not stored in database ❌

---

## 🐛 **THE BUG**

### **Location:** `sna/api/server.go` line 658-714

**buildBackupCommand() Function - MISSING ENV VAR:**

```go
func (s *SNAControlServer) buildBackupCommand(req *BackupRequest) (*exec.Cmd, error) {
    // ... builds command ...
    
    cmd := exec.Command(sbcBinary, args...)
    
    // ❌ MISSING: cmd.Env setting!
    // cmd.Env = append(os.Environ(), ...)
    
    cmd.Stdout = logFile
    cmd.Stderr = logFile
    
    return cmd, nil  // Returns WITHOUT environment variables!
}
```

**Compare with Working Replication Code:** (`sna/vmware/service.go` line 239-244)

```go
// ✅ CORRECT: Sets environment variables
cmd.Env = append(os.Environ(),
    fmt.Sprintf("CLOUDSTACK_API_URL=%s", "http://localhost:8082"),
    fmt.Sprintf("CLOUDSTACK_API_KEY=%s", "test-api-key"),
    fmt.Sprintf("CLOUDSTACK_SECRET_KEY=%s", "test-secret-key"),
    fmt.Sprintf("MIGRATEKIT_JOB_ID=%s", jobID), // ✅ This enables change_id storage!
)
```

---

## ✅ **THE FIX**

### **File:** `sna/api/server.go`
### **Function:** `buildBackupCommand()` (line 658)

**Add environment variables after creating command:**

```go
func (s *SNAControlServer) buildBackupCommand(req *BackupRequest) (*exec.Cmd, error) {
    // ... existing code to build command ...
    
    cmd := exec.Command(sbcBinary, args...)
    
    // 🆕 ADD THIS: Set environment variables for change_id storage
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("MIGRATEKIT_JOB_ID=%s", req.JobID), // Critical for change_id storage
    )
    
    // For incremental backups, also pass previous change_id
    if req.BackupType == "incremental" && req.PreviousChangeID != "" {
        cmd.Env = append(cmd.Env, 
            fmt.Sprintf("MIGRATEKIT_PREVIOUS_CHANGE_ID=%s", req.PreviousChangeID),
        )
    }
    
    // Set up logging (existing code)
    cmd.Stdout = logFile
    cmd.Stderr = logFile
    
    return cmd, nil
}
```

**Lines Changed:** ~7 lines added  
**Complexity:** Trivial  
**Risk:** None (replications already use this pattern)

---

## 🔄 **HOW CHANGE_ID STORAGE WORKS**

### **1. SNA Receives Job from SHA**
```json
POST /api/v1/backup/start
{
  "job_id": "backup-pgtest1-1759901593",
  "vm_name": "pgtest1",
  "nbd_targets": "2000:nbd://...,2001:nbd://...",
  ...
}
```

### **2. SNA Sets Environment Variable**
```go
cmd.Env = append(os.Environ(),
    fmt.Sprintf("MIGRATEKIT_JOB_ID=%s", "backup-pgtest1-1759901593"),
)
```

### **3. sendense-backup-client Reads Env Var**
```go
// internal/target/nbd.go:214
jobID := os.Getenv("MIGRATEKIT_JOB_ID")
if jobID == "" {
    log.Println("⚠️ No MIGRATEKIT_JOB_ID environment variable set")
    return nil  // Change_id NOT stored
}
```

### **4. Client Stores Change_ID via SHA API**
```go
// Call SHA API: POST /api/v1/replications/{job_id}/changeid
// Body: {"change_id": "52d0eb97-27ad-.../446"}
// SHA updates: backup_jobs.change_id = "52d0eb97-27ad-.../446"
```

### **5. SHA Marks Backup Complete**
```go
BackupEngine.CompleteBackup(ctx, backupID, changeID, bytes)
// Updates: status='completed', change_id='...', bytes_transferred
```

---

## 📊 **CURRENT STATUS**

### **Completed Full Backup:**
- **Job ID:** `backup-pgtest1-1759901593` (from your test)
- **Status:** Completed successfully
- **Change ID:** Extracted by client, **DISCARDED** ❌
- **Database:** `backup_jobs.change_id = NULL`

**Proof from Log:**
```
2025/10/08 06:03:07 ⚠️ No MIGRATEKIT_JOB_ID environment variable set, cannot store ChangeID
```

### **Database Check:**
```sql
SELECT id, backup_type, change_id, status 
FROM backup_jobs 
WHERE id = 'backup-pgtest1-1759901593';

-- Result:
-- id                            | backup_type | change_id | status
-- backup-pgtest1-1759901593     | full        | NULL      | started
--                                                   ^^^^ NOT RECORDED!
```

---

## 🎯 **IMPLEMENTATION STEPS**

### **Step 1: Fix SNA Code** (5 minutes)
```bash
cd /home/oma_admin/sendense/source/current/sna/api
# Edit server.go, add environment variable setting to buildBackupCommand()
```

### **Step 2: Build New SNA Binary** (2 minutes)
```bash
cd /home/oma_admin/sendense/source/current/sna/cmd
go build -o /home/oma_admin/sendense/source/builds/sna-api-server-v1.12.0-changeid-fix .
```

### **Step 3: Deploy to SNA** (1 minute)
```bash
# SSH to SNA
ssh vma@10.0.100.231

# Stop old binary
pkill sna-api-server

# Copy new binary
scp /home/oma_admin/sendense/source/builds/sna-api-server-v1.12.0-changeid-fix vma@10.0.100.231:/usr/local/bin/sna-api-server

# Restart (systemd or manual)
systemctl restart sna-api-server
# OR
nohup /usr/local/bin/sna-api-server --port 8081 &
```

### **Step 4: Test with New Full Backup** (10 minutes)
```bash
# Clean environment
/home/oma_admin/sendense/scripts/cleanup-backup-environment.sh

# Start new backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'

# Wait for completion (~10 mins for sparse disk)

# Verify change_id recorded
ssh vma@10.0.100.231 "tail -50 /var/log/sendense/backup-*.log | grep -i changeid"
# Should show: "📋 Stored ChangeID in database: 52d0eb97..."
```

### **Step 5: Verify Database** (1 minute)
```sql
SELECT id, backup_type, change_id, status 
FROM backup_jobs 
WHERE vm_name = 'pgtest1' 
ORDER BY created_at DESC 
LIMIT 1;

-- Expected:
-- id                      | backup_type | change_id              | status
-- backup-pgtest1-...      | full        | 52d0eb97-27ad-.../446  | completed
--                                            ^^^^ NOW RECORDED! ✅
```

### **Step 6: Test Incremental Backup** (5 minutes)
```bash
# Initiate incremental
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"incremental"}'

# SHA automatically queries previous change_id
# Passes it to SNA
# SNA passes it to sendense-backup-client
# Client uses VMware CBT to transfer only changed blocks
# 90%+ space savings! ✅
```

---

## 📋 **ACCEPTANCE CRITERIA**

- [ ] SNA `buildBackupCommand()` sets `MIGRATEKIT_JOB_ID` env var
- [ ] New SNA binary built and deployed
- [ ] Full backup completes with change_id recorded
- [ ] `backup_jobs.change_id` field populated (not NULL)
- [ ] Incremental backup uses previous change_id
- [ ] sendense-backup-client log shows "📋 Stored ChangeID in database"
- [ ] QCOW2 incremental has backing file reference
- [ ] Space savings ~90%+ confirmed

---

## 📈 **EXPECTED RESULTS**

### **Full Backup (First):**
```
Backup Completion Log:
✅ Parallel full copy completed successfully
✅ Stored ChangeID in database: 52d0eb97-27ad-4c3d-87.../446
```

Database:
```sql
change_id = "52d0eb97-27ad-4c3d-87.../446"
status = "completed"
```

### **Incremental Backup (Second):**
```
Backup Start Log:
🔍 Using previous change_id: 52d0eb97-27ad-4c3d-87.../446
🔍 Querying VMware CBT for changed blocks...
✅ Found 1.2GB of 102GB changed (1.2% delta)
✅ Transferring only changed blocks...
```

QCOW2:
```bash
qemu-img info incremental.qcow2
# backing file: /backup/repository/.../full.qcow2
# virtual size: 102GB
# actual size: 1.5GB  (90%+ savings!)
```

---

## 💡 **SUMMARY**

**Problem:** SNA not setting `MIGRATEKIT_JOB_ID` environment variable  
**Impact:** sendense-backup-client can't store change_id in SHA database  
**Fix:** 7 lines of code to set `cmd.Env` in `buildBackupCommand()`  
**Effort:** 20 minutes (code + build + deploy + test)  
**Risk:** None (pattern already used in replications)

**After Fix:**
- ✅ Full backups record change_id
- ✅ Incremental backups use previous change_id
- ✅ VMware CBT transfers only changed blocks
- ✅ 90%+ space savings on incrementals
- ✅ Phase 1 complete

---

**Report Generated:** October 8, 2025 07:00 UTC  
**Full Backup:** Completed but change_id lost (fixable)  
**Next Action:** Implement 7-line fix, redeploy SNA, test again


