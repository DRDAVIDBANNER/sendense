# Bug Fix Report: Restore Mount Query Error

**Date**: October 9, 2025  
**Severity**: Low (Non-Critical - Service Startup Error)  
**Status**: ✅ FIXED  
**Binary**: `sendense-hub-v2.24.1-restore-query-fix`

---

## Problem

SHA service was logging database errors at startup:

```
Error 1054 (42S22): Unknown column 'backup_id' in 'SELECT'
```

This error appeared every time the restore mount cleanup service attempted to list expired mounts.

---

## Root Cause

**Mismatch between database schema and code query**:

1. **Database table** (`restore_mounts`) was **CORRECT** - it had `backup_disk_id BIGINT` column
2. **One raw SQL query** in code was **STALE** - it was selecting `backup_id` (old field name)

**Location**: `source/current/sha/database/restore_mount_repository.go:160`

The stale query in `ListExpired()` method:
```sql
SELECT id, backup_id, mount_path, nbd_device, ... -- ❌ OLD
```

Should have been:
```sql
SELECT id, backup_disk_id, mount_path, nbd_device, ... -- ✅ CORRECT
```

---

## Impact

- **Service Functionality**: ⚠️ Cleanup service couldn't list expired mounts
- **System Stability**: ✅ No impact - other functions worked normally
- **Data Integrity**: ✅ No corruption - table schema was correct
- **User Operations**: ✅ No impact - backups and restores worked

---

## Fix Applied

### Code Changes

**File**: `source/current/sha/database/restore_mount_repository.go`  
**Line**: 160  
**Change**: 
```diff
- SELECT id, backup_id, mount_path, nbd_device, filesystem_type,
+ SELECT id, backup_disk_id, mount_path, nbd_device, filesystem_type,
```

### Documentation Updates

1. **DB_SCHEMA.md** - Updated `restore_mounts` section:
   - Added v2.16.0+ architecture notes
   - Documented FK to `backup_disks.id` (not `backup_jobs.id`)
   - Added CASCADE DELETE chain documentation
   - Listed all indexes and unique constraints

2. **CHANGELOG.md** - Added v2.24.1 entry:
   - Detailed fix description
   - Root cause analysis
   - File locations and binary name

---

## Testing

### Pre-Fix
```bash
sudo journalctl -u sendense-hub --since "5 minutes ago" | grep backup_id
# ERROR: Unknown column 'backup_id' in 'SELECT'
```

### Post-Fix
```bash
sudo journalctl -u sendense-hub --since "5 minutes ago" | grep backup_id
# No errors found ✅
```

### Verification
```bash
# Service startup - clean logs
sudo systemctl restart sendense-hub
sudo journalctl -u sendense-hub --since "1 minute ago" | grep -i error
# No SQL errors ✅

# Cleanup service working
# Logs show: "🧹 Found expired mounts - starting cleanup"
# Successfully cleaned up mount e4805a6f-8ee7-4f3c-8309-2f12362c7398 ✅

# Discovery still working
curl -X POST http://localhost:8082/api/v1/discovery/discover-vms \
  -H "Content-Type: application/json" \
  -d '{"credential_id": 35, "create_context": false}'
# Returns 98 VMs ✅
```

---

## No Migration Required

**Important**: This was a **code fix only** - NO database migration needed.

The table schema was already correct:
```sql
CREATE TABLE restore_mounts (
  id VARCHAR(64) PRIMARY KEY,
  backup_disk_id BIGINT NOT NULL,  -- ✅ Already correct
  ...
  FOREIGN KEY (backup_disk_id) REFERENCES backup_disks(id) ON DELETE CASCADE
);
```

---

## Files Changed

1. `source/current/sha/database/restore_mount_repository.go` - Fixed SQL query
2. `source/current/api-documentation/DB_SCHEMA.md` - Updated documentation
3. `start_here/CHANGELOG.md` - Added v2.24.1 entry
4. `BUGFIX-2025-10-09-restore-query.md` - This report

---

## Deployment

```bash
# Binary built
cd /home/oma_admin/sendense/source/current/sha/cmd
go build -o /home/oma_admin/sendense/source/builds/sendense-hub-v2.24.1-restore-query-fix main.go

# Deployed
sudo pkill -9 sendense-hub
sudo ln -sf /home/oma_admin/sendense/source/builds/sendense-hub-v2.24.1-restore-query-fix /usr/local/bin/sendense-hub
sudo systemctl start sendense-hub

# Verified
systemctl status sendense-hub  # Active and running ✅
```

---

## Lessons Learned

1. **Always check raw SQL queries** when refactoring database schemas
2. **GORM models can be correct** while raw SQL queries are stale
3. **Schema migrations don't catch raw SQL** - need code review
4. **Cleanup services need testing** - they run infrequently so bugs may not surface immediately

---

## Related

- **Original Schema Change**: v2.16.0 (October 8, 2025) - Moved to per-disk architecture
- **Migration File**: `20251008160000_add_restore_tables.up.sql`
- **Previous Refactor**: SHA v2.24.0-restore-v2-refactor

---

**Status**: ✅ COMPLETE - Service running clean, no errors, all functionality operational


