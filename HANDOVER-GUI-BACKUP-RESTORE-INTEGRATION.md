# GUI Integration Handover: Backup & Restore Flows

**Date:** October 8, 2025  
**Backend Status:** ✅ PRODUCTION READY  
**Purpose:** Handover document for GUI integration of backup and restore functionality  
**Target Session:** GUI Developer (Sonnet)  
**Backend Binary:** sendense-hub-v2.24.0-restore-v2-refactor

---

## 🎯 MISSION

Integrate backup and restore functionality into the Sendense GUI (React/Vue/etc.). Backend APIs are production-ready and tested. Your job is to create intuitive UI flows for:

1. **VM Backup Management**
   - Create VM backups (full + incremental)
   - View backup history and chains
   - Monitor backup progress

2. **File-Level Restore**
   - Mount backup disks
   - Browse files (Windows Explorer-style)
   - Download individual files or folders
   - Unmount backups

---

## 🚀 QUICK START

### Backend Service
```bash
# Backend running at:
http://localhost:8082/api/v1

# Test connection:
curl http://localhost:8082/api/v1/health

# Authentication:
# All /api/v1/* endpoints require Bearer token
# Header: Authorization: Bearer <token>
```

### Key Documentation
- **API Reference:** `/home/oma_admin/sendense/source/current/api-documentation/OMA.md`
- **Database Schema:** `/home/oma_admin/sendense/source/current/api-documentation/DB_SCHEMA.md`
- **Backup Context:** `/home/oma_admin/sendense/start_here/PHASE_1_CONTEXT_HELPER.md`
- **Recent Changes:** `/home/oma_admin/sendense/start_here/CHANGELOG.md` (see v2.24.0)

---

## 📋 BACKEND ARCHITECTURE OVERVIEW

### Multi-Disk Backup Architecture (v2.16.0+)

```
vm_backup_contexts (master)
├─ context_id (PK)
├─ vm_name
├─ repository_id
└─ backup_type

    ↓ (FK: vm_backup_context_id)

backup_jobs (parent)
├─ id (PK) "backup-pgtest1-1759947871"
├─ vm_backup_context_id
├─ status (started → in_progress → completed)
└─ created_at

    ↓ (FK: backup_job_id)

backup_disks (per-disk)
├─ id (PK, auto-increment)
├─ backup_job_id (FK)
├─ disk_index (0, 1, 2...)
├─ vmware_disk_key (2000, 2001...)
├─ qcow2_path (actual file path)
├─ disk_change_id (CBT tracking)
├─ size_gb
└─ status (completed, failed)

    ↓ (FK: backup_disk_id, CASCADE DELETE)

restore_mounts (file-level restore)
├─ id (UUID)
├─ backup_disk_id (FK)
├─ mount_path
├─ nbd_device
└─ expires_at (1 hour auto-cleanup)
```

**Key Points:**
- Backups are **VM-level** (all disks backed up together)
- Each disk gets a separate QCOW2 file
- Restore is **disk-level** (select which disk to mount)
- CASCADE DELETE: Deleting backup auto-cleans restore mounts

---

## 🔑 AUTHENTICATION

```javascript
// Login
POST /api/v1/auth/login
{
  "username": "admin",
  "password": "password"
}

// Response:
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-10-09T00:00:00Z"
}

// Use token in all subsequent requests:
headers: {
  'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...',
  'Content-Type': 'application/json'
}
```

---

## 💾 BACKUP API FLOW

### 1. List VMs Available for Backup

```bash
GET /api/v1/vm-contexts
```

**Response:**
```json
{
  "vm_contexts": [
    {
      "context_id": "ctx-backup-pgtest1-20251006-203401",
      "vm_name": "pgtest1",
      "vm_path": "[datastore1] pgtest1/pgtest1.vmx",
      "vcenter_host": "vcenter.example.com",
      "datacenter": "DC1",
      "current_status": "ready",
      "cpu_count": 2,
      "memory_mb": 4096,
      "os_type": "windows",
      "power_state": "poweredOn",
      "created_at": "2025-10-06T20:34:01Z"
    }
  ],
  "count": 1
}
```

**GUI Usage:** Display VM list in table/cards with backup button

---

### 2. Get VM Disks (Optional - for info display)

```bash
GET /api/v1/vm-contexts/{vm_name}/disks
```

**Response:**
```json
{
  "vm_name": "pgtest1",
  "disks": [
    {
      "disk_index": 0,
      "vmware_disk_key": 2000,
      "label": "Hard disk 1",
      "size_gb": 102,
      "datastore": "[datastore1]",
      "path": "pgtest1/pgtest1.vmdk"
    },
    {
      "disk_index": 1,
      "vmware_disk_key": 2001,
      "label": "Hard disk 2",
      "size_gb": 5,
      "datastore": "[datastore1]",
      "path": "pgtest1/pgtest1_1.vmdk"
    }
  ],
  "total_disks": 2
}
```

**GUI Usage:** Show disk info before backup (optional)

---

### 3. Start VM Backup

```bash
POST /api/v1/backups
Content-Type: application/json

{
  "vm_name": "pgtest1",
  "repository_id": "1",
  "backup_type": "full"
}
```

**Request Fields:**
- `vm_name` (required): VM to backup
- `repository_id` (required): Where to store backup
- `backup_type` (required): "full" or "incremental"

**Response:**
```json
{
  "backup_id": "backup-pgtest1-1759947871",
  "vm_context_id": "ctx-backup-pgtest1-20251006-203401",
  "vm_name": "pgtest1",
  "disk_results": [
    {
      "disk_id": 0,
      "disk_index": 0,
      "vmware_disk_key": 2000,
      "nbd_port": 10104,
      "qcow2_path": "/backup/repository/ctx-backup-pgtest1.../disk-0/backup-pgtest1-disk0-20251008-192431.qcow2",
      "status": "qemu_started"
    },
    {
      "disk_id": 1,
      "disk_index": 1,
      "vmware_disk_key": 2001,
      "nbd_port": 10105,
      "qcow2_path": "/backup/repository/ctx-backup-pgtest1.../disk-1/backup-pgtest1-disk1-20251008-192431.qcow2",
      "status": "qemu_started"
    }
  ],
  "backup_type": "full",
  "status": "started",
  "created_at": "2025-10-08T19:24:31+01:00"
}
```

**GUI Usage:** 
- Show "Backup Started" notification
- Store `backup_id` for progress monitoring
- Display per-disk status

---

### 4. Monitor Backup Progress

**Option A: Polling (Simple)**
```bash
GET /api/v1/backups/{backup_id}
```

**Response:**
```json
{
  "backup_id": "backup-pgtest1-1759947871",
  "vm_name": "pgtest1",
  "status": "in_progress",
  "backup_type": "full",
  "disks": [
    {
      "disk_index": 0,
      "status": "completed",
      "size_gb": 102,
      "qcow2_path": "/backup/repository/.../disk-0/..."
    },
    {
      "disk_index": 1,
      "status": "in_progress",
      "size_gb": 5
    }
  ],
  "progress_percent": 50,
  "bytes_transferred": 54760833024,
  "total_bytes": 109521666048,
  "created_at": "2025-10-08T19:24:31Z",
  "completed_at": null
}
```

**GUI Usage:**
- Poll every 2-5 seconds
- Show progress bar
- Display per-disk status
- Show "Completed" when status = "completed"

---

### 5. List Backup History

```bash
GET /api/v1/backups?vm_name=pgtest1
```

**Response:**
```json
{
  "backups": [
    {
      "backup_id": "backup-pgtest1-1759947871",
      "vm_name": "pgtest1",
      "backup_type": "full",
      "status": "completed",
      "total_size_gb": 107,
      "disk_count": 2,
      "created_at": "2025-10-08T19:24:31Z",
      "completed_at": "2025-10-08T19:28:15Z"
    },
    {
      "backup_id": "backup-pgtest1-1759901593",
      "vm_name": "pgtest1",
      "backup_type": "incremental",
      "status": "completed",
      "total_size_gb": 0.055,
      "disk_count": 2,
      "created_at": "2025-10-08T06:33:13Z",
      "completed_at": "2025-10-08T06:34:01Z"
    }
  ],
  "total_count": 2
}
```

**GUI Usage:**
- Display backup history table
- Show backup type badges (Full/Incremental)
- Show size and duration
- Add "Restore" button per backup

---

### 6. Get Backup Chain (Recovery Points)

```bash
GET /api/v1/backups/chain?vm_name=pgtest1&repository_id=1
```

**Response:**
```json
{
  "vm_name": "pgtest1",
  "repository_id": "1",
  "chain": [
    {
      "backup_id": "backup-pgtest1-1759947871",
      "backup_type": "full",
      "sequence_number": 1,
      "created_at": "2025-10-08T19:24:31Z",
      "size_gb": 107,
      "is_restorable": true
    },
    {
      "backup_id": "backup-pgtest1-1759901593",
      "backup_type": "incremental",
      "sequence_number": 2,
      "created_at": "2025-10-08T06:33:13Z",
      "size_gb": 0.055,
      "parent_backup_id": "backup-pgtest1-1759947871",
      "is_restorable": true
    }
  ],
  "total_backups": 2,
  "chain_size_gb": 107.055
}
```

**GUI Usage:**
- Display recovery point timeline
- Show full → incremental chain
- Visual indicator of backup type

---

## 🔄 FILE-LEVEL RESTORE API FLOW

### 1. Mount Backup Disk

```bash
POST /api/v1/restore/mount
Content-Type: application/json

{
  "backup_id": "backup-pgtest1-1759947871",
  "disk_index": 0
}
```

**Request Fields:**
- `backup_id` (required): Backup job ID from backup history
- `disk_index` (required): Which disk to mount (0, 1, 2...)
  - For single-disk VMs: always 0
  - For multi-disk VMs: user selects which disk

**Response:**
```json
{
  "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398",
  "backup_id": "backup-pgtest1-1759947871",
  "backup_disk_id": 44,
  "disk_index": 0,
  "mount_path": "/mnt/sendense/restore/e4805a6f-8ee7-4f3c-8309-2f12362c7398",
  "nbd_device": "/dev/nbd0",
  "filesystem_type": "ntfs",
  "status": "mounted",
  "created_at": "2025-10-08T21:19:37+01:00",
  "expires_at": "2025-10-08T22:19:37+01:00"
}
```

**GUI Usage:**
- Show "Mounting backup..." loader
- Store `mount_id` for file browsing
- Show expiration warning (1 hour auto-cleanup)
- Display filesystem type

**Error Handling:**
```json
{
  "error": "disk not found: backup_id=backup-xxx, disk_index=5"
}
```

---

### 2. Browse Files (Root Directory)

```bash
GET /api/v1/restore/{mount_id}/files?path=/
```

**Response:**
```json
{
  "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398",
  "path": "/",
  "files": [
    {
      "name": "$RECYCLE.BIN",
      "path": "/$RECYCLE.BIN",
      "type": "directory",
      "size": 0,
      "mode": "0777",
      "modified_time": "2024-01-29T11:49:24.0654245Z",
      "is_symlink": false
    },
    {
      "name": "Recovery",
      "path": "/Recovery",
      "type": "directory",
      "size": 0,
      "mode": "0777",
      "modified_time": "2024-01-29T19:35:50.6318693Z",
      "is_symlink": false
    },
    {
      "name": "Windows",
      "path": "/Windows",
      "type": "directory",
      "size": 0,
      "mode": "0777",
      "modified_time": "2025-09-02T06:21:20Z",
      "is_symlink": false
    }
  ],
  "total_count": 3
}
```

**File Object Fields:**
- `name`: Display name
- `path`: Full path (use for navigation/download)
- `type`: "file" or "directory" (for icon selection)
- `size`: File size in bytes (0 for directories)
- `modified_time`: ISO 8601 timestamp
- `is_symlink`: Boolean

**GUI Usage:**
- Display as table or list
- Show folder icon for type="directory"
- Show file icon for type="file"
- Click folder → Browse subdirectory
- Click file → Download file

---

### 3. Navigate into Subdirectory

```bash
GET /api/v1/restore/{mount_id}/files?path=/Recovery/WindowsRE
```

**Response:**
```json
{
  "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398",
  "path": "/Recovery/WindowsRE",
  "files": [
    {
      "name": "ReAgent.xml",
      "path": "/Recovery/WindowsRE/ReAgent.xml",
      "type": "file",
      "size": 1129,
      "mode": "0777",
      "modified_time": "2025-09-02T06:21:20.1985298+01:00",
      "is_symlink": false
    },
    {
      "name": "boot.sdi",
      "path": "/Recovery/WindowsRE/boot.sdi",
      "type": "file",
      "size": 3170304,
      "mode": "0777",
      "modified_time": "2021-05-08T09:14:41.6426299+01:00",
      "is_symlink": false
    },
    {
      "name": "winre.wim",
      "path": "/Recovery/WindowsRE/winre.wim",
      "type": "file",
      "size": 505453500,
      "mode": "0777",
      "modified_time": "2024-01-29T12:59:01.5190276Z",
      "is_symlink": false
    }
  ],
  "total_count": 3
}
```

**GUI Usage:**
- Show breadcrumb navigation (/ → Recovery → WindowsRE)
- Back button navigates to parent path
- Double-click folder navigates deeper

---

### 4. Download Individual File

```bash
GET /api/v1/restore/{mount_id}/download?path=/Recovery/WindowsRE/ReAgent.xml
```

**Response:** File stream with headers
```
Content-Type: application/xml
Content-Disposition: attachment; filename="ReAgent.xml"
Content-Length: 1129
```

**GUI Usage:**
```javascript
// JavaScript example
const downloadFile = async (mountId, filePath, fileName) => {
  const response = await fetch(
    `/api/v1/restore/${mountId}/download?path=${encodeURIComponent(filePath)}`,
    {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }
  );
  
  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = fileName;
  a.click();
  window.URL.revokeObjectURL(url);
};
```

---

### 5. Download Directory as Archive

```bash
GET /api/v1/restore/{mount_id}/download-directory?path=/Recovery&format=zip
```

**Query Params:**
- `path` (required): Directory path
- `format` (optional): "zip" or "tar.gz" (default: "zip")

**Response:** ZIP/TAR.GZ archive stream
```
Content-Type: application/zip
Content-Disposition: attachment; filename="Recovery.zip"
```

**GUI Usage:**
- Show "Download Folder" button on directories
- Format selector (ZIP/TAR.GZ)
- Progress indicator for large folders

---

### 6. List Active Mounts

```bash
GET /api/v1/restore/mounts
```

**Response:**
```json
{
  "mounts": [
    {
      "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398",
      "backup_disk_id": 44,
      "mount_path": "/mnt/sendense/restore/...",
      "nbd_device": "/dev/nbd0",
      "filesystem_type": "ntfs",
      "status": "mounted",
      "created_at": "2025-10-08T21:19:37+01:00",
      "expires_at": "2025-10-08T22:19:37+01:00"
    }
  ],
  "count": 1
}
```

**GUI Usage:**
- Show active mounts in sidebar
- Display countdown timer to expiration
- "Browse Files" button per mount

---

### 7. Unmount Backup

```bash
DELETE /api/v1/restore/{mount_id}
```

**Response:**
```json
{
  "message": "backup unmounted successfully",
  "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398"
}
```

**GUI Usage:**
- "Close" button in file browser
- Auto-unmount after user closes browser
- Show "Unmounting..." loader

---

## 🎨 GUI DESIGN RECOMMENDATIONS

### Backup Flow

**1. VM List Page**
```
┌─────────────────────────────────────────┐
│  Virtual Machines                       │
├─────────────────────────────────────────┤
│  Search: [__________] [+ Discover VMs]  │
│                                          │
│  ┌───────────────────────────────────┐  │
│  │ 🖥️  pgtest1                        │  │
│  │ Status: Ready                      │  │
│  │ 2 CPU | 4GB RAM | 2 Disks         │  │
│  │ Last Backup: 2 hours ago           │  │
│  │ [Backup Now] [View Backups] [...]  │  │
│  └───────────────────────────────────┘  │
│                                          │
│  ┌───────────────────────────────────┐  │
│  │ 🖥️  web-server-01                  │  │
│  │ Status: Ready                      │  │
│  │ [Backup Now] [View Backups]        │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

**2. Backup Wizard**
```
┌─────────────────────────────────────────┐
│  Create Backup: pgtest1                 │
├─────────────────────────────────────────┤
│                                          │
│  Backup Type:                            │
│  ◉ Full Backup                           │
│  ○ Incremental Backup                    │
│                                          │
│  Repository: [Dropdown: Repository 1 ▼] │
│                                          │
│  Disks to backup:                        │
│  ☑ Disk 0 - Hard disk 1 (102 GB)        │
│  ☑ Disk 1 - Hard disk 2 (5 GB)          │
│                                          │
│  Total Size: 107 GB                      │
│                                          │
│  [Cancel] [Start Backup]                 │
└─────────────────────────────────────────┘
```

**3. Backup Progress**
```
┌─────────────────────────────────────────┐
│  Backup in Progress: pgtest1            │
├─────────────────────────────────────────┤
│                                          │
│  Overall Progress:                       │
│  ████████████░░░░░░░░░░░░░░ 50%         │
│  54 GB / 107 GB transferred              │
│                                          │
│  Disk 0 (102 GB): ████████████ Complete │
│  Disk 1 (5 GB):   ████░░░░░░░░ 40%      │
│                                          │
│  Estimated Time: 5 minutes               │
│                                          │
│  [Cancel Backup]                         │
└─────────────────────────────────────────┘
```

**4. Backup History**
```
┌─────────────────────────────────────────┐
│  Backup History: pgtest1                │
├─────────────────────────────────────────┤
│  Date         Type    Size    Status    │
│  ─────────────────────────────────────  │
│  Oct 8 19:24  Full    107GB   ✓ [Restore]│
│  Oct 8 06:33  Incr    55MB    ✓ [Restore]│
│  Oct 7 12:15  Full    107GB   ✓ [Restore]│
│  Oct 6 20:34  Full    107GB   ✓ [Restore]│
│                                          │
│  Total: 321 GB across 4 backups          │
└─────────────────────────────────────────┘
```

### Restore Flow

**1. Restore Options Modal**
```
┌─────────────────────────────────────────┐
│  Restore Options                        │
├─────────────────────────────────────────┤
│                                          │
│  Select disk to restore:                 │
│  ◉ Disk 0 - Hard disk 1 (102 GB)        │
│  ○ Disk 1 - Hard disk 2 (5 GB)          │
│                                          │
│  Restore type:                           │
│  ◉ Browse & Download Files               │
│  ○ Full VM Restore (future)              │
│                                          │
│  [Cancel] [Continue]                     │
└─────────────────────────────────────────┘
```

**2. File Browser**
```
┌─────────────────────────────────────────┐
│  File Browser - pgtest1 (Disk 0)       │
│  Mounted: /dev/nbd0 | Expires: 52 min  │
├─────────────────────────────────────────┤
│  Path: / › Recovery › WindowsRE         │
│  [⬆ Up] [🏠 Root] [⟲ Refresh] [✕ Close]│
├─────────────────────────────────────────┤
│  Name            Size      Modified     │
│  ───────────────────────────────────── │
│  📄 ReAgent.xml   1 KB     Sep 2 2025   │
│  📄 boot.sdi      3 MB     May 8 2021   │
│  📄 winre.wim     505 MB   Jan 29 2024  │
│                                          │
│  [Download Selected] [Download Folder]   │
└─────────────────────────────────────────┘
```

---

## 🧪 TESTING DATA

### Test VM: pgtest1
- **VM Name:** pgtest1
- **Disks:** 2 (102GB + 5GB)
- **OS:** Windows Server 2022
- **Last Backup:** backup-pgtest1-1759947871

### Test Backup IDs
```bash
# Full backup (multi-disk)
backup-pgtest1-1759947871

# Incremental backup
backup-pgtest1-1759901593
```

### Test API Calls
```bash
# List VMs
curl http://localhost:8082/api/v1/vm-contexts

# Start backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'

# Mount for restore
curl -X POST http://localhost:8082/api/v1/restore/mount \
  -H "Content-Type: application/json" \
  -d '{"backup_id":"backup-pgtest1-1759947871","disk_index":0}'

# Browse files
curl "http://localhost:8082/api/v1/restore/{mount_id}/files?path=/"

# Download file
curl "http://localhost:8082/api/v1/restore/{mount_id}/download?path=/Recovery/WindowsRE/ReAgent.xml" \
  -o ReAgent.xml
```

---

## ⚠️ IMPORTANT NOTES

### Multi-Disk Handling
- **Backup:** Always backs up ALL disks together (VM-level consistency)
- **Restore:** User selects which disk to mount (disk_index: 0, 1, 2...)
- **GUI:** If VM has >1 disk, show disk selector before mounting

### Automatic Cleanup
- **Restore Mounts:** Auto-unmount after 1 hour idle
- **GUI:** Show countdown timer, warn user before expiration
- **Recommendation:** Implement activity tracking (keep mount alive while browsing)

### Error Handling
- **Network Errors:** Show user-friendly message, retry button
- **404 Not Found:** "Backup not found" message
- **500 Server Error:** "Server error, please try again"
- **Expired Mounts:** "Mount expired, please remount backup"

### Performance
- **File Browsing:** Fast (local filesystem access)
- **File Download:** Speed depends on file size, show progress bar
- **Large Folders:** Warn user before downloading >1GB folders

### Security
- **Read-Only Mounts:** Users cannot modify backup files
- **Path Traversal:** Backend validates all paths (no ../ escapes)
- **Authentication:** Token required for all API calls

---

## 📚 ADDITIONAL RESOURCES

### Documentation Files
```
/home/oma_admin/sendense/source/current/api-documentation/
├─ OMA.md                    # Complete API reference (restore: lines 287-545)
├─ DB_SCHEMA.md              # Database schema
└─ API_DB_MAPPING.md         # API-to-database mapping

/home/oma_admin/sendense/start_here/
├─ PHASE_1_CONTEXT_HELPER.md # Backup architecture overview
├─ CHANGELOG.md              # Recent changes (v2.16.0-v2.24.0)
└─ PROJECT_RULES.md          # Development rules

/home/oma_admin/sendense/job-sheets/
├─ 2025-10-08-restore-system-v2-refactor.md  # Restore refactor details
└─ 2025-10-08-restore-test-results.txt       # Test validation
```

### Database Tables (for reference)
- `vm_backup_contexts` - VM backup master records
- `backup_jobs` - Parent backup job tracking
- `backup_disks` - Per-disk backup records (QCOW2 paths here)
- `backup_chains` - Backup chain metadata (full → incremental links)
- `restore_mounts` - Active restore mount tracking

### Key Concepts
- **CBT (Changed Block Tracking):** VMware feature for incremental backups
- **QCOW2:** Disk image format with backing file support
- **NBD (Network Block Device):** Protocol for exporting QCOW2 as mountable device
- **Backing Chain:** Incremental backups point to parent full backup
- **CASCADE DELETE:** Database constraint auto-cleaning related records

---

## 🎯 SUCCESS CRITERIA

Your GUI integration is complete when:

1. ✅ User can start full backup from VM list
2. ✅ User can start incremental backup from VM list
3. ✅ User can monitor backup progress in real-time
4. ✅ User can view backup history per VM
5. ✅ User can mount backup for file browsing
6. ✅ User can navigate folder hierarchy (Windows Explorer-style)
7. ✅ User can download individual files
8. ✅ User can download folders as ZIP
9. ✅ User can unmount backup
10. ✅ User sees expiration warnings for mounts
11. ✅ All errors are handled gracefully
12. ✅ UI is responsive and intuitive

---

## 🚨 RULES & CONSTRAINTS

Per `.cursorrules`:

1. **DO NOT modify backend APIs** - They are production-ready and tested
2. **DO NOT create mock APIs** - Use real backend endpoints
3. **DO follow REST conventions** - Backend follows RESTful design
4. **DO handle errors gracefully** - Show user-friendly messages
5. **DO update documentation** - Document GUI components and flows
6. **DO create job sheet** - Track your work in job-sheets/
7. **DO test thoroughly** - Use pgtest1 test data
8. **DO NOT commit without testing** - Verify all flows work

---

## 🆘 GETTING HELP

If you need clarification:

1. **Read API docs:** `source/current/api-documentation/OMA.md` (lines 287-545 for restore)
2. **Check CHANGELOG:** `start_here/CHANGELOG.md` (see v2.24.0 entry)
3. **Review test results:** `job-sheets/2025-10-08-restore-test-results.txt`
4. **Ask user:** If something is unclear or missing

---

**Backend Status:** ✅ READY  
**Your Mission:** Build intuitive GUI for backup & restore flows  
**Expected Duration:** 4-8 hours  
**Good Luck!** 🚀

