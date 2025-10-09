# File-Level Restore - Quick Reference

**Date:** October 9, 2025  
**Phase:** Phase 2 - File-Level Restore GUI  
**Status:** Ready for Implementation  

---

## 📋 KEY DOCUMENTS

| Document | Purpose | Location |
|----------|---------|----------|
| **Job Sheet** | Comprehensive implementation guide | `/home/oma_admin/sendense/job-sheets/2025-10-09-file-level-restore-gui.md` |
| **Grok Prompt** | Concise prompt for Grok | `/home/oma_admin/sendense/job-sheets/GROK-PROMPT-file-level-restore.md` |
| **Backend API Spec** | Complete API documentation | `/home/oma_admin/sendense/HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md` |
| **Cursor Rules** | Code quality standards | `/home/oma_admin/sendense/.cursorrules` |

---

## ✅ BACKEND READINESS CHECK

**Status:** ✅ **100% READY - All APIs Operational**

### Database Tables
- ✅ `restore_mounts` (restore tracking)
- ✅ `backup_jobs` (backup metadata)
- ✅ `backup_disks` (per-disk backup records)
- ✅ `vm_backup_contexts` (VM backup contexts)

### API Endpoints
| Endpoint | Method | Status |
|----------|--------|--------|
| `/api/v1/restore/mount` | POST | ✅ Working |
| `/api/v1/restore/{mount_id}/files` | GET | ✅ Working |
| `/api/v1/restore/{mount_id}/download` | GET | ✅ Working |
| `/api/v1/restore/{mount_id}/download-directory` | GET | ✅ Working |
| `/api/v1/restore/mounts` | GET | ✅ Working |
| `/api/v1/restore/{mount_id}` | DELETE | ✅ Working |

### Test Data
- **VM:** `pgtest1` (2 disks: 102GB + 5GB, Windows Server 2022)
- **Backups:** Multiple completed backups available
- **Last Backup:** `backup-pgtest1-1760011077` (Oct 9, 12:57, incremental)

---

## 🎯 WHAT WE'RE BUILDING

### User Workflow

```
1. Navigate to /restore
2. Select VM: "pgtest1"
3. Select Backup: "Oct 9, 12:57 (Incremental)"
4. Select Disk: "Disk 0 (102GB)"
5. Click "Mount Backup"
   → Backend mounts QCOW2 via qemu-nbd
   → Mounts filesystem read-only
   → Returns mount_id
6. Browse files with breadcrumb navigation
7. Download individual files or folders as ZIP
8. View active mounts with countdown timers
9. Unmount when done
```

### Key Features
- ✅ VM & backup selection with dropdowns
- ✅ Multi-disk support (select which disk to mount)
- ✅ File browser with breadcrumb navigation
- ✅ File download (individual files)
- ✅ Directory download (as ZIP archive)
- ✅ Active mounts panel with countdown timers
- ✅ Auto-unmount after 1 hour idle
- ✅ Light/dark mode support

---

## 🚨 CRITICAL GOTCHAS

### 1. Theme Support
**MUST** use semantic tokens:
- ✅ `bg-background`, `text-foreground`, `border-border`
- ❌ `bg-gray-900`, `text-white`, `border-gray-700`

### 2. API Proxy
**MUST** use Next.js proxy:
- ✅ `const API_BASE = '';` (empty string)
- ❌ `const API_BASE = 'http://localhost:8082';`

### 3. Download Behavior
Use `window.open()` for downloads:
```typescript
const url = getDownloadFileUrl(mountId, path);
window.open(url, '_blank'); // Browser handles download
```

### 4. Mount Limits
- **Max 8 concurrent mounts** (system resource limit)
- **1 mount per disk** (can't mount same disk twice)
- **1-hour auto-expiration** (show countdown timer)

---

## 📦 DELIVERABLES

### New Files (9 total)
```
/app/restore/page.tsx                                  # Main page
/src/features/restore/types/index.ts                   # TypeScript types
/src/features/restore/api/restoreApi.ts                # API client
/src/features/restore/hooks/useRestore.ts              # React Query hooks
/components/features/restore/BackupSelector.tsx        # VM/backup selector
/components/features/restore/FileBrowser.tsx           # File browser table
/components/features/restore/ActiveMountsPanel.tsx     # Active mounts panel
/components/features/restore/BreadcrumbNav.tsx         # Breadcrumb navigation
/components/features/restore/FileRow.tsx               # Individual file row
```

### Modified Files (1)
```
/app/layout.tsx                                        # Add "🔄 Restore" menu item
```

---

## 🧪 TESTING COMMANDS

### Quick Backend Test
```bash
# 1. Get available backups
curl http://localhost:8082/api/v1/backups?vm_name=pgtest1&status=completed

# 2. Mount backup
curl -X POST http://localhost:8082/api/v1/restore/mount \
  -H "Content-Type: application/json" \
  -d '{"backup_id":"backup-pgtest1-1760011077","disk_index":0}'

# 3. List files (replace MOUNT_ID)
curl "http://localhost:8082/api/v1/restore/MOUNT_ID/files?path=/"

# 4. List active mounts
curl http://localhost:8082/api/v1/restore/mounts

# 5. Unmount (replace MOUNT_ID)
curl -X DELETE "http://localhost:8082/api/v1/restore/MOUNT_ID"
```

### GUI Test Checklist
1. ✅ Navigate to `/restore`
2. ✅ Select VM "pgtest1"
3. ✅ Select latest backup
4. ✅ Mount Disk 0
5. ✅ Browse root directory
6. ✅ Navigate into subdirectories
7. ✅ Navigate back using breadcrumbs
8. ✅ Download a file
9. ✅ Download folder as ZIP
10. ✅ Check active mounts panel
11. ✅ Verify countdown timer
12. ✅ Unmount backup
13. ✅ Test light/dark mode

---

## 🔗 ARCHITECTURE FLOW

```
┌─────────────────────────────────────────────────────────────┐
│                          GUI                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ BackupSelector│→│  FileBrowser  │→│ ActiveMounts │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                   │                   │            │
│         ↓                   ↓                   ↓            │
│  ┌──────────────────────────────────────────────────────┐   │
│  │          restoreApi.ts (API Client)                  │   │
│  └──────────────────────────────────────────────────────┘   │
│                          │                                   │
└──────────────────────────┼───────────────────────────────────┘
                           │
                           ↓ (Next.js Proxy)
┌──────────────────────────┼───────────────────────────────────┐
│                          │        Backend                     │
│  ┌─────────────────────────────────────────────┐             │
│  │    restore_handlers.go (API Layer)          │             │
│  └─────────────────────────────────────────────┘             │
│                          │                                    │
│                          ↓                                    │
│  ┌─────────────────────────────────────────────┐             │
│  │    mount_manager.go (Business Logic)        │             │
│  └─────────────────────────────────────────────┘             │
│                          │                                    │
│         ┌────────────────┼────────────────┐                  │
│         ↓                ↓                ↓                  │
│  ┌──────────┐  ┌──────────────┐  ┌─────────────┐           │
│  │ qemu-nbd │  │ Linux Mount  │  │  Database   │           │
│  │(NBD expo)│  │ (Filesystem) │  │ (tracking)  │           │
│  └──────────┘  └──────────────┘  └─────────────┘           │
│       │               │                   │                  │
│       ↓               ↓                   ↓                  │
│  QCOW2 File    /mnt/sendense/restore    restore_mounts      │
└─────────────────────────────────────────────────────────────┘
```

---

## 📊 API REQUEST/RESPONSE EXAMPLES

### Mount Backup
```bash
POST /api/v1/restore/mount
{
  "backup_id": "backup-pgtest1-1760011077",
  "disk_index": 0
}

Response:
{
  "mount_id": "abc123",
  "backup_id": "backup-pgtest1-1760011077",
  "disk_index": 0,
  "filesystem_type": "ntfs",
  "status": "mounted",
  "expires_at": "2025-10-09T14:00:00Z"
}
```

### List Files
```bash
GET /api/v1/restore/abc123/files?path=/Recovery

Response:
{
  "mount_id": "abc123",
  "path": "/Recovery",
  "files": [
    {
      "name": "WindowsRE",
      "path": "/Recovery/WindowsRE",
      "type": "directory",
      "size": 0,
      "modified_time": "2025-09-02T06:21:20Z"
    },
    {
      "name": "file.txt",
      "path": "/Recovery/file.txt",
      "type": "file",
      "size": 1024,
      "modified_time": "2025-09-02T06:22:00Z"
    }
  ],
  "total_count": 2
}
```

---

## 🎓 LEARNING FROM PHASE 1

### What Went Right ✅
- Protection Flows backend: Created with proper database design
- React Query integration: Clean data fetching pattern
- Component structure: Well-organized feature modules

### What Went Wrong ❌
- **Theme Support:** Hardcoded colors broke light mode
- **API Proxy:** Some confusion about using Next.js proxy

### Apply to Phase 2 ✅
1. **Always use semantic tokens** (`bg-background`, not `bg-gray-900`)
2. **Always use Next.js proxy** (`API_BASE = ''`)
3. **Test light + dark mode** before claiming done
4. **Test on mobile** (responsive design)
5. **Handle all error cases** (404, 409, 503)

---

## 🚀 HANDOFF TO GROK

**Copy this to Grok:**

```
Hi Grok! I need you to implement the File-Level Restore GUI for Sendense.

📋 Job Sheet: /home/oma_admin/sendense/job-sheets/2025-10-09-file-level-restore-gui.md
🎯 Grok Prompt: /home/oma_admin/sendense/job-sheets/GROK-PROMPT-file-level-restore.md

The backend is 100% ready - all APIs tested and working. Your job is to build the GUI.

Key requirements:
- Use semantic tokens for light/dark mode support
- Use Next.js proxy (API_BASE = '')
- Production-quality code (no placeholders)
- Test with pgtest1 VM

Please read the job sheet first, then start with the foundation (types, API client, hooks).

Let me know when you're ready to start! 🚀
```

---

## 🎯 SUCCESS METRICS

**Phase 2 is complete when:**

1. ✅ `/restore` page exists and loads
2. ✅ User can select VM and view backup history
3. ✅ User can mount a backup disk
4. ✅ User can browse files with breadcrumb navigation
5. ✅ User can download individual files
6. ✅ User can download directories as ZIP
7. ✅ Active mounts panel shows mounts with countdown timers
8. ✅ User can unmount backups
9. ✅ Light mode works (semantic tokens used)
10. ✅ Dark mode works
11. ✅ Mobile responsive
12. ✅ No console errors
13. ✅ No TypeScript errors
14. ✅ Linter clean

---

**Last Updated:** October 9, 2025  
**Next Phase:** Full VM Restore (restore entire VM to different host)

