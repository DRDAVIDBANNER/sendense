# Existing GUI - Backups Section Integration Plan

**Date:** October 5, 2025  
**Project Overseer:** AI Assistant  
**Target:** Add backups functionality to existing migration-dashboard GUI

---

## 📋 EXISTING GUI ANALYSIS

### **Current Structure (GOOD Foundation)** ✅

**Tech Stack:**
- **Framework:** Next.js 15 + React 19 (excellent)
- **UI:** Flowbite React + Tailwind CSS (professional)
- **State:** React Query (perfect for our APIs)
- **Real-Time:** Socket.io (ideal for live updates)

**Layout System:**
- **Three-Panel Layout:** LeftNavigation → MainContentArea → RightContextPanel
- **VM-Centric Design:** VM selection drives context panel
- **Navigation Pattern:** Section-based routing with clean URL structure

**Current Sections:**
```
🏠 Dashboard              ← System overview
🔍 Discovery              ← VM discovery from vCenter
🖥️  Virtual Machines      ← VM management (primary)
📋 Replication Jobs       ← Migration job tracking
⚡ Failover               ← Failover management
🌐 Network Mapping        ← Network configuration
📅 Schedules              ← Automated replication
👥 Machine Groups         ← VM group management
👤 VM Assignment          ← Group assignment
📄 Logs                   ← System logs
📊 Monitoring             ← Real-time monitoring
⚙️ Settings               ← Configuration
```

**API Structure:**
- ✅ Existing APIClient class with proper error handling
- ✅ TypeScript interfaces for VM contexts, jobs, etc.
- ✅ React Query integration for optimistic updates
- ✅ WebSocket integration for real-time updates

---

## 🎯 BACKUPS SECTION INTEGRATION STRATEGY

### **Where to Add Backups in Navigation** 

**Recommended Position:** Between "Replication Jobs" and "Failover"

**Logical Flow:**
```
🖥️  Virtual Machines      ← Discover VMs
📋 Replication Jobs       ← Migration operations  
💾 Backups               ← NEW: Backup operations (fits logically here)
⚡ Failover               ← Emergency operations
🌐 Network Mapping        ← Configuration
```

**Rationale:**
- Backups are another type of job operation (like replication)
- VM → Replication → **Backup** → Failover is logical workflow
- Keeps all "job" operations grouped together

---

## 🏗️ IMPLEMENTATION PLAN

### **Phase 1: Navigation Integration (Day 1)**

**Files to Modify:**

1. **Add Backups Navigation Item**
   ```typescript
   // File: src/components/layout/LeftNavigation.tsx
   // Add to navigationItems array (line 37+):
   
   {
     id: 'backups',
     label: 'Backups',
     icon: HiDatabase,  // or HiArchive
     href: '/backups',
     description: 'Backup operations and recovery'
   }
   ```

2. **Add Backups Route Handling**
   ```typescript
   // File: src/components/layout/MainContentArea.tsx
   // Add to switch statement (line 34+):
   
   case 'backups':
     return <BackupsManagement onVMSelect={onVMSelect} />;
   ```

3. **Create App Route**
   ```
   mkdir src/app/backups/
   touch src/app/backups/page.tsx
   ```

### **Phase 2: Backup Components (Days 2-3)**

**Following Existing Patterns:**

**1. Main Backups Component** (Similar to JobListView.tsx)
```typescript
// File: src/components/backups/BackupsManagement.tsx
export interface BackupsManagementProps {
  onVMSelect: (vmName: string) => void;
}

export const BackupsManagement = ({ onVMSelect }: BackupsManagementProps) => {
  return (
    <div className="h-full flex flex-col">
      <BackupControlPanel />
      <BackupJobsList onVMSelect={onVMSelect} />
    </div>
  );
};
```

**2. Backup Control Panel** (Action Controls)
```typescript
// File: src/components/backups/BackupControlPanel.tsx
export const BackupControlPanel = () => {
  return (
    <Card className="mb-6">
      <div className="flex justify-between items-center p-4">
        <div>
          <h2 className="text-xl font-bold">Backup Operations</h2>
          <p className="text-gray-600">Manage VM backups and recovery</p>
        </div>
        <div className="flex space-x-2">
          <Button onClick={handleStartBackup}>
            🚀 Start Backup
          </Button>
          <Button variant="outline" onClick={handleRestoreFile}>
            📁 Browse Files
          </Button>
          <Button variant="outline" onClick={handleViewRepositories}>
            💾 Repositories
          </Button>
        </div>
      </div>
    </Card>
  );
};
```

**3. Backup Jobs List** (Similar to existing job patterns)
```typescript
// File: src/components/backups/BackupJobsList.tsx
export const BackupJobsList = ({ onVMSelect }: BackupJobsListProps) => {
  const { data: backups, isLoading } = useQuery(['backups'], api.backup.list);
  
  return (
    <Card>
      <Table>
        <TableHead>
          <tr>
            <th>VM Name</th>
            <th>Backup Type</th>
            <th>Repository</th>
            <th>Status</th>
            <th>Progress</th>
            <th>Started</th>
            <th>Actions</th>
          </tr>
        </TableHead>
        <TableBody>
          {backups?.map(backup => (
            <BackupJobRow 
              key={backup.id}
              backup={backup}
              onVMSelect={onVMSelect}
            />
          ))}
        </TableBody>
      </Table>
    </Card>
  );
};
```

### **Phase 3: API Integration (Day 3-4)**

**Extend Existing API Client:**

```typescript
// File: src/lib/api.ts (extend existing APIClient class)

class APIClient {
  // ... existing methods ...

  // Backup Operations (Task 5 integration)
  async startBackup(params: {
    vm_name: string;
    backup_type: 'full' | 'incremental';
    repository_id: string;
  }): Promise<BackupResponse> {
    const response = await fetch(`${this.baseURL}/api/v1/backup/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params)
    });
    
    if (!response.ok) {
      throw new Error(`Backup start failed: ${response.statusText}`);
    }
    
    return response.json();
  }

  async listBackups(filters?: {
    vm_name?: string;
    backup_type?: string;
    status?: string;
  }): Promise<BackupListResponse> {
    const queryParams = new URLSearchParams(filters);
    const response = await fetch(`${this.baseURL}/api/v1/backup/list?${queryParams}`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch backups: ${response.statusText}`);
    }
    
    return response.json();
  }

  async getBackupDetails(backupId: string): Promise<BackupResponse> {
    const response = await fetch(`${this.baseURL}/api/v1/backup/${backupId}`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch backup details: ${response.statusText}`);
    }
    
    return response.json();
  }

  // File Restore Operations (Task 4 integration)
  async mountBackup(backupId: string): Promise<RestoreMountResponse> {
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

### **Phase 4: TypeScript Interfaces (Day 4)**

**New Types (Following Existing Patterns):**

```typescript
// File: src/lib/types.ts (extend existing interfaces)

export interface BackupJob {
  id: string;
  vm_context_id: string;
  vm_name: string;
  backup_type: 'full' | 'incremental';
  status: 'pending' | 'running' | 'completed' | 'failed';
  repository_id: string;
  file_path: string;
  bytes_transferred: number;
  total_bytes: number;
  created_at: string;
  completed_at?: string;
  error_message?: string;
}

export interface BackupListResponse {
  backups: BackupJob[];
  total: number;
}

export interface BackupChain {
  chain_id: string;
  vm_context_id: string;
  disk_id: number;
  full_backup_id: string;
  backups: BackupJob[];
  total_size_bytes: number;
  backup_count: number;
}

export interface RestoreMount {
  mount_id: string;
  backup_id: string;
  mount_path: string;
  nbd_device: string;
  status: string;
  expires_at: string;
}
```

---

## 📋 DETAILED COMPONENT STRUCTURE

### **Backup Section File Structure**

```
src/app/backups/
├── page.tsx                    # Backup main page
└── [backup-id]/
    └── page.tsx                # Individual backup details

src/components/backups/
├── BackupsManagement.tsx       # Main backup management component
├── BackupControlPanel.tsx      # Action buttons and controls
├── BackupJobsList.tsx          # Backup jobs table
├── BackupJobRow.tsx           # Individual backup job row
├── StartBackupModal.tsx        # Start backup form modal
├── BackupDetailsModal.tsx      # Backup details modal
├── RestoreFileModal.tsx        # File restore modal
└── BackupChainView.tsx         # Backup chain visualization

src/components/restore/
├── FileBrowserModal.tsx        # Browse mounted backup files
├── RestoreMountsList.tsx       # Active restore mounts
└── FileDownloadManager.tsx     # Download files interface
```

### **Integration with Existing Components**

**1. VM Table Integration**
```typescript
// Enhance: src/components/vm/ModernVMTable.tsx
// Add backup-related columns and actions

<TableCell>
  <div className="flex space-x-2">
    {/* Existing replication button */}
    <Button size="sm" onClick={() => handleStartReplication(vm.name)}>
      🔄 Replicate
    </Button>
    
    {/* NEW: Backup button */}
    <Button size="sm" variant="outline" onClick={() => handleStartBackup(vm.name)}>
      💾 Backup
    </Button>
    
    {/* NEW: Restore button (if backups exist) */}
    {vm.has_backups && (
      <Button size="sm" variant="secondary" onClick={() => handleBrowseBackups(vm.name)}>
        📁 Restore Files
      </Button>
    )}
  </div>
</TableCell>
```

**2. Dashboard Integration**
```typescript
// Enhance: src/components/DashboardOverview.tsx  
// Add backup statistics

<div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
  {/* Existing cards */}
  <StatsCard title="Total VMs" value="247" />
  <StatsCard title="Replications" value="23" />
  
  {/* NEW: Backup stats */}
  <StatsCard title="Backups Today" value="12" />
  <StatsCard title="Storage Used" value="1.2TB" />
</div>
```

---

## 🔗 API INTEGRATION POINTS

### **Task 5 Backup APIs (Ready to Use)**
- ✅ `POST /api/v1/backup/start` - Start backup
- ✅ `GET /api/v1/backup/list` - List backups  
- ✅ `GET /api/v1/backup/{id}` - Get backup details
- ✅ `DELETE /api/v1/backup/{id}` - Delete backup
- ✅ `GET /api/v1/backup/chain` - Get backup chain

### **Task 4 Restore APIs (Ready to Use)**
- ✅ `POST /api/v1/restore/mount` - Mount backup
- ✅ `GET /api/v1/restore/{mount_id}/files` - Browse files
- ✅ `GET /api/v1/restore/{mount_id}/download` - Download files
- ✅ `DELETE /api/v1/restore/{mount_id}` - Unmount

### **Task 1 Repository APIs (Ready to Use)**
- ✅ `GET /api/v1/repositories` - List repositories
- ✅ `GET /api/v1/repositories/{id}/storage` - Storage capacity

---

## 🚀 IMPLEMENTATION PHASES

### **Phase 1: Basic Navigation & Skeleton (Day 1)**

**Tasks:**
1. Add "Backups" to navigation menu (between Replication Jobs and Failover)
2. Create `/app/backups/page.tsx` route
3. Create basic `BackupsManagement.tsx` component with placeholder
4. Test navigation works

**Evidence:** Backups section accessible via sidebar navigation

### **Phase 2: Backup Jobs List (Days 2-3)**

**Tasks:**
1. Create `BackupJobsList.tsx` following `JobListView.tsx` patterns
2. Integrate Task 5 backup APIs (`/api/v1/backup/list`)
3. Add real-time updates via React Query
4. Create `BackupJobRow.tsx` with action buttons

**Evidence:** Can view existing backup jobs in table format

### **Phase 3: Start Backup Functionality (Days 3-4)**

**Tasks:**
1. Create `StartBackupModal.tsx` following existing modal patterns
2. Integrate with Task 5 `POST /api/v1/backup/start`
3. Add backup type selection (full vs incremental)
4. Add repository selection
5. Add VM selection (from virtual-machines section)

**Evidence:** Can start new backup operations via GUI

### **Phase 4: File Restore Integration (Days 4-5)**

**Tasks:**
1. Create `FileBrowserModal.tsx` for browsing backup contents
2. Integrate Task 4 restore APIs (`/api/v1/restore/mount`, `/files`, `/download`)
3. Add file download functionality
4. Add directory browsing with tree structure

**Evidence:** Can browse and download files from backups

### **Phase 5: Enhanced VM Integration (Day 5)**

**Tasks:**
1. Add backup status columns to VM table
2. Add "Backup Now" buttons to VM rows
3. Add "Browse Backups" buttons for VMs with existing backups
4. Show backup status in right context panel

**Evidence:** Backup operations accessible from VM management section

---

## 📂 REQUIRED NEW FILES

### **App Routes**
```
src/app/backups/
├── page.tsx                    # Main backups page
├── [backup-id]/
│   └── page.tsx                # Individual backup details
└── restore/
    └── page.tsx                # File restore interface
```

### **Backup Components**
```
src/components/backups/
├── BackupsManagement.tsx       # Main backup management view
├── BackupControlPanel.tsx      # Backup action controls
├── BackupJobsList.tsx          # Backup jobs table
├── BackupJobRow.tsx            # Individual backup job row
├── StartBackupModal.tsx        # Start backup dialog
├── BackupDetailsModal.tsx      # View backup details
├── BackupChainView.tsx         # Backup chain visualization
└── BackupFilters.tsx           # Filter backup jobs
```

### **File Restore Components**
```
src/components/restore/
├── FileBrowserModal.tsx        # Browse backup file contents
├── FileTreeView.tsx            # Directory tree navigation
├── FileDownloadManager.tsx     # Download selected files
├── RestoreMountsList.tsx       # Active restore sessions
└── RestoreProgressModal.tsx    # File restore progress
```

### **API Extensions**
```
src/lib/
├── api.ts                      # Extend with backup/restore methods
├── backup-types.ts             # Backup-specific TypeScript interfaces
└── restore-hooks.ts            # React Query hooks for restore operations
```

---

## 🎨 UI DESIGN CONSISTENCY

### **Following Existing Design Patterns**

**1. Match Current Aesthetic:**
- ✅ Dark theme: `bg-gray-50 dark:bg-gray-900`
- ✅ Flowbite React components (Cards, Buttons, Tables)
- ✅ HeroIcons for consistency
- ✅ Tailwind CSS utility classes

**2. Component Structure:**
```typescript
// Follow existing patterns from vm/ModernVMTable.tsx
export const BackupJobsList = ({ onVMSelect }: BackupJobsListProps) => {
  const { data: backups, isLoading, error } = useQuery(['backups'], api.backup.list);
  
  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorMessage error={error} />;
  
  return (
    <Card>
      <Table>
        {/* Table implementation following existing patterns */}
      </Table>
    </Card>
  );
};
```

**3. Action Buttons:**
```typescript
// Consistent with existing VM and job management buttons
<Button size="sm" className="mr-2" onClick={() => handleStartBackup(vm.name)}>
  💾 Backup
</Button>
<Button size="sm" variant="outline" onClick={() => handleBrowseFiles(backup.id)}>
  📁 Files
</Button>
```

---

## 🔌 REAL-TIME INTEGRATION

### **WebSocket Integration (Following Existing Patterns)**

**Backup Progress Updates:**
```typescript
// Extend existing WebSocket integration
useEffect(() => {
  const socket = io();
  
  // Add backup progress listener
  socket.on('backup_progress', (data: BackupProgressUpdate) => {
    queryClient.setQueryData(['backups'], (oldData: BackupListResponse) => {
      // Update backup progress in real-time
      return updateBackupProgress(oldData, data);
    });
  });
  
  return () => socket.disconnect();
}, []);
```

**Live Status Updates:**
- Backup job progress bars update in real-time
- Status changes (pending → running → completed)
- Throughput and ETA updates
- Error notifications

---

## ✅ SUCCESS CRITERIA

### **Functional Requirements**
- [ ] **Backups navigation item** accessible in sidebar
- [ ] **View backup jobs** in table format with real-time updates
- [ ] **Start new backups** via modal dialog (full and incremental)
- [ ] **Browse backup files** via restore mount interface
- [ ] **Download files** from mounted backups
- [ ] **Integration with VM table** (backup buttons on VM rows)

### **Technical Requirements**
- [ ] **API Integration:** All Task 5 backup endpoints working
- [ ] **Real-Time Updates:** Backup progress visible without refresh
- [ ] **Error Handling:** Proper error messages and retry logic
- [ ] **TypeScript:** All new code strictly typed
- [ ] **Design Consistency:** Matches existing GUI aesthetic
- [ ] **Mobile Responsive:** Works on tablets and mobile

### **User Experience**
- [ ] **Intuitive Workflow:** VM → Start Backup → Monitor Progress → Browse Files
- [ ] **Fast Operations:** Backup start in <2 clicks from any VM
- [ ] **Clear Status:** Backup progress and status always visible
- [ ] **Easy Recovery:** File browsing and download straightforward

---

## 🎯 BENEFITS OF EXISTING GUI INTEGRATION

### **Why This Approach Is Smart:**

**1. Leverages Existing Investment**
- ✅ Professional UI already built
- ✅ Navigation patterns established
- ✅ API client infrastructure ready
- ✅ Real-time updates working

**2. Faster Time to Market**
- ✅ No need to rebuild layout system
- ✅ Existing components provide templates
- ✅ Users already familiar with interface
- ✅ Lower learning curve for customers

**3. Consistent User Experience**
- ✅ Same navigation patterns
- ✅ Same design language
- ✅ Same real-time update behavior
- ✅ Integrated workflow (VM → Backup → Restore)

**4. Technical Benefits**
- ✅ React Query already configured
- ✅ WebSocket infrastructure ready
- ✅ Error handling patterns established
- ✅ TypeScript interfaces in place

---

## 📊 ESTIMATED TIMELINE

**Total Duration:** 5-7 days

```
Day 1: Navigation + Skeleton           [██████████] 100%
Day 2: Backup Jobs List               [██████████] 100% 
Day 3: Start Backup + API Integration [██████████] 100%
Day 4: File Browse + Download         [██████████] 100%
Day 5: VM Integration + Polish        [██████████] 100%
```

**Deliverables:**
- Complete backup section in existing GUI
- Integration with Task 5 backup APIs
- File restore functionality
- Enhanced VM management with backup operations

---

## 🚀 READY TO IMPLEMENT

**This integration plan:**
- ✅ **Leverages existing infrastructure** (smart reuse)
- ✅ **Follows established patterns** (consistency)
- ✅ **Integrates our completed APIs** (Task 5 + Task 4)
- ✅ **Delivers customer value quickly** (faster than rebuilding)

**Want to proceed with this plan?** We can add professional backup functionality to the existing GUI in 5-7 days, giving customers a complete backup management interface without rebuilding everything from scratch.
