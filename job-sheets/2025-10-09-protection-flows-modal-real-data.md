# Protection Flows Modal - Real Data Integration

**Date:** 2025-10-09  
**Status:** 🔴 Ready to Start  
**Priority:** HIGH - User wants to test today  
**Context:** Protection Flows backend is 100% ready, but GUI modal has hardcoded placeholder data

---

## 🎯 Objective

Fix the "Create Protection Flow" modal to use real data from APIs instead of hardcoded placeholders. Enable users to select from actual Protection Groups, VMs, and Repositories to create working backup flows.

---

## 📊 Current State Analysis

### Backend APIs (ALL WORKING ✅)

1. **Machine Groups (Protection Groups):**
   - `GET /api/v1/machine-groups`
   - Returns: Array of groups with `id`, `name`, `description`, `total_vms`, `enabled_vms`, `memberships`
   - Example: `{"id": "8571eb63-a2cc-11f0-b62d-020200cc0023", "name": "FirstGroup", "total_vms": 3}`

2. **VM Contexts (Individual VMs):**
   - `GET /api/v1/vm-contexts`
   - Returns: Array of VMs with `context_id`, `vm_name`, `vmware_vm_id`, `vcenter_host`, `datacenter`, `current_status`, `os_type`, `power_state`, `groups`
   - Example: `{"context_id": "ctx-Quad-Node-Red-20251006-164856", "vm_name": "Quad-Node-Red", "vcenter_host": "quad-vcenter-01.quadris.local"}`

3. **Repositories (Backup Destinations):**
   - `GET /api/v1/repositories`
   - Returns: Array of repos with `id`, `name`, `type`, `enabled`, `storage_info {total_bytes, available_bytes, used_percent, backup_count}`
   - Example: `{"id": "repo-local-1759780872", "name": "sendense-500gb-backups", "type": "local"}`

4. **Protection Flow Creation:**
   - `POST /api/v1/protection-flows`
   - Accepts: `{name, flow_type, target_type, target_id, repository_id, enabled}`
   - Backend correctly handles both `target_type: "vm"` and `target_type: "group"`

### Backend Execution Logic (VERIFIED ✅)

**File:** `/source/current/sha/services/protection_flow_service.go` (lines 307-322)

The `ProcessBackupFlow` function correctly resolves targets:
```go
switch flow.TargetType {
case "vm":
    vmContexts = []string{flow.TargetID}  // Single VM
case "group":
    groupSummary, err := s.machineGroupSvc.GetGroup(ctx, flow.TargetID)
    for _, membership := range groupSummary.Memberships {
        if membership.Enabled {
            vmContexts = append(vmContexts, membership.VMContextID)
        }
    }
```

**Result:** Backend will correctly execute backup jobs for all VMs in a group.

### Current Modal Issues (BROKEN ❌)

**File:** `/source/current/sendense-gui/components/features/protection-flows/CreateFlowModal.tsx`

1. **Lines 104-107:** Hardcoded placeholder sources
   ```typescript
   <SelectItem value="vCenter-ESXi-01">vCenter-ESXi-01</SelectItem>
   ```

2. **Lines 118-121:** Hardcoded placeholder destinations
   ```typescript
   <SelectItem value="CloudStack-Primary">CloudStack-Primary</SelectItem>
   ```

3. **Line 35:** Hardcoded `target_type: 'vm'` - should be dynamic
   ```typescript
   target_type: 'vm' as const,  // ❌ WRONG - should detect from selection
   ```

4. **No API calls** - modal doesn't fetch any real data

5. **No UX for large lists** - need search/filter for potentially hundreds of VMs

---

## 🎨 Design Requirements

### Source Dropdown (Two Sections)

**Section 1: Protection Groups** (Recommended)
```
┌────────────────────────────────────────┐
│ 🛡️  Protection Groups                  │
├────────────────────────────────────────┤
│ ├─ FirstGroup (3 VMs)                  │
│ ├─ second (5 VMs)                      │
│ └─ Production Servers (12 VMs)         │
└────────────────────────────────────────┘
```

**Section 2: Individual VMs**
```
┌────────────────────────────────────────┐
│ 🖥️  Individual VMs                     │
├────────────────────────────────────────┤
│ ├─ Quad-Node-Red                       │
│ │  quad-vcenter-01 • Running • Linux   │
│ ├─ Network_Lab                         │
│ │  quad-vcenter-01 • Running • Linux   │
│ └─ (Search for more...)                │
└────────────────────────────────────────┘
```

**Features:**
- Clear visual separation (groups vs VMs)
- Show VM count for groups
- Show vCenter host, status, OS for VMs
- Status indicators (green dot = running, gray = stopped, red = error)
- Search/filter for large lists (>10 items)
- Selected item shows: "Group: FirstGroup" or "VM: Quad-Node-Red"

### Destination Dropdown (Repositories Only for Backup)

```
┌────────────────────────────────────────┐
│ 📦 Backup Repositories                 │
├────────────────────────────────────────┤
│ ├─ sendense-500gb-backups              │
│ │  Local • 480GB free • 15 backups     │
│ ├─ local-backup-repo                   │
│ │  Local • 480GB free • 0 backups      │
└────────────────────────────────────────┘
```

**Features:**
- Show repository name
- Show type (Local, NFS, CIFS, S3, Azure)
- Show available space (human-readable)
- Show backup count
- Only show enabled repositories
- Disable if no repositories exist (with helpful message)

---

## 🔧 Implementation Plan

### Task 1: Create API Service Layer

**File:** `/source/current/sendense-gui/src/features/protection-flows/api/sourcesApi.ts`

```typescript
import axios from 'axios';

const API_BASE = '';  // Uses Next.js proxy

// Protection Group (Machine Group)
export interface ProtectionGroup {
  id: string;
  name: string;
  description: string;
  total_vms: number;
  enabled_vms: number;
  disabled_vms: number;
  created_at: string;
}

// VM Context
export interface VMContext {
  context_id: string;
  vm_name: string;
  vmware_vm_id: string;
  vm_path: string;
  vcenter_host: string;
  datacenter: string;
  current_status: string;
  os_type: string;
  power_state: string;
  cpu_count: number;
  memory_mb: number;
  groups?: Array<{
    group_id: string;
    group_name: string;
    enabled: boolean;
  }>;
}

// Repository
export interface Repository {
  id: string;
  name: string;
  type: string;  // 'local', 'nfs', 'cifs', 's3', 'azure'
  enabled: boolean;
  storage_info?: {
    total_bytes: number;
    available_bytes: number;
    used_percent: number;
    backup_count: number;
  };
}

// GET /api/v1/machine-groups
export async function listProtectionGroups(): Promise<{ groups: ProtectionGroup[]; total: number }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/machine-groups`);
  return data;
}

// GET /api/v1/vm-contexts
export async function listVMContexts(): Promise<{ vm_contexts: VMContext[] }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/vm-contexts`);
  return data;
}

// GET /api/v1/repositories
export async function listRepositories(): Promise<{ repositories: Repository[]; total: number }> {
  const { data } = await axios.get(`${API_BASE}/api/v1/repositories`);
  return data;
}
```

### Task 2: Create React Query Hooks

**File:** `/source/current/sendense-gui/src/features/protection-flows/hooks/useFlowSources.ts`

```typescript
import { useQuery } from '@tanstack/react-query';
import * as api from '../api/sourcesApi';

export function useProtectionGroups() {
  return useQuery({
    queryKey: ['protection-groups'],
    queryFn: api.listProtectionGroups,
    staleTime: 30000, // 30 seconds - groups don't change often
  });
}

export function useVMContexts() {
  return useQuery({
    queryKey: ['vm-contexts'],
    queryFn: api.listVMContexts,
    staleTime: 10000, // 10 seconds - VMs change more frequently
  });
}

export function useRepositories() {
  return useQuery({
    queryKey: ['repositories'],
    queryFn: api.listRepositories,
    staleTime: 60000, // 60 seconds - repos are relatively static
  });
}
```

### Task 3: Update CreateFlowModal Component

**File:** `/source/current/sendense-gui/components/features/protection-flows/CreateFlowModal.tsx`

**Changes Required:**

1. **Import hooks:**
```typescript
import { useProtectionGroups, useVMContexts, useRepositories } from "@/src/features/protection-flows/hooks/useFlowSources";
import { useState, useMemo } from "react";
```

2. **Fetch data:**
```typescript
const { data: groupsData, isLoading: loadingGroups } = useProtectionGroups();
const { data: vmsData, isLoading: loadingVMs } = useVMContexts();
const { data: reposData, isLoading: loadingRepos } = useRepositories();

const groups = groupsData?.groups || [];
const vms = vmsData?.vm_contexts || [];
const repos = reposData?.repositories?.filter(r => r.enabled) || [];
```

3. **Update form state:**
```typescript
const [formData, setFormData] = useState({
  name: '',
  type: 'backup' as FlowType,
  source: '',  // Will store "group:GROUP_ID" or "vm:CONTEXT_ID"
  sourceType: '' as 'group' | 'vm',  // Track selection type
  destination: '',  // Will store REPOSITORY_ID
  nextRun: '',
  description: ''
});

// Search state for large lists
const [sourceSearch, setSourceSearch] = useState('');
const [repoSearch, setRepoSearch] = useState('');
```

4. **Filter logic for search:**
```typescript
const filteredGroups = useMemo(() => {
  if (!sourceSearch) return groups;
  return groups.filter(g =>
    g.name.toLowerCase().includes(sourceSearch.toLowerCase())
  );
}, [groups, sourceSearch]);

const filteredVMs = useMemo(() => {
  if (!sourceSearch) return vms;
  return vms.filter(vm =>
    vm.vm_name.toLowerCase().includes(sourceSearch.toLowerCase()) ||
    vm.vcenter_host.toLowerCase().includes(sourceSearch.toLowerCase())
  );
}, [vms, sourceSearch]);

const filteredRepos = useMemo(() => {
  if (!repoSearch) return repos;
  return repos.filter(r =>
    r.name.toLowerCase().includes(repoSearch.toLowerCase())
  );
}, [repos, repoSearch]);
```

5. **Source dropdown with sections:**
```typescript
<div className="space-y-2">
  <Label htmlFor="source">Source</Label>
  
  {/* Search input */}
  {(groups.length + vms.length > 10) && (
    <Input
      type="search"
      placeholder="Search protection groups or VMs..."
      value={sourceSearch}
      onChange={(e) => setSourceSearch(e.target.value)}
      className="mb-2"
    />
  )}

  <Select
    value={formData.source}
    onValueChange={(value) => {
      const [type, id] = value.split(':');
      setFormData(prev => ({
        ...prev,
        source: value,
        sourceType: type as 'group' | 'vm'
      }));
    }}
    disabled={loadingGroups || loadingVMs}
  >
    <SelectTrigger>
      <SelectValue placeholder={
        loadingGroups || loadingVMs
          ? "Loading sources..."
          : "Select protection group or VM"
      } />
    </SelectTrigger>
    <SelectContent className="max-h-[400px]">
      {/* Protection Groups Section */}
      {filteredGroups.length > 0 && (
        <>
          <div className="px-2 py-1.5 text-xs font-semibold text-muted-foreground flex items-center gap-2">
            <span className="text-primary">🛡️</span>
            PROTECTION GROUPS
          </div>
          {filteredGroups.map((group) => (
            <SelectItem key={`group:${group.id}`} value={`group:${group.id}`}>
              <div className="flex flex-col">
                <span className="font-medium">{group.name}</span>
                <span className="text-xs text-muted-foreground">
                  {group.total_vms} VM{group.total_vms !== 1 ? 's' : ''}
                  {group.description && ` • ${group.description}`}
                </span>
              </div>
            </SelectItem>
          ))}
        </>
      )}

      {/* Divider if both sections have items */}
      {filteredGroups.length > 0 && filteredVMs.length > 0 && (
        <div className="h-px bg-border my-1" />
      )}

      {/* Individual VMs Section */}
      {filteredVMs.length > 0 && (
        <>
          <div className="px-2 py-1.5 text-xs font-semibold text-muted-foreground flex items-center gap-2">
            <span className="text-primary">🖥️</span>
            INDIVIDUAL VMS
          </div>
          {filteredVMs.map((vm) => (
            <SelectItem key={`vm:${vm.context_id}`} value={`vm:${vm.context_id}`}>
              <div className="flex flex-col">
                <div className="flex items-center gap-2">
                  <span className={`w-2 h-2 rounded-full ${
                    vm.power_state === 'poweredOn' ? 'bg-green-500' :
                    vm.power_state === 'poweredOff' ? 'bg-gray-400' :
                    'bg-red-500'
                  }`} />
                  <span className="font-medium">{vm.vm_name}</span>
                </div>
                <span className="text-xs text-muted-foreground">
                  {vm.vcenter_host} • {vm.power_state === 'poweredOn' ? 'Running' : 'Stopped'} • {vm.os_type}
                </span>
              </div>
            </SelectItem>
          ))}
        </>
      )}

      {/* Empty state */}
      {filteredGroups.length === 0 && filteredVMs.length === 0 && (
        <div className="px-2 py-4 text-sm text-muted-foreground text-center">
          {sourceSearch ? 'No matches found' : 'No protection groups or VMs available'}
        </div>
      )}
    </SelectContent>
  </Select>
</div>
```

6. **Destination dropdown:**
```typescript
<div className="space-y-2">
  <Label htmlFor="destination">Destination</Label>

  {/* Search input for large repo lists */}
  {repos.length > 5 && (
    <Input
      type="search"
      placeholder="Search repositories..."
      value={repoSearch}
      onChange={(e) => setRepoSearch(e.target.value)}
      className="mb-2"
    />
  )}

  <Select
    value={formData.destination}
    onValueChange={(value) => handleInputChange('destination', value)}
    disabled={loadingRepos || repos.length === 0}
  >
    <SelectTrigger>
      <SelectValue placeholder={
        loadingRepos ? "Loading repositories..." :
        repos.length === 0 ? "No repositories configured" :
        "Select backup repository"
      } />
    </SelectTrigger>
    <SelectContent className="max-h-[300px]">
      {filteredRepos.length > 0 ? (
        filteredRepos.map((repo) => (
          <SelectItem key={repo.id} value={repo.id}>
            <div className="flex flex-col">
              <span className="font-medium">{repo.name}</span>
              <span className="text-xs text-muted-foreground">
                {repo.type.toUpperCase()}
                {repo.storage_info && (
                  <>
                    {' • '}
                    {formatBytes(repo.storage_info.available_bytes)} free
                    {' • '}
                    {repo.storage_info.backup_count} backup{repo.storage_info.backup_count !== 1 ? 's' : ''}
                  </>
                )}
              </span>
            </div>
          </SelectItem>
        ))
      ) : (
        <div className="px-2 py-4 text-sm text-muted-foreground text-center">
          {repoSearch ? 'No matches found' : 'No repositories available'}
        </div>
      )}
    </SelectContent>
  </Select>

  {/* Helper message if no repos */}
  {!loadingRepos && repos.length === 0 && (
    <p className="text-xs text-muted-foreground">
      Please configure a repository first in the <a href="/repositories" className="text-primary underline">Repositories</a> page.
    </p>
  )}
</div>
```

7. **Update submit handler:**
```typescript
const handleSubmit = (e: React.FormEvent) => {
  e.preventDefault();

  // Parse source selection
  const [sourceType, sourceId] = formData.source.split(':');

  // Create flow object with correct target_type
  const newFlow = {
    name: formData.name,
    flow_type: formData.type as 'backup' | 'replication',
    target_type: sourceType as 'vm' | 'group',  // ✅ DYNAMIC now!
    target_id: sourceId,
    repository_id: formData.destination,
    enabled: true,
  };

  onCreate(newFlow as any);

  // Reset form
  setFormData({
    name: '',
    type: 'backup',
    source: '',
    sourceType: '',
    destination: '',
    nextRun: '',
    description: ''
  });
  setSourceSearch('');
  setRepoSearch('');
  onClose();
};
```

8. **Add helper function:**
```typescript
// Add at top of component
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
};
```

9. **Update disabled state:**
```typescript
<Button
  type="submit"
  onClick={handleSubmit}
  disabled={
    !formData.name ||
    !formData.source ||
    !formData.destination ||
    loadingGroups ||
    loadingVMs ||
    loadingRepos
  }
>
  Create Flow
</Button>
```

---

## 🧪 Testing Plan

### Test Case 1: Single VM Backup (pgtest1)

**Setup:**
1. Ensure `pgtest1` VM exists in VM Contexts
2. Ensure at least one repository is configured

**Steps:**
1. Open "Create Protection Flow" modal
2. Enter name: "Test pgtest1 Backup"
3. Flow Type: Backup
4. Source: Search for "pgtest1" → Select from Individual VMs section
5. Destination: Select "sendense-500gb-backups"
6. Click "Create Flow"

**Expected Result:**
- Flow created with `target_type: "vm"` and `target_id: "ctx-pgtest1-..."`
- POST request to `/api/v1/protection-flows` succeeds
- Flow appears in Protection Flows list

**Execution Test:**
1. Find the flow in the list
2. Click "Run Now" (or use Execute endpoint)
3. Backend should create ONE backup job for pgtest1
4. Check `/var/lib/sendense/backups/pgtest1/` for backup files

### Test Case 2: Group Backup (FirstGroup)

**Setup:**
1. Ensure "FirstGroup" exists with 3 VMs
2. Ensure repository is configured

**Steps:**
1. Open "Create Protection Flow" modal
2. Enter name: "Test FirstGroup Backup"
3. Flow Type: Backup
4. Source: Select "FirstGroup" from Protection Groups section (shows "3 VMs")
5. Destination: Select "sendense-500gb-backups"
6. Click "Create Flow"

**Expected Result:**
- Flow created with `target_type: "group"` and `target_id: "8571eb63-..."`
- Flow appears in list

**Execution Test:**
1. Execute the flow
2. Backend should create THREE backup jobs (one per VM in group)
3. Check execution record shows `vms_processed: 3`
4. Verify backups for all 3 VMs in group

### Test Case 3: UX Features

**Large Lists:**
1. If >10 VMs exist, search input should appear
2. Typing should filter both groups and VMs
3. Sections should hide if filtered out

**Empty States:**
1. No groups → Only show "Individual VMs" section
2. No VMs → Only show "Protection Groups" section
3. No repos → Show helpful message with link to Repositories page

**Status Indicators:**
1. Running VMs → Green dot
2. Stopped VMs → Gray dot
3. Error state → Red dot

---

## 📚 API Reference

### Next.js Proxy Configuration

Already configured in `next.config.ts`:
```typescript
async rewrites() {
  return [
    {
      source: '/api/v1/:path*',
      destination: 'http://localhost:8082/api/v1/:path*',
    },
  ];
}
```

**Usage:** Use empty string for API_BASE in all API calls.

### API Response Examples

**Machine Groups:**
```json
{
  "groups": [
    {
      "id": "8571eb63-a2cc-11f0-b62d-020200cc0023",
      "name": "FirstGroup",
      "description": "first one",
      "total_vms": 3,
      "enabled_vms": 3,
      "memberships": [...]
    }
  ],
  "total": 1
}
```

**VM Contexts:**
```json
{
  "vm_contexts": [
    {
      "context_id": "ctx-Quad-Node-Red-20251006-164856",
      "vm_name": "Quad-Node-Red",
      "vmware_vm_id": "4205f258-002d-b904-797e-c0c3deeb55c0",
      "vcenter_host": "quad-vcenter-01.quadris.local",
      "current_status": "discovered",
      "os_type": "linux",
      "power_state": "poweredOn",
      "cpu_count": 4,
      "memory_mb": 16384
    }
  ]
}
```

**Repositories:**
```json
{
  "repositories": [
    {
      "id": "repo-local-1759780872",
      "name": "sendense-500gb-backups",
      "type": "local",
      "enabled": true,
      "storage_info": {
        "total_bytes": 527295578112,
        "available_bytes": 480106450944,
        "used_percent": 8.95,
        "backup_count": 15
      }
    }
  ],
  "total": 2
}
```

---

## ✅ Completion Checklist

### Code Implementation
- [ ] Create `sourcesApi.ts` with all 3 API methods
- [ ] Create `useFlowSources.ts` with all 3 React Query hooks
- [ ] Update `CreateFlowModal.tsx` with new dropdowns
- [ ] Add search/filter functionality
- [ ] Add status indicators (green/gray/red dots)
- [ ] Add helper function `formatBytes()`
- [ ] Update form submission to extract `target_type` and `target_id`
- [ ] Add loading states
- [ ] Add empty states with helpful messages

### UX Polish
- [ ] Two-section source dropdown (groups + VMs)
- [ ] Repository dropdown with storage info
- [ ] Search inputs appear for large lists (>10 items)
- [ ] Status dots show power state
- [ ] Disabled state when no repositories exist
- [ ] Link to Repositories page if none configured

### Testing
- [ ] Test creating flow with single VM (pgtest1)
- [ ] Test creating flow with protection group (FirstGroup)
- [ ] Test executing VM flow → 1 backup job created
- [ ] Test executing group flow → N backup jobs created (one per VM)
- [ ] Test search/filter functionality
- [ ] Test empty states
- [ ] Test loading states
- [ ] Verify backend receives correct `target_type` and `target_id`
- [ ] Check browser console for no errors

### Quality Gates
- [ ] No TypeScript errors
- [ ] No console errors
- [ ] `npm run build` succeeds
- [ ] All dropdowns load real data
- [ ] Selection stores correct IDs
- [ ] Backend execution works for both VM and group targets

---

## 🚨 CRITICAL NOTES

1. **Backend is 100% Ready:** The `ProcessBackupFlow` function already handles groups correctly (lines 307-322 in `protection_flow_service.go`)

2. **Target Type Must Be Dynamic:** The current hardcoded `target_type: 'vm'` MUST be changed to extract from the source selection

3. **Use "prefix:id" Format:** Store source as `"group:GROUP_ID"` or `"vm:CONTEXT_ID"` to easily split and determine target_type

4. **Empty String for API_BASE:** All API calls must use `const API_BASE = '';` to use Next.js proxy

5. **Filter Enabled Repos Only:** Only show `enabled: true` repositories in the dropdown

6. **User Testing Today:** Priority is to get this working so user can test pgtest1 backup today

---

**Next Steps:** Hand this to Grok with clear instructions to implement all three tasks sequentially.


