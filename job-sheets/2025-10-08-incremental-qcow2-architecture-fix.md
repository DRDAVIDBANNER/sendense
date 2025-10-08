# Job Sheet: Incremental QCOW2 Architecture Fix

**Job Sheet ID:** 2025-10-08-incremental-qcow2-architecture-fix  
**Created:** October 8, 2025  
**Status:** 🔴 BLOCKED - Architectural refactoring needed  
**Priority:** HIGH - Required for Phase 1 completion  
**Estimated Effort:** 2-3 hours  
**Prerequisite:** change_id recording fix (✅ COMPLETE)

---

## 🎯 TASK LINK TO PROJECT GOALS

**Project Goal:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`  
**Specific Task:** Task 7.6 - Integration Testing → Incremental Backup Support  

**Blocked By:** Architectural issue - backup handlers bypass BackupEngine

**Success Criteria:**
- ✅ Full backup of VMware VM to QCOW2 file (DONE)
- ✅ change_id recording (DONE - October 8, 2025)
- ❌ **Incremental backup using VMware CBT** (BLOCKED)
- ❌ 90%+ data reduction on incrementals (BLOCKED)

**Business Value:**
- Enables incremental backups (90%+ space/time savings)
- Completes Phase 1 requirements
- Critical for production backup operations

---

## 🐛 PROBLEM STATEMENT

**Issue:** Incremental backups fail because backup handlers bypass the BackupEngine and directly create full QCOW2 files, ignoring incremental logic.

**Evidence:**
```
# Attempt incremental backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"incremental"}'

# Error: Failed to create QCOW2 file
# SHA creates FULL QCOW2 instead of incremental with backing file
```

**Impact:**
- Incremental backups don't work
- All backups consume full disk space
- No VMware CBT optimization
- Phase 1 Task 7.6 cannot be completed

---

## 🔍 ROOT CAUSE ANALYSIS

### Current Architecture (BROKEN)

```
StartBackup Handler (backup_handlers.go)
    ↓
    ├─ Manually allocate NBD ports
    ├─ Manually create QCOW2 files via qemuManager  ❌ NO INCREMENTAL LOGIC
    ├─ Manually start qemu-nbd processes
    ├─ Call SNA API directly
    └─ BYPASSES BackupEngine entirely
```

**File:** `source/current/sha/api/handlers/backup_handlers.go`  
**Method:** `StartBackup()` (lines 133-481)

**Problem Code:**
```go
// Line 259-278: Manual QCOW2 creation
for i, vmDisk := range vmDisks {
    exportName := fmt.Sprintf("%s-%s", req.VMName, vmDisk.DiskID)
    diskJobID := fmt.Sprintf("%s-%s", backupJobID, vmDisk.DiskID)

    // Allocate NBD port
    nbdPort, err := bh.portAllocator.Allocate(diskJobID, req.VMName, exportName)
    
    // ❌ PROBLEM: Always creates FULL QCOW2, ignores backup_type
    qcow2Path := fmt.Sprintf("/backup/repository/%s-%s.qcow2", req.VMName, vmDisk.DiskID)
    
    // ❌ PROBLEM: No parent backup lookup for incrementals
    // ❌ PROBLEM: No backing file specified
    err = bh.qemuManager.Start(qcow2Path, nbdPort, exportName, vmDisk.CapacityBytes)
}
```

### Expected Architecture (CORRECT)

```
StartBackup Handler
    ↓
    BackupEngine.ExecuteBackup() ✅ HAS INCREMENTAL LOGIC
    ↓
    LocalRepository.CreateBackup() ✅ CHECKS BACKUP TYPE
    ↓
    QCOW2Manager.CreateIncremental() ✅ CREATES WITH BACKING FILE
    ↓
    QemuNBDManager.Start()
    ↓
    SNA API call
```

**The BackupEngine already has all the logic!**

**File:** `source/current/sha/workflows/backup.go`  
**Method:** `ExecuteBackup()` (lines 102-180)

**Existing Incremental Logic:**
```go
// Line 135-145: Parent backup lookup for incrementals
if req.BackupType == storage.BackupTypeIncremental {
    chain, err := repo.GetBackupChain(ctx, req.VMContextID, req.DiskID)
    if err != nil {
        return nil, fmt.Errorf("failed to get backup chain for incremental: %w", err)
    }
    if chain.LatestBackupID == "" {
        return nil, fmt.Errorf("no parent backup found for incremental - full backup required first")
    }
    backupReq.ParentBackupID = chain.LatestBackupID
    log.WithField("parent_backup_id", chain.LatestBackupID).Info("📎 Using parent backup for incremental")
}
```

**File:** `source/current/sha/storage/local_repository.go`  
**Method:** `CreateBackup()` (lines 76-106)

**Existing Incremental QCOW2 Creation:**
```go
// Line 85-106: Incremental backup with backing file
} else if req.BackupType == BackupTypeIncremental {
    // Incremental backup requires parent
    if req.ParentBackupID == "" {
        return nil, &BackupError{
            BackupID: backupID,
            Op:       "create_incremental",
            Err:      ErrParentBackupRequired,
        }
    }

    // Get parent backup path
    parentBackup, err := lr.GetBackup(ctx, req.ParentBackupID)
    if err != nil {
        return nil, &BackupError{
            BackupID: backupID,
            Op:       "get_parent",
            Err:      fmt.Errorf("parent backup not found: %w", err),
        }
    }

    // ✅ Creates incremental with backing file
    if err := lr.qcowManager.CreateIncremental(ctx, backupPath, parentBackup.FilePath); err != nil {
        return nil, &BackupError{
            BackupID: backupID,
            Op:       "create_incremental",
            Err:      err,
        }
    }
}
```

**File:** `source/current/sha/storage/qcow2_manager.go`  
**Method:** `CreateIncremental()` (lines 68-100)

**QCOW2 Backing File Command:**
```go
// Line 87-93: qemu-img create with backing file
cmd := exec.CommandContext(ctx, q.qemuImgPath, "create",
    "-f", "qcow2",
    "-b", backingFile,    // ✅ Points to parent backup
    "-F", "qcow2",
    path)
```

---

## 🔧 SOLUTION DESIGN

### Approach: Refactor Handlers to Use BackupEngine

**Principle:** Handlers should orchestrate, not implement. Move QCOW2 creation logic from handlers to BackupEngine.

### Current Handler Responsibilities (TOO MUCH)
1. ❌ Allocate NBD ports
2. ❌ Create QCOW2 files
3. ❌ Start qemu-nbd processes
4. ❌ Call SNA API
5. ❌ Create database records

### Proposed Handler Responsibilities (CORRECT)
1. ✅ Validate request
2. ✅ Call BackupEngine.ExecuteBackup() (per disk)
3. ✅ Create database records
4. ✅ Return response

### Proposed BackupEngine Responsibilities
1. ✅ Parent backup lookup (already exists)
2. ✅ QCOW2 creation via repository (already exists)
3. ✅ NBD port allocation (needs to be added)
4. ✅ qemu-nbd process management (needs to be added)
5. ✅ SNA API call (needs to be added)

---

## 📝 IMPLEMENTATION PLAN

### Step 1: Enhance BackupEngine (30 minutes)

**File:** `source/current/sha/workflows/backup.go`

**Add to BackupEngine struct:**
```go
type BackupEngine struct {
    repositoryManager *storage.RepositoryManager
    portAllocator     *services.NBDPortAllocator       // NEW
    qemuManager       *services.QemuNBDManager         // NEW
    snaClient         *sna.Client                      // NEW (for SNA API calls)
}
```

**Enhance ExecuteBackup method:**
```go
func (be *BackupEngine) ExecuteBackup(ctx context.Context, req *BackupRequest) (*BackupResult, error) {
    // ... existing parent backup lookup ...
    
    // Create backup (repository creates QCOW2 file with proper backing if incremental)
    backup, err := repo.CreateBackup(ctx, backupReq)
    
    // NEW: Allocate NBD port
    nbdPort, err := be.portAllocator.Allocate(...)
    
    // NEW: Start qemu-nbd for this QCOW2
    err = be.qemuManager.Start(backup.FilePath, nbdPort, ...)
    
    // NEW: Call SNA API to start backup client
    err = be.snaClient.StartBackup(...)
    
    return &BackupResult{
        BackupID:       backup.ID,
        FilePath:       backup.FilePath,
        NBDExportName:  exportName,
        NBDPort:        nbdPort,
        ...
    }
}
```

### Step 2: Simplify Backup Handlers (45 minutes)

**File:** `source/current/sha/api/handlers/backup_handlers.go`

**Refactor StartBackup method:**
```go
func (bh *BackupHandler) StartBackup(w http.ResponseWriter, r *http.Request) {
    // ... existing validation ...
    
    // Get VM context and disks
    vmContext, _ := bh.vmContextRepo.GetVMContextByName(req.VMName)
    vmDisks, _ := bh.vmDiskRepo.GetByVMContextID(vmContext.ContextID)
    
    backupJobID := fmt.Sprintf("backup-%s-%d", req.VMName, time.Now().Unix())
    diskResults := make([]DiskBackupResult, len(vmDisks))
    
    // Execute backup for EACH disk via BackupEngine
    for i, vmDisk := range vmDisks {
        backupReq := &workflows.BackupRequest{
            VMContextID:  vmContext.ContextID,
            VMName:       req.VMName,
            DiskID:       vmDisk.DiskID,
            BackupType:   storage.BackupType(req.BackupType), // full or incremental
            RepositoryID: req.RepositoryID,
            TotalBytes:   vmDisk.CapacityBytes,
        }
        
        // ✅ BackupEngine handles EVERYTHING
        result, err := bh.backupEngine.ExecuteBackup(ctx, backupReq)
        if err != nil {
            // Cleanup via defer
            return
        }
        
        diskResults[i] = DiskBackupResult{
            DiskID:        vmDisk.DiskID,
            NBDPort:       result.NBDPort,
            NBDExportName: result.NBDExportName,
            QCOW2Path:     result.FilePath,
            Status:        "started",
        }
    }
    
    // Create database record
    backupJob := &database.BackupJob{...}
    bh.backupJobRepo.Create(ctx, backupJob)
    
    // Return response
    response := BackupResponse{...}
    bh.sendJSON(w, http.StatusOK, response)
}
```

### Step 3: Testing (45 minutes)

**Test Plan:**

1. **Full Backup Test**
```bash
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'

# Verify:
# - QCOW2 created with correct size
# - qemu-nbd running
# - change_id recorded
```

2. **Incremental Backup Test**
```bash
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"incremental"}'

# Verify:
# - NEW QCOW2 created with backing file pointing to previous
# - qemu-img info shows backing file
# - Transfer size much smaller (~90% reduction)
# - New change_id recorded
```

3. **Backup Chain Verification**
```bash
qemu-img info /backup/repository/ctx-pgtest1-*/disk-0/backup-*.qcow2

# Expected output for incremental:
# backing file: /backup/repository/.../backup-pgtest1-{previous}-full.qcow2
# backing file format: qcow2
```

### Step 4: Documentation (30 minutes)

1. Update API documentation (`source/current/api-documentation/OMA.md`)
2. Update Phase 1 Context Helper
3. Update CHANGELOG.md
4. Create completion report

---

## 📊 CURRENT STATE ASSESSMENT

### ✅ What Works (Already Complete)
1. **change_id Recording** - 100% operational
   - SNA passes `MIGRATEKIT_JOB_ID` to client
   - SHA creates database records
   - Client calls completion endpoint
   - change_id stored in database
   - **Validated:** backup-pgtest1-1759913694

2. **BackupEngine Incremental Logic** - 100% implemented
   - Parent backup lookup
   - Backup chain management
   - Repository integration

3. **LocalRepository Incremental Logic** - 100% implemented
   - `CreateBackup()` checks backup type
   - Calls `CreateIncremental()` for incremental backups
   - Parent backup path resolution

4. **QCOW2Manager Incremental Support** - 100% implemented
   - `CreateIncremental()` method exists
   - Creates QCOW2 with backing file using `qemu-img create -b`
   - Backing file validation

5. **Database Schema** - Ready for incrementals
   - `backup_jobs.parent_backup_id` field (nullable, FK to self)
   - `backup_jobs.change_id` field (stores VMware CBT ID)
   - `backup_chains` table for tracking chains

### ❌ What's Missing (Needs Implementation)
1. **Backup Handlers** - Bypass BackupEngine
   - Directly create QCOW2s via qemuManager
   - Don't use repository layer
   - Don't check backup type
   - Don't lookup parent backups

2. **BackupEngine Integration** - Not connected to handlers
   - Handlers need to call `BackupEngine.ExecuteBackup()`
   - BackupEngine needs NBD port allocation
   - BackupEngine needs qemu-nbd management
   - BackupEngine needs SNA API client

---

## 📚 KEY FILES & DOCUMENTATION

### Source Code Files

**Backup Handlers (NEEDS REFACTORING):**
- `source/current/sha/api/handlers/backup_handlers.go`
  - Lines 133-481: `StartBackup()` method
  - Lines 23-56: `BackupHandler` struct definition
  - **Issue:** Bypasses BackupEngine, creates QCOW2s directly

**BackupEngine (NEEDS ENHANCEMENT):**
- `source/current/sha/workflows/backup.go`
  - Lines 102-180: `ExecuteBackup()` method (has incremental logic)
  - Lines 135-145: Parent backup lookup for incrementals
  - **Needs:** NBD port allocation, qemu-nbd management, SNA API calls

**LocalRepository (READY - NO CHANGES):**
- `source/current/sha/storage/local_repository.go`
  - Lines 65-120: `CreateBackup()` method
  - Lines 85-106: Incremental QCOW2 creation logic
  - **Status:** ✅ Already implements incremental backup

**QCOW2Manager (READY - NO CHANGES):**
- `source/current/sha/storage/qcow2_manager.go`
  - Lines 33-66: `CreateFull()` method
  - Lines 68-100: `CreateIncremental()` method (creates with backing file)
  - **Status:** ✅ Already supports backing files

**QemuNBDManager:**
- `source/current/sha/services/qemu_nbd_manager.go`
  - Manages qemu-nbd process lifecycle
  - **Status:** Working but called directly by handlers

**NBDPortAllocator:**
- `source/current/sha/services/nbd_port_allocator.go`
  - Allocates ports 10100-10200
  - **Status:** Working but needs integration with BackupEngine

### Database Schema

**File:** `source/current/sha/database/migrations/20251004120000_add_backup_tables.up.sql`

**Key Tables:**
```sql
CREATE TABLE backup_jobs (
    id VARCHAR(191) PRIMARY KEY,
    vm_context_id VARCHAR(191) NOT NULL,
    vm_name VARCHAR(255) NOT NULL,
    disk_id INT NOT NULL DEFAULT 0,
    backup_type ENUM('full', 'incremental') NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed') NOT NULL,
    parent_backup_id VARCHAR(191),  -- NULL for full, references backup_jobs.id for incremental
    change_id VARCHAR(191),          -- VMware CBT change ID
    repository_id VARCHAR(64) NOT NULL,
    ...
    FOREIGN KEY (parent_backup_id) REFERENCES backup_jobs(id) ON DELETE SET NULL
);

CREATE TABLE backup_chains (
    id INT AUTO_INCREMENT PRIMARY KEY,
    chain_id VARCHAR(64) UNIQUE,
    vm_context_id VARCHAR(191) NOT NULL,
    vm_name VARCHAR(255) NOT NULL,
    disk_id INT NOT NULL,
    repository_id VARCHAR(64) NOT NULL,
    full_backup_id VARCHAR(191),    -- First full backup in chain
    latest_backup_id VARCHAR(191),  -- Most recent backup
    ...
    FOREIGN KEY (full_backup_id) REFERENCES backup_jobs(id),
    FOREIGN KEY (latest_backup_id) REFERENCES backup_jobs(id)
);
```

**Database Documentation:**  
`source/current/api-documentation/DB_SCHEMA.md`

### API Documentation

**File:** `source/current/api-documentation/OMA.md`

**Backup Endpoints (Lines 336-418):**
```
POST /api/v1/backups           - Start VM-level backup
POST /api/v1/backups/{id}/complete - Complete backup, record change_id
GET  /api/v1/backups            - List backups
GET  /api/v1/backups/{id}       - Get backup details
GET  /api/v1/backups/chain      - Get backup chain (full + incrementals)
DELETE /api/v1/backups/{id}     - Delete backup
```

**Request Format:**
```json
{
  "vm_name": "pgtest1",
  "repository_id": "1",
  "backup_type": "full"  // or "incremental"
}
```

### Related Job Sheets

**Prerequisites:**
- `job-sheets/2025-10-08-changeid-recording-fix-EXPANDED.md` ✅ COMPLETE
  - change_id recording fully operational
  - Binaries: `sna-api-server-v1.12.0-changeid-fix`, `sendense-hub-v2.23.2-null-fix`

**Multi-Disk Infrastructure:**
- `job-sheets/2025-10-08-phase1-backup-completion.md`
  - Multi-disk VM backup architecture
  - NBD port allocation (10100-10200)
  - QCOW2 management

---

## 🧪 TESTING VERIFICATION

### Full Backup (Currently Works)
```bash
# Cleanup
sudo pkill -9 qemu-nbd
rm -rf /backup/repository/*.qcow2

# Start full backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'

# Verify QCOW2
ls -lh /backup/repository/*.qcow2
qemu-img info /backup/repository/pgtest1-disk-2000.qcow2

# Verify change_id recorded
mysql -u oma_user -p'oma_password' migratekit_oma -e \
  "SELECT id, change_id FROM backup_jobs ORDER BY created_at DESC LIMIT 1;"
```

### Incremental Backup (Currently Broken)
```bash
# Attempt incremental
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"incremental"}'

# Current Error:
# ❌ "Failed to create QCOW2 file" - tries to create full QCOW2
# ❌ Doesn't lookup parent backup
# ❌ Doesn't create backing file
```

### Expected After Fix
```bash
# Incremental should work
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"incremental"}'

# Verify incremental QCOW2 with backing file
qemu-img info /backup/repository/pgtest1-disk-2000.qcow2

# Expected output:
# backing file: /backup/repository/pgtest1-disk-2000-PREVIOUS.qcow2
# backing file format: qcow2
# virtual size: 102 GB
# disk size: 1-2 GB (only changed blocks)
```

---

## ✅ ACCEPTANCE CRITERIA

1. ✅ Handlers refactored to use `BackupEngine.ExecuteBackup()`
2. ✅ BackupEngine enhanced with NBD/qemu-nbd management
3. ✅ Full backups continue to work (no regression)
4. ✅ Incremental backups create QCOW2 with backing files
5. ✅ `qemu-img info` shows backing file for incrementals
6. ✅ Incremental transfers only changed blocks (90%+ reduction)
7. ✅ New change_id recorded after incremental
8. ✅ Backup chains properly linked in database
9. ✅ Unit tests pass
10. ✅ E2E full + incremental test successful

---

## 📝 NOTES FOR NEXT SESSION

### Quick Start Commands
```bash
# Check current state
ls -lh /backup/repository/*.qcow2
mysql -u oma_user -p'oma_password' migratekit_oma -e \
  "SELECT id, backup_type, change_id FROM backup_jobs ORDER BY created_at DESC LIMIT 3;"

# Clean environment
sudo pkill -9 qemu-nbd
rm -rf /backup/repository/*.qcow2

# Test full backup (should work)
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'

# Test incremental (currently broken)
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"incremental"}'
```

### Key Points to Remember
1. ✅ change_id recording is DONE - don't break it!
2. ✅ All incremental logic EXISTS - just not connected
3. ❌ Handlers need refactoring - call BackupEngine instead of qemuManager
4. 🎯 Goal: Make handlers thin orchestrators, BackupEngine does the work

### Binaries in Production
- **SNA:** `sna-api-server-v1.12.0-changeid-fix` (10.0.100.231)
- **SHA:** `sendense-hub-v2.23.2-null-fix` (localhost:8082)
- **Client:** `sendense-backup-client-v1.0.1-port-fix` (on SNA)

---

**Last Updated:** October 8, 2025 12:30 UTC  
**Created By:** Cursor AI Assistant  
**Validated:** Architectural analysis complete, solution designed

