# Fix Backup Listing - Remove Per-Disk Duplicates

**Date:** October 9, 2025  
**Priority:** HIGH  
**Complexity:** Low  
**Estimated Time:** 15-30 minutes  

---

## 🐛 PROBLEM STATEMENT

### Issue 1: Duplicate Backup Listings
**Symptom:** Users see 3x the actual number of backups for pgtest1 (a 2-disk VM)

**Example:**
```
Backup list shows:
- Incremental Backup (backup-pgtest1-1760011077)          ← Parent
- Incremental Backup (backup-pgtest1-disk0-20251009...)   ← Disk 0
- Incremental Backup (backup-pgtest1-disk1-20251009...)   ← Disk 1

User expects to see:
- Incremental Backup (Oct 9, 14:03) - 102 GB - 2 disks   ← Single entry
```

### Issue 2: Display Field Errors
**Symptom:** GUI shows "NaN undefined • disks" instead of proper metadata

**Console Error:**
```
components/features/restore/BackupSelector.tsx (124:21)
Each child in a list should have a unique "key" prop.
```

**Root Cause:**
- API response missing `disks_count` field
- Frontend using wrong field name: `total_size_bytes` (should be `total_bytes`)

---

## 🔍 ROOT CAUSE ANALYSIS

### Backend API Issue
**File:** `sha/api/handlers/backup_handlers.go` - `ListBackups()` method

**Current Behavior:**
```go
// Query returns ALL records from backup_jobs table
SELECT * FROM backup_jobs WHERE vm_name = ? AND status = ?
```

**Result:**
```json
{
  "backups": [
    {"backup_id": "backup-pgtest1-1760011077"},           // Parent (what we want)
    {"backup_id": "backup-pgtest1-disk0-20251009..."},   // Per-disk (should be hidden)
    {"backup_id": "backup-pgtest1-disk1-20251009..."},   // Per-disk (should be hidden)
    ...
  ]
}
```

**Database Architecture:**
- **Parent Job:** `backup-pgtest1-{timestamp}` - Represents entire VM backup
- **Disk Jobs:** `backup-pgtest1-disk{N}-{timestamp}` - Internal per-disk tracking
- **Purpose:** Parent is for user display; per-disk are for internal mount operations

**Detection Pattern:**
- Parent jobs: `backup_id NOT LIKE '%-disk%-%'`
- Per-disk jobs: `backup_id LIKE '%-disk0-%'` OR `%-disk1-%` etc.

---

## 🛠️ REQUIRED FIXES

### Fix 1: Backend - Filter Per-Disk Records

**File:** `/home/oma_admin/sendense/source/current/sha/api/handlers/backup_handlers.go`

**Method:** `ListBackups(w http.ResponseWriter, r *http.Request)`

**Current Query Location:** Around line 500-550 (search for "SELECT" in ListBackups)

**CHANGE REQUIRED:**

**Option A: SQL Filter (Preferred)**
```go
// Add WHERE clause to exclude per-disk records
query := `SELECT * FROM backup_jobs 
          WHERE vm_name = ? 
          AND status = ? 
          AND backup_id NOT LIKE '%-disk%-%%'
          ORDER BY created_at DESC`
```

**Option B: Go Filter (Alternative)**
```go
// After query, filter results before returning
filteredBackups := make([]*database.BackupJob, 0)
for _, backup := range backups {
    // Only include parent jobs (no "-disk" in backup_id)
    if !strings.Contains(backup.ID, "-disk") {
        filteredBackups = append(filteredBackups, backup)
    }
}
backups = filteredBackups
```

**Why Option A is better:** More efficient (database does filtering), less memory usage

---

### Fix 2: Backend - Add `disks_count` Field

**File:** `/home/oma_admin/sendense/source/current/sha/api/handlers/backup_handlers.go`

**Issue:** API response missing `disks_count` field needed by GUI

**SOLUTION:**

**Step 1: Add field to response struct** (around line 80-120)
```go
type BackupResponse struct {
    BackupID          string    `json:"backup_id"`
    VMName            string    `json:"vm_name"`
    BackupType        string    `json:"backup_type"`
    Status            string    `json:"status"`
    RepositoryID      string    `json:"repository_id"`
    TotalBytes        int64     `json:"total_bytes"`
    BytesTransferred  int64     `json:"bytes_transferred"`
    CreatedAt         time.Time `json:"created_at"`
    StartedAt         *time.Time `json:"started_at,omitempty"`
    CompletedAt       *time.Time `json:"completed_at,omitempty"`
    DisksCount        int       `json:"disks_count"` // ADD THIS
}
```

**Step 2: Query disk count when building response** (in `convertToBackupResponse()` or `ListBackups()`)
```go
// For each backup, count associated disks
var disksCount int
err := bh.db.GetGormDB().Table("backup_disks").
    Where("backup_job_id = ?", backup.ID).
    Count(&disksCount).Error

if err != nil {
    log.WithError(err).Warn("Failed to count disks for backup")
    disksCount = 1 // Default to 1 if query fails
}

response := &BackupResponse{
    BackupID:    backup.ID,
    VMName:      backup.VMName,
    // ... other fields ...
    DisksCount:  disksCount, // ADD THIS
}
```

**Alternative (more efficient):** Use JOIN in initial query
```go
// Option: Add JOIN to get disk count in one query
query := `
SELECT 
    bj.*,
    COUNT(bd.id) as disks_count
FROM backup_jobs bj
LEFT JOIN backup_disks bd ON bd.backup_job_id = bj.id
WHERE bj.vm_name = ? 
  AND bj.status = ?
  AND bj.backup_id NOT LIKE '%-disk%-%%'
GROUP BY bj.id
ORDER BY bj.created_at DESC
`
```

---

### Fix 3: Frontend - Field Name Correction

**File:** `/home/oma_admin/sendense/source/current/sendense-gui/components/features/restore/BackupSelector.tsx`

**Line:** 141

**CHANGE:**
```typescript
// BEFORE (line 141):
{formatSize(backup.total_size_bytes)}  // ❌ Wrong field name

// AFTER:
{formatSize(backup.total_bytes)}       // ✅ Correct field name
```

**Why:** Backend API returns `total_bytes`, not `total_size_bytes`

---

### Fix 4: Frontend - Type Definition

**File:** `/home/oma_admin/sendense/source/current/sendense-gui/src/features/restore/types/index.ts`

**CHANGE:**
```typescript
export interface BackupJob {
  id: string;
  backup_id: string;
  vm_name: string;
  backup_type: 'full' | 'incremental' | 'differential';
  status: string;
  repository_id: string;
  total_bytes: number;        // ✅ Correct name
  bytes_transferred: number;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  disks_count: number;        // ADD THIS
}
```

---

## 🧪 TESTING INSTRUCTIONS

### Test 1: Verify Duplicate Removal

**Command:**
```bash
curl "http://localhost:8082/api/v1/backups?vm_name=pgtest1&status=completed" | jq '.backups | length'
```

**Expected Result:**
- BEFORE: Returns 3 records per backup (parent + 2 disks) = 9+ records
- AFTER: Returns 1 record per backup (parent only) = 3-4 records

**Visual Test:**
- Backup selector should show **3-4 backups**, not 9+
- No duplicate timestamps

### Test 2: Verify Disk Count Display

**GUI Check:**
1. Navigate to `/restore` page
2. Select VM: "pgtest1"
3. View backup dropdown

**Expected Display:**
```
✓ Incremental Backup
  Oct 9, 14:03:26 • 102 GB • 2 disks

✓ Incremental Backup
  Oct 8, 21:36:54 • 102 GB • 2 disks
```

**MUST NOT show:**
- "NaN undefined • disks"
- "0 B" (zero bytes)

### Test 3: Verify Parent Backup IDs

**Command:**
```bash
curl "http://localhost:8082/api/v1/backups?vm_name=pgtest1&status=completed" | jq '.backups[].backup_id'
```

**Expected Output:**
```json
"backup-pgtest1-1760011077"
"backup-pgtest1-1759952151"
"backup-pgtest1-1759947871"
```

**MUST NOT contain:**
```json
"backup-pgtest1-disk0-20251009..."  ❌
"backup-pgtest1-disk1-20251009..."  ❌
```

### Test 4: End-to-End Restore Test

**Full Workflow:**
1. Select VM: "pgtest1"
2. Select backup: "Oct 9, 14:03"
3. Select disk: "Disk 0 (System Disk)"
4. Click "Mount Backup"
5. Verify mount succeeds (should show file browser)

**Expected:** Mount API receives **parent backup_id**, backend resolves to correct per-disk QCOW2

---

## 📊 ACCEPTANCE CRITERIA

**Backend:**
- [ ] `ListBackups()` filters out per-disk records (backup_id NOT LIKE '%-disk%-%%')
- [ ] API response includes `disks_count` field
- [ ] API returns correct `total_bytes` (sum of all disks)
- [ ] No console errors or warnings

**Frontend:**
- [ ] Backup selector shows correct number of backups (1 per VM backup, not per disk)
- [ ] Displays proper metadata: date, size, disk count
- [ ] No "NaN undefined" errors
- [ ] No React key prop warnings

**End-to-End:**
- [ ] User can select and mount any backup
- [ ] Mounted backup shows all disks (file browser works)
- [ ] No duplicate listings confusing users

---

## 🔍 VERIFICATION CHECKLIST

**Before claiming "complete", verify:**

1. **API Response Check:**
   ```bash
   curl "http://localhost:8082/api/v1/backups?vm_name=pgtest1&status=completed" | jq '.backups[0]'
   ```
   - ✅ Contains `disks_count` field
   - ✅ `backup_id` does NOT contain "-disk"
   - ✅ `total_bytes` is non-zero

2. **GUI Display Check:**
   - ✅ Backup dropdown shows ~3 entries (not 9+)
   - ✅ Each entry shows: "Oct 9, 14:03 • 102 GB • 2 disks"
   - ✅ No console errors

3. **Mount Test:**
   - ✅ Select backup, click "Mount Backup"
   - ✅ No 404 or 500 errors
   - ✅ File browser appears

4. **Code Quality:**
   - ✅ No commented-out code
   - ✅ No unused imports
   - ✅ Linter passes

---

## 📁 FILES TO MODIFY

### Backend (Go)
1. `/home/oma_admin/sendense/source/current/sha/api/handlers/backup_handlers.go`
   - Modify: `ListBackups()` method
   - Add: SQL filter or Go filter for per-disk records
   - Add: `disks_count` field to response

### Frontend (TypeScript)
2. `/home/oma_admin/sendense/source/current/sendense-gui/components/features/restore/BackupSelector.tsx`
   - Line 141: Change `total_size_bytes` to `total_bytes`

3. `/home/oma_admin/sendense/source/current/sendense-gui/src/features/restore/types/index.ts`
   - Add: `disks_count: number` to `BackupJob` interface
   - Ensure: `total_bytes` field name (not `total_size_bytes`)

---

## 🚨 IMPORTANT NOTES

### DO NOT Break Existing Functionality
- ⚠️ The **mount API** relies on per-disk records in the database - DON'T delete them!
- ⚠️ Only filter per-disk records from the **list API** response
- ⚠️ Per-disk records must remain in `backup_jobs` table for restore functionality

### Database Context
**Table: `backup_jobs`**
- **Parent records:** Used for user-facing backup list (what we display)
- **Per-disk records:** Used internally by restore mount API (hidden from users)
- **Both are required:** Just show parent records to users, use per-disk records for mounts

### Why This Architecture?
**v2.16.0+ Multi-Disk Design:**
1. **Parent job** tracks overall backup success/failure
2. **Per-disk jobs** track individual QCOW2 files (needed for mount API)
3. **Users see:** Parent jobs only (clean UX)
4. **Restore uses:** Per-disk jobs (mount specific QCOW2 files)

---

## 🎯 EXPECTED OUTCOME

**Before Fix:**
```
pgtest1 backups:
- Incremental Backup (NaN undefined • disks)
- Incremental Backup (NaN undefined • disks)
- Incremental Backup (NaN undefined • disks)
- Incremental Backup (NaN undefined • disks)
- Incremental Backup (NaN undefined • disks)
- Incremental Backup (NaN undefined • disks)
[9+ duplicate entries, confusing user]
```

**After Fix:**
```
pgtest1 backups:
✓ Incremental Backup
  Oct 9, 14:03:26 • 102 GB • 2 disks
  
✓ Incremental Backup
  Oct 8, 21:36:54 • 102 GB • 2 disks
  
✓ Full Backup
  Oct 8, 19:54:31 • 102 GB • 2 disks
[3-4 clean entries, user understands]
```

---

## 📚 REFERENCE DOCUMENTS

- **API Spec:** `/home/oma_admin/sendense/HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md`
- **Database Schema:** `/home/oma_admin/sendense/source/current/api-documentation/DB_SCHEMA.md`
- **Multi-Disk Architecture:** See CHANGELOG.md v2.16.0+ notes

---

**CRITICAL:** Test the mount functionality after this fix! Verify that clicking "Mount Backup" still works correctly.

**END OF JOB SHEET**


