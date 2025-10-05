# Task 4: File-Level Restore - Deployment Updates

**Date:** 2025-10-05  
**Status:** ✅ **COMPLETE**  
**Version:** v1.1.0-task4-restore

---

## 📋 Summary

Updated deployment scripts and migrations to support Task 4 (File-Level Restore) infrastructure across all SHA appliance deployments.

## 🔄 Changes Made

### 1. Database Migrations Created

**Location:** `/home/oma_admin/sendense/deployment/sha-appliance/migrations/`

#### Migration 1: `20251005120000_add_restore_tables.*`
- **Purpose:** Create `restore_mounts` table for tracking active QCOW2 mounts
- **Features:**
  - Tracks mount ID, backup ID, mount path, NBD device
  - Filesystem type detection
  - Idle timeout tracking (expires_at)
  - Unique constraints on NBD device and mount path
  - CASCADE DELETE on backup_id foreign key

#### Migration 2: `20251005130000_add_disk_id_to_backup_jobs.*`
- **Purpose:** Add `disk_id` column to `backup_jobs` for multi-disk VM support
- **Features:**
  - Integer column with DEFAULT 0
  - Index on (vm_context_id, disk_id, backup_type) for backup chain queries
  - Required for repository GetBackup() queries

### 2. Deployment Script Updates

**File:** `deployment/sha-appliance/scripts/deploy-sha-complete.sh`  
**Version:** v1.0.0-unified-schema → v1.1.0-task4-restore

**Added Sections:**

#### Database Migrations (after schema import)
```bash
# Run database migrations for additional features
log "${YELLOW}🔄 Running database migrations...${NC}"
if [ -f "${SCRIPT_DIR}/run-migrations.sh" ]; then
    bash "${SCRIPT_DIR}/run-migrations.sh"
    check_success "Database migrations"
fi
```

#### File-Level Restore Infrastructure Setup
```bash
# Create restore mount directory
sudo mkdir -p /mnt/sendense/restore
sudo chown oma_admin:oma_admin /mnt/sendense/restore

# Load NBD kernel module (16 devices)
sudo modprobe nbd max_part=8

# Verify qemu-nbd installation
which qemu-nbd > /dev/null 2>&1
```

### 3. Migration Runner Integration

**Tool:** `deployment/sha-appliance/scripts/run-migrations.sh`

**Features:**
- Tracks applied migrations in `schema_migrations` table
- Idempotent execution (safe to run multiple times)
- Skips already-applied migrations
- Applies migrations in chronological order
- Handles errors gracefully (filters "Duplicate column" warnings)

## 🧪 Testing

### Preprod Validation (10.245.246.136)

✅ **All 9 Restore API Endpoints Tested:**
1. Mount backup - QCOW2 mounted on /dev/nbd0
2. List root files - 4 directories found
3. Browse nested directories - /var/www/html/index.html found
4. Get file metadata - correct size, mode, timestamps
5. Download file - exact content retrieved
6. Download directory as ZIP - valid archive created
7. Path traversal protection - malicious paths rejected
8. Resource monitoring - shows 1 active mount
9. Unmount backup - resources freed

✅ **Migration System Tested:**
- All 5 migrations tracked in `schema_migrations` table
- Skips already-applied migrations correctly
- Idempotent execution verified

✅ **Infrastructure Validated:**
- `/mnt/sendense/restore` directory created with correct permissions
- NBD module loaded (16 devices: /dev/nbd0-15)
- qemu-nbd installation confirmed
- Database tables exist with correct schema

## 📁 File Changes

### New Files
```
deployment/sha-appliance/migrations/
├── 20251005120000_add_restore_tables.up.sql      (NEW)
├── 20251005120000_add_restore_tables.down.sql    (NEW)
├── 20251005130000_add_disk_id_to_backup_jobs.up.sql   (NEW)
└── 20251005130000_add_disk_id_to_backup_jobs.down.sql (NEW)

source/current/control-plane/database/migrations/
├── 20251005120000_add_restore_tables.up.sql      (NEW)
├── 20251005120000_add_restore_tables.down.sql    (NEW)
├── 20251005130000_add_disk_id_to_backup_jobs.up.sql   (NEW)
└── 20251005130000_add_disk_id_to_backup_jobs.down.sql (NEW)
```

### Modified Files
```
deployment/sha-appliance/scripts/deploy-sha-complete.sh
- Version: v1.0.0-unified-schema → v1.1.0-task4-restore
- Added: Migration runner integration
- Added: File-level restore infrastructure setup
- Added: NBD module loading
- Added: Mount directory creation
```

### Unmodified (kept for reference)
```
scripts/deploy-real-production-oma.sh
- Not the primary deployment script for SHA appliances
- Left for legacy/reference purposes
```

## 🚀 Deployment Impact

### Fresh Deployments
- Migrations run automatically after schema import
- Infrastructure created during Phase 2 (Database Setup)
- No manual intervention required

### Existing Deployments
- Run migrations manually: `bash run-migrations.sh`
- Idempotent - safe to run on systems with partial setup
- Existing tables/columns will be skipped
- Infrastructure setup is additive (mkdir -p, modprobe || true)

## 🔧 Issues Fixed During Development

1. **Schema Mismatch:** `disk_id` column missing from `backup_jobs` table
   - **Fixed:** Migration adds column with proper index
   
2. **Permission Issues:** qemu-nbd/mount commands need sudo
   - **Fixed:** Binary code updated to use sudo for system commands
   - **Deployed:** sendense-hub-v2.8.1-sudo-fix binary

3. **Mount Directory:** `/mnt/sendense/restore` didn't exist
   - **Fixed:** Deployment script creates directory with correct permissions

4. **NBD Module:** Not loaded by default
   - **Fixed:** Deployment script loads module with 16 device support

## 📊 Database Schema Changes

### Table: `restore_mounts`
```sql
CREATE TABLE restore_mounts (
    id VARCHAR(64) PRIMARY KEY,
    backup_id VARCHAR(64) NOT NULL,
    mount_path VARCHAR(512) NOT NULL,
    nbd_device VARCHAR(32) NOT NULL,
    filesystem_type VARCHAR(32),
    mount_mode ENUM('read-only') DEFAULT 'read-only',
    status ENUM('mounting','mounted','unmounting','failed') DEFAULT 'mounting',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL,
    
    INDEX idx_backup_id (backup_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_status (status),
    INDEX idx_nbd_device (nbd_device),
    UNIQUE KEY uk_nbd_device_active (nbd_device),
    UNIQUE KEY uk_mount_path_active (mount_path),
    FOREIGN KEY (backup_id) REFERENCES backup_jobs(id) ON DELETE CASCADE
);
```

### Column: `backup_jobs.disk_id`
```sql
ALTER TABLE backup_jobs 
ADD COLUMN disk_id INT NOT NULL DEFAULT 0 AFTER vm_name;

CREATE INDEX idx_backup_vm_disk ON backup_jobs(vm_context_id, disk_id, backup_type);
```

## ✅ Acceptance Criteria Met

- [x] Migrations created for schema changes
- [x] Deployment script updated with migration runner
- [x] Infrastructure setup automated (mount directory, NBD, qemu-nbd)
- [x] Idempotent execution (safe to run multiple times)
- [x] Tested on preprod (10.245.246.136)
- [x] All 9 restore API endpoints validated
- [x] No breaking changes to existing functionality
- [x] Documentation complete

## 🎯 Next Steps

1. **Deploy to Production:** Run updated `deploy-sha-complete.sh` on production SHA appliances
2. **Update Package:** Include new migrations in deployment packages
3. **Update Binaries:** Deploy sendense-hub-v2.8.1-sudo-fix or later
4. **Task 5:** Begin Backup API Endpoints implementation (trigger backups via API)

## 📝 Notes

- Migration system uses timestamp-based versioning (YYYYMMDDHHmmss)
- All migrations have up/down scripts for rollback capability
- `run-migrations.sh` is reentrant - can be run multiple times safely
- Deployment script checks for qemu-utils but doesn't auto-install (Phase 1 responsibility)
- NBD device allocation: /dev/nbd0-7 for restore, /dev/nbd8-15 for backup operations

---

**Deployment Script Version:** v1.1.0-task4-restore  
**Binary Version:** sendense-hub-v2.8.1-sudo-fix  
**Testing Status:** ✅ PASSED (preprod 10.245.246.136)  
**Production Ready:** ✅ YES
