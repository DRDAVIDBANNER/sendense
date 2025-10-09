# Backend API Integration Guide: Backup & Restore

**Date:** October 8, 2025  
**Backend Status:** ✅ PRODUCTION READY  
**Purpose:** Technical API reference for integrating backup and restore functionality  
**Backend Binary:** sendense-hub-v2.24.0-restore-v2-refactor

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

## 📋 BACKEND ARCHITECTURE

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

**Endpoint:** `POST /api/v1/auth/login`

**Backend Logic:**
- Handler: `api/handlers/auth_handlers.go:Login()`
- Validates username/password against database
- Generates JWT token with 24-hour expiration
- Returns token for use in subsequent API calls

**Request:**
```json
{
  "username": "admin",
  "password": "password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-10-09T00:00:00Z"
}
```

**Usage:** Include token in all subsequent requests:
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

## 💾 BACKUP API ENDPOINTS

### 1. List VMs Available for Backup

**Endpoint:** `GET /api/v1/vm-contexts`

**Backend Logic:**
- Handler: `api/handlers/vm_context_handlers.go:ListVMContexts()`
- Queries: `vm_replication_contexts` table
- Filters: Returns all VMs with auto_added=true
- Joins: Includes group memberships
- Returns: VM metadata (CPU, memory, OS, power state)

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

---

### 2. Get VM Disks

**Endpoint:** `GET /api/v1/vm-contexts/{vm_name}/disks`

**Backend Logic:**
- Handler: `api/handlers/vm_context_handlers.go:GetVMDisks()`
- Queries: `vm_disks` table
- Filters: `WHERE vm_name = ?`
- Returns: Disk metadata from VMware discovery

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

---

### 3. Start VM Backup

**Endpoint:** `POST /api/v1/backups`

**Backend Logic:**
- Handler: `api/handlers/backup_handlers.go:StartBackup()`
- Service: `backup/backup_service.go:StartVMBackup()`
- Process:
  1. Validates VM exists in `vm_replication_contexts`
  2. Queries `vm_disks` to get all disks for VM
  3. Creates `backup_jobs` parent record
  4. Creates `backup_disks` child records (one per disk)
  5. For each disk:
     - Creates QCOW2 file (full) or backing chain (incremental)
     - Starts qemu-nbd process on unique port
     - Records NBD port, export name, PID
  6. Calls VMA to start VMware NBD export via SSH tunnel
  7. Returns multi-disk job details

**Database Writes:**
- `backup_jobs` - Parent job record
- `backup_disks` - Per-disk records with QCOW2 paths
- `backup_chains` - Chain metadata (if first backup)

**Request:**
```json
{
  "vm_name": "pgtest1",
  "repository_id": "1",
  "backup_type": "full"
}
```

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
      "nbd_export_name": "pgtest1-disk-2000",
      "qcow2_path": "/backup/repository/ctx-backup-pgtest1.../disk-0/backup-pgtest1-disk0-20251008-192431.qcow2",
      "qemu_nbd_pid": 3956432,
      "status": "qemu_started"
    },
    {
      "disk_id": 1,
      "disk_index": 1,
      "vmware_disk_key": 2001,
      "nbd_port": 10105,
      "nbd_export_name": "pgtest1-disk-2001",
      "qcow2_path": "/backup/repository/ctx-backup-pgtest1.../disk-1/backup-pgtest1-disk1-20251008-192431.qcow2",
      "qemu_nbd_pid": 3956438,
      "status": "qemu_started"
    }
  ],
  "nbd_targets_string": "2000:nbd://127.0.0.1:10104/pgtest1-disk-2000,2001:nbd://127.0.0.1:10105/pgtest1-disk-2001",
  "backup_type": "full",
  "repository_id": "1",
  "status": "started",
  "created_at": "2025-10-08T19:24:31+01:00"
}
```

---

### 4. Monitor Backup Progress

**Endpoint:** `GET /api/v1/backups/{backup_id}`

**Backend Logic:**
- Handler: `api/handlers/backup_handlers.go:GetBackup()`
- Queries:
  1. `backup_jobs` - Parent job status
  2. `backup_disks` - Per-disk status and progress
- Aggregates: Total bytes transferred, progress percentage
- Status Flow: `started` → `in_progress` → `completed` or `failed`

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
      "qcow2_path": "/backup/repository/.../disk-0/...",
      "bytes_transferred": 109521666048,
      "progress_percent": 100
    },
    {
      "disk_index": 1,
      "status": "in_progress",
      "size_gb": 5,
      "bytes_transferred": 2684354560,
      "progress_percent": 50
    }
  ],
  "progress_percent": 75,
  "bytes_transferred": 112206020608,
  "total_bytes": 114806120448,
  "created_at": "2025-10-08T19:24:31Z",
  "completed_at": null
}
```

**Polling Recommendation:** Poll every 2-5 seconds during backup

---

### 5. Complete Backup (Internal)

**Endpoint:** `POST /api/v1/backups/{backup_id}/complete`

**Backend Logic:**
- Handler: `api/handlers/backup_handlers.go:CompleteBackup()`
- Service: `backup/completion_service.go:CompleteBackup()`
- Called by: VMA after VMware NBD export completes
- Process:
  1. Validates backup exists and is in_progress
  2. Updates `backup_disks` status to "completed"
  3. Records disk_change_id from VMware CBT
  4. Stops qemu-nbd processes
  5. Updates `backup_jobs` status to "completed"
  6. Updates `backup_chains` total_backups count
  7. Triggers cleanup of old backups (retention policy)

**Database Writes:**
- `backup_disks.status` = "completed"
- `backup_disks.disk_change_id` = VMware CBT ID
- `backup_jobs.status` = "completed"
- `backup_chains.total_backups` incremented

---

### 6. List Backup History

**Endpoint:** `GET /api/v1/backups?vm_name={vm_name}`

**Backend Logic:**
- Handler: `api/handlers/backup_handlers.go:ListBackups()`
- Queries:
  1. `backup_jobs` - Filter by VM name
  2. `backup_disks` - Aggregate size per job
- Joins: Counts disk_count per backup
- Sorting: Newest first (ORDER BY created_at DESC)
- Filtering: Optional status filter

**Query:**
```sql
SELECT 
  bj.id AS backup_id,
  bj.vm_name,
  bj.backup_type,
  bj.status,
  COUNT(bd.id) AS disk_count,
  SUM(bd.size_gb) AS total_size_gb,
  bj.created_at,
  bj.completed_at
FROM backup_jobs bj
LEFT JOIN backup_disks bd ON bd.backup_job_id = bj.id
WHERE bj.vm_name = ?
GROUP BY bj.id
ORDER BY bj.created_at DESC
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

---

### 7. Get Backup Chain

**Endpoint:** `GET /api/v1/backups/chain?vm_name={vm_name}&repository_id={repository_id}`

**Backend Logic:**
- Handler: `api/handlers/backup_handlers.go:GetBackupChain()`
- Queries:
  1. `backup_chains` - Get chain metadata
  2. `backup_jobs` - Get all backups in chain
  3. `backup_disks` - Aggregate sizes
- Returns: Ordered list from full → incrementals
- Links: Parent-child relationships via parent_backup_id

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
      "parent_backup_id": null,
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

---

## 🔄 FILE-LEVEL RESTORE API ENDPOINTS

### 1. Mount Backup Disk

**Endpoint:** `POST /api/v1/restore/mount`

**Backend Logic:**
- Handler: `api/handlers/restore_handlers.go:MountBackup()`
- Service: `restore/mount_manager.go:MountBackup()`
- Process:
  1. Validates backup_id exists in `backup_jobs`
  2. Queries `backup_disks` for QCOW2 path:
     ```sql
     SELECT id, qcow2_path, status, disk_index
     FROM backup_disks
     WHERE backup_job_id = ? AND disk_index = ? AND status = 'completed'
     ```
  3. Checks for existing mount (one mount per disk)
  4. Allocates NBD device from pool (/dev/nbd0-7)
  5. Starts qemu-nbd with read-only flag:
     ```bash
     qemu-nbd --read-only --format=qcow2 --connect=/dev/nbd0 /path/to/backup.qcow2
     ```
  6. Detects filesystem type (ntfs, ext4, xfs, etc.)
  7. Mounts filesystem read-only:
     ```bash
     mount -o ro /dev/nbd0p1 /mnt/sendense/restore/{mount_id}
     ```
  8. Creates `restore_mounts` record with 1-hour expiration
  9. Returns mount details

**Database Writes:**
- `restore_mounts` - New record with FK to `backup_disks.id`

**Request:**
```json
{
  "backup_id": "backup-pgtest1-1759947871",
  "disk_index": 0
}
```

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

**Error Cases:**
- `404`: Backup not found or disk_index invalid
- `409`: Disk already mounted
- `503`: No NBD devices available (max 8 restore mounts)

---

### 2. Browse Files

**Endpoint:** `GET /api/v1/restore/{mount_id}/files?path={path}`

**Backend Logic:**
- Handler: `api/handlers/restore_handlers.go:ListFiles()`
- Service: `restore/file_browser.go:ListFiles()`
- Process:
  1. Validates mount_id exists and is active
  2. Validates path is within mount root (security)
  3. Updates last_accessed_at timestamp
  4. Reads directory using `ioutil.ReadDir()`
  5. For each entry:
     - Gets file info (size, modified time, mode)
     - Determines type (file vs directory)
     - Checks for symlinks
  6. Returns sorted list (directories first)

**Security:**
- Path traversal protection (blocks ../ escapes)
- Read-only access enforced
- Validates all paths against mount root

**Request:**
```bash
GET /api/v1/restore/e4805a6f-8ee7-4f3c-8309-2f12362c7398/files?path=/Recovery/WindowsRE
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

---

### 3. Download File

**Endpoint:** `GET /api/v1/restore/{mount_id}/download?path={path}`

**Backend Logic:**
- Handler: `api/handlers/restore_handlers.go:DownloadFile()`
- Service: `restore/file_downloader.go:DownloadFile()`
- Process:
  1. Validates mount_id and path
  2. Verifies path points to file (not directory)
  3. Updates last_accessed_at timestamp
  4. Opens file for reading
  5. Detects MIME type from extension
  6. Sets response headers:
     - Content-Type
     - Content-Disposition: attachment
     - Content-Length
  7. Streams file to client (http.ServeContent)

**Response Headers:**
```
Content-Type: application/xml
Content-Disposition: attachment; filename="ReAgent.xml"
Content-Length: 1129
```

**Response Body:** File stream

---

### 4. Download Directory

**Endpoint:** `GET /api/v1/restore/{mount_id}/download-directory?path={path}&format={format}`

**Backend Logic:**
- Handler: `api/handlers/restore_handlers.go:DownloadDirectory()`
- Service: `restore/file_downloader.go:DownloadDirectory()`
- Process:
  1. Validates mount_id and path
  2. Verifies path points to directory
  3. Updates last_accessed_at timestamp
  4. Creates archive (ZIP or TAR.GZ):
     - ZIP: Uses `archive/zip` package
     - TAR.GZ: Uses `archive/tar` + `compress/gzip`
  5. Walks directory tree recursively
  6. Adds each file/folder to archive
  7. Streams archive to client

**Query Params:**
- `path` (required): Directory path
- `format` (optional): "zip" or "tar.gz" (default: "zip")

**Response Headers:**
```
Content-Type: application/zip
Content-Disposition: attachment; filename="Recovery.zip"
Transfer-Encoding: chunked
```

**Response Body:** Archive stream

---

### 5. List Active Mounts

**Endpoint:** `GET /api/v1/restore/mounts`

**Backend Logic:**
- Handler: `api/handlers/restore_handlers.go:ListMounts()`
- Service: `restore/mount_manager.go:ListMounts()`
- Queries: `restore_mounts` WHERE status IN ('mounting', 'mounted')
- Returns: All active mounts with expiration times

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
      "expires_at": "2025-10-08T22:19:37+01:00",
      "last_accessed_at": "2025-10-08T21:25:12+01:00"
    }
  ],
  "count": 1
}
```

---

### 6. Unmount Backup

**Endpoint:** `DELETE /api/v1/restore/{mount_id}`

**Backend Logic:**
- Handler: `api/handlers/restore_handlers.go:UnmountBackup()`
- Service: `restore/mount_manager.go:UnmountBackup()`
- Process:
  1. Validates mount_id exists
  2. Unmounts filesystem:
     ```bash
     umount /mnt/sendense/restore/{mount_id}
     ```
  3. Disconnects qemu-nbd:
     ```bash
     qemu-nbd --disconnect /dev/nbd0
     ```
  4. Removes mount directory
  5. Deletes `restore_mounts` record
  6. Releases NBD device back to pool

**Response:**
```json
{
  "message": "backup unmounted successfully",
  "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398"
}
```

---

### 7. Automatic Cleanup (Background Service)

**Service:** `restore/cleanup_service.go`

**Backend Logic:**
- Runs every 5 minutes
- Process:
  1. Queries `restore_mounts` WHERE expires_at < NOW()
  2. For each expired mount:
     - Unmounts filesystem
     - Disconnects qemu-nbd
     - Deletes database record
  3. Also cleans mounts idle > 1 hour (last_accessed_at)
  4. Logs cleanup actions

**Monitoring Endpoint:** `GET /api/v1/restore/cleanup-status`

**Response:**
```json
{
  "running": true,
  "cleanup_interval": "5m",
  "idle_timeout": "1h",
  "active_mount_count": 1,
  "expired_mount_count": 0,
  "last_cleanup": "2025-10-08T21:20:00+01:00",
  "total_cleanups": 15
}
```

---

## 🧪 TESTING DATA

### Test VM: pgtest1
- **VM Name:** pgtest1
- **Disks:** 2 (102GB + 5GB)
- **OS:** Windows Server 2022
- **Backup IDs:**
  - Full: `backup-pgtest1-1759947871`
  - Incremental: `backup-pgtest1-1759901593`

### Quick Test Commands

```bash
# 1. List VMs
curl http://localhost:8082/api/v1/vm-contexts

# 2. Start backup
curl -X POST http://localhost:8082/api/v1/backups \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"vm_name":"pgtest1","repository_id":"1","backup_type":"full"}'

# 3. Check progress
curl http://localhost:8082/api/v1/backups/backup-pgtest1-1759947871 \
  -H "Authorization: Bearer $TOKEN"

# 4. Mount for restore
curl -X POST http://localhost:8082/api/v1/restore/mount \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"backup_id":"backup-pgtest1-1759947871","disk_index":0}'

# 5. Browse files
curl "http://localhost:8082/api/v1/restore/$MOUNT_ID/files?path=/" \
  -H "Authorization: Bearer $TOKEN"

# 6. Download file
curl "http://localhost:8082/api/v1/restore/$MOUNT_ID/download?path=/Recovery/WindowsRE/ReAgent.xml" \
  -H "Authorization: Bearer $TOKEN" \
  -o ReAgent.xml

# 7. Unmount
curl -X DELETE "http://localhost:8082/api/v1/restore/$MOUNT_ID" \
  -H "Authorization: Bearer $TOKEN"
```

---

## ⚠️ IMPORTANT NOTES

### Multi-Disk Handling
- **Backup:** Always backs up ALL disks together (VM-level consistency)
- **Restore:** User selects which disk to mount (disk_index: 0, 1, 2...)
- **Logic:** Query `vm_disks` to get disk count, show selector if >1 disk

### Automatic Cleanup
- **Restore Mounts:** Auto-unmount after 1 hour idle
- **Recommendation:** Update `last_accessed_at` on every file browse/download
- **Expiration:** Show countdown timer to user

### Error Handling
- **404 Not Found:** Backup/mount doesn't exist
- **409 Conflict:** Disk already mounted
- **500 Server Error:** Internal error (qemu-nbd failure, filesystem issue)
- **503 Service Unavailable:** No NBD devices available (max 8 concurrent mounts)

### Performance
- **File Browsing:** Fast (local filesystem access via mount)
- **File Download:** Speed = disk I/O speed
- **Large Folders:** Warn user before downloading >1GB folders as archives

### Security
- **Read-Only Mounts:** Users cannot modify backup files
- **Path Traversal:** Backend validates all paths (no ../ escapes)
- **Authentication:** Bearer token required for all operations
- **Isolation:** Each mount gets unique directory

---

## 📚 ADDITIONAL RESOURCES

### Source Code Locations
```
Backend Handlers:
- api/handlers/backup_handlers.go       # Backup endpoints
- api/handlers/restore_handlers.go      # Restore endpoints
- api/handlers/vm_context_handlers.go   # VM listing

Business Logic:
- backup/backup_service.go              # Backup orchestration
- backup/completion_service.go          # Backup completion
- restore/mount_manager.go              # Mount operations
- restore/file_browser.go               # File browsing
- restore/file_downloader.go            # File downloads
- restore/cleanup_service.go            # Auto cleanup

Database:
- database/backup_repository.go         # Backup queries
- database/restore_mount_repository.go  # Restore mount queries
- database/migrations/20251008160000_add_restore_tables.up.sql
```

### Database Tables
- `vm_backup_contexts` - VM backup master records
- `backup_jobs` - Parent backup job tracking
- `backup_disks` - Per-disk backup records (QCOW2 paths)
- `backup_chains` - Backup chain metadata
- `restore_mounts` - Active restore mount tracking
- `vm_disks` - VM disk metadata from VMware

### Key Concepts
- **CBT (Changed Block Tracking):** VMware feature for incremental backups
- **QCOW2:** Disk image format with backing file support
- **NBD (Network Block Device):** Protocol for mounting QCOW2 as block device
- **Backing Chain:** Incremental backups point to parent full backup
- **CASCADE DELETE:** Database constraint auto-cleaning related records

---

## 🎯 INTEGRATION CHECKLIST

Backend integration is complete when:

1. ✅ Can call backup API to start full backup
2. ✅ Can call backup API to start incremental backup
3. ✅ Can poll backup progress endpoint
4. ✅ Can list backup history per VM
5. ✅ Can call restore mount API
6. ✅ Can call file browse API with hierarchical paths
7. ✅ Can download individual files
8. ✅ Can download directories as archives
9. ✅ Can unmount backups
10. ✅ Handle all error responses gracefully
11. ✅ Include Bearer token in all requests
12. ✅ Understand multi-disk architecture

---

**Backend Status:** ✅ PRODUCTION READY  
**Documentation:** Complete API reference with backend logic  
**Testing:** All endpoints tested with pgtest1 VM  
**Support:** See `/home/oma_admin/sendense/source/current/api-documentation/OMA.md` for full details
