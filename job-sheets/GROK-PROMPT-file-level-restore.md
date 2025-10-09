# GROK PROMPT: File-Level Restore GUI Implementation

**Session:** Phase 2 - File-Level Restore  
**Date:** October 9, 2025  
**Context:** Backend fully operational, GUI needs implementation  

---

## 📋 YOUR MISSION

Implement a production-ready **File-Level Restore** interface in the Sendense GUI that allows users to browse and download files from VM backups.

**Primary Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-09-file-level-restore-gui.md`

---

## 🎯 WHAT YOU'RE BUILDING

A three-step workflow for file-level recovery:

1. **Select & Mount:** User selects VM → Backup → Disk → Click "Mount"
2. **Browse Files:** User navigates filesystem with breadcrumbs, sees file tree
3. **Download:** User downloads individual files or folders as ZIP

**Plus:** Active Mounts panel showing all mounted backups with countdown timers

---

## ✅ BACKEND STATUS: FULLY READY

All these APIs are **tested and working**:

| Endpoint | Purpose | Status |
|----------|---------|--------|
| `POST /api/v1/restore/mount` | Mount backup disk | ✅ |
| `GET /api/v1/restore/{mount_id}/files?path={path}` | Browse files | ✅ |
| `GET /api/v1/restore/{mount_id}/download?path={path}` | Download file | ✅ |
| `GET /api/v1/restore/{mount_id}/download-directory?path={path}&format=zip` | Download folder | ✅ |
| `GET /api/v1/restore/mounts` | List active mounts | ✅ |
| `DELETE /api/v1/restore/{mount_id}` | Unmount backup | ✅ |

**Database:**
- `restore_mounts` table ✅ EXISTS
- `backup_jobs`, `backup_disks`, `vm_backup_contexts` ✅ ALL EXIST

**Test Data:**
- VM: `pgtest1` (2 disks: 102GB + 5GB)
- Backups: Multiple completed backups available
- You can test mount/unmount in your implementation

---

## 📐 DETAILED SPECIFICATION

**Read the full job sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-09-file-level-restore-gui.md`

**Key sections:**
1. **Section: Backend API Verification** - Exact request/response formats
2. **Section: GUI Design Specification** - Wireframe and component breakdown
3. **Section: Component Breakdown** - Detailed component specs
4. **Section: TypeScript Interfaces** - All types you need
5. **Section: API Client** - Exact API call code
6. **Section: React Query Hooks** - Data fetching patterns
7. **Section: Theme Consistency** - CRITICAL color token usage

---

## 🚨 CRITICAL RULES

### 1. **Theme Support (MANDATORY)**

You **MUST** use semantic color tokens that adapt to light/dark mode:

✅ **CORRECT:**
```tsx
<div className="bg-background text-foreground border-border">
  <Button className="bg-primary text-primary-foreground hover:bg-primary/90">
    Mount Backup
  </Button>
</div>
```

❌ **WRONG:**
```tsx
<div className="bg-gray-900 text-white border-gray-700">
  <Button className="bg-blue-600 text-white hover:bg-blue-700">
    Mount Backup
  </Button>
</div>
```

**Why this matters:** The Protection Flows page lost light mode support because hardcoded colors were used. Don't repeat this mistake.

**Reference:** See `/app/protection-flows/page.tsx` for correct semantic token usage.

---

### 2. **Use Next.js API Proxy**

All API calls go through Next.js proxy (configured in `next.config.ts`):

✅ **CORRECT:**
```typescript
const API_BASE = ''; // Empty string = uses Next.js proxy

await fetch(`${API_BASE}/api/v1/restore/mount`, { ... });
```

❌ **WRONG:**
```typescript
const API_BASE = 'http://localhost:8082'; // Direct backend call
```

**Why:** Proxy handles CORS, authentication, and works in production.

---

### 3. **Code Quality Standards**

- ✅ **No placeholder code** - All functions must be fully implemented
- ✅ **No commented-out code** - Clean, production-ready only
- ✅ **No unused imports** - Linter-clean code
- ✅ **TypeScript strict mode** - All types correct
- ✅ **Component size <200 lines** - Extract sub-components if needed
- ✅ **Error handling** - Try/catch with user-friendly messages
- ✅ **Loading states** - Spinners for async operations

---

## 📦 DELIVERABLES

### **New Files to Create:**

1. **Main Page:**
   - `/app/restore/page.tsx`

2. **Feature Module:**
   - `/src/features/restore/types/index.ts`
   - `/src/features/restore/api/restoreApi.ts`
   - `/src/features/restore/hooks/useRestore.ts`

3. **Components:**
   - `/components/features/restore/BackupSelector.tsx`
   - `/components/features/restore/FileBrowser.tsx`
   - `/components/features/restore/ActiveMountsPanel.tsx`
   - `/components/features/restore/BreadcrumbNav.tsx`
   - `/components/features/restore/FileRow.tsx`

### **Files to Modify:**

1. **Navigation:**
   - `/app/layout.tsx` - Add "🔄 Restore" menu item after "Protection Flows"

2. **Existing APIs (if needed):**
   - Check if `/src/features/backups/api/backupsApi.ts` exists
   - If not, you may need to create basic VM/backup listing functions

---

## 🎨 UI/UX REQUIREMENTS

### **Page Layout:**

```
┌─────────────────────────────────────────────────────────┐
│ File-Level Restore                                       │
├─────────────────────────────────────────────────────────┤
│ [Active Mounts: 1] [Total Restores: 47] [Free Slots: 7] │
├─────────────────────────────────────────────────────────┤
│ Step 1: Select VM and Backup                            │
│ VM: [pgtest1 ▼]  Backup: [Oct 9, 12:57 ▼]             │
│ Disk: [Disk 0 (102GB) ▼]  [ Mount Backup ]            │
├─────────────────────────────────────────────────────────┤
│ Step 2: Browse Files                                    │
│ 📁 / > Recovery > WindowsRE       [🔍 Search]          │
│                                                         │
│ ┌─────┬──────────────┬────────┬────────────────────┐  │
│ │ ☑️  │ Name         │ Size   │ Modified           │  │
│ ├─────┼──────────────┼────────┼────────────────────┤  │
│ │ 📁  │ ..           │ -      │ -                  │  │
│ │ 📄  │ ReAgent.xml  │ 1.1 KB │ Sep 2, 2025        │  │
│ │ 📄  │ winre.wim    │ 482 MB │ Jan 29, 2024       │  │
│ └─────┴──────────────┴────────┴────────────────────┘  │
│                                                         │
│ [ Download Selected ]  [ Download Folder as ZIP ]      │
├─────────────────────────────────────────────────────────┤
│ Active Mounts (1)                                       │
│ pgtest1 | Disk 0 | NTFS | Expires in 52 min [Unmount] │
└─────────────────────────────────────────────────────────┘
```

### **Key Features:**

1. **VM & Backup Selection:**
   - Dropdown with all VMs from `/api/v1/vm-contexts`
   - Backup dropdown shows history for selected VM
   - Disk selector (only show if VM has multiple disks)

2. **File Browser:**
   - Breadcrumb navigation (/ > Folder > Subfolder)
   - File table with checkboxes for multi-select
   - Icons: 📁 for folders, 📄 for files
   - Double-click folder to navigate
   - Download buttons (single file or selected files)
   - Download folder as ZIP button

3. **Active Mounts Panel:**
   - Shows all mounted backups
   - Countdown timer: "Expires in 52 minutes"
   - Unmount button per mount
   - Auto-refresh every 30 seconds

4. **Download Behavior:**
   ```typescript
   // For file download:
   const url = getDownloadFileUrl(mountId, filePath);
   window.open(url, '_blank'); // Browser handles download
   
   // For folder download:
   const url = getDownloadDirectoryUrl(mountId, folderPath, 'zip');
   window.open(url, '_blank');
   ```

---

## 🧪 TESTING REQUIREMENTS

**You MUST test these scenarios:**

1. ✅ Mount a backup (disk 0 of pgtest1)
2. ✅ Browse root directory
3. ✅ Navigate into subdirectories
4. ✅ Navigate back using breadcrumbs
5. ✅ Download a single file
6. ✅ Select multiple files
7. ✅ Download folder as ZIP
8. ✅ Check active mounts panel shows mount
9. ✅ Verify countdown timer updates
10. ✅ Unmount backup
11. ✅ Verify mount removed from panel
12. ✅ Test light mode + dark mode

**Error scenarios:**
1. ✅ Try to mount disk that's already mounted (409 error)
2. ✅ Navigate to invalid path (404 error)
3. ✅ Download file that doesn't exist

---

## 📚 KEY REFERENCES

### **Must Read:**
1. **Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-09-file-level-restore-gui.md`
   - Complete API specs
   - Component breakdown
   - TypeScript interfaces
   - React Query hooks

2. **Backend API Docs:** `/home/oma_admin/sendense/HANDOVER-GUI-BACKUP-RESTORE-INTEGRATION.md`
   - Section: "🔄 FILE-LEVEL RESTORE API ENDPOINTS" (lines 441-728)
   - Request/response examples
   - Error handling

3. **Cursor Rules:** `/home/oma_admin/sendense/.cursorrules`
   - Code quality standards
   - Prohibited practices
   - Completion checklist

### **Reference Implementations:**
- **Protection Flows Page:** `/app/protection-flows/page.tsx`
  - Correct semantic token usage
  - Next.js proxy pattern
  - React Query integration

- **API Client Pattern:** `/src/features/protection-flows/api/protectionFlowsApi.ts`
  - How to structure API calls
  - Error handling
  - Type definitions

---

## 🚀 IMPLEMENTATION SEQUENCE

**Step 1: Foundation (30 min)**
1. Create types: `/src/features/restore/types/index.ts`
2. Create API client: `/src/features/restore/api/restoreApi.ts`
3. Create hooks: `/src/features/restore/hooks/useRestore.ts`
4. Test API calls with mock data

**Step 2: Components (1 hour)**
1. Create `BackupSelector` component
2. Create `FileBrowser` component
3. Create `ActiveMountsPanel` component
4. Create `BreadcrumbNav` and `FileRow` sub-components

**Step 3: Integration (30 min)**
1. Create main page: `/app/restore/page.tsx`
2. Add navigation menu item in `layout.tsx`
3. Wire up components with state management

**Step 4: Testing (30 min)**
1. Test mount/unmount flow
2. Test file browsing and downloads
3. Test error scenarios
4. Verify light/dark mode
5. Check mobile responsiveness

**Step 5: Polish (15 min)**
1. Fix linter errors
2. Remove console.logs
3. Add loading states
4. Improve error messages
5. Final commit

---

## ✅ COMPLETION CHECKLIST

Before you claim "done", verify:

- [ ] All files created (9 new files)
- [ ] Navigation menu updated
- [ ] Can mount a backup
- [ ] Can browse files with breadcrumbs
- [ ] Can download individual files
- [ ] Can download folders as ZIP
- [ ] Active mounts panel works
- [ ] Countdown timers update
- [ ] Can unmount backups
- [ ] Light mode works (no hardcoded colors)
- [ ] Dark mode works
- [ ] No console errors
- [ ] No TypeScript errors
- [ ] No linter warnings
- [ ] Mobile responsive
- [ ] Loading states present
- [ ] Error handling works
- [ ] Toast notifications work

---

## 💬 COMMUNICATION

**When you're done:**
1. Summarize what you built
2. List any issues encountered
3. Provide testing instructions
4. Highlight any deviations from spec (if any)

**If you get stuck:**
1. Check the job sheet for detailed specs
2. Check the backend API docs for request/response formats
3. Check existing Protection Flows code for patterns
4. Ask specific questions about what's blocking you

---

## 🎯 SUCCESS CRITERIA

**You succeed when:**
- User can click "🔄 Restore" in sidebar
- User can mount a backup and browse its files
- User can download files from mounted backups
- User can see active mounts with countdown timers
- Everything works in both light and dark mode
- Code is production-ready (no placeholders, no commented code)
- All tests pass

---

**LET'S BUILD THIS! 🚀**

**START HERE:** Read `/home/oma_admin/sendense/job-sheets/2025-10-09-file-level-restore-gui.md` for full details, then begin with the foundation (types, API client, hooks).

