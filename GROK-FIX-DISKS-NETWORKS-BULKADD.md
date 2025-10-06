# 🚨 CRITICAL GUI FIXES - VM Discovery Missing Disks/Networks + Bulk Add Broken

## Problem Summary

**User Report:**
1. ✅ VM names now working
2. ❌ **NOT showing disks/sizes** - Backend sends this data, GUI ignores it
3. ❌ **NOT showing networks** - Backend sends this data, GUI ignores it  
4. ❌ **Bulk add failing** - Frontend payload doesn't match backend expectations

## Analysis

### Issue #1: Disks Not Displayed

**Backend sends (enhanced_discovery.go line 247-248):**
```go
Disks:      vm.Disks,      // Include disk information
Networks:   vm.Networks,   // Include network information
```

**Backend disk structure (enhanced_discovery_service.go lines 69-77):**
```go
type VMADiskInfo struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Path          string `json:"path"`
	SizeGB        int    `json:"size_gb"`
	CapacityBytes int64  `json:"capacity_bytes"`
	Datastore     string `json:"datastore"`
}
```

**Backend network structure (enhanced_discovery_service.go lines 79-84):**
```go
type VMANetworkInfo struct {
	Label       string `json:"label"`
	NetworkName string `json:"network_name"`
	MACAddress  string `json:"mac_address"`
}
```

**Frontend interface has the fields (VMDiscoveryModal.tsx lines 39-40):**
```typescript
disks?: any[];     // ✅ Defined
networks?: any[];  // ✅ Defined
```

**BUT GUI doesn't render them!**

Current VM display (lines 383-385 and 431-433):
```typescript
<div className="text-xs text-muted-foreground">
  {vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM
</div>
```

**Missing:** Disk count, total disk size, network count

### Issue #2: Bulk Add API Mismatch

**Frontend sends (VMDiscoveryModal.tsx lines 168-177):**
```typescript
await fetch('/api/v1/discovery/bulk-add', {
  method: 'POST',
  body: JSON.stringify({
    vm_ids: selectedVMIds,        // ❌ WRONG FIELD NAME
    credential_id: selectedCredentialId // ❌ WRONG PAYLOAD STRUCTURE
  }),
});
```

**Backend expects (enhanced_discovery.go lines 83-91):**
```go
type BulkAddVMsRequest struct {
	VCenter     string   `json:"vcenter" binding:"required"`
	Username    string   `json:"username" binding:"required"`
	Password    string   `json:"password" binding:"required"`
	Datacenter  string   `json:"datacenter" binding:"required"`
	Filter      string   `json:"filter,omitempty"`
	SelectedVMs []string `json:"selected_vms" binding:"required"` // ❌ NOT "vm_ids"
}
```

**Problem:** `bulk-add` endpoint DOESN'T support `credential_id` - it requires manual credentials!

**CORRECT endpoint to use:** `POST /api/v1/discovery/add-vms` (lines 347-466)

This endpoint properly handles `credential_id`:
```go
type AddVMsRequest struct {
	CredentialID *int     `json:"credential_id,omitempty"` // ✅ Supports credential_id!
	VCenter      string   `json:"vcenter,omitempty"`
	Username     string   `json:"username,omitempty"`
	Password     string   `json:"password,omitempty"`
	Datacenter   string   `json:"datacenter,omitempty"`
	VMNames      []string `json:"vm_names" binding:"required,min=1"` // ✅ VM names array
	AddedBy      string   `json:"added_by,omitempty"`
}
```

## Required Fixes

### Fix #1: Add Disk & Network Display

**File:** `source/current/sendense-gui/components/features/protection-groups/VMDiscoveryModal.tsx`

**Location 1: Step 2 VM Preview (around lines 383-386)**

BEFORE:
```typescript
<div className="text-xs text-muted-foreground">
  {vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM
</div>
```

AFTER:
```typescript
<div className="text-xs text-muted-foreground space-y-0.5">
  <div>{vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM</div>
  {vm.disks && vm.disks.length > 0 && (
    <div>
      💾 {vm.disks.length} disk{vm.disks.length !== 1 ? 's' : ''} 
      ({vm.disks.reduce((total: number, disk: any) => total + (disk.size_gb || 0), 0)} GB total)
    </div>
  )}
  {vm.networks && vm.networks.length > 0 && (
    <div>🌐 {vm.networks.length} network{vm.networks.length !== 1 ? 's' : ''}</div>
  )}
</div>
```

**Location 2: Step 3 VM Selection (around lines 431-434)**

BEFORE:
```typescript
<div className="text-sm text-muted-foreground">
  {vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM
</div>
```

AFTER:
```typescript
<div className="text-sm text-muted-foreground space-y-0.5">
  <div>{vm.guest_os} • {vm.num_cpu} CPU • {vm.memory_mb} MB RAM</div>
  {vm.disks && vm.disks.length > 0 && (
    <div className="text-xs">
      💾 {vm.disks.length} disk{vm.disks.length !== 1 ? 's' : ''} 
      ({vm.disks.reduce((total: number, disk: any) => total + (disk.size_gb || 0), 0)} GB total)
    </div>
  )}
  {vm.networks && vm.networks.length > 0 && (
    <div className="text-xs">
      🌐 {vm.networks.length} network{vm.networks.length !== 1 ? 's' : ''} 
      {vm.networks.map((net: any, i: number) => ` ${net.network_name}`).join(',')}
    </div>
  )}
</div>
```

### Fix #2: Change Bulk Add Endpoint

**File:** `source/current/sendense-gui/components/features/protection-groups/VMDiscoveryModal.tsx`

**Location: addVMsToManagement function (lines 161-193)**

BEFORE:
```typescript
const addVMsToManagement = async () => {
  if (selectedVMIds.length === 0) return;

  setIsAddingVMs(true);
  setError(null);

  try {
    const response = await fetch('/api/v1/discovery/bulk-add', {  // ❌ WRONG ENDPOINT
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        vm_ids: selectedVMIds,              // ❌ WRONG FIELD
        credential_id: selectedCredentialId // ❌ Wrong structure for this endpoint
      }),
    });

    if (response.ok) {
      onDiscoveryComplete();
      onClose();
      resetModal();
    } else {
      const errorResult = await response.json();
      setError(errorResult.message || 'Failed to add VMs to management');
    }
  } catch (err) {
    setError('Failed to add VMs to management');
    console.error('Error adding VMs:', err);
  } finally {
    setIsAddingVMs(false);
  }
};
```

AFTER:
```typescript
const addVMsToManagement = async () => {
  if (selectedVMIds.length === 0) return;

  setIsAddingVMs(true);
  setError(null);

  try {
    // Get VM names from IDs
    const selectedVMNames = selectedVMIds
      .map(id => discoveredVMs.find(vm => vm.id === id)?.name)
      .filter((name): name is string => !!name);

    const response = await fetch('/api/v1/discovery/add-vms', {  // ✅ CORRECT ENDPOINT
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        credential_id: selectedCredentialId, // ✅ Supported by this endpoint
        vm_names: selectedVMNames,           // ✅ Correct field name
        added_by: 'gui-user'                 // ✅ Track who added these VMs
      }),
    });

    if (response.ok) {
      const result = await response.json();
      console.log(`Successfully added ${result.vms_added} VMs to management`);
      onDiscoveryComplete();
      onClose();
      resetModal();
    } else {
      const errorResult = await response.json();
      setError(errorResult.error || errorResult.message || 'Failed to add VMs to management');
    }
  } catch (err) {
    setError('Failed to add VMs to management');
    console.error('Error adding VMs:', err);
  } finally {
    setIsAddingVMs(false);
  }
};
```

## Summary of Changes

### 1. Disk/Network Display (2 locations)
- **Step 2 Preview**: Show disk count + total GB, network count
- **Step 3 Selection**: Show disk count + total GB, network names

### 2. Bulk Add Fix (1 function)
- **Change endpoint**: `/api/v1/discovery/bulk-add` → `/api/v1/discovery/add-vms`
- **Change payload**: 
  - `vm_ids` → `vm_names` (convert IDs to names)
  - Keep `credential_id` (properly supported)
  - Add `added_by` field

## Expected Results

✅ **Disks visible:** "💾 3 disks (500 GB total)"  
✅ **Networks visible:** "🌐 2 networks VM Network, Production"  
✅ **Bulk add working:** VMs properly added to vm_replication_contexts table  
✅ **Proper error messages:** Backend validation errors displayed correctly

## Backend Endpoints Reference

```
POST /api/v1/discovery/discover-vms     - Discovery with optional context creation
POST /api/v1/discovery/add-vms          - Add specific VMs (supports credential_id) ✅ USE THIS
POST /api/v1/discovery/bulk-add         - Legacy bulk add (requires manual creds) ❌ DON'T USE
```

The `/add-vms` endpoint:
- Accepts `credential_id` or manual credentials
- Returns detailed success/failure per VM
- Properly creates vm_replication_contexts entries
- Includes audit trail with `added_by` field
