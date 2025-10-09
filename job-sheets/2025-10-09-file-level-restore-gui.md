# File-Level Restore GUI Implementation - Job Sheet

**Date:** October 9, 2025  
**Session:** Phase 2 - File-Level Restore  
**Prerequisites:** Backend fully operational (restore_handlers.go + restore/mount_manager.go)  
**Backend Binary:** sendense-hub-v2.25.2-backup-type-fix (running)  
**Backend Port:** http://localhost:8082/api/v1  

---

## 🎯 OBJECTIVE

Implement a production-ready File-Level Restore interface in the Sendense GUI that allows users to:
1. Select a VM and view its completed backup history
2. Select a backup and choose which disk to mount (multi-disk support)
3. Browse the mounted filesystem with breadcrumb navigation
4. Download individual files or directories as archives
5. Unmount backups when finished
6. View active mounts with auto-expiration countdown timers

---

## 📋 BACKEND API VERIFICATION

### ✅ Database Verification

**Table:** `restore_mounts` ✅ EXISTS

```sql
CREATE TABLE restore_mounts (
    id VARCHAR(64) PRIMARY KEY,
    backup_disk_id BIGINT NOT NULL UNIQUE,
    mount_path VARCHAR(512) NOT NULL,
    nbd_device VARCHAR(32) NOT NULL UNIQUE,
    filesystem_type VARCHAR(32),
    mount_mode ENUM('read-only','read-write') DEFAULT 'read-only',
    status ENUM('mounting','mounted','unmounting','failed','unmounted') NOT NULL DEFAULT 'mounting',
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    unmounted_at TIMESTAMP,
    KEY idx_status (status),
    KEY idx_last_accessed (last_accessed_at),
    KEY idx_expires (expires_at)
);
```

**Foreign Key:**  
`backup_disk_id` → `backup_disks.id` (CASCADE DELETE)

**Other Required Tables:**
- ✅ `backup_jobs` - Parent backup job records
- ✅ `backup_disks` - Per-disk backup tracking with qcow2_path
- ✅ `vm_backup_contexts` - VM backup master context

---

### ✅ API Endpoints Verification

All endpoints tested and operational via `/home/oma_admin/sendense/HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md`

#### 1. Mount Backup Disk
**Endpoint:** `POST /api/v1/restore/mount`  
**Handler:** `restore_handlers.go:MountBackup()`  
**Status:** ✅ VERIFIED

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

**Errors:**
- `404`: Backup not found or disk_index invalid
- `409`: Disk already mounted (limit: 1 mount per disk)
- `503`: No mount slots available (max 8 concurrent mounts)

**Key Logic:**
- Validates backup exists in `backup_jobs`
- Queries `backup_disks` for qcow2_path by backup_job_id + disk_index
- Allocates NBD device (/dev/nbd0-7)
- Starts qemu-nbd --read-only
- Mounts filesystem read-only
- Creates restore_mounts record with 1-hour expiration

---

#### 2. Browse Files
**Endpoint:** `GET /api/v1/restore/{mount_id}/files?path={path}`  
**Handler:** `restore_handlers.go:ListFiles()`  
**Status:** ✅ VERIFIED

**Example:**
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
    }
  ],
  "total_count": 2
}
```

**Key Features:**
- Path traversal protection (blocks ../)
- Read-only access enforced
- Updates last_accessed_at timestamp
- Sorted: directories first, then files

---

#### 3. Download File
**Endpoint:** `GET /api/v1/restore/{mount_id}/download?path={path}`  
**Handler:** `restore_handlers.go:DownloadFile()`  
**Status:** ✅ VERIFIED

**Example:**
```bash
GET /api/v1/restore/e4805a6f/download?path=/Recovery/WindowsRE/ReAgent.xml
```

**Response Headers:**
```
Content-Type: application/xml
Content-Disposition: attachment; filename="ReAgent.xml"
Content-Length: 1129
```

**Response Body:** File stream (binary data)

---

#### 4. Download Directory
**Endpoint:** `GET /api/v1/restore/{mount_id}/download-directory?path={path}&format={format}`  
**Handler:** `restore_handlers.go:DownloadDirectory()`  
**Status:** ✅ VERIFIED

**Query Params:**
- `path` (required): Directory path
- `format` (optional): "zip" or "tar.gz" (default: "zip")

**Example:**
```bash
GET /api/v1/restore/e4805a6f/download-directory?path=/Recovery&format=zip
```

**Response Headers:**
```
Content-Type: application/zip
Content-Disposition: attachment; filename="Recovery.zip"
Transfer-Encoding: chunked
```

**Response Body:** Archive stream (binary data)

**Performance Note:** Warn users before downloading >1GB directories

---

#### 5. List Active Mounts
**Endpoint:** `GET /api/v1/restore/mounts`  
**Handler:** `restore_handlers.go:ListMounts()`  
**Status:** ✅ VERIFIED

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

**Use Cases:**
- Show active mounts panel
- Display countdown timers (expires_at)
- Quick unmount access

---

#### 6. Unmount Backup
**Endpoint:** `DELETE /api/v1/restore/{mount_id}`  
**Handler:** `restore_handlers.go:UnmountBackup()`  
**Status:** ✅ VERIFIED

**Example:**
```bash
DELETE /api/v1/restore/e4805a6f-8ee7-4f3c-8309-2f12362c7398
```

**Response:**
```json
{
  "message": "backup unmounted successfully",
  "mount_id": "e4805a6f-8ee7-4f3c-8309-2f12362c7398"
}
```

**Cleanup Actions:**
- Unmounts filesystem: `umount /mnt/sendense/restore/{mount_id}`
- Disconnects qemu-nbd: `qemu-nbd --disconnect /dev/nbd0`
- Removes mount directory
- Deletes restore_mounts record
- Releases NBD device back to pool

---

#### 7. Get File Info (Bonus)
**Endpoint:** `GET /api/v1/restore/{mount_id}/file-info?path={path}`  
**Handler:** `restore_handlers.go:GetFileInfo()`  
**Status:** ✅ AVAILABLE

**Use Case:** Show detailed file metadata before download

---

## 🎨 GUI DESIGN SPECIFICATION

### **Page Structure:** `/app/restore/page.tsx`

**Layout:** Three-column responsive layout

```
┌─────────────────────────────────────────────────────────────┐
│  File-Level Restore                                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Active       │  │ Total        │  │ Available    │      │
│  │ Mounts: 1    │  │ Restores: 47 │  │ Slots: 7     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Step 1: Select VM and Backup                           │ │
│  │                                                         │ │
│  │ VM: [pgtest1           ▼]  Backup: [Oct 9, 12:57 ▼]   │ │
│  │                                                         │ │
│  │ Disk: [Disk 0 (102GB) ▼]   [ Mount Backup ]           │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Step 2: Browse Files                                   │ │
│  │                                                         │ │
│  │ 📁 / > Recovery > WindowsRE          [🔍 Search]       │ │
│  │                                                         │ │
│  │ ┌────────┬──────────────┬────────┬──────────────────┐ │ │
│  │ │ ☑️     │ Name         │ Size   │ Modified         │ │ │
│  │ ├────────┼──────────────┼────────┼──────────────────┤ │ │
│  │ │ 📁     │ ..           │ -      │ -                │ │ │
│  │ │ 📄     │ ReAgent.xml  │ 1.1 KB │ Sep 2, 2025      │ │ │
│  │ │ 📄     │ boot.sdi     │ 3.0 MB │ May 8, 2021      │ │ │
│  │ │ 📄     │ winre.wim    │ 482 MB │ Jan 29, 2024     │ │ │
│  │ └────────┴──────────────┴────────┴──────────────────┘ │ │
│  │                                                         │ │
│  │ 3 files selected (486 MB)                              │ │
│  │ [ Download Selected ]  [ Download Folder as ZIP ]      │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Active Mounts (1)                                      │ │
│  │                                                         │ │
│  │ pgtest1 | Disk 0 | NTFS | Expires in 52 min [Unmount] │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

---

### **Component Breakdown**

#### 1. `/app/restore/page.tsx`
**Purpose:** Main restore page container  
**Features:**
- Statistics cards (Active Mounts, Total Restores, Available Slots)
- Three-step workflow: Select → Mount → Browse → Download
- Active mounts panel at bottom
- Responsive layout (stacks on mobile)

**State Management:**
```typescript
const [selectedVM, setSelectedVM] = useState<string | null>(null);
const [selectedBackup, setSelectedBackup] = useState<string | null>(null);
const [selectedDiskIndex, setSelectedDiskIndex] = useState<number>(0);
const [activeMountId, setActiveMountId] = useState<string | null>(null);
const [currentPath, setCurrentPath] = useState<string>("/");
const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
```

---

#### 2. `BackupSelector` Component
**File:** `components/features/restore/BackupSelector.tsx`  
**Props:**
```typescript
interface BackupSelectorProps {
  onBackupSelected: (backupId: string, diskCount: number) => void;
}
```

**Features:**
- VM dropdown (fetch from `/api/v1/vm-contexts`)
- Backup history for selected VM (fetch from `/api/v1/backups?vm_name={vm}&status=completed`)
- Disk selector (if multi-disk VM)
  - Query `backup_disks` via backup_id to get disk count
  - Show dropdown: "Disk 0 (102GB)", "Disk 1 (5GB)"
- "Mount Backup" button
  - Disabled if no selection
  - Shows loading spinner during mount
  - Calls `POST /api/v1/restore/mount`

**API Calls:**
1. `GET /api/v1/vm-contexts` → List VMs
2. `GET /api/v1/backups?vm_name={vm}&status=completed` → List backups
3. `GET /api/v1/backups/{backup_id}` → Get disk count (backup_disks array)
4. `POST /api/v1/restore/mount` → Mount selected disk

**Error Handling:**
- 404: "Backup not found or disk invalid"
- 409: "Disk already mounted. Unmount existing mount first."
- 503: "Maximum concurrent mounts reached (limit: 8). Please unmount an existing backup first."

---

#### 3. `FileBrowser` Component
**File:** `components/features/restore/FileBrowser.tsx`  
**Props:**
```typescript
interface FileBrowserProps {
  mountId: string;
  onUnmount: () => void;
}
```

**Features:**
- **Breadcrumb navigation**
  - `/` > `Recovery` > `WindowsRE`
  - Clickable path segments
  - "Home" icon for root

- **File table**
  - Columns: Checkbox, Icon, Name, Size, Modified, Actions
  - Icons: 📁 for directories, 📄 for files
  - Sortable columns (name, size, date)
  - Row hover effects
  - Double-click folder to navigate

- **Selection**
  - Checkbox per file/folder
  - "Select All" checkbox in header
  - Multi-select with Shift+Click
  - Display selected count: "3 files selected (486 MB)"

- **Actions per row**
  - Files: "Download" button
  - Folders: "Browse" button

- **Bulk actions**
  - "Download Selected" button (if files selected)
  - "Download Folder as ZIP" button

**API Calls:**
1. `GET /api/v1/restore/{mount_id}/files?path={path}` → List files
2. `GET /api/v1/restore/{mount_id}/download?path={path}` → Download file
3. `GET /api/v1/restore/{mount_id}/download-directory?path={path}&format=zip` → Download folder

**Download Logic:**
```typescript
const handleDownloadFile = (path: string) => {
  const url = `/api/v1/restore/${mountId}/download?path=${encodeURIComponent(path)}`;
  window.open(url, '_blank'); // Opens in new tab, triggers browser download
};
```

**Search Feature:**
- Filter files by name (client-side)
- Case-insensitive
- Highlight matches

---

#### 4. `ActiveMountsPanel` Component
**File:** `components/features/restore/ActiveMountsPanel.tsx`  

**Features:**
- List all active mounts from `GET /api/v1/restore/mounts`
- Display per mount:
  - VM name (fetch from backup_id → backup_jobs)
  - Disk index
  - Filesystem type (NTFS, ext4, etc.)
  - Countdown timer: "Expires in 52 minutes"
  - "Unmount" button
- Refresh every 30 seconds
- Auto-remove unmounted entries

**Timer Logic:**
```typescript
const getTimeRemaining = (expiresAt: string) => {
  const now = new Date();
  const expiry = new Date(expiresAt);
  const diff = expiry.getTime() - now.getTime();
  
  if (diff <= 0) return "Expired";
  
  const minutes = Math.floor(diff / 60000);
  return `Expires in ${minutes} min`;
};
```

**Unmount Action:**
```typescript
const handleUnmount = async (mountId: string) => {
  await fetch(`/api/v1/restore/${mountId}`, { method: 'DELETE' });
  queryClient.invalidateQueries(['active-mounts']);
  toast.success('Backup unmounted successfully');
};
```

---

## 🔧 TYPESCRIPT INTERFACES

**File:** `src/features/restore/types/index.ts`

```typescript
// Restore Mount
export interface RestoreMount {
  mount_id: string;
  backup_id: string;
  backup_disk_id: number;
  disk_index: number;
  mount_path: string;
  nbd_device: string;
  filesystem_type: string;
  status: 'mounting' | 'mounted' | 'unmounting' | 'failed' | 'unmounted';
  created_at: string;
  expires_at: string;
  last_accessed_at: string;
}

// Mount Request
export interface MountBackupRequest {
  backup_id: string;
  disk_index: number;
}

// File Info
export interface FileInfo {
  name: string;
  path: string;
  type: 'file' | 'directory';
  size: number;
  mode: string;
  modified_time: string;
  is_symlink: boolean;
}

// File List Response
export interface FileListResponse {
  mount_id: string;
  path: string;
  files: FileInfo[];
  total_count: number;
}

// Active Mounts Response
export interface ActiveMountsResponse {
  mounts: RestoreMount[];
  count: number;
}
```

---

## 🌐 API CLIENT

**File:** `src/features/restore/api/restoreApi.ts`

```typescript
const API_BASE = ''; // Uses Next.js proxy

// 1. Mount backup disk
export const mountBackup = async (request: MountBackupRequest): Promise<RestoreMount> => {
  const response = await fetch(`${API_BASE}/api/v1/restore/mount`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
  
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to mount backup');
  }
  
  return response.json();
};

// 2. Browse files
export const listFiles = async (mountId: string, path: string): Promise<FileListResponse> => {
  const response = await fetch(
    `${API_BASE}/api/v1/restore/${mountId}/files?path=${encodeURIComponent(path)}`
  );
  
  if (!response.ok) {
    throw new Error('Failed to list files');
  }
  
  return response.json();
};

// 3. List active mounts
export const listActiveMounts = async (): Promise<ActiveMountsResponse> => {
  const response = await fetch(`${API_BASE}/api/v1/restore/mounts`);
  
  if (!response.ok) {
    throw new Error('Failed to list active mounts');
  }
  
  return response.json();
};

// 4. Unmount backup
export const unmountBackup = async (mountId: string): Promise<void> => {
  const response = await fetch(`${API_BASE}/api/v1/restore/${mountId}`, {
    method: 'DELETE',
  });
  
  if (!response.ok) {
    throw new Error('Failed to unmount backup');
  }
};

// 5. Get download URL (file)
export const getDownloadFileUrl = (mountId: string, path: string): string => {
  return `${API_BASE}/api/v1/restore/${mountId}/download?path=${encodeURIComponent(path)}`;
};

// 6. Get download URL (directory)
export const getDownloadDirectoryUrl = (mountId: string, path: string, format: 'zip' | 'tar.gz' = 'zip'): string => {
  return `${API_BASE}/api/v1/restore/${mountId}/download-directory?path=${encodeURIComponent(path)}&format=${format}`;
};
```

---

## 🎣 REACT QUERY HOOKS

**File:** `src/features/restore/hooks/useRestore.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as restoreApi from '../api/restoreApi';

// Fetch active mounts
export const useActiveMounts = () => {
  return useQuery({
    queryKey: ['active-mounts'],
    queryFn: restoreApi.listActiveMounts,
    refetchInterval: 30000, // Refresh every 30 seconds
  });
};

// Fetch files for a mount
export const useFiles = (mountId: string, path: string) => {
  return useQuery({
    queryKey: ['files', mountId, path],
    queryFn: () => restoreApi.listFiles(mountId, path),
    enabled: !!mountId, // Only fetch if mountId exists
  });
};

// Mount backup mutation
export const useMountBackup = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: restoreApi.mountBackup,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['active-mounts'] });
    },
  });
};

// Unmount backup mutation
export const useUnmountBackup = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: restoreApi.unmountBackup,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['active-mounts'] });
    },
  });
};
```

---

## 🎨 THEME CONSISTENCY

**CRITICAL:** Must support both light and dark modes.

**Use Semantic Tokens:**
```css
/* Backgrounds */
bg-background        /* Main background */
bg-card              /* Card/panel background */
bg-muted             /* Subtle background */

/* Text */
text-foreground      /* Primary text */
text-muted-foreground /* Secondary text */

/* Borders */
border-border        /* Default borders */

/* Interactive */
bg-primary           /* Primary button */
text-primary-foreground
hover:bg-primary/90
```

**DO NOT use hardcoded colors:**
- ❌ `bg-gray-900`, `text-white`, `border-gray-700`
- ✅ `bg-background`, `text-foreground`, `border-border`

---

## ✅ TESTING CHECKLIST

### Backend API Testing
- [ ] 1. Mount backup: `curl -X POST http://localhost:8082/api/v1/restore/mount -d '{"backup_id":"backup-pgtest1-1759947871","disk_index":0}'`
- [ ] 2. List files: `curl http://localhost:8082/api/v1/restore/{mount_id}/files?path=/`
- [ ] 3. Download file: `curl http://localhost:8082/api/v1/restore/{mount_id}/download?path=/file.txt -o file.txt`
- [ ] 4. List mounts: `curl http://localhost:8082/api/v1/restore/mounts`
- [ ] 5. Unmount: `curl -X DELETE http://localhost:8082/api/v1/restore/{mount_id}`

### GUI Testing
- [ ] 1. Navigate to `/restore` page
- [ ] 2. Select VM "pgtest1" from dropdown
- [ ] 3. Select backup from history
- [ ] 4. Mount Disk 0
- [ ] 5. Browse root directory `/`
- [ ] 6. Navigate into subdirectories
- [ ] 7. Download a small file
- [ ] 8. Select multiple files
- [ ] 9. Download folder as ZIP
- [ ] 10. Check active mounts panel shows countdown timer
- [ ] 11. Unmount backup
- [ ] 12. Verify mount removed from panel

### Edge Cases
- [ ] 1. Mount when 8 slots already in use (503 error - show friendly message)
- [ ] 2. Mount disk that's already mounted (409 error)
- [ ] 3. Navigate to invalid path (404 error)
- [ ] 4. Download file that no longer exists
- [ ] 5. Unmount while file browser is open
- [ ] 6. Theme switch (light/dark mode)

---

## 🚨 CRITICAL REQUIREMENTS

1. **Security:**
   - Never expose raw mount paths to users
   - All file paths URL-encoded
   - Path traversal protection (no ../)

2. **Performance:**
   - Lazy load file lists (don't fetch until mount active)
   - Debounce search input (300ms)
   - Show loading spinners for async operations

3. **UX:**
   - Clear error messages
   - Loading states for all async actions
   - Countdown timers for mount expiration
   - Toast notifications for success/failure
   - Breadcrumb navigation
   - File type icons

4. **Responsiveness:**
   - Mobile-friendly layout
   - Stack columns on small screens
   - Touch-friendly buttons

5. **Accessibility:**
   - Keyboard navigation
   - ARIA labels
   - Focus management
   - Screen reader support

---

## 📦 DELIVERABLES

### Files to Create:
1. `/app/restore/page.tsx` (main page)
2. `/src/features/restore/types/index.ts` (TypeScript interfaces)
3. `/src/features/restore/api/restoreApi.ts` (API client)
4. `/src/features/restore/hooks/useRestore.ts` (React Query hooks)
5. `/components/features/restore/BackupSelector.tsx`
6. `/components/features/restore/FileBrowser.tsx`
7. `/components/features/restore/ActiveMountsPanel.tsx`
8. `/components/features/restore/BreadcrumbNav.tsx`
9. `/components/features/restore/FileRow.tsx`

### Files to Modify:
1. `/app/layout.tsx` - Add "🔄 Restore" menu item to sidebar
2. `/src/features/restore/api/backupsApi.ts` - Import for VM/backup listing (if not already exists)

### Documentation:
1. Update `/job-sheets/2025-10-09-file-level-restore-gui.md` with implementation notes
2. Create `/job-sheets/FILE-LEVEL-RESTORE-TESTING.md` with test results

---

## 📊 SUCCESS CRITERIA

✅ **Phase 2 Complete When:**
1. User can navigate to `/restore` page
2. User can select VM and view backup history
3. User can mount a backup disk
4. User can browse filesystem with breadcrumbs
5. User can download individual files
6. User can download directories as ZIP archives
7. User can see active mounts with countdown timers
8. User can unmount backups
9. All errors handled gracefully with user-friendly messages
10. Light/dark mode support working
11. No console errors
12. All TypeScript types correct
13. Code passes linter
14. Responsive design works on mobile

---

## 🔗 REFERENCE DOCUMENTS

- **Backend API Spec:** `/home/oma_admin/sendense/HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md`
- **Database Schema:** Check `restore_mounts`, `backup_jobs`, `backup_disks` tables
- **Handler Code:** `/home/oma_admin/sendense/source/current/sha/api/handlers/restore_handlers.go`
- **Cursor Rules:** `/home/oma_admin/sendense/.cursorrules`

---

## 🎯 NEXT STEPS (After Phase 2)

- **Phase 3:** Full VM Restore (restore entire VM to different host)
- **Phase 4:** Schedule Restore Tests (automated restore verification)
- **Phase 5:** Restore from Replication Targets

---

**END OF JOB SHEET**

