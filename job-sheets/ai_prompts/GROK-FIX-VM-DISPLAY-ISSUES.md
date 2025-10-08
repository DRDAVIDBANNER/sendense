# Grok Prompt: Fix VM Display and Selection Issues

## Critical Issues Found

After the previous fixes, VMs are being discovered but:
1. **VM names don't display** - Shows blank/undefined
2. **Selecting one VM selects all VMs** - Selection logic broken
3. **VM details missing** - CPU count, datacenter not showing

## Root Cause: More Field Name Mismatches

### Backend API Response (Actual):
```json
{
  "discovered_vms": [
    {
      "id": "4205759b-fc08-fa55-bea6-7ee650028188",
      "name": "PhilB Test machine",
      "path": "/DatabanxDC/vm/PhilB Test machine",
      "power_state": "poweredOn",
      "guest_os": "windows",
      "memory_mb": 4096,
      "num_cpu": 2,
      "vmx_version": "vmx-19"
    }
  ]
}
```

### Frontend Interface (Line 30-38) - WRONG:
```typescript
interface DiscoveredVM {
  vmware_vm_id: string;  // ❌ Backend uses "id"
  vm_name: string;       // ❌ Backend uses "name"
  datacenter: string;
  power_state: 'poweredOn' | 'poweredOff' | 'suspended';
  guest_os: string;
  cpu_count: number;     // ❌ Backend uses "num_cpu"
  memory_mb: number;
}
```

## Required Fixes

### 1. Update DiscoveredVM Interface (Line 30-38)

**Replace with:**
```typescript
interface DiscoveredVM {
  id: string;                    // ✅ Match backend field name
  name: string;                  // ✅ Match backend field name
  path: string;                  // ✅ Add path field
  power_state: 'poweredOn' | 'poweredOff' | 'suspended';
  guest_os: string;
  num_cpu: number;               // ✅ Match backend field name
  memory_mb: number;
  vmx_version?: string;          // ✅ Add version field
  disks?: any[];                 // ✅ Add disk info
  networks?: any[];              // ✅ Add network info
  existing?: boolean;            // ✅ Add existing flag
}
```

### 2. Fix ALL References Throughout Component

**Search and replace these patterns:**

#### VM ID References:
- `vm.vmware_vm_id` → `vm.id` (appears ~10 times)
- `key={vm.vmware_vm_id}` → `key={vm.id}`
- `id={\`vm-\${vm.vmware_vm_id}\`}` → `id={\`vm-\${vm.id}\`}`
- `selectedVMIds.includes(vm.vmware_vm_id)` → `selectedVMIds.includes(vm.id)`

#### VM Name References:
- `vm.vm_name` → `vm.name` (appears ~5 times)
- `<span>{vm.vm_name}</span>` → `<span>{vm.name}</span>`

#### CPU Count References:
- `vm.cpu_count` → `vm.num_cpu` (appears ~3 times)
- `{vm.cpu_count} CPU` → `{vm.num_cpu} CPU`

### 3. Specific Line Fixes

**Line 372:** (Discovery preview list)
```typescript
<div key={vm.vmware_vm_id} className="flex items-center gap-3 p-2 rounded border">
```
**Change to:**
```typescript
<div key={vm.id} className="flex items-center gap-3 p-2 rounded border">
```

**Line 376:** (VM name display)
```typescript
<span className="font-medium text-sm">{vm.vm_name}</span>
```
**Change to:**
```typescript
<span className="font-medium text-sm">{vm.name}</span>
```

**Line 380:** (VM details)
```typescript
{vm.datacenter} • {vm.guest_os} • {vm.cpu_count} CPU • {vm.memory_mb} MB RAM
```
**Change to:**
```typescript
{vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM
```
Note: Remove `vm.datacenter` as it's not in the VM object

**Line 414:** (Selection list)
```typescript
<div key={vm.vmware_vm_id} className="flex items-center space-x-3 p-3 rounded-lg border hover:bg-muted/50">
```
**Change to:**
```typescript
<div key={vm.id} className="flex items-center space-x-3 p-3 rounded-lg border hover:bg-muted/50">
```

**Line 416-417:** (Checkbox)
```typescript
<Checkbox
  id={`vm-${vm.vmware_vm_id}`}
  checked={selectedVMIds.includes(vm.vmware_vm_id)}
  onCheckedChange={(checked) => handleVMSelection(vm.vmware_vm_id, checked as boolean)}
/>
```
**Change to:**
```typescript
<Checkbox
  id={`vm-${vm.id}`}
  checked={selectedVMIds.includes(vm.id)}
  onCheckedChange={(checked) => handleVMSelection(vm.id, checked as boolean)}
/>
```

**Line 424:** (VM name in selection list)
```typescript
<span className="font-medium">{vm.vm_name}</span>
```
**Change to:**
```typescript
<span className="font-medium">{vm.name}</span>
```

**Line 427:** (VM specs)
```typescript
<div className="text-xs text-muted-foreground">
  {vm.guest_os} • {vm.cpu_count} CPU • {vm.memory_mb} MB RAM
</div>
```
**Change to:**
```typescript
<div className="text-xs text-muted-foreground">
  {vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM
</div>
```

**Line 445:** (Selected VMs display)
```typescript
const vm = discoveredVMs.find(v => v.vmware_vm_id === vmId);
```
**Change to:**
```typescript
const vm = discoveredVMs.find(v => v.id === vmId);
```

**Line 449:** (Selected VM name)
```typescript
<span>{vm.vm_name}</span>
```
**Change to:**
```typescript
<span>{vm.name}</span>
```

## Testing Checklist

After fixes:

1. ✅ VM names display correctly (e.g., "PhilB Test machine", "Network_Lab")
2. ✅ Selecting one VM only selects that VM (not all VMs)
3. ✅ CPU count displays correctly (e.g., "2 CPU", "24 CPU")
4. ✅ Memory displays correctly (e.g., "4096 MB RAM")
5. ✅ Power state badges show (Running/Stopped)
6. ✅ Selection count updates correctly
7. ✅ "Add to Management" button works with selected VMs

## Summary of Changes

Replace **ALL** occurrences of:
- `vmware_vm_id` → `id`
- `vm_name` → `name`
- `cpu_count` → `num_cpu`

This affects approximately **15-20 locations** in the file.

---

**Priority:** CRITICAL - Blocks VM selection workflow  
**Difficulty:** LOW - Simple find/replace operations  
**File:** `/home/oma_admin/sendense/source/current/sendense-gui/components/features/protection-groups/VMDiscoveryModal.tsx`
