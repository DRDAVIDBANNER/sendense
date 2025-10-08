# Backup Change ID API Endpoint Plan
**Created:** October 8, 2025  
**Priority:** 🔴 CRITICAL - Blocks incremental backups  
**Status:** 📋 PLANNING - Ready for implementation

---

## 🐛 PROBLEM CONFIRMED

### **What Happened:**
- ✅ Full backup completed successfully
- ✅ `MIGRATEKIT_JOB_ID` environment variable set correctly (our fix works!)
- ❌ Change ID NOT stored in database
- ❌ sendense-backup-client called wrong API endpoint

### **Error Message:**
```
Error: failed to write ChangeID to SHA database: 
SHA API returned status 404: 
{"details":"Replication job not found","error":"Job not found","timestamp":"2025-10-08T08:09:12+01:00"}
```

### **Root Cause:**
```go
// sendense-backup-client/internal/target/nbd.go:411
apiURL := fmt.Sprintf("%s/api/v1/replications/%s/changeid", shaURL, jobID)
```

**The Problem:**
- Client calls `/api/v1/replications/{job_id}/changeid`
- Looks for job in `replication_jobs` table
- But backup jobs are in `backup_jobs` table
- SHA returns 404: "Replication job not found"

---

## ✅ SOLUTION: Add Backup-Specific Completion Endpoint

### **Option A: Dedicated Backup Endpoint** (RECOMMENDED)
**Why:** Clean separation, follows REST conventions, future-proof

**Changes Required:**
1. **SHA API Handler** - Add `CompleteBackup` handler
2. **SHA Route Registration** - Add backup completion route
3. **sendense-backup-client** - Update API endpoint logic
4. **Build & Deploy** - 2 new binaries (SHA + sendense-backup-client)

**Pros:**
- ✅ Clean REST API design
- ✅ Proper separation of concerns
- ✅ Easy to extend (add bytes_transferred, etc.)
- ✅ Follows existing backup API patterns

**Cons:**
- ❌ Requires updating 2 binaries (SHA + client)
- ❌ More code changes

### **Option B: Unified Endpoint** (QUICK FIX)
**Why:** Minimal changes, works for both job types

**Changes Required:**
1. **SHA Replication Handler** - Check both job tables
2. **Build & Deploy** - 1 binary (SHA only)

**Pros:**
- ✅ Minimal code changes
- ✅ Only 1 binary to rebuild
- ✅ Client unchanged

**Cons:**
- ❌ Mixes concerns (replications + backups)
- ❌ Less clean architecture
- ❌ Harder to maintain

---

## 🎯 RECOMMENDED APPROACH: Option A

**Reasoning:**
- Following project rules: modular, clean separation
- Backup system should have its own completion endpoint
- Aligns with existing `/api/v1/backups` pattern
- Future-proof for additional backup-specific fields

---

## 📋 IMPLEMENTATION PLAN (Option A)

### **STEP 1: Add SHA Backup Completion Endpoint** (20 minutes)

#### **File:** `source/current/sha/api/handlers/backup_handlers.go`

**Add Handler Method:**
```go
// CompleteBackup handles POST /api/v1/backups/{backup_id}/complete
// Called by sendense-backup-client when backup finishes to record change_id
func (bh *BackupHandler) CompleteBackup(w http.ResponseWriter, r *http.Request) {
    // Extract backup ID from URL path
    backupID := mux.Vars(r)["backup_id"]
    if backupID == "" {
        bh.sendError(w, http.StatusBadRequest, "missing backup_id", "backup_id path parameter is required")
        return
    }

    // Parse request body
    var req struct {
        ChangeID         string `json:"change_id"`
        BytesTransferred int64  `json:"bytes_transferred"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        bh.sendError(w, http.StatusBadRequest, "invalid request body", err.Error())
        return
    }

    // Validate required fields
    if req.ChangeID == "" {
        bh.sendError(w, http.StatusBadRequest, "missing change_id", "change_id field is required")
        return
    }

    log.WithFields(log.Fields{
        "backup_id":         backupID,
        "change_id":         req.ChangeID,
        "bytes_transferred": req.BytesTransferred,
    }).Info("📝 Completing backup job and storing change_id")

    // Call BackupEngine.CompleteBackup()
    err := bh.backupEngine.CompleteBackup(r.Context(), backupID, req.ChangeID, req.BytesTransferred)
    if err != nil {
        // Check if backup job not found
        if strings.Contains(err.Error(), "not found") {
            bh.sendError(w, http.StatusNotFound, "backup job not found", err.Error())
            return
        }
        bh.sendError(w, http.StatusInternalServerError, "failed to complete backup", err.Error())
        return
    }

    // Success response
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":     "completed",
        "backup_id":  backupID,
        "change_id":  req.ChangeID,
        "message":    "Backup completed successfully, change_id recorded",
        "timestamp":  time.Now().Format(time.RFC3339),
    })
}
```

**Register Route in `RegisterRoutes()` method:**
```go
// In backup_handlers.go RegisterRoutes() method, add:
r.HandleFunc("/backups/{backup_id}/complete", bh.CompleteBackup).Methods("POST")
```

---

### **STEP 2: Update sendense-backup-client** (15 minutes)

#### **File:** `source/current/sendense-backup-client/internal/target/nbd.go`

**Update `storeChangeIDInOMA()` method (line 403-440):**

**Current Code:**
```go
func (t *NBDTarget) storeChangeIDInOMA(jobID, changeID string) error {
    shaURL := os.Getenv("SHA_API_URL")
    if shaURL == "" {
        shaURL = "http://localhost:8082"
    }

    apiURL := fmt.Sprintf("%s/api/v1/replications/%s/changeid", shaURL, jobID)
    // ... rest of code
}
```

**New Code:**
```go
func (t *NBDTarget) storeChangeIDInOMA(jobID, changeID string) error {
    shaURL := os.Getenv("SHA_API_URL")
    if shaURL == "" {
        shaURL = "http://localhost:8082"
    }

    // Determine API endpoint based on job ID prefix
    var apiURL string
    if strings.HasPrefix(jobID, "backup-") {
        // Backup job - use backup completion endpoint
        apiURL = fmt.Sprintf("%s/api/v1/backups/%s/complete", shaURL, jobID)
    } else {
        // Replication job - use replication endpoint
        apiURL = fmt.Sprintf("%s/api/v1/replications/%s/changeid", shaURL, jobID)
    }

    // CRITICAL FIX: Calculate correct disk ID from VMware disk.Key
    diskID := t.getCurrentDiskID()

    // Create request payload
    payload := map[string]interface{}{
        "change_id": changeID,
    }
    
    // Only add disk_id for replication jobs (backups are VM-level)
    if strings.HasPrefix(jobID, "replication-") {
        payload["disk_id"] = diskID
    }
    
    // Add bytes_transferred if available (for backups)
    if t.BytesTransferred > 0 {
        payload["bytes_transferred"] = t.BytesTransferred
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal request data: %w", err)
    }

    log.Printf("📡 Storing ChangeID via %s API for job type", 
        strings.Split(jobID, "-")[0])
    log.Printf("🔄 API URL: %s", apiURL)
    log.Printf("🔄 ChangeID: %s", changeID)

    resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to call SHA API: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("SHA API returned status %d: %s", resp.StatusCode, string(body))
    }

    log.Printf("✅ Successfully stored ChangeID in SHA database")
    return nil
}
```

---

### **STEP 3: Build Binaries** (5 minutes)

```bash
# Build SHA
cd /home/oma_admin/sendense/source/current/sha/cmd
go build -o /home/oma_admin/sendense/source/builds/sendense-hub-v2.22.0-backup-completion main.go

# Build sendense-backup-client
cd /home/oma_admin/sendense/source/current/sendense-backup-client
go build -o /home/oma_admin/sendense/source/builds/sendense-backup-client-v1.0.2-backup-api .
```

---

### **STEP 4: Deploy SHA** (5 minutes)

```bash
# Stop SHA
sudo pkill sendense-hub

# Deploy new binary
sudo ln -sf /home/oma_admin/sendense/source/builds/sendense-hub-v2.22.0-backup-completion /usr/local/bin/sendense-hub

# Start SHA
nohup /usr/local/bin/sendense-hub -port=8082 -auth=false \
  -db-host=localhost -db-port=3306 -db-name=migratekit_oma \
  -db-user=oma_user -db-pass=oma_password >/tmp/sha-backup-api.log 2>&1 &
```

---

### **STEP 5: Deploy sendense-backup-client to SNA** (10 minutes)

```bash
# Copy binary to SNA
sshpass -p 'Password1' scp \
  /home/oma_admin/sendense/source/builds/sendense-backup-client-v1.0.2-backup-api \
  vma@10.0.100.231:/tmp/sendense-backup-client-new

# Deploy on SNA
sshpass -p 'Password1' ssh vma@10.0.100.231 << 'EOF'
  sudo mv /tmp/sendense-backup-client-new /usr/local/bin/sendense-backup-client
  sudo chmod +x /usr/local/bin/sendense-backup-client
  sudo chown root:root /usr/local/bin/sendense-backup-client
  ls -lh /usr/local/bin/sendense-backup-client
EOF
```

---

### **STEP 6: Test Full Backup** (60 minutes)

```bash
# Clean environment
/home/oma_admin/sendense/scripts/cleanup-backup-environment.sh

# Start full backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'

# Monitor logs for change_id storage
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "tail -f /var/log/sendense/backup-*.log | grep -i changeid"

# Expected: "✅ Successfully stored ChangeID in SHA database"
# NOT: "Error: failed to write ChangeID to SHA database"
```

---

### **STEP 7: Verify Database** (2 minutes)

```sql
SELECT id, vm_name, backup_type, change_id, status, created_at, completed_at
FROM backup_jobs
WHERE vm_name = 'pgtest1'
ORDER BY created_at DESC
LIMIT 1;

-- EXPECTED:
-- change_id = "52d0eb97-27ad-4c3d-874a-c34e85f2ea95/446" (VMware format)
-- status = "completed"
-- completed_at = <timestamp>
```

---

### **STEP 8: Test Incremental Backup** (30 minutes)

```bash
# Start incremental backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"incremental"}'

# Monitor for CBT usage
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "tail -f /var/log/sendense/backup-*.log | grep -i 'changed\|CBT\|incremental'"

# Expected: "Using previous change_id..." and "Querying changed disk areas"

# Verify QCOW2 backing file
qemu-img info /backup/repository/ctx-pgtest1-*/disk-0/backup-*-incr.qcow2

# Expected: backing file: .../full.qcow2

# Check space savings
du -h /backup/repository/ctx-pgtest1-*/disk-0/*.qcow2

# Expected: full ~12GB, incremental ~1GB (90%+ savings)
```

---

## 📊 TIMELINE

| Task | Duration | Complexity |
|------|----------|------------|
| SHA Handler Code | 20 min | Medium |
| Client Code Update | 15 min | Medium |
| Build Binaries | 5 min | Easy |
| Deploy SHA | 5 min | Easy |
| Deploy Client | 10 min | Easy |
| Clean Environment | 2 min | Easy |
| Test Full Backup | 60 min | Medium |
| Verify Database | 2 min | Easy |
| Test Incremental | 30 min | Medium |
| Documentation | 15 min | Easy |
| **TOTAL** | **164 min** | **~2.5 hours** |

---

## ✅ ACCEPTANCE CRITERIA

- [ ] SHA backup completion endpoint exists: `POST /api/v1/backups/{id}/complete`
- [ ] sendense-backup-client detects backup vs replication jobs
- [ ] Full backup completes and stores change_id (not NULL)
- [ ] sendense-backup-client log shows "✅ Successfully stored ChangeID"
- [ ] Database shows change_id in VMware format (UUID/sequence)
- [ ] Incremental backup uses previous change_id
- [ ] VMware CBT transfers only changed blocks
- [ ] QCOW2 incremental has backing file
- [ ] Space savings 90%+ confirmed

---

## 📝 DOCUMENTATION UPDATES

### **Files to Update:**
1. `start_here/CHANGELOG.md` - Add backup completion API entry
2. `api-documentation/OMA.md` - Document new endpoint
3. `api-documentation/API_DB_MAPPING.md` - Add completion endpoint mapping
4. `job-sheets/2025-10-08-backup-completion-api.md` - Create new job sheet

### **API Documentation Entry:**
```markdown
### Backup Completion
POST /api/v1/backups/{backup_id}/complete

**Purpose:** Record change_id and mark backup as completed

**Request:**
```json
{
  "change_id": "52d0eb97-27ad-4c3d-874a-c34e85f2ea95/446",
  "bytes_transferred": 107374182400
}
```

**Response (200):**
```json
{
  "status": "completed",
  "backup_id": "backup-pgtest1-1759905433",
  "change_id": "52d0eb97-27ad-4c3d-874a-c34e85f2ea95/446",
  "message": "Backup completed successfully, change_id recorded",
  "timestamp": "2025-10-08T08:30:00Z"
}
```

**Called By:** sendense-backup-client after backup finishes
**Database Impact:** Updates backup_jobs.change_id, backup_jobs.status, backup_jobs.completed_at
```

---

## ⚠️ ROLLBACK PLAN

**If New API Fails:**
```bash
# Rollback SHA
sudo pkill sendense-hub
sudo ln -sf /home/oma_admin/sendense/source/builds/sendense-hub-v2.21.0-error-handling /usr/local/bin/sendense-hub
nohup /usr/local/bin/sendense-hub -port=8082 ... &

# Rollback sendense-backup-client on SNA
sshpass -p 'Password1' ssh vma@10.0.100.231 \
  "sudo cp /usr/local/bin/sendense-backup-client.backup /usr/local/bin/sendense-backup-client"
```

---

## 🎯 IMPACT

### **What This Enables:**
- ✅ **Incremental backups** - 90%+ space/time savings
- ✅ **Production-grade backup system** - Full + incremental complete
- ✅ **Phase 1 completion** - All requirements met
- ✅ **VMware CBT optimization** - Transfer only changed blocks

### **Why This Matters:**
- **Customer Value:** Faster backups, less storage, lower costs
- **Product Completeness:** Core backup feature fully functional
- **Competitive Advantage:** On par with Veeam incremental capabilities

---

**Plan Created:** October 8, 2025 08:00 UTC  
**Estimated Implementation:** 2.5 hours  
**Priority:** CRITICAL - Blocks Phase 1 completion

