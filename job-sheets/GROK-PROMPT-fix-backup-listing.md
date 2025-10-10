# GROK: Fix Backup Listing Duplicates

**Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-09-fix-backup-listing-duplicates.md`

---

## 🎯 MISSION

Fix backup listing showing 3x duplicate entries. Users see parent + per-disk records as separate backups.

**Current:** 9+ backups for pgtest1 (3 records per backup: parent, disk0, disk1)  
**Expected:** 3-4 backups for pgtest1 (1 parent record per backup)

---

## 🔧 REQUIRED CHANGES

### Backend Fix 1: Filter Per-Disk Records

**File:** `sha/api/handlers/backup_handlers.go`  
**Method:** `ListBackups()`

**Add SQL filter to exclude per-disk records:**

```go
// In the database query, add:
AND backup_id NOT LIKE '%-disk%-%%'

// Example:
query := db.Where("vm_name = ?", vmName).
    Where("status = ?", status).
    Where("backup_id NOT LIKE ?", "%-disk%-%").  // ADD THIS LINE
    Order("created_at DESC")
```

**Why:** Per-disk records (`backup-pgtest1-disk0-...`) are for internal use. Only show parent records (`backup-pgtest1-...`) to users.

---

### Backend Fix 2: Add `disks_count` Field

**File:** `sha/api/handlers/backup_handlers.go`

**Step 1: Update response struct**
```go
type BackupResponse struct {
    // ... existing fields ...
    DisksCount int `json:"disks_count"`  // ADD THIS
}
```

**Step 2: Query disk count**
```go
// When building response, count disks:
var disksCount int
db.Table("backup_disks").
    Where("backup_job_id = ?", backup.ID).
    Count(&disksCount)

response.DisksCount = disksCount
```

---

### Frontend Fix 1: Field Name

**File:** `sendense-gui/components/features/restore/BackupSelector.tsx`  
**Line:** ~141

```typescript
// CHANGE:
{formatSize(backup.total_size_bytes)}  // ❌ Wrong

// TO:
{formatSize(backup.total_bytes)}       // ✅ Correct
```

---

### Frontend Fix 2: Type Definition

**File:** `sendense-gui/src/features/restore/types/index.ts`

```typescript
export interface BackupJob {
  // ... existing fields ...
  total_bytes: number;      // Ensure correct name
  disks_count: number;      // ADD THIS
}
```

---

## 🧪 TEST COMMANDS

**Verify duplicate removal:**
```bash
curl "http://localhost:8082/api/v1/backups?vm_name=pgtest1&status=completed" | jq '.backups | length'
```
**Expected:** ~3 backups (not 9+)

**Verify `disks_count` field:**
```bash
curl "http://localhost:8082/api/v1/backups?vm_name=pgtest1&status=completed" | jq '.backups[0]'
```
**Expected:** Response includes `"disks_count": 2`

**GUI Test:**
- Navigate to `/restore`
- Select "pgtest1"
- Should show: "Oct 9, 14:03 • 102 GB • 2 disks"
- Should NOT show: "NaN undefined • disks"

---

## ✅ ACCEPTANCE CRITERIA

- [ ] Backup selector shows ~3 entries (not 9+)
- [ ] No backup_id containing "-disk" in API response
- [ ] Each backup displays: "Date • Size • N disks"
- [ ] No "NaN undefined" in GUI
- [ ] No console errors
- [ ] Mount functionality still works

---

## 🚨 CRITICAL

**DON'T delete per-disk records from database!** Only filter them from the LIST API response. The mount API needs per-disk records to work.

**Architecture:**
- **Parent records:** For user display (show these)
- **Per-disk records:** For mount operations (hide these from list, but keep in DB)

---

## 📁 FILES TO MODIFY

1. `sha/api/handlers/backup_handlers.go` - Add filter + disks_count
2. `sendense-gui/components/features/restore/BackupSelector.tsx` - Fix field name
3. `sendense-gui/src/features/restore/types/index.ts` - Add disks_count to interface

---

**START HERE:** Read full job sheet, then make changes. Test thoroughly before claiming complete.

**Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-09-fix-backup-listing-duplicates.md`


