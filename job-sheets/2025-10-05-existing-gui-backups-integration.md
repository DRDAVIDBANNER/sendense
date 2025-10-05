# Job Sheet: Existing GUI Backups Section Integration

**Date Created:** 2025-10-05  
**Status:** 🔴 **READY TO START**  
**Project Goal Link:** [project-goals/phases/phase-1-vmware-backup.md → GUI Integration for Backup Operations]  
**Duration:** 5-7 days  
**Priority:** High (Customer-facing backup management interface)  
**Completed:** TBD

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`  
**Task Context:** **GUI Integration for completed backup infrastructure**  
**Business Value:** Customer-facing backup management interface enabling self-service operations  
**Success Criteria:** Professional backup management integrated into existing migration-dashboard

**Strategic Decision:**
Instead of building new cockpit GUI from scratch (3+ weeks), integrate backup functionality into existing professional GUI (5-7 days) to deliver customer value quickly while preserving design time for future innovations.

---

## 🔗 DEPENDENCY STATUS

### **Required Before Starting:**
- ✅ **Task 5:** Backup API Endpoints (POST /backup/start, GET /backup/list, etc.) - OPERATIONAL
- ✅ **Task 4:** File-Level Restore API (POST /restore/mount, GET /files, etc.) - OPERATIONAL  
- ✅ **Existing GUI:** migration-dashboard (Next.js 15 + Flowbite) - OPERATIONAL
- ✅ **API Infrastructure:** APIClient class with error handling and React Query

### **Foundation Analysis:**
- ✅ **Tech Stack:** Next.js 15 + React 19 + Flowbite + Tailwind (excellent)
- ✅ **Layout System:** Three-panel layout (LeftNavigation → MainContent → RightContext)  
- ✅ **Navigation:** 12 existing sections with clean routing
- ✅ **Components:** Professional VM tables, job lists, modal patterns
- ✅ **Real-Time:** Socket.io + React Query for live updates

### **Enables These Features:**
- 🎯 **Customer Backup Management:** Self-service backup operations via GUI
- 🎯 **File Recovery Interface:** Browse and download files from backups
- 🎯 **Backup Monitoring:** Real-time backup job progress
- 🎯 **Complete Workflow:** VM selection → Backup → Monitor → Restore files

---

## 📋 JOB BREAKDOWN (Detailed Implementation)

### **Phase 1: Navigation & Skeleton (Day 1)**

- [ ] **Add Backups Navigation Item**
  - **File:** `src/components/layout/LeftNavigation.tsx`
  - **Position:** Between "Replication Jobs" and "Failover" (line 65+)
  - **Icon:** HiDatabase or HiArchive (consistent with HeroIcons pattern)
  - **Evidence:** Backups section accessible via sidebar navigation

- [ ] **Create App Route Structure**
  - **Directory:** `src/app/backups/`
  - **Files:** `page.tsx` (main), `[backup-id]/page.tsx` (details)
  - **Pattern:** Follow existing `src/app/virtual-machines/` structure
  - **Evidence:** `/backups` URL route working

- [ ] **Add MainContentArea Route Handler**
  - **File:** `src/components/layout/MainContentArea.tsx`
  - **Addition:** `case 'backups': return <BackupsManagement onVMSelect={onVMSelect} />;`
  - **Integration:** Follow existing switch statement pattern (line 34+)
  - **Evidence:** Backups section renders in main content area

- [ ] **Create Skeleton BackupsManagement Component**
  - **File:** `src/components/backups/BackupsManagement.tsx`
  - **Structure:** Follow `components/jobs/JobListView.tsx` pattern
  - **Placeholder:** Professional "Coming Soon" with backup context
  - **Evidence:** Basic backups section renders without errors

### **Phase 2: Backup Jobs List (Days 2-3)**

- [ ] **Create BackupJobsList Component**
  - **File:** `src/components/backups/BackupJobsList.tsx`
  - **Pattern:** Follow `components/jobs/UnifiedJobList.tsx` structure
  - **Integration:** React Query with Task 5 `GET /api/v1/backup/list`
  - **Evidence:** Backup jobs displayed in table format

- [ ] **Create BackupJobRow Component**
  - **File:** `src/components/backups/BackupJobRow.tsx`
  - **Columns:** VM Name, Type, Repository, Status, Progress, Started, Actions
  - **Actions:** View Details, Browse Files, Delete (if allowed)
  - **Evidence:** Individual backup jobs with action buttons

- [ ] **Extend APIClient with Backup Methods**
  - **File:** `src/lib/api.ts`
  - **Methods:** `listBackups()`, `getBackupDetails()`, `deleteBackup()`
  - **Integration:** Use Task 5 endpoints with existing error handling patterns
  - **Evidence:** API calls working from GUI components

- [ ] **Add Backup TypeScript Interfaces**
  - **File:** `src/lib/types.ts` (extend existing)
  - **Interfaces:** `BackupJob`, `BackupListResponse`, `BackupChain`
  - **Pattern:** Follow existing `VM`, `Migration` interface structure
  - **Evidence:** All backup data properly typed

### **Phase 3: Start Backup Functionality (Days 3-4)**

- [ ] **Create StartBackupModal Component**
  - **File:** `src/components/backups/StartBackupModal.tsx`
  - **Pattern:** Follow existing modal patterns from VM/failover components
  - **Fields:** VM Selection, Backup Type (full/incremental), Repository
  - **Evidence:** Can start backup operations via GUI modal

- [ ] **Create BackupControlPanel Component**
  - **File:** `src/components/backups/BackupControlPanel.tsx`
  - **Features:** Start Backup, View Repositories, Backup Analytics
  - **Layout:** Card header with action buttons (follow VM table patterns)
  - **Evidence:** Backup control buttons functional

- [ ] **Integrate startBackup API Method**
  - **File:** `src/lib/api.ts` (extend APIClient)
  - **Method:** `startBackup(vm_name, backup_type, repository_id)`
  - **Integration:** Task 5 `POST /api/v1/backup/start` endpoint
  - **Evidence:** Backup jobs triggered successfully from GUI

- [ ] **Add Real-Time Backup Progress**
  - **Integration:** Extend existing WebSocket listeners
  - **Updates:** Backup progress, status changes, completion notifications
  - **Pattern:** Follow existing replication job progress patterns
  - **Evidence:** Backup progress visible without refresh

### **Phase 4: File Restore Integration (Days 4-5)**

- [ ] **Create FileBrowserModal Component**
  - **File:** `src/components/restore/FileBrowserModal.tsx`
  - **Features:** Mount backup, browse directories, download files
  - **Integration:** Task 4 restore APIs (mount, files, download)
  - **Evidence:** Can browse backup file contents via GUI

- [ ] **Create FileTreeView Component**
  - **File:** `src/components/restore/FileTreeView.tsx`
  - **Features:** Directory tree navigation, file selection
  - **UI:** Follow existing tree/table patterns in GUI
  - **Evidence:** Intuitive file browsing interface

- [ ] **Add Restore API Methods**
  - **File:** `src/lib/api.ts` (extend APIClient)
  - **Methods:** `mountBackup()`, `browseFiles()`, `downloadFile()`, `unmountBackup()`
  - **Integration:** Task 4 restore endpoints
  - **Evidence:** File restore operations working from GUI

- [ ] **Create RestoreMountsList Component**
  - **File:** `src/components/restore/RestoreMountsList.tsx`
  - **Features:** Show active restore mounts, unmount controls
  - **Pattern:** Follow existing job/status list patterns
  - **Evidence:** Active restore sessions manageable via GUI

### **Phase 5: VM Integration & Enhancement (Day 5)**

- [ ] **Enhance VM Table with Backup Actions**
  - **File:** `src/components/vm/ModernVMTable.tsx`
  - **Addition:** "Backup Now" and "Browse Files" buttons per VM row
  - **Integration:** Connect to backup APIs from VM context
  - **Evidence:** Can start backups directly from VM table

- [ ] **Add Backup Status to VM Context Panel**
  - **File:** `src/components/layout/RightContextPanel.tsx`
  - **Addition:** Last backup time, backup count, storage used
  - **Integration:** Query backup status when VM selected
  - **Evidence:** VM backup information visible in context panel

- [ ] **Enhance Dashboard with Backup Stats**
  - **File:** `src/components/DashboardOverview.tsx`
  - **Addition:** Backup statistics cards (today's backups, storage used)
  - **Integration:** Aggregate backup data from APIs
  - **Evidence:** Backup metrics visible on main dashboard

---

## 🎨 DESIGN INTEGRATION STRATEGY

### **Following Existing Patterns**

**UI Components (Consistent with Current GUI):**
- **Cards:** Flowbite `<Card>` components for sections
- **Tables:** Existing table patterns with action buttons
- **Modals:** Follow existing modal sizing and styling
- **Icons:** HeroIcons (consistent with current icon set)
- **Colors:** Dark theme with existing color palette

**Component Architecture:**
```typescript
// Follow existing component patterns
export const BackupJobsList = ({ onVMSelect }: BackupJobsListProps) => {
  const { data: backups, isLoading } = useQuery(['backups'], api.backup.list);
  
  // Use existing loading/error patterns
  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorMessage />;
  
  // Use existing Flowbite components
  return (
    <Card>
      <Table>
        {/* Follow existing table structure */}
      </Table>
    </Card>
  );
};
```

---

## 🔌 API INTEGRATION POINTS

### **Task 5 Backup APIs (Ready)**
- ✅ `POST /api/v1/backup/start` - Start backup operations
- ✅ `GET /api/v1/backup/list` - List all backups with filtering
- ✅ `GET /api/v1/backup/{id}` - Get backup details
- ✅ `DELETE /api/v1/backup/{id}` - Delete backup
- ✅ `GET /api/v1/backup/chain` - Get backup chain

### **Task 4 Restore APIs (Ready)**
- ✅ `POST /api/v1/restore/mount` - Mount backup for file browsing
- ✅ `GET /api/v1/restore/{mount_id}/files` - Browse files in backup
- ✅ `GET /api/v1/restore/{mount_id}/download` - Download files
- ✅ `DELETE /api/v1/restore/{mount_id}` - Unmount backup

### **Existing Infrastructure (Reusable)**
- ✅ VM Context APIs - VM selection and management
- ✅ Repository APIs - Storage backend information
- ✅ WebSocket integration - Real-time updates
- ✅ Authentication - Bearer token system

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **Navigation Integration:** Backups section accessible via sidebar
- [ ] **Backup Jobs Display:** List all backup jobs with real-time status
- [ ] **Start Backup Workflow:** Can initiate backups from GUI (full and incremental)
- [ ] **File Restore Workflow:** Can browse and download files from backups
- [ ] **VM Integration:** Backup operations accessible from VM management
- [ ] **Real-Time Updates:** Backup progress visible without manual refresh
- [ ] **Design Consistency:** Matches existing GUI aesthetic and patterns

### **Testing Evidence Required**
- [ ] Navigate to backups section via sidebar
- [ ] View existing backup jobs in table format
- [ ] Start new backup operation via GUI modal
- [ ] Monitor backup progress in real-time
- [ ] Browse files in completed backup
- [ ] Download individual files successfully
- [ ] Start backup from VM table row
- [ ] View backup status in VM context panel

---

## 🚨 PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All GUI code in existing `deployment/sha-appliance/gui/` structure
- ✅ **API Integration:** Use Task 5 and Task 4 endpoints without modification
- ✅ **Design Consistency:** Follow existing Flowbite + Tailwind patterns
- ✅ **No Simulations:** Real backup operations via completed backend APIs
- ✅ **Error Handling:** Use existing error handling and notification patterns
- ✅ **TypeScript:** All new code strictly typed following existing interfaces
- ✅ **Component Patterns:** Follow existing component structure and naming

### **Integration Constraints:**
- **Existing Layout:** Work within three-panel layout system
- **Navigation Pattern:** Follow existing sidebar navigation structure
- **API Client:** Extend existing APIClient class (don't replace)
- **Component Style:** Use Flowbite React components for consistency
- **Real-Time:** Integrate with existing WebSocket infrastructure

---

## 📂 DELIVERABLES

### **New Navigation Integration**
- Enhanced `src/components/layout/LeftNavigation.tsx` - Add backups menu item
- Enhanced `src/components/layout/MainContentArea.tsx` - Add backups route handler

### **Backup Management Components**
- `src/components/backups/BackupsManagement.tsx` - Main backup management view
- `src/components/backups/BackupControlPanel.tsx` - Backup action controls
- `src/components/backups/BackupJobsList.tsx` - Backup jobs table
- `src/components/backups/BackupJobRow.tsx` - Individual backup job display
- `src/components/backups/StartBackupModal.tsx` - Start backup dialog

### **File Restore Components**
- `src/components/restore/FileBrowserModal.tsx` - Browse backup file contents
- `src/components/restore/FileTreeView.tsx` - Directory tree navigation
- `src/components/restore/FileDownloadManager.tsx` - Download files interface
- `src/components/restore/RestoreMountsList.tsx` - Active restore sessions

### **App Routes**
- `src/app/backups/page.tsx` - Main backups page
- `src/app/backups/[backup-id]/page.tsx` - Individual backup details

### **API Extensions**
- Enhanced `src/lib/api.ts` - Backup and restore methods
- Enhanced `src/lib/types.ts` - Backup-related TypeScript interfaces

### **VM Enhancement**
- Enhanced `src/components/vm/ModernVMTable.tsx` - Add backup action buttons
- Enhanced `src/components/layout/RightContextPanel.tsx` - Add backup status info

---

## 🏗️ TECHNICAL ARCHITECTURE

### **Component Integration Pattern**

```typescript
// Main backup management component (follows JobListView pattern)
export interface BackupsManagementProps {
  onVMSelect: (vmName: string) => void;
}

export const BackupsManagement = ({ onVMSelect }: BackupsManagementProps) => {
  const [selectedBackup, setSelectedBackup] = useState<string | null>(null);
  const [showStartModal, setShowStartModal] = useState(false);
  const [showFileBrowser, setShowFileBrowser] = useState<string | null>(null);

  return (
    <div className="h-full flex flex-col">
      <BackupControlPanel 
        onStartBackup={() => setShowStartModal(true)}
        onBrowseFiles={() => {/* Handle file browsing */}}
      />
      
      <BackupJobsList 
        onVMSelect={onVMSelect}
        onBackupSelect={setSelectedBackup}
        onBrowseFiles={setShowFileBrowser}
      />
      
      {showStartModal && (
        <StartBackupModal 
          isOpen={showStartModal}
          onClose={() => setShowStartModal(false)}
          onBackupStarted={(backup) => {/* Handle success */}}
        />
      )}
      
      {showFileBrowser && (
        <FileBrowserModal
          backupId={showFileBrowser}
          isOpen={!!showFileBrowser}
          onClose={() => setShowFileBrowser(null)}
        />
      )}
    </div>
  );
};
```

### **API Integration (Extending Existing APIClient)**

```typescript
// Extend existing APIClient class in src/lib/api.ts
class APIClient {
  // ... existing methods ...

  // Backup Operations (Task 5 Integration)
  async listBackups(filters?: BackupListFilters): Promise<BackupListResponse> {
    const queryParams = filters ? new URLSearchParams(filters) : '';
    const response = await fetch(`${this.baseURL}/api/v1/backup/list?${queryParams}`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch backups: ${response.statusText}`);
    }
    
    return response.json();
  }

  async startBackup(request: StartBackupRequest): Promise<BackupResponse> {
    const response = await fetch(`${this.baseURL}/api/v1/backup/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request)
    });
    
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(`Backup start failed: ${errorData.message || response.statusText}`);
    }
    
    return response.json();
  }

  // File Restore Operations (Task 4 Integration)
  async mountBackupForRestore(backupId: string): Promise<RestoreMountResponse> {
    const response = await fetch(`${this.baseURL}/api/v1/restore/mount`, {
      method: 'POST', 
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ backup_id: backupId })
    });
    
    if (!response.ok) {
      throw new Error(`Mount failed: ${response.statusText}`);
    }
    
    return response.json();
  }
}
```

### **Real-Time Integration (Extending Existing WebSocket)**

```typescript
// Enhance existing WebSocket integration for backup progress
useEffect(() => {
  const socket = io();
  
  // Existing replication progress listener
  socket.on('replication_progress', handleReplicationProgress);
  
  // NEW: Backup progress listener
  socket.on('backup_progress', (data: BackupProgressUpdate) => {
    queryClient.setQueryData(['backups'], (oldData: BackupListResponse) => {
      if (!oldData) return oldData;
      
      return {
        ...oldData,
        backups: oldData.backups.map(backup => 
          backup.id === data.backup_id 
            ? { ...backup, ...data.progress }
            : backup
        )
      };
    });
  });
  
  return () => socket.disconnect();
}, []);
```

---

## 🎨 UI DESIGN SPECIFICATIONS

### **Backup Jobs Table (Following Existing Patterns)**

```
┌─ BACKUP OPERATIONS ────────────────────────────────────────────────────────┐
│                                                                            │
│ ┌─ CONTROL PANEL ─────────────────────────────────────────────────────────┐ │
│ │ Backup Operations                               [🚀 Start Backup]      │ │
│ │ Manage VM backups and file recovery             [📁 Browse Files]      │ │
│ │                                                 [💾 Repositories]     │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                            │
│ ┌─ BACKUP JOBS TABLE ────────────────────────────────────────────────────┐ │
│ │ VM Name        │ Type     │ Repository  │ Status    │ Progress  │ Actions  │ │
│ │ ───────────────┼──────────┼─────────────┼───────────┼───────────┼──────── │ │
│ │ database-prod  │ Full     │ Local SSD   │ 🟢 Running │ ████▓▓ 83% │ [⏸️][🔍] │ │
│ │ exchange-srv   │ Incremen │ AWS S3      │ ✅ Complete│ ██████ 100%│ [📁][🗑️] │ │
│ │ web-server-01  │ Full     │ Local SSD   │ 🔴 Failed  │ ███▓▓▓ 45% │ [🔄][🔍] │ │
│ │ file-server    │ Incremen │ Azure Blob  │ ⏳ Queued  │ ▓▓▓▓▓▓ 0%  │ [▶️][⏹️] │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                            │
│ 📊 Filters: [All VMs ▼] [All Types ▼] [All Status ▼] [🔍 Search...      ] │
└────────────────────────────────────────────────────────────────────────────┘
```

### **Start Backup Modal (Following Existing Modal Patterns)**

```
┌─ START BACKUP OPERATION ──────────────────────────────────────────────────┐
│                                                                           │
│ ┌─ VM SELECTION ────────────────────────────────────────────────────────┐ │
│ │ Select VM: [database-prod-01 ▼]                                        │ │
│ │ Status: ✅ Ready for backup (Last: 6h ago)                            │ │
│ │ Disk: 45GB used / 100GB total                                         │ │
│ └───────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
│ ┌─ BACKUP CONFIGURATION ───────────────────────────────────────────────┐ │
│ │ Type: ○ Full Backup    ● Incremental Backup                          │ │
│ │ Repository: [Local SSD Primary ▼]                                     │ │ 
│ │ Storage: 1.2TB used / 2.0TB total (60%)                              │ │
│ │ Estimated Time: ~8 minutes (based on last backup)                     │ │
│ └───────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
│ ┌─ ADVANCED OPTIONS ───────────────────────────────────────────────────┐ │
│ │ ☑ Enable compression                                                  │ │
│ │ ☑ Verify backup integrity                                             │ │
│ │ ☐ Apply backup policy (3-2-1 rule)                                   │ │
│ └───────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
│ [Cancel] [🚀 Start Backup]                                               │
└───────────────────────────────────────────────────────────────────────────┘
```

---

## 📊 ENHANCEMENT TO EXISTING SECTIONS

### **VM Table Enhancement**

**Add Backup Columns:**
```typescript
// Enhance ModernVMTable.tsx
<TableRow>
  <TableCell>{vm.name}</TableCell>
  <TableCell>{vm.platform}</TableCell>
  <TableCell>{vm.power_state}</TableCell>
  
  {/* NEW: Backup Status Column */}
  <TableCell>
    {vm.last_backup ? (
      <div>
        <Badge color="success">✅ {vm.last_backup_age}</Badge>
        <div className="text-xs text-gray-500">
          {vm.backup_count} backups, {vm.backup_size}
        </div>
      </div>
    ) : (
      <Badge color="warning">No backups</Badge>
    )}
  </TableCell>
  
  <TableCell>
    <div className="flex space-x-1">
      {/* Existing buttons */}
      <Button size="xs" onClick={() => handleReplicate(vm.name)}>
        🔄 Replicate
      </Button>
      
      {/* NEW: Backup buttons */}
      <Button size="xs" variant="outline" onClick={() => handleBackup(vm.name)}>
        💾 Backup
      </Button>
      
      {vm.has_backups && (
        <Button size="xs" variant="secondary" onClick={() => handleBrowseFiles(vm.name)}>
          📁 Files
        </Button>
      )}
    </div>
  </TableCell>
</TableRow>
```

### **Dashboard Enhancement**

**Add Backup Statistics Cards:**
```typescript
// Enhance DashboardOverview.tsx
<div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
  {/* Existing cards */}
  <StatsCard 
    title="Total VMs" 
    value="247" 
    icon={<HiServer />}
  />
  <StatsCard 
    title="Active Replications" 
    value="23" 
    icon={<HiLightningBolt />}
    trend="+2 today"
  />
  
  {/* NEW: Backup statistics */}
  <StatsCard 
    title="Backups Today" 
    value="12" 
    icon={<HiDatabase />}
    trend="8 completed, 4 running"
  />
  <StatsCard 
    title="Storage Used" 
    value="1.2TB / 2.0TB" 
    icon={<HiArchive />}
    trend="60% capacity"
  />
</div>
```

---

## 🔗 INTEGRATION TESTING PLAN

### **Phase 1 Testing: Navigation**
- [ ] Click "Backups" in sidebar navigation
- [ ] Verify backups page loads without errors
- [ ] Confirm URL routing works (`/backups`)
- [ ] Test navigation state management

### **Phase 2 Testing: Backup Jobs**
- [ ] Verify backup jobs list displays correctly
- [ ] Test real-time progress updates
- [ ] Validate sorting and filtering
- [ ] Test error handling for API failures

### **Phase 3 Testing: Start Backup**
- [ ] Open start backup modal
- [ ] Select VM and configure backup
- [ ] Submit backup job successfully
- [ ] Verify job appears in list with "pending" status
- [ ] Monitor progress updates in real-time

### **Phase 4 Testing: File Restore**
- [ ] Click "Browse Files" on completed backup
- [ ] Mount backup successfully
- [ ] Navigate directory tree
- [ ] Download individual file
- [ ] Unmount backup when finished

### **Phase 5 Testing: VM Integration**
- [ ] Start backup from VM table row
- [ ] View backup status in context panel
- [ ] Browse files from VM context
- [ ] Verify backup operations don't interfere with replication

---

## 🚀 COMPETITIVE VALUE

### **Customer Benefits**
- ✅ **Familiar Interface:** Uses existing professional GUI customers already know
- ✅ **Complete Workflow:** VM management → Backup → File recovery in single interface
- ✅ **Self-Service Operations:** Customers can manage backups without IT intervention
- ✅ **Real-Time Monitoring:** Live backup progress like existing replication jobs

### **Business Benefits**
- ✅ **Faster Delivery:** 5-7 days vs 3+ weeks for new GUI
- ✅ **Lower Risk:** Builds on proven interface and patterns
- ✅ **Customer Adoption:** Familiar navigation reduces training requirements
- ✅ **Revenue Enablement:** GUI-driven backups unlock $10-25/VM pricing tiers

### **Technical Benefits**
- ✅ **Code Reuse:** Leverages existing component patterns and API infrastructure
- ✅ **Maintenance:** Single GUI codebase instead of multiple interfaces
- ✅ **Consistency:** Same error handling, styling, and behavior patterns
- ✅ **Future Ready:** Can enhance with cockpit elements later

---

## ✅ ACCEPTANCE CRITERIA

### **Functional Requirements**
- [ ] **Complete Backup Workflow:** Start → Monitor → Complete → Browse Files
- [ ] **VM Integration:** Backup operations accessible from VM management
- [ ] **Real-Time Updates:** Progress visible without manual refresh
- [ ] **File Recovery:** Browse and download files from any backup
- [ ] **Error Handling:** Clear error messages and recovery guidance

### **Technical Requirements**
- [ ] **API Integration:** All Task 5 backup and Task 4 restore endpoints working
- [ ] **Design Consistency:** Matches existing GUI aesthetic perfectly
- [ ] **TypeScript:** All new code strictly typed
- [ ] **Performance:** No regressions in existing functionality
- [ ] **Mobile Responsive:** Works on tablets (following existing responsive patterns)

### **Business Requirements**
- [ ] **Professional Quality:** Interface suitable for enterprise customer demos
- [ ] **User Experience:** Intuitive workflow requiring minimal training
- [ ] **Customer Value:** Complete backup management capabilities via GUI
- [ ] **Revenue Ready:** Interface supports $10-25/VM tier customer adoption

---

## 🚨 PROJECT RULES COMPLIANCE CHECKLIST

- [x] **Source Authority:** Using existing `deployment/sha-appliance/gui/` structure ✅
- [x] **API Integration:** Task 5 + Task 4 endpoints (no modifications required) ✅
- [x] **No Simulations:** Real backup operations via operational backend ✅
- [x] **Documentation Updates:** Will update GUI documentation with new sections ✅
- [x] **Design Consistency:** Following existing Flowbite + Tailwind patterns ✅
- [x] **Error Handling:** Using existing error handling and notification systems ✅

---

## 🎯 SUCCESS METRICS

### **Implementation Success**
- [ ] **5-7 Day Delivery:** Complete backup section within estimated timeline
- [ ] **Zero Regressions:** Existing GUI functionality unaffected
- [ ] **Professional Quality:** Interface matches existing sections' polish
- [ ] **API Coverage:** All backup and restore endpoints accessible via GUI

### **Customer Success**
- [ ] **Intuitive Operation:** Backup workflow discoverable without training
- [ ] **Complete Functionality:** Start backup → Monitor → Recover files
- [ ] **Performance:** Backup operations as responsive as existing replication
- [ ] **Error Recovery:** Clear guidance when operations fail

---

## 🔗 READY FOR IMPLEMENTATION

### **Foundation Ready (Existing GUI)**
- ✅ **Professional Interface:** Next.js 15 + Flowbite production-ready
- ✅ **Component Patterns:** Established patterns for tables, modals, forms
- ✅ **API Infrastructure:** APIClient with proper error handling
- ✅ **Real-Time Updates:** WebSocket integration operational
- ✅ **TypeScript:** Strict typing with proper interfaces

### **Backend Ready (Tasks 1-5)**
- ✅ **Backup APIs:** Complete REST endpoints for backup operations
- ✅ **Restore APIs:** Complete file-level restore functionality
- ✅ **Repository System:** Multi-repository backend operational
- ✅ **Database Schema:** All backup tables and relationships ready

---

## 🚀 IMPLEMENTATION APPROACH

**This job integrates backup functionality into the existing professional GUI, delivering customer value quickly while maintaining design quality and user experience consistency.**

**Customer Journey Enabled:**
```
Virtual Machines → Select VM → Backup Now → Monitor Progress → 
Browse Backup Files → Download Specific Files → Complete Recovery
```

**Ready to start implementation?** This approach will give customers a complete backup management interface in 5-7 days using the proven foundation we already have.

---

**Job Owner:** Frontend Engineering Team  
**Reviewer:** Project Overseer + UX Review  
**Status:** 🔴 Ready to Start  
**Last Updated:** 2025-10-05  
**Integration Target:** Existing migration-dashboard GUI  
**Delivery Timeline:** 5-7 days
