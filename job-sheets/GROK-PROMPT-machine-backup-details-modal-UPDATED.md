# Grok Prompt: Machine Backup Details Modal (UPDATED WITH TELEMETRY)

## 🎯 **MISSION**
Add clickable machine rows in Protection Flows that open a modal showing complete backup history, KPIs, and VM details with **accurate real-time data**.

---

## 🚨 **CRITICAL CONTEXT: TELEMETRY NOW WORKING**

**Session Date:** 2025-10-10

**What Was Fixed:**
1. ✅ **`bytes_transferred` bug** - Was always 0, now accurately reports backup size
2. ✅ **Real-time telemetry** - SBC now sends progress/speed/phase data to SHA
3. ✅ **Data persistence** - Telemetry data survives job completion (no more zeros!)
4. ✅ **JSON decoding** - SHA now properly decodes telemetry from SBC

**Root Causes Fixed:**
- Missing JSON tags in SHA's `TelemetryUpdate` struct
- Completion logic overwriting telemetry data with zeros
- Smart update logic now preserves good data

**Result:** ALL backup_jobs fields now populate with real data:
- `bytes_transferred` - ✅ Accurate backup size (was 0 before today)
- `progress_percent` - ✅ Real-time progress during backup
- `transfer_speed_bps` - ✅ Actual transfer speed
- `current_phase` - ✅ "transferring", "completed", etc.
- `last_telemetry_at` - ✅ Last update timestamp

**IMPORTANT:** The modal can now show **accurate backup sizes** - this was the original blocker!

---

## 📋 **WHAT EXISTS**

### **Components:**
- ✅ `FlowMachinesTable` - Shows machines (needs click handler)
- ✅ `FlowDetailsPanel` - Parent component with flow context
- ✅ Backend API ready: `GET /api/v1/backups?vm_name={name}&repository_id={repo}`

### **Data Available (NOW WITH ACCURATE VALUES):**
- ✅ `FlowMachineInfo` type - VM specs + backup_stats
- ✅ Backend returns per backup:
  - `backup_id` - Unique identifier
  - `type` - "full" or "incremental"
  - `status` - "completed", "failed", "running"
  - `bytes_transferred` - ✅ **NOW ACCURATE** (was 0 before 2025-10-10 fix)
  - `progress_percent` - Real-time progress (0-100)
  - `transfer_speed_bps` - Speed in bytes/sec
  - `current_phase` - "transferring", "completed", etc.
  - `created_at` - Start timestamp
  - `started_at` - Actual start time
  - `completed_at` - End timestamp
  - `error_message` - If failed
  - `last_telemetry_at` - Last real-time update

### **Backend Endpoints:**
```
GET /api/v1/backups?vm_name={name}&repository_id={repo}
```
**Returns:** Array of backup jobs with ALL fields populated correctly

**Handler:** `sha/api/handlers/backup_handlers.go:483` (ListBackups)

**Key Filters:**
- Filters by `vm_name` (exact match)
- Filters by `repository_id` 
- Returns parent jobs only (not per-disk records)
- Ordered by newest first

---

## 🎨 **WHAT TO BUILD**

### **1. Make Table Rows Clickable**
**File:** `components/features/protection-flows/FlowMachinesTable.tsx`

Add:
- `onMachineClick` prop: `(machine: FlowMachineInfo) => void`
- `onClick` handler on `<TableRow>` (around line 48)
- Hover styling: `cursor-pointer hover:bg-accent/50`
- Accessible: `role="button" tabIndex={0}` for keyboard nav

**Example:**
```typescript
<TableRow 
  key={machine.vm_name}
  className="cursor-pointer hover:bg-accent/50 transition-colors"
  onClick={() => onMachineClick(machine)}
>
```

### **2. Create Modal Component**
**New File:** `components/features/protection-flows/MachineDetailsModal.tsx`

**Props:**
```typescript
interface MachineDetailsModalProps {
  machine: FlowMachineInfo | null;
  repositoryId: string;
  isOpen: boolean;
  onClose: () => void;
}
```

**Modal Layout:**

#### **A. Header**
```
┌─────────────────────────────────────────┐
│  🖥️ pgtest1                       [X]  │
└─────────────────────────────────────────┘
```

#### **B. VM Summary Card** (Top section)
```
┌──────────────────────────────────────────────────────────┐
│  CPU: 2 cores | Memory: 4 GB | Disks: 2 (107 GB total)  │
│  OS: Ubuntu 20.04 | Power State: poweredOn              │
└──────────────────────────────────────────────────────────┘
```

**Data Source:** `machine` prop (already has these fields)

#### **C. KPI Cards** (4 columns)
```
┌──────────────┬──────────────┬──────────────┬──────────────┐
│  📊 Total    │  ✅ Success  │  💾 Avg      │  ⏱️ Avg     │
│   Backups    │    Rate      │   Size       │  Duration    │
│     12       │     92%      │   42.3 GB    │    18 min    │
└──────────────┴──────────────┴──────────────┴──────────────┘
```

**Calculations:**
```typescript
// Total backups
const totalBackups = backups.length;

// Success rate
const completedBackups = backups.filter(b => b.status === 'completed').length;
const successRate = totalBackups > 0 
  ? ((completedBackups / totalBackups) * 100).toFixed(0) + '%'
  : 'N/A';

// Average size (completed backups only, using bytes_transferred)
const completedWithSize = backups.filter(b => 
  b.status === 'completed' && b.bytes_transferred > 0
);
const avgSize = completedWithSize.length > 0
  ? completedWithSize.reduce((sum, b) => sum + b.bytes_transferred, 0) / completedWithSize.length
  : 0;
const avgSizeFormatted = formatBytes(avgSize);

// Average duration (completed backups only)
const completedWithTime = backups.filter(b => 
  b.status === 'completed' && b.started_at && b.completed_at
);
const avgDuration = completedWithTime.length > 0
  ? completedWithTime.reduce((sum, b) => {
      const duration = getDuration(b.started_at, b.completed_at);
      return sum + duration;
    }, 0) / completedWithTime.length
  : 0;
const avgDurationFormatted = formatDuration(avgDuration);
```

#### **D. Backup History Table**

**Columns:** Type | Size | Duration | Status | Timestamp

**Table Structure:**
```
┌──────────────┬─────────────┬──────────┬────────────┬──────────────────┐
│ Type         │ Size        │ Duration │ Status     │ Date             │
├──────────────┼─────────────┼──────────┼────────────┼──────────────────┤
│ Full         │ 45.2 GB     │ 2h 15m   │ ✅ Success │ Oct 10, 13:40    │
│ Incremental  │ 2.3 GB      │ 18m      │ ✅ Success │ Oct 10, 12:30    │
│ Full         │ 0 B         │ 5m       │ ❌ Failed  │ Oct 10, 11:00    │
│   └─ Error: Connection timeout to vCenter                          │
└──────────────┴─────────────┴──────────┴────────────┴──────────────────┘
```

**Per Backup Row:**

1. **Type Badge:**
   - Full: `<Badge variant="outline" className="bg-blue-500/10 text-blue-400">Full</Badge>`
   - Incremental: `<Badge variant="outline" className="bg-green-500/10 text-green-400">Incremental</Badge>`

2. **Size:**
   - Use `bytes_transferred` field (NOW ACCURATE!)
   - Format with `formatBytes()` helper
   - Show "0 B" if failed or bytes not available

3. **Duration:**
   - Calculate: `completed_at - started_at`
   - Format with `formatDuration()` helper
   - Show "N/A" if no started_at/completed_at

4. **Status Badge:**
   - Success: `<Badge className="bg-green-500/10 text-green-400 border-green-500/20">✅ Success</Badge>`
   - Failed: `<Badge className="bg-red-500/10 text-red-400 border-red-500/20">❌ Failed</Badge>`
   - Running: `<Badge className="bg-blue-500/10 text-blue-400 border-blue-500/20">🔄 Running</Badge>`

5. **Timestamp:**
   - Format `created_at` as "Oct 10, 13:40"
   - Use `format(new Date(backup.created_at), 'MMM dd, HH:mm')`

6. **Error Display** (if status === 'failed'):
   - Expandable row or inline error message below
   - Show `error_message` field
   - Style: `text-sm text-red-400 italic`

**Sorting:** Newest first (default) - already sorted by backend

### **3. API Hook**
**File:** `src/features/protection-flows/hooks/useProtectionFlows.ts`

```typescript
import { useQuery } from '@tanstack/react-query';

export function useMachineBackups(vmName: string | null, repositoryId: string) {
  return useQuery({
    queryKey: ['machine-backups', vmName, repositoryId],
    queryFn: async () => {
      if (!vmName) return null;
      
      const params = new URLSearchParams({
        vm_name: vmName,
        repository_id: repositoryId,
      });
      
      const response = await fetch(
        `${API_BASE}/backups?${params.toString()}`
      );
      
      if (!response.ok) {
        throw new Error(`Failed to fetch backups: ${response.statusText}`);
      }
      
      const data = await response.json();
      return data.backups || [];
    },
    enabled: !!vmName && !!repositoryId,
    staleTime: 30000, // 30 seconds - reasonable for backup data
  });
}
```

### **4. Integration**
**File:** `components/features/protection-flows/FlowDetailsPanel.tsx`

Add state (around existing modal state):
```typescript
const [selectedMachine, setSelectedMachine] = useState<FlowMachineInfo | null>(null);
const [isMachineModalOpen, setIsMachineModalOpen] = useState(false);
```

Update FlowMachinesTable props:
```typescript
<FlowMachinesTable 
  machines={flowMachines}
  onMachineClick={(machine) => {
    setSelectedMachine(machine);
    setIsMachineModalOpen(true);
  }}
/>
```

Add modal render (after existing RestoreWorkflowModal):
```typescript
<MachineDetailsModal
  machine={selectedMachine}
  repositoryId={flow.repository_id || ''}
  isOpen={isMachineModalOpen}
  onClose={() => {
    setIsMachineModalOpen(false);
    setSelectedMachine(null); // Reset state on close
  }}
/>
```

---

## 🎨 **UI/UX REQUIREMENTS**

### **Modal Structure:**
- Use shadcn/ui `Dialog` component
- Max width: `1000px`
- Max height: `80vh`
- Dark theme consistent with existing UI
- `ScrollArea` component for backup list

### **Responsive:**
- Stack KPI cards on mobile (2x2 grid)
- Horizontal scroll for table on small screens
- Touch-friendly click targets

### **Status Badges:**
```typescript
const statusStyles = {
  completed: "bg-green-500/10 text-green-400 border-green-500/20",
  failed: "bg-red-500/10 text-red-400 border-red-500/20",
  running: "bg-blue-500/10 text-blue-400 border-blue-500/20",
};
```

### **Type Badges:**
```typescript
const typeStyles = {
  full: "bg-blue-500/10 text-blue-400 border-blue-500/20",
  incremental: "bg-green-500/10 text-green-400 border-green-500/20",
};
```

### **Loading States:**
- Show skeleton loader while `isLoading`
- Spinner for KPI cards
- Skeleton rows for backup table

### **Empty States:**
```typescript
{backups.length === 0 && (
  <div className="text-center py-12 text-muted-foreground">
    <p>No backups found for this machine.</p>
    <p className="text-sm mt-2">Backups will appear here once protection runs.</p>
  </div>
)}
```

### **Error State:**
```typescript
{isError && (
  <Alert variant="destructive">
    <AlertCircle className="h-4 w-4" />
    <AlertDescription>
      Failed to load backup history: {error.message}
    </AlertDescription>
  </Alert>
)}
```

---

## 🧪 **TESTING**

### **Test VMs:**
- **pgtest1:** Individual VM with multiple backups (VERIFIED WORKING)
- **pgtest2:** Another individual VM
- **pgtest3:** Part of group-based flow

### **Test Data Verification:**
After today's fixes, verify these fields are NON-ZERO for completed backups:
- ✅ `bytes_transferred` (was 0 before, now accurate)
- ✅ `progress_percent` (should be 100 for completed)
- ✅ `transfer_speed_bps` (average speed)

### **Test Scenarios:**

1. **Happy Path:**
   - Click pgtest1 row → modal opens
   - Verify VM summary shows correct CPU/Memory/Disks
   - Verify KPIs calculate correctly
   - Verify backup sizes are non-zero (e.g., "45.2 GB" not "0 B")
   - Close modal → modal closes, state resets

2. **Multiple Backups:**
   - Verify list shows all backups for VM
   - Verify sorting (newest first)
   - Verify full vs incremental badges
   - Verify sizes differ between full and incremental

3. **Failed Backup:**
   - Find a failed backup
   - Verify red "Failed" badge
   - Verify error message displays
   - Verify size shows "0 B" or "N/A"

4. **Empty State:**
   - Click VM with no backups
   - Verify "No backups found" message

5. **Repository Filtering:**
   - Switch repositories (if applicable)
   - Verify only backups from selected repo show

6. **Edge Cases:**
   - Running backup (if any) - shows progress
   - Very large backup (>1 TB) - formats correctly
   - Very fast backup (<1 min) - duration formats correctly

---

## 💡 **HELPER FUNCTIONS**

```typescript
// Format bytes (supports up to PB)
export const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  if (bytes < 0) return 'N/A';
  
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
};

// Format duration (seconds to human-readable)
export const formatDuration = (seconds: number): string => {
  if (seconds < 0 || isNaN(seconds)) return 'N/A';
  if (seconds < 60) return `${Math.floor(seconds)}s`;
  
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
};

// Calculate duration from timestamps
export const getDuration = (startedAt: string | null, completedAt: string | null): number => {
  if (!startedAt || !completedAt) return 0;
  
  const start = new Date(startedAt).getTime();
  const end = new Date(completedAt).getTime();
  
  if (isNaN(start) || isNaN(end)) return 0;
  
  return (end - start) / 1000; // Convert to seconds
};

// Format timestamp for display
export const formatTimestamp = (timestamp: string): string => {
  try {
    return format(new Date(timestamp), 'MMM dd, HH:mm');
  } catch {
    return 'Invalid date';
  }
};
```

---

## 📂 **FILES TO CREATE/MODIFY**

### **Create:**
- `components/features/protection-flows/MachineDetailsModal.tsx` (NEW)

### **Modify:**
- `components/features/protection-flows/FlowMachinesTable.tsx`
  - Add `onMachineClick` prop
  - Add `onClick` handler to `<TableRow>`
  - Add hover styles

- `components/features/protection-flows/FlowDetailsPanel.tsx`
  - Add modal state (selectedMachine, isMachineModalOpen)
  - Pass `onMachineClick` to FlowMachinesTable
  - Render MachineDetailsModal

- `src/features/protection-flows/hooks/useProtectionFlows.ts`
  - Add `useMachineBackups` hook

### **Imports Needed:**
```typescript
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';
import { AlertCircle } from 'lucide-react';
import { format } from 'date-fns';
```

---

## 🚨 **CURSORULES COMPLIANCE**

### **Before Claiming Complete:**
- [ ] Code compiles with no TypeScript errors
- [ ] Linter passes (`npm run lint`)
- [ ] End-to-end test: Click VM → modal opens → data accurate
- [ ] Verify `bytes_transferred` shows real values (not 0)
- [ ] Verify KPIs calculate correctly
- [ ] No placeholder/commented code
- [ ] Screenshot evidence of working modal

### **Documentation:**
- [ ] Update `CHANGELOG.md`:
  ```markdown
  ## [GUI v1.X.X] - 2025-10-10
  
  ### Added - Machine Backup Details Modal
  - Clickable machine rows in Protection Flows open detailed modal
  - Shows VM summary (CPU, memory, disks, OS, power state)
  - Displays backup KPIs (total, success rate, avg size, avg duration)
  - Lists complete backup history with size, duration, status
  - Failed backups show error messages
  - Uses accurate `bytes_transferred` data from telemetry fix (2025-10-10)
  ```

- [ ] Provide screenshot showing:
  - Modal open with VM summary
  - All 4 KPI cards visible
  - Backup list with real sizes (non-zero values)
  - At least one failed backup with error message

---

## 🎯 **SUCCESS CRITERIA**

✅ Modal opens on row click  
✅ VM summary shows correct specs (CPU/Memory/Disks/OS/Power)  
✅ KPIs calculate accurately:
  - Total backups count
  - Success rate percentage
  - Average size (using bytes_transferred)
  - Average duration (in minutes/hours)
✅ Backup list shows all backups with:
  - Type badge (Full/Incremental)
  - **Accurate size** (non-zero for completed backups)
  - Duration (formatted)
  - Status badge (Success/Failed/Running)
  - Timestamp (formatted)
✅ Failed backups show error messages  
✅ Empty state works ("No backups found")  
✅ Loading states show skeletons  
✅ Error state shows alert  
✅ Repository filtering works correctly  
✅ No console errors or warnings  
✅ Responsive on mobile  
✅ Keyboard accessible (Esc to close, Tab navigation)  
✅ Screenshot evidence provided  

---

## 🔗 **REFERENCES**

### **Documentation:**
- **Tech Spec:** `/home/oma_admin/sendense/job-sheets/TECH-SPEC-machine-backup-details-modal.md`
- **Reference Data Flow:** `/home/oma_admin/sendense/job-sheets/REFERENCE-machine-modal-data-flow.md`
- **Telemetry Debug Findings:** `/home/oma_admin/sendense/TELEMETRY-DEBUG-FINDINGS.md`

### **Backend:**
- **API Endpoint:** `GET /api/v1/backups?vm_name={name}&repository_id={repo}`
- **Handler:** `source/current/sha/api/handlers/backup_handlers.go:483` (ListBackups)
- **Database Schema:** `source/current/api-documentation/DB_SCHEMA.md:142-149`

### **Frontend:**
- **Existing Modal Pattern:** `components/features/protection-flows/RestoreWorkflowModal.tsx`
- **Machine Table:** `components/features/protection-flows/FlowMachinesTable.tsx`
- **Flow Panel:** `components/features/protection-flows/FlowDetailsPanel.tsx`

### **Key Telemetry Fix (2025-10-10):**
- **SHA Fix:** `source/current/sha/services/telemetry_service.go` (Added JSON tags)
- **SHA Fix:** `source/current/sha/workflows/backup.go` (Preserve telemetry on completion)
- **Result:** `bytes_transferred` now accurately reports backup sizes (was 0 before)

---

## 🎉 **READY TO IMPLEMENT**

**Blocker Removed:** The original blocker (zero bytes_transferred) was fixed today (2025-10-10).  
**Backend Ready:** All APIs exist and return accurate data.  
**Frontend Only:** This is purely a GUI feature - no backend changes needed.  
**Data Verified:** Real backup data available with accurate sizes, speeds, and durations.

**Estimated Effort:** 2-4 hours for experienced React developer

---

**Let's make this modal awesome! 🚀**

