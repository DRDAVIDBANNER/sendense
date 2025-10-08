# Job Sheet: Backup API Endpoint Integration

**Job ID:** JS-2025-10-06-BACKUP-API  
**Phase:** Phase 1 - VMware Backup Implementation (Task 7: Testing & Validation)  
**Related Phase Link:** `project-goals/phases/phase-1-vmware-backup.md`  
**Status:** 🔴 **NOT STARTED**  
**Created:** October 6, 2025  
**Assigned:** TBD  
**Priority:** High  
**Estimated Time:** 4-6 hours  

---

## 🎯 Job Objective

**Primary Goal:** Wire up existing backup API handlers to server routes and validate E2E backup workflow

**Context:** 
- Backup handlers already implemented (`source/current/oma/api/handlers/backup_handlers.go`)
- Backend backup engine operational (`source/current/oma/workflows/backup.go`)
- Repository infrastructure complete (Task 1 ✅)
- NBD file export complete (Task 2 ✅)
- Backup workflow complete (Task 3 ✅)
- File-level restore complete (Task 4 ✅)
- **Critical Fix Complete:** vm_disks table now populated at discovery (v2.11.1)

**What's Missing:** Route registration in `source/current/oma/cmd/main.go` for backup endpoints

---

## 📋 Acceptance Criteria

- [ ] 5 backup API endpoints registered and responding
- [ ] Can trigger full backup of pgtest1 disk-2000 (102 GB)
- [ ] Backup job tracked in backup_jobs table with disk_id
- [ ] QCOW2 file created in repository with correct size
- [ ] Can list backups for pgtest1
- [ ] Can get backup details by backup_id
- [ ] Can delete backup and verify QCOW2 removal
- [ ] Can query backup chain for VM
- [ ] Backup operation uses vm_disks data (size_gb, capacity_bytes) populated from discovery
- [ ] All endpoints return consistent JSON format
- [ ] Error responses include proper HTTP status codes

---

## 🔧 Technical Implementation

### **Files to Modify**

#### 1. **source/current/oma/cmd/main.go**
**Action:** Register backup routes in InitializeRoutes()

**Required Routes:**
```go
// Backup Management
backupGroup := router.Group("/api/v1/backups")
{
    backupGroup.POST("", handlers.Backup.StartBackup)           // Start new backup
    backupGroup.GET("", handlers.Backup.ListBackups)            // List all backups
    backupGroup.GET("/:backup_id", handlers.Backup.GetBackupDetails) // Get backup details
    backupGroup.DELETE("/:backup_id", handlers.Backup.DeleteBackup)  // Delete backup
    backupGroup.GET("/:vm_name/chain", handlers.Backup.GetBackupChain) // Get backup chain
}
```

**Dependencies:**
- BackupHandler already initialized in handlers.go (line 46)
- BackupEngine already initialized in handlers.go
- No new handler initialization needed

---

### **Task Breakdown**

#### **Task 1: Register Backup Routes** ⏱️ 30 minutes
**Sub-Tasks:**
1. Open `source/current/oma/cmd/main.go`
2. Locate `InitializeRoutes()` function
3. Add backup route group after repository routes
4. Register 5 backup endpoints with correct HTTP methods
5. Verify handlers.Backup is accessible

**Validation:**
```bash
curl http://localhost:8082/api/v1/debug/endpoints | jq '.debug_data.endpoints' | grep backup
```

---

#### **Task 2: Test Backup Start Endpoint** ⏱️ 1 hour
**Sub-Tasks:**
1. Verify pgtest1 exists with vm_disks populated
2. Verify repository exists (local-repo-1 or create new)
3. Trigger full backup via API
4. Monitor backup job creation in backup_jobs table
5. Verify QCOW2 file creation in repository path
6. Check disk metadata query from vm_disks table

**Test Commands:**
```bash
# 1. Check VM and disks
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT vd.* FROM vm_disks vd 
   JOIN vm_replication_contexts vrc ON vd.vm_context_id = vrc.context_id 
   WHERE vrc.vm_name='pgtest1';"

# 2. Check repositories
curl -s http://localhost:8082/api/v1/repositories | jq '.repositories'

# 3. Start backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{
    "vm_name": "pgtest1",
    "disk_id": 0,
    "repository_id": "local-repo-1",
    "backup_type": "full",
    "tags": {"test": "e2e_validation"}
  }' | jq '.'

# 4. Check backup job
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT * FROM backup_jobs ORDER BY created_at DESC LIMIT 1;"

# 5. Verify QCOW2 file
ls -lh /var/lib/sendense/backups/*/disk-0/*.qcow2
```

**Expected Response:**
```json
{
  "success": true,
  "message": "Backup started successfully",
  "backup": {
    "id": "backup-20251006-203500",
    "vm_name": "pgtest1",
    "vm_context_id": "ctx-pgtest1-20251006-203401",
    "disk_id": 0,
    "repository_id": "local-repo-1",
    "backup_type": "full",
    "status": "running",
    "qcow2_path": "/var/lib/sendense/backups/{uuid}/disk-0/full-20251006-203500.qcow2"
  }
}
```

---

#### **Task 3: Test Backup List Endpoint** ⏱️ 30 minutes
**Sub-Tasks:**
1. Create 2-3 backups for different VMs/disks
2. Query list endpoint with no filters
3. Query list with vm_name filter
4. Query list with repository_id filter
5. Verify pagination if implemented

**Test Commands:**
```bash
# List all backups
curl -s http://localhost:8082/api/v1/backups | jq '.'

# List backups for specific VM
curl -s "http://localhost:8082/api/v1/backups?vm_name=pgtest1" | jq '.'

# List backups in specific repository
curl -s "http://localhost:8082/api/v1/backups?repository_id=local-repo-1" | jq '.'
```

---

#### **Task 4: Test Backup Details Endpoint** ⏱️ 30 minutes
**Sub-Tasks:**
1. Get backup_id from list endpoint
2. Query details endpoint
3. Verify metadata includes: size, creation time, backing file (if incremental)
4. Test error handling for non-existent backup_id

**Test Commands:**
```bash
# Get backup details
BACKUP_ID=$(curl -s http://localhost:8082/api/v1/backups | jq -r '.backups[0].id')
curl -s "http://localhost:8082/api/v1/backups/${BACKUP_ID}" | jq '.'

# Test non-existent backup
curl -s http://localhost:8082/api/v1/backups/backup-nonexistent | jq '.'
```

---

#### **Task 5: Test Backup Chain Endpoint** ⏱️ 30 minutes
**Sub-Tasks:**
1. Create full backup for pgtest1
2. Create 2 incremental backups
3. Query chain endpoint for pgtest1
4. Verify chain shows: full → incr1 → incr2
5. Verify total chain size calculation

**Test Commands:**
```bash
# Get backup chain
curl -s "http://localhost:8082/api/v1/backups/pgtest1/chain" | jq '.'
```

**Expected Response:**
```json
{
  "success": true,
  "vm_name": "pgtest1",
  "disk_id": 0,
  "chain": [
    {
      "backup_id": "backup-20251006-203500",
      "type": "full",
      "size_bytes": 109521666048,
      "created_at": "2025-10-06T20:35:00Z"
    },
    {
      "backup_id": "backup-20251006-210000",
      "type": "incremental",
      "size_bytes": 2147483648,
      "backing_file": "backup-20251006-203500",
      "created_at": "2025-10-06T21:00:00Z"
    }
  ],
  "total_size_bytes": 111669149696,
  "chain_length": 2
}
```

---

#### **Task 6: Test Backup Delete Endpoint** ⏱️ 45 minutes
**Sub-Tasks:**
1. Create test backup
2. Verify QCOW2 file exists
3. Delete backup via API
4. Verify QCOW2 file removed
5. Verify backup_jobs record deleted
6. Test error handling for:
   - Non-existent backup
   - Backup with dependent incrementals (chain validation)

**Test Commands:**
```bash
# Delete backup
curl -X DELETE "http://localhost:8082/api/v1/backups/${BACKUP_ID}" | jq '.'

# Verify deletion
ls -la /var/lib/sendense/backups/*/disk-0/*.qcow2
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT * FROM backup_jobs WHERE id='${BACKUP_ID}';"
```

---

#### **Task 7: E2E Integration Test** ⏱️ 1 hour
**Sub-Tasks:**
1. Clean test environment (delete existing backups)
2. Run complete backup workflow:
   - Discover pgtest1 (verify vm_disks populated)
   - Create full backup (102 GB disk)
   - Verify backup completion
   - Create incremental backup (simulate changes)
   - Query backup chain
   - Delete old backup
3. Monitor logs for errors
4. Document any issues or improvements

**Integration Test Script:**
```bash
#!/bin/bash
# E2E Backup Test Script

echo "=== E2E BACKUP TEST ==="

# 1. Verify pgtest1 discovered with disks
echo "1. Verifying VM discovery..."
DISK_COUNT=$(mysql -u oma_user -poma_password migratekit_oma -Nse \
  "SELECT COUNT(*) FROM vm_disks vd 
   JOIN vm_replication_contexts vrc ON vd.vm_context_id = vrc.context_id 
   WHERE vrc.vm_name='pgtest1';")
echo "   Disks found: ${DISK_COUNT}"

# 2. Start full backup
echo "2. Starting full backup..."
BACKUP_RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{
    "vm_name": "pgtest1",
    "disk_id": 0,
    "repository_id": "local-repo-1",
    "backup_type": "full"
  }')
BACKUP_ID=$(echo "$BACKUP_RESPONSE" | jq -r '.backup.id')
echo "   Backup ID: ${BACKUP_ID}"

# 3. Wait for completion (poll status)
echo "3. Monitoring backup progress..."
# ... status polling logic ...

# 4. Verify QCOW2 file
echo "4. Verifying QCOW2 file creation..."
# ... file verification ...

# 5. Query backup chain
echo "5. Querying backup chain..."
curl -s "http://localhost:8082/api/v1/backups/pgtest1/chain" | jq '.'

echo "=== TEST COMPLETE ==="
```

---

## 📊 Success Metrics

### **Functional Metrics**
- [ ] All 5 endpoints return 200 OK for valid requests
- [ ] Backup QCOW2 files created with correct size (±5% of source disk)
- [ ] Incremental backups use backing files correctly
- [ ] Backup deletion removes files and database records
- [ ] Chain queries show correct parent-child relationships

### **Performance Metrics**
- [ ] Full backup throughput: ≥ 2.5 GiB/s (target: 3.2 GiB/s)
- [ ] API response time: < 200ms (excluding backup execution)
- [ ] Incremental backup size: < 10% of full backup (with typical changes)

### **Error Handling Metrics**
- [ ] Invalid backup_id returns 404 with JSON error
- [ ] Missing required fields return 400 with validation errors
- [ ] Non-existent VM returns 404
- [ ] Non-existent repository returns 400

---

## 🔗 Dependencies

### **Prerequisite Completed Work**
✅ Task 1: Repository Abstraction (Complete)  
✅ Task 2: NBD File Export (Complete)  
✅ Task 3: Backup Workflow (Complete)  
✅ Task 4: File-Level Restore (Complete)  
✅ Task 5: API Handlers (Complete)  
✅ **Critical Fix**: vm_disks population at discovery (v2.11.1)

### **External Dependencies**
- VMA service operational for NBD streaming
- Repository storage available (local or network)
- MariaDB database operational
- NBD server supporting QCOW2 file exports

---

## 📝 Testing Checklist

### **Pre-Integration Testing**
- [ ] Verify BackupHandler initialized in handlers.go
- [ ] Verify BackupEngine initialized with dependencies
- [ ] Check existing backup handler tests pass
- [ ] Verify vm_disks populated for test VMs

### **Route Registration Testing**
- [ ] `/api/v1/backups` POST returns valid response
- [ ] `/api/v1/backups` GET returns backup list
- [ ] `/api/v1/backups/:id` GET returns backup details
- [ ] `/api/v1/backups/:id` DELETE removes backup
- [ ] `/api/v1/backups/:vm_name/chain` GET returns chain

### **Error Scenario Testing**
- [ ] Invalid JSON body returns 400
- [ ] Missing required field returns 400 with validation message
- [ ] Non-existent VM returns 404
- [ ] Non-existent repository returns 400
- [ ] Non-existent backup_id returns 404
- [ ] Delete backup with dependents returns 409 (conflict)

### **Integration Testing**
- [ ] Full backup creates QCOW2 file
- [ ] Incremental backup uses backing file
- [ ] Backup chain query shows all backups
- [ ] Delete cascade removes dependent records
- [ ] Concurrent backups don't conflict
- [ ] Repository storage capacity checked before backup

---

## 📖 Documentation Updates Required

After completion, update:
1. ✅ `start_here/CHANGELOG.md` - Add route registration entry
2. ✅ `docs/database-schema.md` - Document backup_jobs.disk_id column
3. 🔴 `source/current/api-documentation/OMA.md` - Update backup endpoints section with working examples
4. 🔴 `project-goals/phases/phase-1-vmware-backup.md` - Mark Task 7 complete
5. 🔴 Create `docs/backup-api-testing-guide.md` with E2E test procedures

---

## 🚨 Known Issues / Risks

### **Potential Issues**
1. **Backup endpoint 404**: Routes not registered in main.go
   - **Solution**: Add backup route group to InitializeRoutes()

2. **vm_disks query fails**: NULL job_id handling
   - **Solution**: Already fixed in v2.11.1 - query by vm_context_id

3. **Repository not found**: Repository ID mismatch
   - **Solution**: Verify repository exists via GET /api/v1/repositories

4. **QCOW2 creation fails**: Insufficient disk space
   - **Solution**: Add storage capacity check before backup

### **Risk Mitigation**
- Test with small VM first (< 10 GB disk)
- Monitor repository disk space during backup
- Implement job cancellation for stuck backups
- Add backup timeout (default: 4 hours)

---

## 🎓 Learning / Notes

### **Key Insights from vm_disks Fix**
- Discovery service now stores disk metadata immediately
- Backup workflow can query disk info without replication job
- job_id field nullable - NULL = from discovery, populated = from replication
- vm_context_id is the stable identifier for VM disk relationships

### **Backup Architecture Patterns**
- Handlers delegate to BackupEngine for business logic
- BackupEngine uses Repository abstraction for storage
- QCOW2Manager handles backing file chains
- JobLog provides operation tracking and audit trail

### **API Design Standards**
- Consistent JSON response format: `{ success: bool, data: {}, error: string }`
- HTTP status codes: 200 (OK), 400 (validation), 404 (not found), 409 (conflict), 500 (server error)
- Query parameters for filtering: `?vm_name=X&repository_id=Y`
- RESTful resource naming: `/backups/:id` not `/get-backup/:id`

---

## ✅ Completion Criteria

This job sheet is considered **COMPLETE** when:
1. ✅ All 5 backup endpoints registered and responding
2. ✅ E2E backup test passes (full backup → incremental → delete)
3. ✅ All error scenarios handled gracefully
4. ✅ QCOW2 files created with correct backing file chains
5. ✅ Documentation updated (CHANGELOG, API docs, Phase 1 status)
6. ✅ Binary deployed with version tag (e.g., v2.12.0-backup-api-wired)
7. ✅ Test results documented in completion summary

---

## 📎 Related Documents

- Phase 1 Plan: `project-goals/phases/phase-1-vmware-backup.md`
- Backup Handlers: `source/current/oma/api/handlers/backup_handlers.go`
- Backup Engine: `source/current/oma/workflows/backup.go`
- API Documentation: `source/current/api-documentation/OMA.md`
- Database Schema: `docs/database-schema.md`
- vm_disks Architecture: `VM_DISKS_ARCHITECTURE_ASSESSMENT.md`
- Task 5 Summary: `job-sheets/TASK5-COMPLETE-SUMMARY.md`
- **🔥 HANDOVER:** `MIGRATEKIT-HANG-INVESTIGATION-HANDOVER.md` (372 lines, complete troubleshooting guide)

---

## 🚨 CURRENT SESSION STATUS (Oct 7, 2025 05:35 UTC)

**BLOCKED** by migratekit NBD hang - comprehensive handover document created for next session.

### Progress Summary
- **75% Complete** - All infrastructure and APIs working
- **Blocking Issue:** migratekit hangs after "NBD metadata context enabled" message
- **7 Fixes Attempted:** All failed (qemu-nbd flags, config changes, SSH tunnel adjustments)
- **Handover Created:** Complete 372-line investigation guide in `MIGRATEKIT-HANG-INVESTIGATION-HANDOVER.md`

### Next Steps (for new session)
1. **Option 1:** Add debug logging to migratekit (recompile with logs between lines 65-73 of parallel_full_copy.go)
2. **Option 3:** Research qemu-nbd + libnbd metadata context compatibility issues
3. Analyze results and determine fix path

### Achievements This Session
- ✅ RESTful backup API endpoints working (`/backups`)
- ✅ vm_disks architecture fixed (populated at discovery)
- ✅ Backend FK constraint bug fixed (backup_job ordering)
- ✅ 500GB repository configured and writable
- ✅ NBD server fully operational
- ✅ qemu-nbd exporting QCOW2 correctly (109GB size)
- ✅ SNA successfully connects to SHA via tunnel
- ✅ All connectivity issues resolved

**Waiting on:** Debug logging investigation to pinpoint exact hang location in migratekit code.

