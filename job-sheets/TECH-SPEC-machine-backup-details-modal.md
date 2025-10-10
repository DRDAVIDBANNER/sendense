# Tech Spec: Machine Backup Details Modal

**Date:** October 10, 2025  
**Feature:** Click machine in flow → show detailed backup history modal  
**Complexity:** Medium (GUI only, all backend APIs exist)  
**Estimated Effort:** 3-4 hours

---

## 🎯 **OBJECTIVE**

Add clickable rows to `FlowMachinesTable` that open a modal showing:
1. Machine summary (VM specs)
2. Complete backup history list with size, duration, timestamp, status
3. KPI metrics (success rate, average size, average duration)

---

## 📊 **CURRENT STATE**

### **Existing Components**
- ✅ `FlowMachinesTable.tsx` - Shows machines in table (line 75 shows backup count + size)
- ✅ `FlowDetailsPanel.tsx` - Parent component with flow context
- ✅ `FlowMachineInfo` type - Has all VM data including backup_stats

### **Existing Backend APIs**
- ✅ `GET /api/v1/backups` - List backups with filtering
  - Query params: `vm_name`, `repository_id`, `status`, `backup_type`
  - Returns: Full backup details array
- ✅ `GET /api/v1/backups/stats` - Already used for table summary

### **Database Schema**
- ✅ `backup_jobs` table has all required fields:
  - `bytes_transferred` (actual backup size)
  - `created_at`, `started_at`, `completed_at` (for duration calc)
  - `status` (completed/failed/running)
  - `backup_type` (full/incremental)
  - `error_message` (if failed)

---

## 🎨 **FEATURE SPECIFICATION**

### **1. Make Table Rows Clickable**

**File:** `components/features/protection-flows/FlowMachinesTable.tsx`

**Changes:**
- Add `onClick` handler to `<TableRow>` (line 48)
- Add `cursor-pointer hover:bg-accent/50` classes for UX
- Add `onMachineClick` prop to component
- Emit machine data on row click

```typescript
interface FlowMachinesTableProps {
  machines: FlowMachineInfo[];
  onMachineClick?: (machine: FlowMachineInfo) => void; // NEW
}

// In JSX:
<TableRow 
  key={machine.context_id}
  onClick={() => onMachineClick?.(machine)}
  className="cursor-pointer hover:bg-accent/50 transition-colors"
>
```

---

### **2. Create Machine Details Modal Component**

**New File:** `components/features/protection-flows/MachineDetailsModal.tsx`

**Props:**
```typescript
interface MachineDetailsModalProps {
  machine: FlowMachineInfo | null;
  repositoryId: string;  // From flow.repository_id for filtering
  isOpen: boolean;
  onClose: () => void;
}
```

**Modal Structure:**

```
┌─────────────────────────────────────────────────────────┐
│ Machine: pgtest1                               [X]      │
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────┐ │
│ │ VM SUMMARY                                          │ │
│ │ CPU: 2c | Memory: 4GB | Disks: 2 (107GB)           │ │
│ │ OS: Ubuntu Linux | Power: On                        │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ BACKUP KPIs                                         │ │
│ │ ┌───────────┬───────────┬───────────┬───────────┐  │ │
│ │ │  Total    │  Success  │   Avg     │   Avg     │  │ │
│ │ │  Backups  │   Rate    │   Size    │ Duration  │  │ │
│ │ │    12     │    92%    │  42.3GB   │   18min   │  │ │
│ │ └───────────┴───────────┴───────────┴───────────┘  │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ BACKUP HISTORY                                      │ │
│ │ ┌─────────────────────────────────────────────────┐ │ │
│ │ │ Type | Size | Duration | Status | Date         │ │ │
│ │ ├─────────────────────────────────────────────────┤ │ │
│ │ │ Full | 102GB | 45m | ✅ Success | Oct 10 02:00│ │ │
│ │ │ Incr | 4.2GB | 8m  | ✅ Success | Oct 9 02:00 │ │ │
│ │ │ Incr | 3.8GB | 7m  | ❌ Failed | Oct 8 02:00  │ │ │
│ │ │      └─> Error: qemu-nbd process exited       │ │ │
│ │ └─────────────────────────────────────────────────┘ │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

### **3. API Integration**

**Hook:** Create `useMachineBackups` hook

**File:** `src/features/protection-flows/hooks/useProtectionFlows.ts`

```typescript
export function useMachineBackups(vmName: string | null, repositoryId: string) {
  return useQuery({
    queryKey: ['machine-backups', vmName, repositoryId],
    queryFn: async () => {
      if (!vmName) return null;
      
      const response = await fetch(
        `${API_BASE}/backups?vm_name=${encodeURIComponent(vmName)}&repository_id=${repositoryId}`
      );
      
      if (!response.ok) throw new Error('Failed to fetch backups');
      
      const data = await response.json();
      return data.backups; // Array of BackupResponse
    },
    enabled: !!vmName && !!repositoryId,
  });
}
```

**API Response Format (from backend):**
```json
{
  "backups": [
    {
      "backup_id": "backup-pgtest1-1728518400",
      "vm_name": "pgtest1",
      "backup_type": "full",
      "status": "completed",
      "bytes_transferred": 109521739776,
      "total_bytes": 109521739776,
      "disks_count": 2,
      "created_at": "2025-10-10T02:00:00Z",
      "started_at": "2025-10-10T02:00:15Z",
      "completed_at": "2025-10-10T02:45:23Z",
      "error_message": ""
    }
  ],
  "total": 12
}
```

---

### **4. Data Processing & KPI Calculations**

**In Modal Component:**

```typescript
const calculateKPIs = (backups: BackupResponse[]) => {
  const total = backups.length;
  const successful = backups.filter(b => b.status === 'completed').length;
  const successRate = total > 0 ? (successful / total * 100).toFixed(0) : 0;
  
  const completedBackups = backups.filter(b => b.status === 'completed');
  
  // Average size (bytes_transferred)
  const avgSize = completedBackups.length > 0
    ? completedBackups.reduce((sum, b) => sum + b.bytes_transferred, 0) / completedBackups.length
    : 0;
  
  // Average duration (completed_at - started_at in seconds)
  const avgDuration = completedBackups.length > 0
    ? completedBackups.reduce((sum, b) => {
        const start = new Date(b.started_at).getTime();
        const end = new Date(b.completed_at).getTime();
        return sum + (end - start) / 1000; // seconds
      }, 0) / completedBackups.length
    : 0;
  
  return {
    total,
    successRate: `${successRate}%`,
    avgSize: formatBytes(avgSize),
    avgDuration: formatDuration(avgDuration),
  };
};

const formatDuration = (seconds: number): string => {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m`;
  return `${Math.floor(seconds)}s`;
};
```

---

### **5. Backup History Table**

**Features:**
- Sortable by date (default: newest first)
- Color-coded status badges
- Expandable error messages for failed backups
- Duration calculated from timestamps
- Size formatted (GB/TB)

**Columns:**
1. **Type** - Badge: Full (blue) / Incremental (green)
2. **Size** - `bytes_transferred` formatted
3. **Duration** - Calculated from `completed_at - started_at`
4. **Status** - Badge: Success (green) / Failed (red) / Running (blue)
5. **Date** - `created_at` formatted as "Oct 10, 02:00"

**Per Row:**
- ✅ Success: Green checkmark + timestamp
- ❌ Failed: Red X + error message expandable
- 🔄 Running: Blue spinner + progress (if available)

---

### **6. Parent Component Integration**

**File:** `components/features/protection-flows/FlowDetailsPanel.tsx`

**Changes:**
```typescript
import { MachineDetailsModal } from "./MachineDetailsModal";

// Add state
const [selectedMachine, setSelectedMachine] = useState<FlowMachineInfo | null>(null);
const [isMachineModalOpen, setIsMachineModalOpen] = useState(false);

// Handler
const handleMachineClick = (machine: FlowMachineInfo) => {
  setSelectedMachine(machine);
  setIsMachineModalOpen(true);
};

// In JSX (line 266):
<FlowMachinesTable 
  machines={flowMachines}
  onMachineClick={handleMachineClick}  // NEW
/>

// Add modal at bottom (after RestoreWorkflowModal)
<MachineDetailsModal
  machine={selectedMachine}
  repositoryId={flow.repository_id || ''}
  isOpen={isMachineModalOpen}
  onClose={() => setIsMachineModalOpen(false)}
/>
```

---

## 🎨 **UI/UX REQUIREMENTS**

### **Modal Design:**
- Dark theme consistent with existing GUI
- Max width: 1000px
- Max height: 80vh with scroll
- shadcn/ui components: Dialog, Card, Badge, ScrollArea, Table

### **Status Badges:**
- ✅ Success: `bg-green-500/10 text-green-400 border-green-500/20`
- ❌ Failed: `bg-red-500/10 text-red-400 border-red-500/20`
- 🔄 Running: `bg-blue-500/10 text-blue-400 border-blue-500/20`

### **Type Badges:**
- Full: `bg-blue-500/10 text-blue-400`
- Incremental: `bg-green-500/10 text-green-400`

### **Loading States:**
- Show skeleton loader while fetching backups
- "No backups found" empty state

### **Error Handling:**
- API error: Show error message in modal
- Failed backup: Expandable error message row

---

## 📝 **IMPLEMENTATION CHECKLIST**

### **Phase 1: Table Interaction (30 min)**
- [ ] Add `onMachineClick` prop to `FlowMachinesTable`
- [ ] Make rows clickable with hover effect
- [ ] Add state management in `FlowDetailsPanel`
- [ ] Test click handler fires correctly

### **Phase 2: API Hook (30 min)**
- [ ] Create `useMachineBackups` hook
- [ ] Test API endpoint with existing data
- [ ] Handle loading/error states
- [ ] Verify data shape matches expectations

### **Phase 3: Modal Component (2 hours)**
- [ ] Create `MachineDetailsModal.tsx`
- [ ] Build VM summary section
- [ ] Build KPI cards with calculations
- [ ] Build backup history table
- [ ] Add status badges and formatting
- [ ] Handle error message expansion

### **Phase 4: Integration (30 min)**
- [ ] Wire modal into `FlowDetailsPanel`
- [ ] Test end-to-end flow
- [ ] Verify repository filtering works
- [ ] Test with multiple VMs

### **Phase 5: Polish (30 min)**
- [ ] Add loading skeletons
- [ ] Add empty states
- [ ] Test responsive layout
- [ ] Verify dark theme consistency

---

## 🧪 **TESTING REQUIREMENTS**

### **Test Data:**
- **pgtest1**: Has multiple backups (full + incrementals)
- **pgtest2**: Individual VM with credential_id=35
- **pgtest3**: Group-based flow

### **Test Scenarios:**
1. Click VM with backups → modal opens with data ✅
2. Click VM with no backups → modal shows "No backups found" ✅
3. Failed backup → error message displays ✅
4. API error → error state shows ✅
5. Close modal → state resets ✅
6. Repository filtering → only shows backups from flow's repo ✅

### **Visual Testing:**
- Modal fits on screen (no overflow)
- Table scrolls correctly with many backups
- Status badges render correctly
- Hover effects work
- Mobile responsive (if needed)

---

## 📚 **FILES TO CREATE/MODIFY**

### **New Files:**
1. `components/features/protection-flows/MachineDetailsModal.tsx` (main component)

### **Modified Files:**
1. `components/features/protection-flows/FlowMachinesTable.tsx` (add click handler)
2. `components/features/protection-flows/FlowDetailsPanel.tsx` (integrate modal)
3. `src/features/protection-flows/hooks/useProtectionFlows.ts` (add hook)
4. `src/features/protection-flows/types/index.ts` (add BackupResponse type if needed)

---

## 🚨 **CURSORULES COMPLIANCE**

### **Before Claiming Complete:**
- [ ] Code compiles cleanly (no errors)
- [ ] Linter passes with zero errors
- [ ] End-to-end test succeeds (click VM → see data)
- [ ] No commented code blocks >10 lines
- [ ] No placeholder/simulation data
- [ ] Evidence provided (screenshots of working modal)

### **Documentation:**
- [ ] Update `CHANGELOG.md` with feature entry
- [ ] Update `API_REFERENCE.md` if new endpoints (N/A - using existing)
- [ ] Update project goals document

### **Status Reporting:**
```markdown
**Status:** [X]% Complete - [state]
**Evidence:** [screenshot/description]
**Blockers:** [if any]
**Next:** [specific action]
```

---

## 🎯 **SUCCESS CRITERIA**

✅ **Feature is complete when:**
1. Clicking any VM row opens modal
2. Modal shows VM summary correctly
3. KPIs calculate accurately (success rate, avg size, avg duration)
4. Backup history lists all backups with correct data
5. Status badges show correct colors
6. Error messages display for failed backups
7. Modal closes cleanly
8. No console errors
9. Repository filtering works
10. Screenshot evidence provided

---

## 🔗 **REFERENCES**

### **Existing APIs:**
- Backend: `/home/oma_admin/sendense/source/current/sha/api/handlers/backup_handlers.go`
  - Line 483: `ListBackups` handler
  - Line 1014: `GetBackupStats` handler

### **Database Schema:**
- `/home/oma_admin/sendense/source/current/api-documentation/DB_SCHEMA.md`
  - Lines 142-149: `backup_jobs` table definition

### **Existing Components:**
- `FlowMachinesTable.tsx` - Table to modify
- `FlowDetailsPanel.tsx` - Parent component
- `RestoreWorkflowModal.tsx` - Modal pattern reference

### **API Documentation:**
- `/home/oma_admin/sendense/source/current/api-documentation/OMA.md`
  - Line 611: GET /api/v1/backups endpoint

---

**End of Tech Spec**

