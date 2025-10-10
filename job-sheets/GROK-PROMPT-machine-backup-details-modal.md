# Grok Prompt: Machine Backup Details Modal

## 🎯 **MISSION**
Add clickable machine rows in Protection Flows that open a modal showing complete backup history, KPIs, and VM details.

---

## 📋 **WHAT EXISTS**

### **Components:**
- ✅ `FlowMachinesTable` - Shows machines (needs click handler)
- ✅ `FlowDetailsPanel` - Parent component with flow context
- ✅ Backend API ready: `GET /api/v1/backups?vm_name={name}&repository_id={repo}`

### **Data Available:**
- ✅ `FlowMachineInfo` type - VM specs + backup_stats
- ✅ Backend returns: backup_id, type, status, bytes_transferred, timestamps, error_message
- ✅ All calculations can be done client-side

---

## 🎨 **WHAT TO BUILD**

### **1. Make Table Rows Clickable**
**File:** `components/features/protection-flows/FlowMachinesTable.tsx`

Add:
- `onMachineClick` prop
- `onClick` handler on `<TableRow>` (line 48)
- Hover styling: `cursor-pointer hover:bg-accent/50`

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

**Modal Sections:**

**A. VM Summary Card**
```
CPU: 2c | Memory: 4GB | Disks: 2 (107GB)
OS: Ubuntu Linux | Power: On
```

**B. KPI Cards (4 columns)**
```
┌───────────┬───────────┬───────────┬───────────┐
│  Total    │  Success  │   Avg     │   Avg     │
│  Backups  │   Rate    │   Size    │ Duration  │
│    12     │    92%    │  42.3GB   │   18min   │
└───────────┴───────────┴───────────┴───────────┘
```

**Calculations:**
- Total: `backups.length`
- Success Rate: `(completed / total * 100).toFixed(0) + '%'`
- Avg Size: `sum(bytes_transferred) / count` (completed only)
- Avg Duration: `sum(completed_at - started_at) / count` (seconds → formatted)

**C. Backup History Table**

Columns: Type | Size | Duration | Status | Date

**Per backup:**
- Type: Badge (Full=blue, Incremental=green)
- Size: `bytes_transferred` formatted (GB/TB)
- Duration: `completed_at - started_at` formatted (Xh Xm)
- Status: Badge (Success=green, Failed=red)
- Date: `created_at` as "Oct 10, 02:00"
- If failed: Expandable error message

**Sort:** Newest first (default)

### **3. API Hook**
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
      return (await response.json()).backups;
    },
    enabled: !!vmName && !!repositoryId,
  });
}
```

### **4. Integration**
**File:** `components/features/protection-flows/FlowDetailsPanel.tsx`

Add state:
```typescript
const [selectedMachine, setSelectedMachine] = useState<FlowMachineInfo | null>(null);
const [isMachineModalOpen, setIsMachineModalOpen] = useState(false);
```

Pass to table:
```typescript
<FlowMachinesTable 
  machines={flowMachines}
  onMachineClick={(machine) => {
    setSelectedMachine(machine);
    setIsMachineModalOpen(true);
  }}
/>
```

Add modal (after RestoreWorkflowModal):
```typescript
<MachineDetailsModal
  machine={selectedMachine}
  repositoryId={flow.repository_id || ''}
  isOpen={isMachineModalOpen}
  onClose={() => setIsMachineModalOpen(false)}
/>
```

---

## 🎨 **UI REQUIREMENTS**

### **Modal:**
- shadcn/ui `Dialog` component
- Max width: 1000px, Max height: 80vh
- Dark theme consistent with existing
- ScrollArea for backup list

### **Status Badges:**
- Success: `bg-green-500/10 text-green-400 border-green-500/20`
- Failed: `bg-red-500/10 text-red-400 border-red-500/20`
- Running: `bg-blue-500/10 text-blue-400 border-blue-500/20`

### **Type Badges:**
- Full: `bg-blue-500/10 text-blue-400`
- Incremental: `bg-green-500/10 text-green-400`

### **Loading/Error:**
- Skeleton loader while fetching
- "No backups found" empty state
- API error message display

---

## 🧪 **TESTING**

### **Test VMs:**
- pgtest1: Multiple backups available
- pgtest2: Individual VM
- pgtest3: Group-based flow

### **Test Scenarios:**
1. ✅ Click VM → modal opens with data
2. ✅ Click VM with no backups → "No backups found"
3. ✅ Failed backup → error message displays
4. ✅ KPIs calculate correctly
5. ✅ Close modal → state resets

---

## 📂 **FILES**

### **Create:**
- `components/features/protection-flows/MachineDetailsModal.tsx`

### **Modify:**
- `components/features/protection-flows/FlowMachinesTable.tsx`
- `components/features/protection-flows/FlowDetailsPanel.tsx`
- `src/features/protection-flows/hooks/useProtectionFlows.ts`

---

## 🚨 **CURSORULES COMPLIANCE**

### **Before Claiming Complete:**
- [ ] Code compiles (no errors)
- [ ] Linter passes
- [ ] End-to-end test: Click VM → see accurate data
- [ ] No placeholder/commented code
- [ ] Screenshot evidence provided

### **Documentation:**
- [ ] Update `CHANGELOG.md` with feature
- [ ] Screenshot of working modal

---

## 💡 **HELPER FUNCTIONS**

```typescript
// Format bytes
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
};

// Format duration
const formatDuration = (seconds: number): string => {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m`;
  return `${Math.floor(seconds)}s`;
};

// Calculate duration from timestamps
const getDuration = (startedAt: string, completedAt: string): number => {
  const start = new Date(startedAt).getTime();
  const end = new Date(completedAt).getTime();
  return (end - start) / 1000; // seconds
};
```

---

## 🎯 **SUCCESS CRITERIA**

✅ Modal opens on row click  
✅ VM summary shows correct specs  
✅ KPIs calculate accurately  
✅ Backup list shows all backups with size/duration/status  
✅ Failed backups show error messages  
✅ No console errors  
✅ Repository filtering works  
✅ Screenshot evidence provided  

---

## 🔗 **REFERENCES**

**Tech Spec:** `/home/oma_admin/sendense/job-sheets/TECH-SPEC-machine-backup-details-modal.md`

**Backend API:**
- Endpoint: `GET /api/v1/backups?vm_name={name}&repository_id={repo}`
- Handler: `source/current/sha/api/handlers/backup_handlers.go:483`

**Database:**
- Schema: `source/current/api-documentation/DB_SCHEMA.md:142-149`

**Existing Modal Pattern:**
- Reference: `components/features/protection-flows/RestoreWorkflowModal.tsx`

---

**Ready to implement! All backend APIs exist, purely GUI feature.** 🚀

