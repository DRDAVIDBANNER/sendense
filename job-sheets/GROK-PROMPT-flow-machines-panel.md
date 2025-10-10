# GROK PROMPT: Flow Machines Panel - Real Data

**Mission:** Replace placeholder machine cards with real VM data in Protection Flows lower panel

---

## Context

Location: `source/current/sendense-gui/components/features/protection-flows/FlowDetailsPanel.tsx`  
Current State: Shows 3 fake machines (web-server-01, database-01, app-server-01)  
Target: Show REAL VMs from selected flow with specs and backup stats

---

## Quick Summary

**What you're building:**
A compact table showing VMs in the selected protection flow with:
- VM name + power state (● green = on, ○ gray = off)
- OS icon (🪟 Windows, 🐧 Linux, 💿 Other)
- CPU/Memory (e.g., "2c / 8GB")
- Disk count + total size (e.g., "2 (112GB)")
- Backup count + total size (e.g., "5 (1.2TB)")

**Space-efficient design for 100+ VMs:**
- Use table (not cards)
- Compact inline stats
- Scrollable in panel

---

## Architecture

### Step 1: Check If Backend APIs Exist

**Required Endpoints:**

1. **Get Group Members:** `GET /api/v1/vm-groups/{group_id}/members`
   ```json
   {
     "members": [
       {
         "context_id": "ctx-...",
         "vm_name": "pgtest1",
         "cpu_count": 2,
         "memory_mb": 8192,
         "os_type": "windows",
         "power_state": "poweredOn"
       }
     ]
   }
   ```

2. **Get Backup Stats:** `GET /api/v1/backups/stats?vm_name={name}&repository_id={repo}`
   ```json
   {
     "backup_count": 5,
     "total_size_bytes": 1200000000000,
     "last_backup_at": "2025-10-09T16:51:45Z"
   }
   ```

**If missing, create them:**
- `source/current/sha/api/handlers/vm_group_handlers.go`
- `source/current/sha/api/handlers/backup_handlers.go`
- See full job sheet for SQL queries

### Step 2: Create Frontend API Client

```typescript
// source/current/sendense-gui/src/features/protection-flows/api/protectionFlowsApi.ts

export const getFlowMachines = async (flowId: string): Promise<FlowMachineInfo[]> => {
  // 1. Get flow (has target_type: "vm"|"group", target_id)
  const flow = await getProtectionFlow(flowId);
  
  // 2. Get VMs (from group or single VM)
  let vms = [];
  if (flow.target_type === 'group') {
    const res = await axios.get(`/api/v1/vm-groups/${flow.target_id}/members`);
    vms = res.data.members;
  } else {
    const res = await axios.get(`/api/v1/vm-contexts/${flow.target_id}`);
    vms = [res.data];
  }
  
  // 3. Get disks for each VM
  const enriched = await Promise.all(vms.map(async (vm) => {
    const disks = await axios.get(`/api/v1/vm-contexts/${vm.context_id}/disks`);
    const stats = await axios.get(`/api/v1/backups/stats?vm_name=${vm.vm_name}&repository_id=${flow.repository_id}`);
    
    return {
      ...vm,
      disks: disks.data.disks,
      backup_stats: stats.data
    };
  }));
  
  return enriched;
};
```

### Step 3: Create React Query Hook

```typescript
// source/current/sendense-gui/src/features/protection-flows/hooks/useProtectionFlows.ts

export const useFlowMachines = (flowId: string | null) => {
  return useQuery({
    queryKey: ['protection-flows', flowId, 'machines'],
    queryFn: () => getFlowMachines(flowId!),
    enabled: !!flowId,
    staleTime: 30000
  });
};
```

### Step 4: Create Compact Table Component

```typescript
// NEW FILE: source/current/sendense-gui/components/features/protection-flows/FlowMachinesTable.tsx

export const FlowMachinesTable = ({ machines }: { machines: FlowMachineInfo[] }) => {
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '—';
    const k = 1024;
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${['B','KB','MB','GB','TB'][i]}`;
  };
  
  const getOSIcon = (os: string) => 
    os.toLowerCase().includes('windows') ? '🪟' : 
    os.toLowerCase().includes('linux') ? '🐧' : '💿';
  
  const totalDisksGB = (disks: VMDiskInfo[]) => 
    disks.reduce((sum, d) => sum + d.size_gb, 0);
  
  return (
    <Table>
      <Table.Head>
        <Table.HeadCell>VM Name</Table.HeadCell>
        <Table.HeadCell>OS</Table.HeadCell>
        <Table.HeadCell>CPU/Memory</Table.HeadCell>
        <Table.HeadCell>Disks</Table.HeadCell>
        <Table.HeadCell>Backups</Table.HeadCell>
      </Table.Head>
      <Table.Body>
        {machines.map(m => (
          <Table.Row key={m.context_id}>
            <Table.Cell>
              <span className={m.power_state === 'poweredOn' ? 'text-green-400' : 'text-gray-400'}>
                ●
              </span>{' '}
              {m.vm_name}
            </Table.Cell>
            <Table.Cell>{getOSIcon(m.os_type)} {m.os_type}</Table.Cell>
            <Table.Cell>{m.cpu_count}c / {Math.round(m.memory_mb/1024)}GB</Table.Cell>
            <Table.Cell>
              {m.disks.length} ({totalDisksGB(m.disks)}GB)
            </Table.Cell>
            <Table.Cell>
              {m.backup_stats.backup_count} ({formatBytes(m.backup_stats.total_size_bytes)})
            </Table.Cell>
          </Table.Row>
        ))}
      </Table.Body>
    </Table>
  );
};
```

### Step 5: Wire into FlowDetailsPanel

```typescript
// source/current/sendense-gui/components/features/protection-flows/FlowDetailsPanel.tsx
// In "Machines" tab content (replace mockMachines):

const { data: machines, isLoading } = useFlowMachines(selectedFlow?.id);

// Render:
{isLoading ? (
  <Spinner />
) : (
  <FlowMachinesTable machines={machines || []} />
)}
```

---

## TypeScript Interfaces

```typescript
// source/current/sendense-gui/src/features/protection-flows/types/index.ts

export interface FlowMachineInfo {
  context_id: string;
  vm_name: string;
  cpu_count: number;
  memory_mb: number;
  os_type: string;
  power_state: string;
  disks: VMDiskInfo[];
  backup_stats: VMBackupStats;
}

export interface VMDiskInfo {
  disk_id: string;
  size_gb: number;
}

export interface VMBackupStats {
  backup_count: number;
  total_size_bytes: number;
  last_backup_at?: string;
}
```

---

## Backend APIs (If Missing)

### 1. Get Group Members

```go
// source/current/sha/api/handlers/vm_group_handlers.go

func (h *VMGroupHandler) GetGroupMembers(w http.ResponseWriter, r *http.Request) {
    groupID := chi.URLParam(r, "group_id")
    
    var members []struct {
        ContextID   string `json:"context_id"`
        VMName      string `json:"vm_name"`
        CPUCount    int    `json:"cpu_count"`
        MemoryMB    int    `json:"memory_mb"`
        OSType      string `json:"os_type"`
        PowerState  string `json:"power_state"`
    }
    
    err := h.db.GetGormDB().
        Table("vm_group_memberships vgm").
        Select("vrc.context_id, vrc.vm_name, vrc.cpu_count, vrc.memory_mb, vrc.os_type, vrc.power_state").
        Joins("JOIN vm_replication_contexts vrc ON vgm.vm_context_id = vrc.context_id").
        Where("vgm.group_id = ? AND vgm.enabled = 1", groupID).
        Order("vrc.vm_name").
        Find(&members).Error
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "members": members,
        "count":   len(members),
    })
}
```

**Register Route:**
```go
// source/current/sha/api/server.go (in RegisterRoutes)
r.Get("/vm-groups/{group_id}/members", handlers.VMGroup.GetGroupMembers)
```

### 2. Get VM Disks

```go
// source/current/sha/api/handlers/vm_context_handlers.go (or create)

func (h *VMContextHandler) GetVMDisks(w http.ResponseWriter, r *http.Request) {
    contextID := chi.URLParam(r, "context_id")
    
    var disks []struct {
        DiskID string `json:"disk_id"`
        SizeGB int    `json:"size_gb"`
    }
    
    h.db.GetGormDB().
        Table("vm_disks").
        Select("disk_id, size_gb").
        Where("vm_context_id = ?", contextID).
        Order("disk_id").
        Find(&disks)
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "disks": disks,
        "count": len(disks),
    })
}
```

**Register Route:**
```go
r.Get("/vm-contexts/{context_id}/disks", handlers.VMContext.GetVMDisks)
```

### 3. Get Backup Stats

```go
// source/current/sha/api/handlers/backup_handlers.go

func (h *BackupHandler) GetBackupStats(w http.ResponseWriter, r *http.Request) {
    vmName := r.URL.Query().Get("vm_name")
    repoID := r.URL.Query().Get("repository_id")
    
    var stats struct {
        BackupCount      int    `json:"backup_count"`
        TotalSizeBytes   int64  `json:"total_size_bytes"`
        LastBackupAt     string `json:"last_backup_at"`
    }
    
    h.db.GetGormDB().
        Table("backup_jobs").
        Select("COUNT(*) as backup_count, SUM(IFNULL(bytes_transferred, 0)) as total_size_bytes, MAX(completed_at) as last_backup_at").
        Where("vm_name = ? AND repository_id = ? AND status = ? AND id NOT LIKE ?",
            vmName, repoID, "completed", "%-disk%").  // 🔥 CRITICAL: Filter out per-disk records
        Scan(&stats)
    
    json.NewEncoder(w).Encode(stats)
}
```

**Register Route:**
```go
r.Get("/backups/stats", handlers.Backup.GetBackupStats)
```

---

## Testing

**Test with:**
- `pgtest3` flow (group with 1 VM) - should show pgtest3
- `pgtest1` flow (single VM) - should show pgtest1 only
- Empty group - should show "No machines"

**Verify:**
- ✅ Power state colors: green = on, gray = off
- ✅ OS icons: 🪟 Windows, 🐧 Linux
- ✅ CPU/Memory: "2c / 8GB" format
- ✅ Disks: "2 (112GB)" format
- ✅ Backups: "5 (1.2TB)" format (no duplicates!)
- ✅ Panel scrolls for 10+ VMs
- ✅ Theme works (light + dark mode)

---

## CRITICAL RULES

1. **Multi-disk backups:** Filter `id NOT LIKE '%-disk%'` to avoid showing parent + per-disk jobs as separate backups
2. **Semantic colors:** Use `text-green-400`, `bg-muted`, NOT hardcoded colors
3. **React Query:** Follow existing patterns in `useProtectionFlows.ts`
4. **Component size:** Keep FlowMachinesTable.tsx < 200 lines
5. **No placeholder data:** Delete ALL mockMachines code
6. **Database credentials:** `oma_user:oma_password@localhost:3306/migratekit_oma`

---

## Definition of Done

- [ ] Backend APIs exist (check first, create if missing)
- [ ] Frontend shows real VMs from flow
- [ ] All VM specs accurate (OS, CPU, Memory, Disks)
- [ ] Backup stats correct (count and size)
- [ ] Table is space-efficient (works with 100+ VMs)
- [ ] No placeholder data remains
- [ ] Light + dark mode both work
- [ ] No linter errors

---

**Full Job Sheet:** `/home/oma_admin/sendense/job-sheets/2025-10-10-flow-machines-panel.md`  
**Database Creds:** `oma_user:oma_password@localhost:3306/migratekit_oma`  
**Test Flows:** pgtest1, pgtest3, pgtest2group

