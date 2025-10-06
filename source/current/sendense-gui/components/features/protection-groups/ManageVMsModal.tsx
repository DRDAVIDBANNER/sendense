"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Server, Search, Plus, X, Loader2 } from "lucide-react";

interface VMContext {
  context_id: string;
  vm_name: string;
  vmware_vm_id: string;
  vcenter_host: string;
  current_status: 'discovered' | 'replicating' | 'ready_for_failover';
  datacenter: string;
  power_state: 'poweredOn' | 'poweredOff' | 'suspended';
  last_discovered_at: string;
}

interface ProtectionGroup {
  id: string;
  name: string;
  description: string | null;
  schedule_id: string | null;
  schedule_name: string | null;
  max_concurrent_vms: number;
  priority: number;
  total_vms: number;
  enabled_vms: number;
  disabled_vms: number;
  active_jobs: number;
  last_execution: string | null;
  created_by: string;
  created_at: string;
  updated_at: string;
  status: 'active' | 'inactive' | 'error';
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

interface ManageVMsModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdate: () => void;
  group: ProtectionGroup | null;
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
