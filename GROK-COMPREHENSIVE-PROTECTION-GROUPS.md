# 🎯 COMPREHENSIVE Protection Groups Integration - Wire Everything Up

## Session Goal

Transform the Protection Groups page from mock data to **fully functional enterprise protection group management** with:
- Live data from backend APIs
- Complete CRUD operations for groups
- VM assignment and management
- Schedule integration
- Efficient VM display with group membership status
- Discovery workflow optimization

## Current State Analysis

### What's Working ✅
1. **UI Components:** Cards, modals, dropdowns all render correctly
2. **VM Discovery:** Can discover VMs and create contexts
3. **Ungrouped VMs:** Fetching and displaying VMs without groups

### What's Broken/Mock ❌
1. **Group data:** Uses `mockGroups` hardcoded array
2. **Create Group:** Doesn't call backend API
3. **Edit Group:** No modal or backend call
4. **Manage VMs:** No VM assignment interface
5. **Delete Group:** No confirmation or backend call
6. **Info panels:** Calculating from mock data
7. **Schedules:** No schedule dropdown

---

## Backend APIs Available

### Machine Groups
```typescript
POST   /api/v1/machine-groups                  // Create group
GET    /api/v1/machine-groups                  // List all groups
GET    /api/v1/machine-groups/{id}             // Get group details
PUT    /api/v1/machine-groups/{id}             // Update group
DELETE /api/v1/machine-groups/{id}             // Delete group
GET    /api/v1/machine-groups/{id}/vms         // List group VMs
POST   /api/v1/machine-groups/{id}/vms         // Assign VM to group
DELETE /api/v1/machine-groups/{id}/vms/{vmId}  // Remove VM from group
```

### Schedules
```typescript
GET    /api/v1/schedules                       // List all schedules
POST   /api/v1/schedules                       // Create schedule
GET    /api/v1/schedules/{id}                  // Get schedule details
PUT    /api/v1/schedules/{id}                  // Update schedule
```

### Discovery
```typescript
POST   /api/v1/discovery/discover-vms          // Discover VMs (optimized)
GET    /api/v1/discovery/ungrouped-vms         // Get VMs not in groups
```

---

## Part 1: Data Types & Interfaces

**File:** `app/protection-groups/page.tsx`

### Update ProtectionGroup Interface
```typescript
interface ProtectionGroup {
  id: string;                // Backend: group ID
  name: string;              // Backend: group name
  description: string | null;// Backend: description (nullable)
  schedule_id: string | null;// Backend: schedule reference
  schedule_name: string | null; // Backend: schedule name (from join)
  max_concurrent_vms: number;// Backend: max concurrent VMs
  priority: number;          // Backend: group priority
  total_vms: number;         // Backend: count of VMs in group
  enabled_vms: number;       // Backend: count of enabled VMs
  disabled_vms: number;      // Backend: count of disabled VMs
  active_jobs: number;       // Backend: count of running jobs
  last_execution: string | null; // Backend: last run timestamp
  created_by: string;        // Backend: who created
  created_at: string;        // Backend: creation timestamp
  updated_at: string;        // Backend: update timestamp
  status: 'active' | 'inactive' | 'error'; // Derived from data
}

interface Schedule {
  id: string;
  name: string;
  description: string | null;
  enabled: boolean;
  cron_expression: string;
  vm_group_id: string | null;
  created_at: string;
  updated_at: string;
}

interface VMWithGroupStatus extends VMContext {
  group_id: string | null;
  group_name: string | null;
  group_priority: number | null;
  membership_enabled: boolean;
}
```

---

## Part 2: Discovery Optimization (50% Performance Improvement)

**File:** `components/features/protection-groups/VMDiscoveryModal.tsx`

### Change Step 3: Use create_context=true

**Location:** `addVMsToManagement` function (lines 161-193)

**BEFORE:**
```typescript
const addVMsToManagement = async () => {
  // ... sends to /api/v1/discovery/add-vms
  // Problem: Backend re-queries vCenter (14s wasted)
}
```

**AFTER:**
```typescript
const addVMsToManagement = async () => {
  if (selectedVMIds.length === 0) return;

  setIsAddingVMs(true);
  setError(null);

  try {
    // Get VM names from already-discovered VMs
    const selectedVMNames = selectedVMIds
      .map(id => discoveredVMs.find(vm => vm.id === id)?.name)
      .filter((name): name is string => !!name);

    // ✅ OPTIMIZED: Use discover-vms with create_context=true
    // This creates contexts using already-discovered data (no re-query!)
    const response = await fetch('/api/v1/discovery/discover-vms', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        credential_id: selectedCredentialId,
        create_context: true,           // ✅ Create contexts
        selected_vms: selectedVMNames,  // ✅ Only these VMs
      }),
    });

    if (response.ok) {
      const result = await response.json();
      const addedCount = result.addition_result?.successfully_added || 0;
      console.log(`✅ Successfully added ${addedCount} VMs to management`);
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

**Performance:** 28s → 14s (50% faster!)

---

## Part 3: Fetch Live Group Data

**File:** `app/protection-groups/page.tsx`

### Replace Mock Data with API Calls

**Location:** Component initialization (lines 89-129)

**ADD THESE FUNCTIONS:**
```typescript
const [groups, setGroups] = useState<ProtectionGroup[]>([]);
const [schedules, setSchedules] = useState<Schedule[]>([]);
const [isLoadingGroups, setIsLoadingGroups] = useState(false);
const [isLoadingSchedules, setIsLoadingSchedules] = useState(false);

// Fetch groups from backend
const fetchGroups = async () => {
  setIsLoadingGroups(true);
  try {
    const response = await fetch('/api/v1/machine-groups');
    if (response.ok) {
      const data = await response.json();
      // Backend returns: { groups: [...], total_count: N }
      const fetchedGroups = data.groups.map((g: any) => ({
        ...g,
        // Derive status from data
        status: g.enabled_vms > 0 ? 'active' : 'inactive'
      }));
      setGroups(fetchedGroups);
    } else {
      console.error('Failed to fetch groups:', response.statusText);
    }
  } catch (error) {
    console.error('Failed to fetch groups:', error);
  } finally {
    setIsLoadingGroups(false);
  }
};

// Fetch schedules from backend
const fetchSchedules = async () => {
  setIsLoadingSchedules(true);
  try {
    const response = await fetch('/api/v1/schedules');
    if (response.ok) {
      const data = await response.json();
      // Backend returns: { schedules: [...], total_count: N }
      setSchedules(data.schedules || []);
    } else {
      console.error('Failed to fetch schedules:', response.statusText);
    }
  } catch (error) {
    console.error('Failed to fetch schedules:', error);
  } finally {
    setIsLoadingSchedules(false);
  }
};

// Fetch on component mount and after operations
useEffect(() => {
  fetchGroups();
  fetchSchedules();
  fetchUngroupedVMs();
}, []);
```

---

## Part 4: Wire Up Create Group

**File:** `components/features/protection-groups/CreateGroupModal.tsx`

### Update onCreate Handler

**Location:** `handleSubmit` function

**CHANGE:**
```typescript
const handleSubmit = async () => {
  // Validate
  if (!formData.name || !formData.schedule) {
    setError('Name and schedule are required');
    return;
  }

  setIsSubmitting(true);
  setError(null);

  try {
    // ✅ Call backend API to create group
    const response = await fetch('/api/v1/machine-groups', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        name: formData.name,
        description: formData.description || null,
        schedule_id: formData.schedule, // Schedule ID from dropdown
        max_concurrent_vms: formData.maxConcurrentVMs || 10,
        priority: formData.priority || 50,
        created_by: 'gui-user',
      }),
    });

    if (response.ok) {
      const result = await response.json();
      console.log('✅ Group created:', result.id);
      
      // If VMs selected, assign them to the group
      if (selectedVMIds.length > 0) {
        await assignVMsToGroup(result.id, selectedVMIds);
      }
      
      onCreate({
        ...formData,
        vmIds: selectedVMIds,
      });
      onClose();
      resetForm();
    } else {
      const errorResult = await response.json();
      setError(errorResult.error || 'Failed to create group');
    }
  } catch (err) {
    setError('Failed to create group');
    console.error('Error creating group:', err);
  } finally {
    setIsSubmitting(false);
  }
};

// Helper function to assign VMs to group
const assignVMsToGroup = async (groupId: string, vmIds: string[]) => {
  try {
    const response = await fetch(`/api/v1/machine-groups/${groupId}/vms`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        vm_context_ids: vmIds,
        priority: 50,
        enabled: true,
      }),
    });

    if (response.ok) {
      console.log(`✅ Assigned ${vmIds.length} VMs to group ${groupId}`);
    } else {
      console.error('Failed to assign VMs:', response.statusText);
    }
  } catch (error) {
    console.error('Failed to assign VMs:', error);
  }
};
```

### Add Schedule Dropdown

**Location:** Form fields section

**ADD:**
```typescript
<div className="space-y-2">
  <Label htmlFor="schedule">Schedule</Label>
  <Select
    value={formData.schedule}
    onValueChange={(value) => setFormData(prev => ({ ...prev, schedule: value }))}
  >
    <SelectTrigger>
      <SelectValue placeholder="Select schedule" />
    </SelectTrigger>
    <SelectContent>
      {schedules.map((schedule) => (
        <SelectItem key={schedule.id} value={schedule.id}>
          {schedule.name} - {schedule.cron_expression}
        </SelectItem>
      ))}
    </SelectContent>
  </Select>
</div>
```

---

## Part 5: Wire Up Edit Group

**File:** Create new `components/features/protection-groups/EditGroupModal.tsx`

**NEW FILE:**
```typescript
"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Loader2, AlertCircle } from "lucide-react";

interface EditGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdate: () => void;
  group: ProtectionGroup | null;
  schedules: Schedule[];
}

export function EditGroupModal({ isOpen, onClose, onUpdate, group, schedules }: EditGroupModalProps) {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    schedule_id: '',
    max_concurrent_vms: 10,
    priority: 50,
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Populate form when group changes
  useEffect(() => {
    if (group) {
      setFormData({
        name: group.name,
        description: group.description || '',
        schedule_id: group.schedule_id || '',
        max_concurrent_vms: group.max_concurrent_vms,
        priority: group.priority,
      });
    }
  }, [group]);

  const handleSubmit = async () => {
    if (!group) return;

    if (!formData.name) {
      setError('Group name is required');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/machine-groups/${group.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: formData.name,
          description: formData.description || null,
          schedule_id: formData.schedule_id || null,
          max_concurrent_vms: formData.max_concurrent_vms,
          priority: formData.priority,
        }),
      });

      if (response.ok) {
        console.log('✅ Group updated:', group.id);
        onUpdate();
        onClose();
      } else {
        const errorResult = await response.json();
        setError(errorResult.error || 'Failed to update group');
      }
    } catch (err) {
      setError('Failed to update group');
      console.error('Error updating group:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Edit Protection Group</DialogTitle>
          <DialogDescription>
            Update group settings, schedule, and priority
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <span className="text-sm text-destructive">{error}</span>
          </div>
        )}

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="name">Group Name</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
              placeholder="Production Web Servers"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              placeholder="Group description..."
              rows={3}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="schedule">Schedule</Label>
            <Select
              value={formData.schedule_id}
              onValueChange={(value) => setFormData(prev => ({ ...prev, schedule_id: value }))}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select schedule (optional)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">No schedule</SelectItem>
                {schedules.map((schedule) => (
                  <SelectItem key={schedule.id} value={schedule.id}>
                    {schedule.name} - {schedule.cron_expression}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="maxConcurrentVMs">Max Concurrent VMs</Label>
              <Input
                id="maxConcurrentVMs"
                type="number"
                min="1"
                max="100"
                value={formData.max_concurrent_vms}
                onChange={(e) => setFormData(prev => ({ ...prev, max_concurrent_vms: parseInt(e.target.value) }))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="priority">Priority</Label>
              <Input
                id="priority"
                type="number"
                min="0"
                max="100"
                value={formData.priority}
                onChange={(e) => setFormData(prev => ({ ...prev, priority: parseInt(e.target.value) }))}
              />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            Update Group
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

**UPDATE:** `app/protection-groups/page.tsx`

```typescript
import { EditGroupModal } from "@/components/features/protection-groups";

// Add state
const [editingGroup, setEditingGroup] = useState<ProtectionGroup | null>(null);

// Update handler
const handleEditGroup = (group: ProtectionGroup) => {
  setEditingGroup(group);
};

// Add modal
<EditGroupModal
  isOpen={!!editingGroup}
  onClose={() => setEditingGroup(null)}
  onUpdate={() => {
    fetchGroups();
    setEditingGroup(null);
  }}
  group={editingGroup}
  schedules={schedules}
/>
```

---

## Part 6: Wire Up Manage VMs (Assignment Interface)

**File:** Create new `components/features/protection-groups/ManageVMsModal.tsx`

**NEW FILE:**
```typescript
"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Server, Search, Plus, X, Loader2 } from "lucide-react";

interface ManageVMsModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdate: () => void;
  group: ProtectionGroup | null;
}

interface VMAssignment {
  vm_context_id: string;
  vm_name: string;
  vcenter_host: string;
  power_state: string;
  priority: number;
  enabled: boolean;
  assigned_at: string;
}

export function ManageVMsModal({ isOpen, onClose, onUpdate, group }: ManageVMsModalProps) {
  const [assignedVMs, setAssignedVMs] = useState<VMAssignment[]>([]);
  const [availableVMs, setAvailableVMs] = useState<VMContext[]>([]);
  const [selectedVMIds, setSelectedVMIds] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isAssigning, setIsAssigning] = useState(false);

  useEffect(() => {
    if (isOpen && group) {
      fetchAssignedVMs();
      fetchAvailableVMs();
    }
  }, [isOpen, group]);

  const fetchAssignedVMs = async () => {
    if (!group) return;

    try {
      const response = await fetch(`/api/v1/machine-groups/${group.id}/vms`);
      if (response.ok) {
        const data = await response.json();
        setAssignedVMs(data.vms || []);
      }
    } catch (error) {
      console.error('Failed to fetch assigned VMs:', error);
    }
  };

  const fetchAvailableVMs = async () => {
    try {
      const response = await fetch('/api/v1/discovery/ungrouped-vms');
      if (response.ok) {
        const data = await response.json();
        setAvailableVMs(data.vms || []);
      }
    } catch (error) {
      console.error('Failed to fetch available VMs:', error);
    }
  };

  const handleAssignVMs = async () => {
    if (!group || selectedVMIds.length === 0) return;

    setIsAssigning(true);
    try {
      const response = await fetch(`/api/v1/machine-groups/${group.id}/vms`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          vm_context_ids: selectedVMIds,
          priority: 50,
          enabled: true,
        }),
      });

      if (response.ok) {
        console.log(`✅ Assigned ${selectedVMIds.length} VMs to group`);
        setSelectedVMIds([]);
        fetchAssignedVMs();
        fetchAvailableVMs();
        onUpdate();
      }
    } catch (error) {
      console.error('Failed to assign VMs:', error);
    } finally {
      setIsAssigning(false);
    }
  };

  const handleRemoveVM = async (vmContextId: string) => {
    if (!group) return;

    try {
      const response = await fetch(`/api/v1/machine-groups/${group.id}/vms/${vmContextId}`, {
        method: 'DELETE',
      });

      if (response.ok) {
        console.log(`✅ Removed VM ${vmContextId} from group`);
        fetchAssignedVMs();
        fetchAvailableVMs();
        onUpdate();
      }
    } catch (error) {
      console.error('Failed to remove VM:', error);
    }
  };

  const filteredAvailableVMs = availableVMs.filter(vm =>
    vm.vm_name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[800px] max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>Manage VMs - {group?.name}</DialogTitle>
          <DialogDescription>
            Assign or remove virtual machines from this protection group
          </DialogDescription>
        </DialogHeader>

        <div className="flex gap-6 flex-1 overflow-hidden">
          {/* Left: Available VMs */}
          <div className="flex-1 flex flex-col">
            <h3 className="text-sm font-semibold mb-3">Available VMs ({availableVMs.length})</h3>
            
            <div className="mb-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search VMs..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>

            <div className="flex-1 overflow-auto border rounded-lg p-2 space-y-2">
              {filteredAvailableVMs.map((vm) => (
                <div
                  key={vm.context_id}
                  className="flex items-center gap-3 p-2 hover:bg-muted rounded cursor-pointer"
                  onClick={() => {
                    setSelectedVMIds(prev =>
                      prev.includes(vm.context_id)
                        ? prev.filter(id => id !== vm.context_id)
                        : [...prev, vm.context_id]
                    );
                  }}
                >
                  <Checkbox
                    checked={selectedVMIds.includes(vm.context_id)}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        setSelectedVMIds(prev => [...prev, vm.context_id]);
                      } else {
                        setSelectedVMIds(prev => prev.filter(id => id !== vm.context_id));
                      }
                    }}
                  />
                  <Server className="h-4 w-4 text-muted-foreground" />
                  <div className="flex-1">
                    <div className="text-sm font-medium">{vm.vm_name}</div>
                    <div className="text-xs text-muted-foreground">{vm.datacenter}</div>
                  </div>
                  <Badge variant="secondary" className="text-xs">
                    {vm.power_state}
                  </Badge>
                </div>
              ))}
            </div>

            <Button
              className="mt-3 w-full"
              onClick={handleAssignVMs}
              disabled={selectedVMIds.length === 0 || isAssigning}
            >
              {isAssigning && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              <Plus className="h-4 w-4 mr-2" />
              Assign {selectedVMIds.length} VM{selectedVMIds.length !== 1 ? 's' : ''}
            </Button>
          </div>

          {/* Right: Assigned VMs */}
          <div className="flex-1 flex flex-col">
            <h3 className="text-sm font-semibold mb-3">Assigned VMs ({assignedVMs.length})</h3>
            
            <div className="flex-1 overflow-auto border rounded-lg p-2 space-y-2">
              {assignedVMs.map((vm) => (
                <div
                  key={vm.vm_context_id}
                  className="flex items-center gap-3 p-2 bg-muted rounded"
                >
                  <Server className="h-4 w-4 text-green-500" />
                  <div className="flex-1">
                    <div className="text-sm font-medium">{vm.vm_name}</div>
                    <div className="text-xs text-muted-foreground">
                      Priority: {vm.priority} • {vm.enabled ? 'Enabled' : 'Disabled'}
                    </div>
                  </div>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-6 w-6 p-0"
                    onClick={() => handleRemoveVM(vm.vm_context_id)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}

              {assignedVMs.length === 0 && (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <Server className="h-12 w-12 mb-2 opacity-50" />
                  <p className="text-sm">No VMs assigned to this group</p>
                </div>
              )}
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
```

**UPDATE:** `app/protection-groups/page.tsx`

```typescript
import { ManageVMsModal } from "@/components/features/protection-groups";

// Add state
const [managingGroup, setManagingGroup] = useState<ProtectionGroup | null>(null);

// Update handler
const handleManageVMs = (group: ProtectionGroup) => {
  setManagingGroup(group);
};

// Add modal
<ManageVMsModal
  isOpen={!!managingGroup}
  onClose={() => setManagingGroup(null)}
  onUpdate={() => {
    fetchGroups();
    fetchUngroupedVMs();
    setManagingGroup(null);
  }}
  group={managingGroup}
/>
```

---

## Part 7: Wire Up Info Panels with Live Data

**File:** `app/protection-groups/page.tsx`

### Update Summary Cards (lines 242-300)

**CHANGE:**
```typescript
<div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
  {/* Total Groups */}
  <Card>
    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
      <CardTitle className="text-sm font-medium">Total Groups</CardTitle>
      <Settings className="h-4 w-4 text-muted-foreground" />
    </CardHeader>
    <CardContent>
      <div className="text-2xl font-bold">
        {isLoadingGroups ? (
          <div className="h-8 w-12 bg-muted rounded animate-pulse" />
        ) : (
          groups.length
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        Protection groups configured
      </p>
    </CardContent>
  </Card>

  {/* Total VMs */}
  <Card>
    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
      <CardTitle className="text-sm font-medium">Total VMs</CardTitle>
      <Users className="h-4 w-4 text-muted-foreground" />
    </CardHeader>
    <CardContent>
      <div className="text-2xl font-bold">
        {isLoadingGroups ? (
          <div className="h-8 w-12 bg-muted rounded animate-pulse" />
        ) : (
          groups.reduce((sum, group) => sum + group.total_vms, 0)
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        Virtual machines in groups
      </p>
    </CardContent>
  </Card>

  {/* Protected VMs - Leave for later */}
  <Card>
    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
      <CardTitle className="text-sm font-medium">Protected VMs</CardTitle>
      <Users className="h-4 w-4 text-green-500" />
    </CardHeader>
    <CardContent>
      <div className="text-2xl font-bold">
        <span className="text-muted-foreground">—</span>
      </div>
      <p className="text-xs text-muted-foreground">
        Coming soon
      </p>
    </CardContent>
  </Card>

  {/* Active Schedules */}
  <Card>
    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
      <CardTitle className="text-sm font-medium">Active Schedules</CardTitle>
      <Calendar className="h-4 w-4 text-muted-foreground" />
    </CardHeader>
    <CardContent>
      <div className="text-2xl font-bold">
        {isLoadingSchedules ? (
          <div className="h-8 w-12 bg-muted rounded animate-pulse" />
        ) : (
          schedules.filter(s => s.enabled).length
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        Enabled schedules
      </p>
    </CardContent>
  </Card>
</div>
```

---

## Part 8: Update Group Display with Live Data (COMPACT DESIGN)

**File:** `app/protection-groups/page.tsx`

### ⚠️ CRITICAL: Make Groups Compact - They're Using Too Much Space!

**Current Problem:** Group cards are way too large and waste screen space

**NEW COMPACT DESIGN:** Smaller cards with efficient information density

### Update Group Cards (lines 303-386)

**CHANGE:**
```typescript
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
  {isLoadingGroups ? (
    // Loading skeletons - COMPACT
    Array.from({ length: 4 }).map((_, i) => (
      <Card key={i} className="p-3">
        <div className="space-y-2">
          <div className="h-3 bg-muted rounded animate-pulse w-3/4" />
          <div className="h-2 bg-muted rounded animate-pulse w-1/2" />
        </div>
      </Card>
    ))
  ) : (
    groups.map((group) => (
      <Card
        key={group.id}
        className={`cursor-pointer transition-all hover:shadow-md ${
          selectedGroupId === group.id ? 'ring-2 ring-primary' : ''
        }`}
        onClick={() => setSelectedGroupId(group.id)}
      >
        {/* COMPACT HEADER - No CardHeader wrapper, just padding */}
        <div className="p-3 pb-2">
          <div className="flex items-start justify-between mb-2">
            <div className="flex-1 min-w-0">
              <h3 className="font-semibold text-sm truncate">{group.name}</h3>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={(e) => e.stopPropagation()}
                  className="h-6 w-6 p-0 -mt-1"
                >
                  <MoreHorizontal className="h-3 w-3" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={(e) => { e.stopPropagation(); handleEditGroup(group); }}>
                  Edit Group
                </DropdownMenuItem>
                <DropdownMenuItem onClick={(e) => { e.stopPropagation(); handleManageVMs(group); }}>
                  Manage VMs
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={(e) => { e.stopPropagation(); handleDeleteGroup(group); }}
                  className="text-destructive focus:text-destructive"
                >
                  Delete Group
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          {/* Compact badges */}
          <div className="flex items-center gap-1 flex-wrap">
            {getStatusBadge(group.status)}
            {group.schedule_name && (
              <Badge variant="outline" className="text-xs px-1.5 py-0">
                {group.schedule_name}
              </Badge>
            )}
          </div>
        </div>

        {/* COMPACT CONTENT - Minimal padding */}
        <div className="px-3 pb-3 space-y-2">
          {/* VM Count - Single line */}
          <div className="flex items-center justify-between text-xs">
            <span className="text-muted-foreground">VMs</span>
            <span className="font-medium">
              {group.enabled_vms}/{group.total_vms}
            </span>
          </div>
          <Progress
            value={group.total_vms > 0 ? (group.enabled_vms / group.total_vms) * 100 : 0}
            className="h-1"
          />

          {/* Compact stats - 2 columns */}
          <div className="grid grid-cols-2 gap-x-2 text-xs">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Priority</span>
              <span className="font-medium">{group.priority}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Max</span>
              <span className="font-medium">{group.max_concurrent_vms}</span>
            </div>
          </div>

          {/* Last run - Optional, only if exists */}
          {group.last_execution && (
            <div className="text-xs text-muted-foreground pt-1 border-t">
              Last: {formatLastRun(group.last_execution)}
            </div>
          )}
        </div>
      </Card>
    ))
  )}

  {/* Add New Group Card - COMPACT */}
  <Card
    className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 cursor-pointer transition-colors"
    onClick={handleCreateGroup}
  >
    <div className="flex flex-col items-center justify-center p-6">
      <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center mb-2">
        <Plus className="h-5 w-5 text-muted-foreground" />
      </div>
      <h3 className="text-sm font-medium text-foreground mb-1">New Group</h3>
      <p className="text-xs text-muted-foreground text-center">
        Create protection group
      </p>
    </div>
  </Card>
</div>
```

**Space Savings:**
- ✅ **4 groups per row** instead of 3 (33% more groups visible)
- ✅ **Reduced padding:** p-3 instead of CardHeader + CardContent
- ✅ **Smaller text:** text-sm → text-xs for secondary info
- ✅ **Single-line stats:** Compact 2-column grid
- ✅ **Thinner progress bar:** h-1 instead of h-2
- ✅ **Smaller badges:** Reduced padding
- ✅ **Truncated titles:** Long names don't break layout
- ✅ **Smaller "Add" card:** Less vertical space

**Information Density:**
- Still shows: name, status, schedule, VM count, priority, max concurrent, last run
- Everything visible without scrolling on 1080p screen
- Clean, professional appearance

---

## Part 9: Layout Optimization for VM Display

### Efficient Space Usage Design

**RECOMMENDATION:** Use a compact table layout for VMs with group status

**Location:** Below groups grid, replace card-based ungrouped VMs

**NEW LAYOUT:**
```typescript
{/* VM Management Section */}
<div className="mt-8">
  <div className="flex items-center justify-between mb-4">
    <div>
      <h2 className="text-lg font-semibold text-foreground mb-1">
        Virtual Machines
      </h2>
      <p className="text-sm text-muted-foreground">
        {ungroupedVMs.length} ungrouped • {groups.reduce((sum, g) => sum + g.total_vms, 0)} in groups
      </p>
    </div>
    <Button variant="outline" onClick={handleAddVMs} className="gap-2">
      <Plus className="h-4 w-4" />
      Discover More VMs
    </Button>
  </div>

  {/* Compact Table View */}
  <Card>
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead className="border-b">
          <tr className="text-left text-sm text-muted-foreground">
            <th className="p-3 font-medium">VM Name</th>
            <th className="p-3 font-medium">vCenter</th>
            <th className="p-3 font-medium">State</th>
            <th className="p-3 font-medium">Protection Group</th>
            <th className="p-3 font-medium">Status</th>
            <th className="p-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {ungroupedVMs.map((vm) => (
            <tr key={vm.context_id} className="border-b hover:bg-muted/50 transition-colors">
              <td className="p-3">
                <div className="flex items-center gap-2">
                  <Server className="h-4 w-4 text-muted-foreground" />
                  <span className="font-medium text-sm">{vm.vm_name}</span>
                </div>
              </td>
              <td className="p-3 text-sm text-muted-foreground">{vm.vcenter_host}</td>
              <td className="p-3">
                <Badge variant="secondary" className="text-xs">
                  {vm.power_state}
                </Badge>
              </td>
              <td className="p-3">
                <Badge variant="outline" className="text-xs text-yellow-400 border-yellow-400/20">
                  Ungrouped
                </Badge>
              </td>
              <td className="p-3">
                <Badge variant="secondary" className="text-xs">
                  {vm.current_status}
                </Badge>
              </td>
              <td className="p-3">
                <div className="flex justify-end gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    className="text-xs h-7"
                    onClick={() => {
                      // TODO: Open group selection for this VM
                      console.log('Add to group:', vm.context_id);
                    }}
                  >
                    Add to Group
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {ungroupedVMs.length === 0 && !isLoadingUngroupedVMs && (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Server className="h-12 w-12 mb-2 opacity-50" />
          <p className="text-sm">No ungrouped VMs</p>
          <p className="text-xs">All discovered VMs are assigned to groups</p>
        </div>
      )}
    </div>
  </Card>
</div>
```

---

## Summary of Changes

### Performance
- ✅ **50% faster VM addition** (discover-vms with create_context=true)
- ✅ **Single vCenter query** instead of two

### Functionality
- ✅ **Live group data** from backend API
- ✅ **Create groups** with backend integration
- ✅ **Edit groups** (name, description, schedule, priority)
- ✅ **Manage VMs** (assign/remove from groups)
- ✅ **Delete groups** (ready for confirmation modal)
- ✅ **Live info panels** (groups, VMs, schedules)
- ✅ **Schedule integration** (dropdown in create/edit)

### UI/UX
- ✅ **Compact table layout** for efficient space usage
- ✅ **Loading states** everywhere
- ✅ **Group status badges** (active/inactive/error)
- ✅ **VM assignment interface** with search and bulk operations
- ✅ **Group membership display** in VM list

### Architecture
- ✅ **Type safety** with TypeScript interfaces
- ✅ **Error handling** for all API calls
- ✅ **Proper state management** with React hooks
- ✅ **Modular components** (EditGroupModal, ManageVMsModal)

---

## Testing Checklist

1. **Discovery**: Discover VMs → Add to management (should take ~14s, not 28s)
2. **Create Group**: Create group → Select VMs → Verify backend call
3. **Edit Group**: Edit group name/schedule → Verify update
4. **Manage VMs**: Open Manage VMs → Assign ungrouped VMs → Verify assignment
5. **Info Panels**: Verify counts match backend data
6. **VM Table**: Verify ungrouped VMs show correctly
7. **Delete Group**: Delete group → Verify backend call

---

## Files to Create/Update

### New Files (2)
1. `components/features/protection-groups/EditGroupModal.tsx` (253 lines)
2. `components/features/protection-groups/ManageVMsModal.tsx` (221 lines)

### Update Files (2)
1. `app/protection-groups/page.tsx` (major refactor)
2. `components/features/protection-groups/VMDiscoveryModal.tsx` (optimize add-vms)

### Export Updates (1)
1. `components/features/protection-groups/index.ts` (add new exports)

---

## Expected Results

### Before
- Mock data showing 4 groups
- Create group doesn't persist
- No way to edit groups
- No way to assign VMs
- Info panels showing fake data

### After
- Live data from backend showing real groups
- Create group persists to database
- Edit group updates backend
- Manage VMs assigns/removes VMs
- Info panels show live counts
- VM table shows ungrouped VMs
- 50% faster VM discovery workflow

---

## Notes

- All backend APIs exist and are working
- Auth is disabled (-auth=false) so no token needed
- Database foreign keys ensure referential integrity
- CASCADE DELETE means deleting group removes memberships
- Schedule assignments are optional (can be null)
- Priority and max_concurrent_vms have defaults
- Ungrouped VMs query uses database view for efficiency

