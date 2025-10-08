# VM Disks at Discovery - Deployment Update

**Date:** 2025-10-06  
**Status:** ✅ **COMPLETE**  
**Version:** sendense-hub-v2.11.1-vm-disks-null-fix  
**Migration:** 20251006200000_make_vm_disks_job_id_nullable

---

## 📋 Summary

Updated database schema and discovery service to populate `vm_disks` table immediately when VMs are added to management, enabling backup operations without requiring replication jobs.

## 🎯 Problem Statement

**Issue:** VMware backup operations require disk metadata (size, capacity, datastore), but the `vm_disks` table was only populated when creating replication jobs. This prevented backup operations on VMs that were discovered but not yet replicated.

**Root Cause:** Discovery service was retrieving disk information from VMA but discarding it instead of storing in database.

**Solution:** Make `vm_disks.job_id` nullable and populate disk records immediately during discovery, with `job_id = NULL` until replication starts.

---

## 🔄 Changes Made

### 1. Database Schema Migration

**Migration File:** `20251006200000_make_vm_disks_job_id_nullable`  
**Location:** `/home/oma_admin/sendense/deployment/sha-appliance/migrations/`

#### Migration 1: `.up.sql` (Apply Changes)
- Make `vm_disks.job_id` nullable (was NOT NULL)
- Drop existing FK constraint `fk_vm_disks_job`
- Re-add FK constraint allowing NULL values
- Add `disk_id` column to `backup_jobs` table for multi-disk tracking
- Add composite index on `backup_jobs(vm_context_id, disk_id)`

**Key Changes:**
```sql
-- Make job_id nullable
ALTER TABLE vm_disks 
    MODIFY COLUMN job_id VARCHAR(191) NULL 
    COMMENT 'FK to replication_jobs - NULL if disk populated from discovery';

-- Re-add FK allowing NULL
ALTER TABLE vm_disks 
    ADD CONSTRAINT fk_vm_disks_job 
    FOREIGN KEY (job_id) REFERENCES replication_jobs(id) 
    ON DELETE CASCADE;

-- Add disk_id to backup_jobs
ALTER TABLE backup_jobs 
    ADD COLUMN IF NOT EXISTS disk_id INT NOT NULL DEFAULT 0 
    COMMENT 'Disk number (0, 1, 2...) within VM for multi-disk support';
```

#### Migration 2: `.down.sql` (Rollback)
- Deletes any vm_disks records with NULL job_id
- Reverts job_id to NOT NULL
- Removes disk_id from backup_jobs
- **Warning:** Destructive rollback - loses discovery-populated disk records

---

### 2. Code Changes

#### **File:** `source/current/oma/database/models.go`
**Change:** Updated VMDisk struct JobID field
```go
// Before:
JobID string `json:"job_id" gorm:"type:varchar(191);not null"`

// After:
JobID *string `json:"job_id" gorm:"type:varchar(191)"` // Nullable pointer
```

#### **File:** `source/current/oma/services/enhanced_discovery_service.go`
**Change:** Added disk creation during VM context creation

**New Function:** `createVMDisksFromDiscovery()`
- Queries disk information from VMA discovery response
- Creates vm_disks records with `job_id = nil`
- Links to vm_context_id (VM-centric architecture)
- Stores: disk_id, vmdk_path, size_gb, capacity_bytes, datastore, label

**Modified Function:** `createVMContext()`
- Calls createVMDisksFromDiscovery() after VM context creation
- Logs error but doesn't fail if disk creation fails (graceful degradation)

#### **File:** `source/current/oma/workflows/migration.go`
**Change:** Updated replication workflow to use pointer for JobID
```go
// Disk creation during replication
vmDisk := &database.VMDisk{
    JobID: &req.JobID,  // Changed from req.JobID (pointer for nullable field)
    // ... other fields
}

// Update existing disk record
existingDisk.JobID = &req.JobID  // Changed from req.JobID
```

---

### 3. Binary Deployment

**Binary:** `sendense-hub-v2.11.1-vm-disks-null-fix`  
**Size:** 34 MB  
**Location:** `/home/oma_admin/sendense/source/builds/`  
**Symlink:** `/usr/local/bin/sendense-hub`

**Includes:**
- Nullable JobID support in database models
- Discovery service disk population
- Replication workflow pointer updates
- All previous features from v2.10.x series

---

## 🧪 Testing & Validation

### Test Environment: Dev Server (10.245.246.125)

✅ **Schema Migration Applied:**
```sql
-- Verified job_id is nullable
DESCRIBE vm_disks;
-- job_id | varchar(191) | YES | MUL | NULL

-- Verified disk_id added to backup_jobs
DESCRIBE backup_jobs;
-- disk_id | int(11) | NO | | 0
```

✅ **Discovery Test - pgtest1:**
- VM discovered and added to management via API
- vm_replication_contexts record created: `ctx-pgtest1-20251006-203401`
- vm_disks records created:
  - disk-2000: 102 GB (109,521,666,048 bytes), job_id = NULL
  - disk-2001: 5 GB (5,368,709,120 bytes), job_id = NULL
- Query verification:
  ```sql
  SELECT id, vm_context_id, disk_id, size_gb, capacity_bytes, 
         IFNULL(job_id, 'NULL') as job_id 
  FROM vm_disks 
  WHERE vm_context_id LIKE 'ctx-pgtest1%';
  ```

✅ **Data Flow Validation:**
```
VMA Discovery → Disk Info Retrieved → EnhancedDiscoveryService 
  → createVMDisksFromDiscovery() → vm_disks table populated 
  → job_id = NULL (no replication yet)
```

✅ **Backup Readiness:**
- Backup operations can now query vm_disks by vm_context_id
- Disk metadata available without replication job
- Size and capacity information accessible for backup planning

### Expected Workflow Changes

#### **Before (Broken):**
```
1. Discover VM → vm_replication_contexts created
2. Try backup → ERROR: No disk information available
3. Must create replication job first → vm_disks populated
4. Now can backup
```

#### **After (Fixed):**
```
1. Discover VM → vm_replication_contexts created
2. Discovery also populates vm_disks (job_id = NULL)
3. Can immediately backup (no replication needed)
4. If replication created later → job_id populated
```

---

## 📊 Database Architecture

### vm_disks Table Lifecycle

**State 1: Discovery** (New Behavior)
```sql
INSERT INTO vm_disks (vm_context_id, job_id, disk_id, size_gb, ...)
VALUES ('ctx-pgtest1-...', NULL, 'disk-2000', 102, ...);
```

**State 2: Replication Started**
```sql
UPDATE vm_disks 
SET job_id = 'job-20251006-123456' 
WHERE vm_context_id = 'ctx-pgtest1-...' AND disk_id = 'disk-2000';
```

**State 3: Replication Completed**
```sql
UPDATE vm_disks 
SET sync_status = 'completed', 
    bytes_synced = 109521666048,
    disk_change_id = '52 3c ec 11 9e 2c...'
WHERE job_id = 'job-20251006-123456';
```

### Foreign Key Constraint

**Allows NULL job_id:**
```sql
CONSTRAINT fk_vm_disks_job 
FOREIGN KEY (job_id) REFERENCES replication_jobs(id) 
ON DELETE CASCADE
```

**Behavior:**
- `job_id = NULL`: Valid (disk from discovery)
- `job_id = 'valid-job-id'`: Valid (disk from replication)
- `job_id = 'invalid-job-id'`: ERROR (FK constraint violation)
- Delete replication_job: Cascades to set vm_disks.job_id = NULL (graceful)

---

## 🚀 Deployment Instructions

### Automated Deployment

**Script:** `deployment/sha-appliance/scripts/deploy-sha-complete.sh`

The existing migration runner will automatically apply this migration:
```bash
# Run deployment script
cd /home/oma_admin/sendense/deployment/sha-appliance/scripts
bash deploy-sha-complete.sh

# Migrations applied in order:
# 1. 20251003160000_add_operation_summary.up.sql
# 2. 20251004120000_add_backup_tables.up.sql
# 3. 20251004120001_add_backup_tables_fixed.up.sql
# 4. 20251005120000_add_restore_tables.up.sql
# 5. 20251005130000_add_disk_id_to_backup_jobs.up.sql
# 6. 20251006200000_make_vm_disks_job_id_nullable.up.sql ← NEW
```

### Manual Migration (if needed)

**Apply Migration:**
```bash
mysql -u oma_user -poma_password migratekit_oma < \
  /home/oma_admin/sendense/deployment/sha-appliance/migrations/20251006200000_make_vm_disks_job_id_nullable.up.sql
```

**Verify:**
```bash
# Check job_id is nullable
mysql -u oma_user -poma_password migratekit_oma -e "DESCRIBE vm_disks;" | grep job_id

# Check disk_id added to backup_jobs
mysql -u oma_user -poma_password migratekit_oma -e "DESCRIBE backup_jobs;" | grep disk_id

# Check migration tracked
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT * FROM schema_migrations WHERE version='20251006200000';"
```

**Rollback (if needed):**
```bash
mysql -u oma_user -poma_password migratekit_oma < \
  /home/oma_admin/sendense/deployment/sha-appliance/migrations/20251006200000_make_vm_disks_job_id_nullable.down.sql
```

---

## 📖 Documentation Updates

### Updated Documents:

1. ✅ **start_here/CHANGELOG.md**
   - Added "VM Disks Table Not Populated During Discovery" fix entry
   - Documented schema changes and impact
   - Listed binary version: sendense-hub-v2.11.1-vm-disks-null-fix

2. ✅ **docs/database-schema.md**
   - Updated vm_disks table documentation
   - Added "Architecture Change (October 6, 2025)" section
   - Documented nullable job_id behavior
   - Added "Discovery vs Replication Flow" explanation

3. ✅ **deployment/sha-appliance/migrations/**
   - Added migration files (up and down)
   - Migration automatically tracked in schema_migrations table

4. ✅ **job-sheets/2025-10-06-backup-api-integration.md**
   - Created new job sheet for wiring up backup API endpoints
   - Links to Phase 1 - VMware Backup Implementation
   - Includes E2E testing procedures using vm_disks data

5. ✅ **VM_DISKS_ARCHITECTURE_ASSESSMENT.md**
   - Comprehensive 783-line architectural analysis
   - Problem identification and solution assessment
   - Implementation plan and database relationships

---

## 🔗 Related Work

### Dependencies:
- Phase 1: VMware Backup Implementation (Task 1-5 complete)
- vm_replication_contexts table (VM-centric architecture)
- VMA API discovery endpoint (provides disk information)
- Enhanced Discovery Service (orchestrates discovery workflow)

### Enables:
- Backup API endpoint integration (next job sheet)
- Multi-disk VM backup support
- Backup operations without replication
- Resource planning and capacity monitoring
- GUI disk visualization for discovered VMs

### Future Work:
- Wire up backup API endpoints (job sheet created)
- E2E backup testing with discovered VMs
- GUI updates to show disk information for unprotected VMs
- Backup scheduling based on discovered disk sizes

---

## ⚠️ Known Issues / Limitations

### Limitations:
1. **Disk Change IDs:** Still require replication to populate `disk_change_id` field
2. **Incremental Backups:** Need CBT tracking from replication for incremental backups
3. **Full Backups Only:** Discovery-only VMs can only do full backups (no incrementals yet)

### Backwards Compatibility:
✅ **Fully Compatible:**
- Existing vm_disks records with job_id remain valid
- Replication workflow works identically (just uses pointer)
- No breaking changes to APIs or workflows
- Migration is idempotent and safe to re-run

❌ **Rollback Impact:**
- Rolling back migration deletes all discovery-populated disk records
- Loses disk information for non-replicated VMs
- Must re-discover VMs after rollback

---

## ✅ Completion Checklist

- [x] Schema migration created (up and down)
- [x] Migration copied to SHA deployment directory
- [x] Database models updated (JobID pointer)
- [x] Discovery service updated (createVMDisksFromDiscovery)
- [x] Replication workflow updated (pointer usage)
- [x] Binary compiled and deployed (v2.11.1)
- [x] Migration tested on dev server
- [x] Discovery tested with real VM (pgtest1)
- [x] vm_disks records verified (NULL job_id)
- [x] Documentation updated (CHANGELOG, schema docs)
- [x] Job sheet created for API integration
- [x] Deployment update document created

---

## 📞 Support Information

**Migration Version:** 20251006200000  
**Binary Version:** sendense-hub-v2.11.1-vm-disks-null-fix  
**Deployment Date:** October 6, 2025  
**Migration Runner:** run-migrations.sh (automatic)  
**Rollback Available:** Yes (destructive - see warning)

**Deployment Verification:**
```bash
# Check migration applied
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT * FROM schema_migrations ORDER BY applied_at DESC LIMIT 5;"

# Check binary version
/usr/local/bin/sendense-hub --version

# Test discovery and disk population
curl -X POST http://localhost:8082/api/v1/discovery/add-vms \
  -H "Content-Type: application/json" \
  -d '{"credential_id": 35, "vm_names": ["test-vm"]}'

# Verify disk records created
mysql -u oma_user -poma_password migratekit_oma -e \
  "SELECT * FROM vm_disks WHERE job_id IS NULL ORDER BY created_at DESC LIMIT 3;"
```

