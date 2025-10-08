# Job Sheet: Backup Context Architecture Refactoring

**Job Sheet ID:** 2025-10-08-backup-context-architecture  
**Created:** October 8, 2025  
**Status:** 🟡 IN PROGRESS - Phase 1 Complete  
**Priority:** HIGH - Eliminates time-window hack, aligns with replication architecture  
**Estimated Effort:** 6-8 hours (database migration + code refactoring + testing)  
**Prerequisites:** 
- ✅ Multi-disk incremental backups working (with time-window hack)
- ✅ QCOW2 backing chains operational
- ✅ Current backup system understood

**Phase 1 Status: ✅ COMPLETE (October 8, 2025)**
- Database tables created: vm_backup_contexts, backup_disks
- Existing data migrated: 1 context, 21 disks, 27 jobs linked
- CASCADE DELETE verified
- Evidence: Migration logs, row counts, FK tests

---

## 🎯 OBJECTIVE

**Goal:** Refactor backup system to use proper VM-centric context architecture matching the replication system pattern, eliminating the time-window hack for parent job ID routing.

**Business Value:**
- Eliminates fragile time-window matching (currently 1 hour)
- Supports arbitrarily large backups (100TB+)
- Proper foreign key relationships with CASCADE DELETE
- Consistent architecture across replication and backup systems
- Per-disk change_id tracking in dedicated table
- Professional database design

---

## 🐛 CURRENT PROBLEM

### **The Time-Window Hack:**

```go
// sha/workflows/backup.go line 596
timeWindow := parentJob.CreatedAt.Add(1 * time.Hour)  // ❌ FRAGILE!

// Matches per-disk jobs by:
WHERE vm_name = ? 
  AND disk_id = ? 
  AND created_at >= ?
  AND created_at <= ?  // ← What if backup takes 2 hours?
```

**Failure Scenarios:**
- 100TB VM takes 5 hours to backup → completion API fails after 1 hour
- System clock skew → jobs don't match
- Concurrent backups of same VM → wrong job matched
- Database timezone issues → matching fails

### **Architecture Mismatch:**

**Replication (PROPER):**
```
vm_replication_contexts
  ├─ replication_jobs (FK: vm_context_id)
     └─ vm_disks (FK: vm_context_id, job_id)
        └─ disk_change_id (per-disk tracking)
```

**Backup (BROKEN):**
```
backup_jobs (flat structure)
  ├─ backup-pgtest1-1759940386 (parent)
  ├─ backup-pgtest1-disk0-... (orphan, matched by time!)
  └─ backup-pgtest1-disk1-... (orphan, matched by time!)
```

---

## 📋 IMPLEMENTATION PLAN

### **Phase 1: Database Schema (2-3 hours)**

#### Step 1.1: Create `vm_backup_contexts` table
```sql
CREATE TABLE vm_backup_contexts (
  context_id VARCHAR(64) PRIMARY KEY DEFAULT (uuid()),
  vm_name VARCHAR(255) NOT NULL,
  vmware_vm_id VARCHAR(255) NOT NULL,
  vm_path VARCHAR(500) NOT NULL,
  vcenter_host VARCHAR(255) NOT NULL,
  datacenter VARCHAR(255) NOT NULL,
  repository_id VARCHAR(64) NOT NULL,
  
  -- Backup statistics
  total_backups_run INT DEFAULT 0,
  successful_backups INT DEFAULT 0,
  failed_backups INT DEFAULT 0,
  last_backup_id VARCHAR(64),
  last_backup_type ENUM('full', 'incremental'),
  last_backup_at TIMESTAMP,
  
  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  
  -- Foreign keys
  FOREIGN KEY (repository_id) REFERENCES backup_repositories(id),
  
  -- Constraints
  UNIQUE KEY uk_vm_backup (vm_name, repository_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

#### Step 1.2: Create `backup_disks` table
```sql
CREATE TABLE backup_disks (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  vm_backup_context_id VARCHAR(64) NOT NULL,
  backup_job_id VARCHAR(64) NOT NULL,
  
  -- Disk identification
  disk_index INT NOT NULL,  -- 0, 1, 2... (consistent with SHA handler)
  vmware_disk_key INT NOT NULL,  -- 2000, 2001... (VMware's key)
  size_gb BIGINT NOT NULL,
  unit_number INT,
  
  -- Backup tracking
  disk_change_id VARCHAR(255),  -- VMware CBT change ID
  qcow2_path VARCHAR(512),  -- Path to QCOW2 file
  bytes_transferred BIGINT DEFAULT 0,
  status ENUM('pending', 'running', 'completed', 'failed') DEFAULT 'pending',
  
  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP NULL,
  
  -- Foreign keys
  FOREIGN KEY (vm_backup_context_id) REFERENCES vm_backup_contexts(context_id) ON DELETE CASCADE,
  FOREIGN KEY (backup_job_id) REFERENCES backup_jobs(id) ON DELETE CASCADE,
  
  -- Constraints
  UNIQUE KEY uk_backup_disk (backup_job_id, disk_index),
  INDEX idx_change_id_lookup (vm_backup_context_id, disk_index, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

#### Step 1.3: Migrate `backup_jobs` table
```sql
-- Add new columns
ALTER TABLE backup_jobs 
  ADD COLUMN vm_backup_context_id VARCHAR(64) AFTER id,
  ADD FOREIGN KEY fk_backup_context (vm_backup_context_id) 
      REFERENCES vm_backup_contexts(context_id) ON DELETE CASCADE;

-- disk_id and change_id will be deprecated (moved to backup_disks)
-- Keep them temporarily for backward compatibility during migration
-- Will drop in Phase 3 after validation
```

#### Step 1.4: Migration script
```sql
-- File: migrations/20251008_backup_context_architecture.sql

START TRANSACTION;

-- Create new tables (from Steps 1.1 and 1.2)
-- ...

-- Migrate existing data
-- For each unique VM in backup_jobs:
INSERT INTO vm_backup_contexts (
  context_id, vm_name, vmware_vm_id, vm_path, 
  vcenter_host, datacenter, repository_id
)
SELECT 
  CONCAT('ctx-backup-', vm_name, '-', DATE_FORMAT(MIN(created_at), '%Y%m%d-%H%i%s')) as context_id,
  vm_name,
  '' as vmware_vm_id,  -- Populate from vm_replication_contexts if available
  '' as vm_path,
  '' as vcenter_host,
  '' as datacenter,
  repository_id
FROM backup_jobs
GROUP BY vm_name, repository_id;

-- Link existing backup_jobs to contexts
UPDATE backup_jobs bj
JOIN vm_backup_contexts vbc ON bj.vm_name = vbc.vm_name AND bj.repository_id = vbc.repository_id
SET bj.vm_backup_context_id = vbc.context_id;

-- Migrate disk records to backup_disks
-- (Only if we have disk_id populated - skip parent jobs)
INSERT INTO backup_disks (
  vm_backup_context_id, backup_job_id, disk_index,
  vmware_disk_key, size_gb, disk_change_id,
  qcow2_path, bytes_transferred, status, completed_at
)
SELECT 
  bj.vm_backup_context_id,
  bj.id,
  COALESCE(bj.disk_id, 0) as disk_index,
  COALESCE(bj.disk_id, 0) + 2000 as vmware_disk_key,  -- Estimate
  0 as size_gb,  -- Will be populated from vm_disks on next backup
  bj.change_id,
  bj.repository_path as qcow2_path,
  bj.bytes_transferred,
  CASE 
    WHEN bj.status = 'completed' THEN 'completed'
    WHEN bj.status = 'failed' THEN 'failed'
    WHEN bj.status = 'running' THEN 'running'
    ELSE 'pending'
  END as status,
  bj.completed_at
FROM backup_jobs bj
WHERE bj.vm_backup_context_id IS NOT NULL
  AND bj.disk_id IS NOT NULL;  -- Only migrate per-disk jobs

COMMIT;
```

---

### **Phase 2: Code Refactoring (3-4 hours)**

#### Step 2.1: Update BackupHandler - StartBackup
**File:** `sha/api/handlers/backup_handlers.go`

```go
// STEP 3: Get or create VM backup context
func (bh *BackupHandler) getOrCreateBackupContext(
    ctx context.Context, 
    req *BackupRequest, 
    vmDisks []database.VMDisk,
) (*database.VMBackupContext, error) {
    
    // Try to find existing context
    var backupContext database.VMBackupContext
    err := bh.db.GetGormDB().
        Where("vm_name = ? AND repository_id = ?", req.VMName, req.RepositoryID).
        First(&backupContext).Error
    
    if err == gorm.ErrRecordNotFound {
        // Create new context
        backupContext = database.VMBackupContext{
            ContextID:      fmt.Sprintf("ctx-backup-%s-%s", req.VMName, time.Now().Format("20060102-150405")),
            VMName:         req.VMName,
            VMwareVMID:     vmContext.VMwareVMID,  // From vm_replication_contexts
            VMPath:         vmContext.VMPath,
            VCenterHost:    creds.VCenterHost,
            Datacenter:     vmContext.Datacenter,
            RepositoryID:   req.RepositoryID,
            CreatedAt:      time.Now(),
        }
        
        if err := bh.db.GetGormDB().Create(&backupContext).Error; err != nil {
            return nil, fmt.Errorf("failed to create backup context: %w", err)
        }
    } else if err != nil {
        return nil, fmt.Errorf("failed to query backup context: %w", err)
    }
    
    return &backupContext, nil
}
```

#### Step 2.2: Update BackupEngine - CompleteBackup
**File:** `sha/workflows/backup.go`

```go
// NEW: No more time-window hack!
func (be *BackupEngine) CompleteBackup(ctx context.Context, backupID string, diskID int, changeID string, bytesTransferred int64) error {
    log.WithFields(log.Fields{
        "backup_id": backupID,
        "disk_id":   diskID,
        "change_id": changeID,
    }).Info("📝 Completing backup disk")

    now := time.Now()
    
    // Update backup_disks table directly (no time-window matching!)
    result := be.db.GetGormDB().
        Model(&database.BackupDisk{}).
        Where("backup_job_id = ? AND disk_index = ?", backupID, diskID).
        Updates(map[string]interface{}{
            "status":            "completed",
            "disk_change_id":    changeID,
            "bytes_transferred": bytesTransferred,
            "completed_at":      now,
        })
    
    if result.Error != nil {
        return fmt.Errorf("failed to update backup disk: %w", result.Error)
    }
    
    if result.RowsAffected == 0 {
        return fmt.Errorf("backup disk not found: job_id=%s disk_index=%d", backupID, diskID)
    }

    // Check if all disks completed
    var totalDisks, completedDisks int64
    be.db.GetGormDB().Model(&database.BackupDisk{}).
        Where("backup_job_id = ?", backupID).
        Count(&totalDisks)
    be.db.GetGormDB().Model(&database.BackupDisk{}).
        Where("backup_job_id = ? AND status = ?", backupID, "completed").
        Count(&completedDisks)
    
    // If all disks completed, mark parent job complete
    if totalDisks == completedDisks {
        be.db.GetGormDB().
            Model(&database.BackupJob{}).
            Where("id = ?", backupID).
            Updates(map[string]interface{}{
                "status":       "completed",
                "completed_at": now,
            })
        
        log.WithField("backup_id", backupID).Info("✅ All disks completed, backup job finished")
    }
    
    return nil
}
```

#### Step 2.3: Update GetChangeID endpoint
**File:** `sha/api/handlers/backup_handlers.go`

```go
// GET /api/v1/backups/changeid?vm_name=pgtest1&disk_id=0
func (bh *BackupHandler) GetChangeID(w http.ResponseWriter, r *http.Request) {
    vmName := r.URL.Query().Get("vm_name")
    diskIDStr := r.URL.Query().Get("disk_id")
    
    diskID, _ := strconv.Atoi(diskIDStr)
    
    // Query backup_disks table for most recent completed backup
    var backupDisk database.BackupDisk
    err := bh.db.GetGormDB().
        Joins("JOIN vm_backup_contexts vbc ON backup_disks.vm_backup_context_id = vbc.context_id").
        Joins("JOIN backup_jobs bj ON backup_disks.backup_job_id = bj.id").
        Where("vbc.vm_name = ? AND backup_disks.disk_index = ? AND backup_disks.status = ? AND backup_disks.disk_change_id IS NOT NULL", 
              vmName, diskID, "completed").
        Order("backup_disks.completed_at DESC").
        First(&backupDisk).Error
    
    if err == gorm.ErrRecordNotFound {
        bh.sendJSON(w, http.StatusOK, map[string]string{
            "vm_name":   vmName,
            "disk_id":   fmt.Sprintf("%d", diskID),
            "change_id": "",
            "message":   "No previous backup found",
        })
        return
    }
    
    if err != nil {
        bh.sendError(w, http.StatusInternalServerError, "database error", err.Error())
        return
    }
    
    bh.sendJSON(w, http.StatusOK, map[string]string{
        "vm_name":   vmName,
        "disk_id":   fmt.Sprintf("%d", diskID),
        "change_id": backupDisk.DiskChangeID,
        "message":   "Previous change_id found",
    })
}
```

---

### **Phase 3: Testing (1-2 hours)**

#### Test 3.1: Fresh full backup
```bash
# Expected: Creates context + parent job + 2 disk records
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{
    "vm_name": "pgtest1",
    "repository_id": "repo-local-1759780081",
    "backup_type": "full"
  }'

# Verify database:
mysql -e "SELECT * FROM vm_backup_contexts WHERE vm_name='pgtest1'\G"
mysql -e "SELECT * FROM backup_disks WHERE backup_job_id='<job_id>'\G"
```

#### Test 3.2: Incremental backup
```bash
# Expected: Reuses context, creates new job + disk records, proper change_id lookup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{
    "vm_name": "pgtest1",
    "repository_id": "repo-local-1759780081",
    "backup_type": "incremental"
  }'

# Verify change_ids stored per-disk:
mysql -e "SELECT disk_index, disk_change_id FROM backup_disks WHERE backup_job_id='<job_id>'\G"
```

#### Test 3.3: Multi-disk completion
```bash
# Simulate backup client completion (both disks)
curl -X POST http://localhost:8082/api/v1/backups/<job_id>/complete \
  -d '{"disk_id": 0, "change_id": "TEST_DISK0", "bytes_transferred": 1000}'

curl -X POST http://localhost:8082/api/v1/backups/<job_id>/complete \
  -d '{"disk_id": 1, "change_id": "TEST_DISK1", "bytes_transferred": 2000}'

# Expected: Both disk records updated, parent job marked completed
```

#### Test 3.4: Large backup (simulate time)
```bash
# Create backup, wait 2 hours (simulated by manually setting created_at)
# Then complete - should work without time-window limitation

mysql -e "UPDATE backup_jobs SET created_at = DATE_SUB(NOW(), INTERVAL 2 HOUR) WHERE id='<job_id>'"

# Completion should still work (no time window!)
curl -X POST http://localhost:8082/api/v1/backups/<job_id>/complete \
  -d '{"disk_id": 0, "change_id": "LATE_COMPLETION", "bytes_transferred": 1000}'
```

---

## 📊 SUCCESS CRITERIA

**Phase 1 Complete:**
- ✅ All 3 tables created without errors
- ✅ Migration script runs successfully
- ✅ Existing backup data migrated to new structure
- ✅ Foreign key constraints working

**Phase 2 Complete:**
- ✅ Full backup creates context + parent job + disk records
- ✅ Incremental backup reuses context
- ✅ Completion API updates backup_disks directly
- ✅ GetChangeID queries backup_disks table
- ✅ No time-window code remaining

**Phase 3 Complete:**
- ✅ Fresh full backup test passes
- ✅ Incremental backup test passes
- ✅ Multi-disk completion test passes
- ✅ Large backup (2+ hour) test passes
- ✅ CASCADE DELETE working (delete context → all jobs/disks deleted)

**Documentation:**
- ✅ PHASE_1_CONTEXT_HELPER.md updated
- ✅ DB_SCHEMA.md updated with new tables
- ✅ API_REFERENCE.md updated (completion API changes)
- ✅ Job sheet completed with evidence

---

## ⚠️ RISKS & MITIGATION

**Risk 1:** Existing backups become orphaned
- **Mitigation:** Migration script creates contexts for all existing VMs

**Risk 2:** GUI breaks (depends on backup_jobs structure)
- **Mitigation:** Keep backup_jobs columns temporarily, mark as deprecated

**Risk 3:** Incomplete migration leaves inconsistent data
- **Mitigation:** Use SQL transaction, rollback on any error

**Risk 4:** Performance degradation (more JOINs)
- **Mitigation:** Add indexes on foreign keys and lookup paths

---

## 🎓 LESSONS FROM REPLICATION SYSTEM

**Good Patterns to Copy:**
1. ✅ Context table with statistics (total_backups_run, etc.)
2. ✅ FK relationships with CASCADE DELETE
3. ✅ Per-disk tracking in separate table
4. ✅ Unique constraints prevent duplicates
5. ✅ Status ENUM for consistency

**Improvements for Backup:**
1. ✅ Simpler disk identification (just disk_index, not multiple IDs)
2. ✅ QCOW2 path stored in backup_disks (not parent job)
3. ✅ Repository relationship at context level

---

## 📚 FILES TO MODIFY

**Database:**
- `migrations/20251008_backup_context_architecture.sql` (NEW)
- `sha/database/models.go` (add VMBackupContext, BackupDisk models)

**Backend:**
- `sha/api/handlers/backup_handlers.go` (StartBackup, CompleteBackup, GetChangeID)
- `sha/workflows/backup.go` (BackupEngine methods)
- `sha/storage/local_repository.go` (CreateBackup - link to context)

**Documentation:**
- `api-documentation/DB_SCHEMA.md`
- `api-documentation/API_REFERENCE.md`
- `start_here/PHASE_1_CONTEXT_HELPER.md`

---

## 📅 TIMELINE

**Session 1 (Today):** Planning + Job Sheet ✅  
**Session 2:** Phase 1 - Database schema + migration  
**Session 3:** Phase 2 - Code refactoring  
**Session 4:** Phase 3 - Testing + Documentation  

**Estimated Total:** 6-8 hours across 3-4 sessions

---

## 🔗 REFERENCES

- **Replication Schema:** `SHOW CREATE TABLE vm_replication_contexts`
- **Current Backup Schema:** `SHOW CREATE TABLE backup_jobs`
- **Time-Window Hack:** `sha/workflows/backup.go` lines 582-621
- **.cursorrules:** `/home/oma_admin/sendense/.cursorrules`

---

**Job Sheet Status:** 🟡 PLANNING → Ready for implementation approval


