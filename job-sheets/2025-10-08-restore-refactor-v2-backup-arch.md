# File-Level Restore Refactor for v2.16.0+ Backup Architecture

**Date:** October 8, 2025  
**Status:** 🔴 IN PROGRESS  
**Priority:** HIGH (Blocks file-level restore functionality)

---

## 🎯 Problem Statement

File-level restore code was written in October 5, 2025 for the OLD backup architecture. The v2.16.0-v2.22.0 backup refactoring (October 8, 2025) fundamentally changed the database schema, breaking all restore functionality.

---

## 📊 Architecture Comparison

### OLD Architecture (restore was written for):
```
backup_jobs table:
├─ id (PK)
├─ vm_context_id
├─ repository_path  ← QCOW2 file path (SINGLE FILE)
├─ disk_id          ← Simple disk number
├─ change_id        ← CBT change ID
└─ status

One backup_jobs record = One QCOW2 file
```

### NEW Architecture (v2.16.0+):
```
vm_backup_contexts (master):
├─ context_id (PK)
├─ vm_name
├─ repository_id
└─ backup statistics

backup_jobs (parent job):
├─ id (PK)
├─ vm_backup_context_id (FK)
├─ vm_name
├─ backup_type
└─ status (VM-level status)

backup_disks (per-disk records):  ← ACTUAL QCOW2 PATHS HERE
├─ id (PK)
├─ vm_backup_context_id (FK)
├─ backup_job_id (FK)           ← Links to parent job
├─ disk_index (0, 1, 2...)      ← Which disk
├─ vmware_disk_key (2000, 2001...)
├─ qcow2_path                   ← ACTUAL QCOW2 FILE PATH
├─ disk_change_id               ← Per-disk CBT tracking
└─ status (per-disk status)

backup_chains (chain metadata):
├─ vm_context_id
├─ disk_id
├─ full_backup_id
├─ latest_backup_id
└─ total_backups

One backup_jobs record = Multiple backup_disks records
```

---

## 💥 What's Broken

### 1. Database Schema (`restore_mounts` table)
**Issue:** FK to wrong table, not integrated with CASCADE DELETE chain  
**Impact:** Can't properly track which disk is mounted, no automatic cleanup

**OLD (Wrong):**
```sql
CREATE TABLE restore_mounts (
    id VARCHAR(64) PRIMARY KEY,
    backup_id VARCHAR(64) NOT NULL,  -- FK to backup_jobs (parent job) ❌ WRONG
    ...
    FOREIGN KEY (backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE
);
```

**FIXED (Correct):**
```sql
CREATE TABLE restore_mounts (
    id VARCHAR(64) PRIMARY KEY,
    backup_disk_id BIGINT NOT NULL,  -- FK to backup_disks.id ✅ CORRECT
    ...
    FOREIGN KEY (backup_disk_id) REFERENCES backup_disks(id) ON DELETE CASCADE,
    UNIQUE KEY uk_backup_disk (backup_disk_id)  -- One mount per disk
);
```

**CASCADE DELETE Chain:**
```
vm_backup_contexts → backup_jobs → backup_disks → restore_mounts ✅
When backup deleted, all mounts automatically cleaned up
```

### 2. API Request Structure (`MountRequest`)
**Issue:** No disk selection parameter  
**Impact:** Can't mount specific disks

**Current:**
```go
type MountRequest struct {
    BackupID string `json:"backup_id"` // Parent job ID
}
```

**Needed:**
```go
type MountRequest struct {
    BackupID   string `json:"backup_id"`   // Parent job ID
    DiskIndex  int    `json:"disk_index"`  // Which disk to mount (0, 1, 2...)
}
```

### 3. Backup File Lookup (`findBackupFile()`)
**Issue:** Queries non-existent `backup_jobs.repository_path` field  
**Impact:** Complete failure to find QCOW2 files

**Current Logic:**
```go
// ❌ BROKEN: Expects repository_path on backup_jobs
backup, err := mm.repositoryManager.GetBackupFromAnyRepository(ctx, backupID)
return backup.FilePath // Doesn't exist!
```

**Needed Logic:**
```go
// ✅ CORRECT: Query backup_disks table
SELECT qcow2_path 
FROM backup_disks 
WHERE backup_job_id = ? 
  AND disk_index = ?
  AND status = 'completed'
LIMIT 1;
```

### 4. Repository Manager Integration
**Issue:** `RepositoryManager.GetBackupFromAnyRepository()` doesn't know about new schema  
**Impact:** Can't locate backup files

**Options:**
- **A) Update RepositoryManager** - Add backup_disks awareness (impacts other code)
- **B) Bypass RepositoryManager** - Query backup_disks directly in mount_manager.go (cleaner)

**Recommendation:** Option B - Direct database query

### 5. Multi-Disk UX
**Issue:** No way to discover available disks for a backup  
**Impact:** Users don't know which disk_index values are valid

**Needed API:**
```http
GET /api/v1/backups/{backup_id}/disks
Response:
{
  "backup_id": "backup-pgtest1-1759947871",
  "disks": [
    {
      "disk_index": 0,
      "vmware_disk_key": 2000,
      "size_gb": 102,
      "qcow2_path": ".../disk-0/backup-pgtest1-disk0-20251008-192431.qcow2",
      "status": "completed"
    },
    {
      "disk_index": 1,
      "vmware_disk_key": 2001,
      "size_gb": 5,
      "qcow2_path": ".../disk-1/backup-pgtest1-disk1-20251008-192431.qcow2",
      "status": "completed"
    }
  ]
}
```

---

## ✅ Implementation Tasks

### Phase 1: Database Schema Update
- [ ] Update `20251008160000_add_restore_tables.up.sql` migration
  - Add `disk_index INT NOT NULL DEFAULT 0`
  - Update `UNIQUE KEY uk_nbd_device (nbd_device, backup_id, disk_index)` for proper uniqueness
- [ ] Create `.down.sql` migration for rollback
- [ ] Run migration on development database

### Phase 2: Core Restore Logic Refactor
- [ ] Update `restore/mount_manager.go`:
  - [ ] Add `disk_index` to `MountRequest` struct
  - [ ] Rewrite `findBackupFile()` to query `backup_disks` table
  - [ ] Remove `RepositoryManager` dependency (or make it optional)
  - [ ] Update `MountBackup()` to handle disk_index
  - [ ] Update database record creation to include disk_index
- [ ] Update `database/restore_mount_repository.go`:
  - [ ] Add `DiskIndex` field to `RestoreMount` model
  - [ ] Update all Create/Update/Query methods
- [ ] Update `restore/file_browser.go` (if needed):
  - [ ] Verify no hard dependencies on old schema
- [ ] Update `restore/file_downloader.go` (if needed):
  - [ ] Verify no hard dependencies on old schema

### Phase 3: API Endpoints Update
- [ ] Update `api/handlers/restore_handlers.go`:
  - [ ] `MountBackup()` - Parse `disk_index` from request body
  - [ ] Add input validation for disk_index
  - [ ] Add error handling for invalid disk_index
- [ ] Add new endpoint `GET /api/v1/backups/{backup_id}/disks`
  - [ ] List all disks for a backup
  - [ ] Show which disks are available for mounting
  - [ ] Include disk metadata (size, status, qcow2_path)

### Phase 4: Testing
- [ ] Unit tests:
  - [ ] Test findBackupFile() with v2.16.0+ schema
  - [ ] Test disk_index parameter handling
  - [ ] Test multi-disk mount scenarios
- [ ] Integration tests:
  - [ ] Mount disk 0 of pgtest1 backup
  - [ ] Mount disk 1 of pgtest1 backup
  - [ ] List files from each mounted disk
  - [ ] Download files from mounted disks
  - [ ] Verify automatic cleanup
- [ ] Error handling tests:
  - [ ] Invalid backup_id
  - [ ] Invalid disk_index
  - [ ] Missing QCOW2 file
  - [ ] NBD device exhaustion

### Phase 5: Documentation
- [ ] Update `restore/README.md` with v2.16.0+ architecture
- [ ] Update `api-documentation/API_REFERENCE.md` with new endpoints
- [ ] Add examples for multi-disk restore scenarios
- [ ] Update CHANGELOG.md with restore refactor details

---

## 🔍 Testing Data (Production Backups)

From Phase 1 testing (October 8, 2025), we have real multi-disk backup data:

```sql
-- pgtest1 backup (2 disks)
SELECT * FROM backup_jobs WHERE id = 'backup-pgtest1-1759947871';
-- id: backup-pgtest1-1759947871
-- vm_backup_context_id: ctx-backup-pgtest1-20251006-203401
-- status: completed

SELECT * FROM backup_disks WHERE backup_job_id = 'backup-pgtest1-1759947871';
-- disk_index=0, qcow2_path=.../disk-0/backup-pgtest1-disk0-20251008-192431.qcow2, size_gb=102
-- disk_index=1, qcow2_path=.../disk-1/backup-pgtest1-disk1-20251008-192431.qcow2, size_gb=5
```

**Test Scenarios:**
1. Mount disk 0: `{"backup_id": "backup-pgtest1-1759947871", "disk_index": 0}`
2. Mount disk 1: `{"backup_id": "backup-pgtest1-1759947871", "disk_index": 1}`
3. List available disks: `GET /api/v1/backups/backup-pgtest1-1759947871/disks`

---

## 📝 Implementation Notes

### Migration Strategy
- **Backward Compatibility:** Old `backup_id`-only mounts default to `disk_index=0`
- **Validation:** API validates disk_index against backup_disks table
- **Error Messages:** Clear errors for invalid disk selections

### Database Query Pattern
```go
// Query backup_disks for QCOW2 path
func findBackupFile(ctx context.Context, backupID string, diskIndex int) (string, error) {
    var disk BackupDisk
    err := db.Where("backup_job_id = ? AND disk_index = ? AND status = 'completed'", 
                    backupID, diskIndex).
            First(&disk).Error
    if err != nil {
        return "", fmt.Errorf("disk not found: backup_id=%s, disk_index=%d", backupID, diskIndex)
    }
    return disk.QCOW2Path, nil
}
```

### API Request Examples

**Mount Single-Disk Backup (legacy compatibility):**
```bash
curl -X POST http://localhost:8082/api/v1/restore/mount \
  -H "Content-Type: application/json" \
  -d '{"backup_id": "backup-singledisk-123"}'
# disk_index defaults to 0
```

**Mount Multi-Disk Backup (explicit disk selection):**
```bash
curl -X POST http://localhost:8082/api/v1/restore/mount \
  -H "Content-Type: application/json" \
  -d '{
    "backup_id": "backup-pgtest1-1759947871",
    "disk_index": 0
  }'
```

**List Available Disks:**
```bash
curl http://localhost:8082/api/v1/backups/backup-pgtest1-1759947871/disks
```

---

## 🚀 Rollout Plan

1. **Development Testing** - Test with pgtest1 multi-disk backup (October 8 data)
2. **Migration Validation** - Verify schema updates work correctly
3. **API Testing** - Test all restore endpoints with new schema
4. **Documentation** - Update all docs before marking complete
5. **Production Deploy** - Roll out with v2.23.0+ SHA binary

---

## 📚 Related Documentation

- `start_here/PHASE_1_CONTEXT_HELPER.md` - Backup architecture details
- `start_here/CHANGELOG.md` - v2.16.0-v2.22.0 backup evolution
- `project-goals/phases/phase-1-vmware-backup.md` - Phase 1 status
- `project-goals/modules/04-restore-engine.md` - Module 04 enhanced context
- `api-documentation/DB_SCHEMA.md` - Database schema reference

---

**Status:** 🔴 Ready for implementation  
**Estimated Time:** 4-6 hours  
**Complexity:** Medium (schema changes + logic refactor)  
**Risk:** Low (restore is new feature, no production usage yet)

