# Flow Machines Panel - Real Data Integration

**Date:** October 10, 2025  
**Component:** Protection Flows Page - Lower Panel "Machines" Tab  
**Status:** Ready for Implementation  
**Priority:** High

---

## Objective

Replace placeholder machine cards with **real data** from the Protection Flow API showing:
- Actual VMs in the flow (via group membership or single VM)
- VM specifications (OS, CPU, Memory, Disks)
- Backup statistics (count and total size)
- **Space-efficient** design (flows can have 100+ VMs)
- **Responsive** to panel size changes

---

## Current State (Placeholder Cards)

```typescript
// source/current/sendense-gui/components/features/protection-flows/FlowDetailsPanel.tsx
// Lines ~150-250 (Machines tab content)
const mockMachines = [
  {
    id: 1,
    name: "web-server-01",
    host: "esxi-01",
    os: "Ubuntu 22.04",
    cpu: "2 cores",
    memory: "4 GB",
    cpuUsage: 45,
    memoryUsage: 62,
    lastActivity: "Oct 06, 15:30"
  },
  // ...more placeholder data
];
```

Currently renders hardcoded cards with progress bars and fake metrics.

---

## Target State (Real API Data)

### API Endpoints to Use

#### 1. Get Flow Details
```http
GET /api/v1/protection-flows/{flow_id}
```

**Response:**
```json
{
  "id": "d9537d4d-a527-11f0-b62d-020200cc0023",
  "name": "pgtest3",
  "target_type": "group",  // or "vm"
  "target_id": "8571eb63-a2cc-11f0-b62d-020200cc0023",  // group_id or vm_context_id
  "repository_id": "repo-local-1759780081",
  "enabled": true
}
```

#### 2a. If `target_type === "group"` - Get Group Members
```sql
-- Query: vm_group_memberships + vm_replication_contexts
SELECT 
  vrc.context_id,
  vrc.vm_name,
  vrc.cpu_count,
  vrc.memory_mb,
  vrc.os_type,
  vrc.power_state
FROM vm_group_memberships vgm
JOIN vm_replication_contexts vrc ON vgm.vm_context_id = vrc.context_id
WHERE vgm.group_id = ? AND vgm.enabled = 1
ORDER BY vrc.vm_name;
```

**Suggested API:** `GET /api/v1/vm-groups/{group_id}/members` *(create if missing)*

#### 2b. If `target_type === "vm"` - Get Single VM
```sql
SELECT context_id, vm_name, cpu_count, memory_mb, os_type, power_state
FROM vm_replication_contexts
WHERE context_id = ?;
```

**Existing API:** `GET /api/v1/vm-contexts/{context_id}`

#### 3. Get VM Disks
```sql
SELECT disk_id, size_gb
FROM vm_disks
WHERE vm_context_id = ?
ORDER BY disk_id;
```

**Suggested:** Include in VM details response or create `GET /api/v1/vm-contexts/{context_id}/disks`

#### 4. Get Backup Statistics
```sql
SELECT 
  COUNT(*) as backup_count,
  SUM(IFNULL(bytes_transferred, 0)) as total_backup_size_bytes
FROM backup_jobs
WHERE vm_name = ? 
  AND status = 'completed'
  AND id NOT LIKE '%-disk%'  -- Parent backups only (multi-disk)
  AND repository_id = ?;  -- Match flow's repository
```

**Suggested API:** `GET /api/v1/backups/stats?vm_name={name}&repository_id={repo}`

---

## Data Structure (TypeScript Interfaces)

```typescript
// Add to: source/current/sendense-gui/src/features/protection-flows/types/index.ts

export interface FlowMachineInfo {
  context_id: string;
  vm_name: string;
  cpu_count: number;
  memory_mb: number;
  os_type: string;  // "windows", "linux", "other"
  power_state: string;  // "poweredOn", "poweredOff"
  disks: VMDiskInfo[];
  backup_stats: VMBackupStats;
}

export interface VMDiskInfo {
  disk_id: string;
  size_gb: number;
}

export interface VMBackupStats {
  backup_count: number;
  total_size_bytes: number;  // Sum of bytes_transferred
  last_backup_at?: string;  // Optional: most recent completed_at
}
```

---

## Component Design (Space-Efficient)

### Layout Requirements

**Current:** 3-column card grid (bulky for 100+ VMs)  
**Target:** Compact table or list with inline stats

#### Option A: Compact Table (Recommended)
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Machines (12)                                              [Search...] [⟳]  │
├──────────────┬───────────────────┬────────────┬───────────┬─────────────────┤
│ VM Name      │ OS                │ CPU/Memory │ Disks     │ Backups         │
├──────────────┼───────────────────┼────────────┼───────────┼─────────────────┤
│ ● pgtest1    │ 🪟 Windows Server │ 2c / 8GB   │ 2 (112GB) │ 5 (1.2TB)      │
│ ● pgtest2    │ 🐧 CentOS 8       │ 4c / 16GB  │ 1 (50GB)  │ 3 (150GB)      │
│ ○ philb-test │ 🪟 Windows 10     │ 2c / 4GB   │ 1 (40GB)  │ 0 (—)          │
└──────────────┴───────────────────┴────────────┴───────────┴─────────────────┘
```

**Key:**
- ● Green = poweredOn
- ○ Gray = poweredOff
- OS icons: 🪟 Windows, 🐧 Linux, 💿 Other
- CPU/Memory: `{cpu_count}c / {memory_gb}GB`
- Disks: `{count} ({total_gb}GB)`
- Backups: `{count} ({total_size_formatted})`

#### Option B: Compact Cards (If table too dense)
```
┌──────────────────────────────────────┐ ┌──────────────────────────────────────┐
│ ● pgtest1                 🪟 Windows │ │ ● pgtest2                    🐧 Linux │
│ 2c / 8GB • 2 disks (112GB)           │ │ 4c / 16GB • 1 disk (50GB)            │
│ Backups: 5 (1.2TB)                   │ │ Backups: 3 (150GB)                   │
└──────────────────────────────────────┘ └──────────────────────────────────────┘
```

### Responsive Behavior

**Panel Heights:**
- < 300px: Show count only ("12 machines")
- 300-500px: Show table with 5-10 rows + scroll
- > 500px: Show table with 15+ rows + scroll

**Panel Widths:**
- < 800px: Stack columns vertically (mobile)
- 800-1200px: Show all columns compact
- > 1200px: Expand columns with more spacing

---

## Implementation Steps

### Backend (If APIs Missing)

**1. Create Group Members API** *(if not exists)*
```go
// source/current/sha/api/handlers/vm_group_handlers.go
func (h *VMGroupHandler) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
    groupID := chi.URLParam(r, "group_id")
    
    var members []struct {
        database.VMReplicationContext
        Disks []database.VMDisk `gorm:"foreignKey:VMContextID;references:ContextID"`
    }
    
    err := h.db.GetGormDB().
        Table("vm_group_memberships vgm").
        Select("vrc.*").
        Joins("JOIN vm_replication_contexts vrc ON vgm.vm_context_id = vrc.context_id").
        Where("vgm.group_id = ? AND vgm.enabled = 1", groupID).
        Preload("Disks").  // Load associated disks
        Find(&members).Error
    
    // ... handle response
}
```

**2. Create Backup Stats API** *(if not exists)*
```go
// source/current/sha/api/handlers/backup_handlers.go
func (h *BackupHandler) GetBackupStats(w http.ResponseWriter, r *http.Request) {
    vmName := r.URL.Query().Get("vm_name")
    repoID := r.URL.Query().Get("repository_id")
    
    var stats struct {
        BackupCount      int     `json:"backup_count"`
        TotalSizeBytes   int64   `json:"total_size_bytes"`
        LastBackupAt     *string `json:"last_backup_at"`
    }
    
    h.db.GetGormDB().
        Table("backup_jobs").
        Select("COUNT(*) as backup_count, SUM(IFNULL(bytes_transferred, 0)) as total_size_bytes, MAX(completed_at) as last_backup_at").
        Where("vm_name = ? AND repository_id = ? AND status = ? AND id NOT LIKE ?",
            vmName, repoID, "completed", "%-disk%").
        Scan(&stats)
    
    // ... handle response
}
```

### Frontend

**1. Update API Client**
```typescript
// source/current/sendense-gui/src/features/protection-flows/api/protectionFlowsApi.ts

export const getFlowMachines = async (flowId: string): Promise<FlowMachineInfo[]> => {
  // Step 1: Get flow details
  const flow = await getProtectionFlow(flowId);
  
  // Step 2: Get VMs based on target_type
  let vms: VMReplicationContext[];
  if (flow.target_type === 'group') {
    const response = await axios.get(`/api/v1/vm-groups/${flow.target_id}/members`);
    vms = response.data.members;
  } else {
    const response = await axios.get(`/api/v1/vm-contexts/${flow.target_id}`);
    vms = [response.data];
  }
  
  // Step 3: Enrich with backup stats
  const enriched = await Promise.all(
    vms.map(async (vm) => {
      const [disks, stats] = await Promise.all([
        axios.get(`/api/v1/vm-contexts/${vm.context_id}/disks`),
        axios.get(`/api/v1/backups/stats?vm_name=${vm.vm_name}&repository_id=${flow.repository_id}`)
      ]);
      
      return {
        ...vm,
        disks: disks.data.disks,
        backup_stats: stats.data
      };
    })
  );
  
  return enriched;
};
```

**2. Create React Query Hook**
```typescript
// source/current/sendense-gui/src/features/protection-flows/hooks/useProtectionFlows.ts

export const useFlowMachines = (flowId: string | null) => {
  return useQuery({
    queryKey: ['protection-flows', flowId, 'machines'],
    queryFn: () => getFlowMachines(flowId!),
    enabled: !!flowId,
    staleTime: 30000,  // 30 seconds
    refetchOnWindowFocus: false
  });
};
```

**3. Update FlowDetailsPanel Component**
```typescript
// source/current/sendense-gui/components/features/protection-flows/FlowDetailsPanel.tsx

const FlowDetailsPanel = ({ selectedFlow }: { selectedFlow: Flow | null }) => {
  const { data: machines, isLoading, error } = useFlowMachines(selectedFlow?.id);
  
  // In "Machines" tab content:
  if (!selectedFlow) {
    return <div>Select a flow to view machines</div>;
  }
  
  if (isLoading) {
    return <Spinner />;
  }
  
  if (error) {
    return <Alert color="failure">Failed to load machines: {error.message}</Alert>;
  }
  
  return (
    <div className="overflow-y-auto">
      <FlowMachinesTable machines={machines} />
    </div>
  );
};
```

**4. Create FlowMachinesTable Component**
```typescript
// source/current/sendense-gui/components/features/protection-flows/FlowMachinesTable.tsx

export const FlowMachinesTable = ({ machines }: { machines: FlowMachineInfo[] }) => {
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '—';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const k = 1024;
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${units[i]}`;
  };
  
  const getOSIcon = (osType: string) => {
    if (osType.toLowerCase().includes('windows')) return '🪟';
    if (osType.toLowerCase().includes('linux') || osType.toLowerCase().includes('centos') || osType.toLowerCase().includes('ubuntu')) return '🐧';
    return '💿';
  };
  
  const getPowerStateColor = (state: string) => {
    return state === 'poweredOn' ? 'text-green-400' : 'text-gray-400';
  };
  
  const totalDisksGB = (disks: VMDiskInfo[]) => {
    return disks.reduce((sum, disk) => sum + disk.size_gb, 0);
  };
  
  return (
    <div className="relative overflow-auto">
      <Table className="min-w-full">
        <Table.Head>
          <Table.HeadCell>VM Name</Table.HeadCell>
          <Table.HeadCell>OS</Table.HeadCell>
          <Table.HeadCell>CPU/Memory</Table.HeadCell>
          <Table.HeadCell>Disks</Table.HeadCell>
          <Table.HeadCell>Backups</Table.HeadCell>
        </Table.Head>
        <Table.Body className="divide-y divide-border">
          {machines.map((machine) => (
            <Table.Row key={machine.context_id} className="hover:bg-muted/50">
              <Table.Cell className="font-medium">
                <span className={getPowerStateColor(machine.power_state)}>●</span>{' '}
                {machine.vm_name}
              </Table.Cell>
              <Table.Cell>
                {getOSIcon(machine.os_type)} {machine.os_type}
              </Table.Cell>
              <Table.Cell className="text-sm">
                {machine.cpu_count}c / {Math.round(machine.memory_mb / 1024)}GB
              </Table.Cell>
              <Table.Cell className="text-sm">
                {machine.disks.length} ({totalDisksGB(machine.disks)}GB)
              </Table.Cell>
              <Table.Cell className="text-sm">
                {machine.backup_stats.backup_count}{' '}
                ({formatBytes(machine.backup_stats.total_size_bytes)})
              </Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table>
      
      {machines.length === 0 && (
        <div className="p-8 text-center text-muted-foreground">
          No machines in this flow
        </div>
      )}
    </div>
  );
};
```

---

## Testing Checklist

### Data Accuracy
- [ ] Group flows show all group members
- [ ] Single VM flows show 1 machine
- [ ] CPU/Memory values match vCenter
- [ ] Disk counts and sizes are correct
- [ ] Backup counts exclude per-disk records (no duplicates)
- [ ] Backup sizes sum bytes_transferred (not total_bytes)

### UI/UX
- [ ] Power state indicator (● green/gray) works
- [ ] OS icons render correctly (Windows/Linux/Other)
- [ ] Table fits in panel (no horizontal scroll unless <800px width)
- [ ] Vertical scroll works for 100+ VMs
- [ ] Empty state shows when no machines
- [ ] Loading spinner shows during data fetch
- [ ] Error alert shows on API failure

### Performance
- [ ] Loads <1s for 10 VMs
- [ ] Loads <3s for 100 VMs
- [ ] No memory leaks on repeated selection changes
- [ ] React Query caching prevents redundant API calls

### Responsive Design
- [ ] Panel height < 300px: shows count only
- [ ] Panel height 300-500px: shows scrollable table (5-10 rows visible)
- [ ] Panel height > 500px: shows full table (15+ rows visible)
- [ ] Panel width < 800px: columns stack or condense
- [ ] Theme consistency: works in light + dark mode

---

## Acceptance Criteria

✅ **Done When:**
1. Selecting a flow shows **real VMs** from the database
2. All VM specs are **accurate** (OS, CPU, Memory, Disks)
3. Backup stats are **correct** (count and size)
4. Design is **space-efficient** (handles 100+ VMs gracefully)
5. Panel is **responsive** to size changes
6. No placeholder data remains
7. Code follows existing GUI patterns (React Query, TypeScript strict, semantic tokens)

---

## Files to Modify

### Backend (If APIs Missing)
- `source/current/sha/api/handlers/vm_group_handlers.go` - Add GetGroupMembers
- `source/current/sha/api/handlers/backup_handlers.go` - Add GetBackupStats
- `source/current/sha/api/server.go` - Register new routes

### Frontend
- `source/current/sendense-gui/src/features/protection-flows/types/index.ts` - Add interfaces
- `source/current/sendense-gui/src/features/protection-flows/api/protectionFlowsApi.ts` - Add getFlowMachines
- `source/current/sendense-gui/src/features/protection-flows/hooks/useProtectionFlows.ts` - Add useFlowMachines
- `source/current/sendense-gui/components/features/protection-flows/FlowDetailsPanel.tsx` - Wire real data
- `source/current/sendense-gui/components/features/protection-flows/FlowMachinesTable.tsx` - **NEW FILE** (table component)

---

## Notes for Grok

1. **Backend APIs may not exist yet** - check first, create if missing
2. **Multi-disk backups** - filter `id NOT LIKE '%-disk%'` to avoid showing parent + disk jobs as separate backups
3. **Size formatting** - use `formatBytes()` utility for human-readable sizes
4. **Semantic colors** - use `text-green-400`, `text-gray-400`, `bg-muted`, `text-muted-foreground` (not hardcoded hex)
5. **React Query** - follow existing patterns in `useProtectionFlows.ts`
6. **Component size** - keep FlowMachinesTable.tsx < 200 lines
7. **Testing** - use `pgtest1`, `pgtest2`, `pgtest3` flows for validation

---

**Created:** October 10, 2025 04:52 BST  
**Author:** AI Assistant (per user requirements)  
**Related:** Protection Flows Engine (job-sheets/2025-10-09-protection-flows-engine.md)


