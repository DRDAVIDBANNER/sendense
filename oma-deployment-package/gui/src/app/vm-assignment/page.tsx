'use client';

import React, { useState, useEffect } from 'react';
import { VMCentricLayout } from '@/components/layout/VMCentricLayout';
import { Button, Alert, Badge } from 'flowbite-react';
import { 
  HiRefresh, 
  HiExclamationCircle,
  HiCollection,
  HiServer,
  HiCheck,
  HiX,
  HiArrowRight,
  HiUsers,
  HiClock
} from 'react-icons/hi';

interface VMContext {
  context_id: string;
  vm_name: string;
  vmware_vm_id: string;
  vm_path: string;
  vcenter_host: string;
  datacenter: string;
  current_status: string;
  current_job_id?: string;
  total_jobs_run: number;
  successful_jobs: number;
  failed_jobs: number;
  cpu_count?: number;
  memory_mb?: number;
  os_type?: string;
  created_at: string;
  updated_at: string;
}

interface MachineGroup {
  id: string;
  name: string;
  description?: string;
  schedule_id?: string;
  max_concurrent_vms: number;
  priority: number;
  created_at: string;
  updated_at: string;
  vm_count?: number;
  schedule_name?: string;
  assigned_vms?: VMContext[];
}

interface GroupMembership {
  id: string;
  group_id: string;
  vm_context_id: string;
  enabled: boolean;
  priority: number;
  schedule_override_id?: string;
}

export default function VMAssignmentPage() {
  const [ungroupedVMs, setUngroupedVMs] = useState<VMContext[]>([]);
  const [groups, setGroups] = useState<MachineGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [selectedVMs, setSelectedVMs] = useState<Set<string>>(new Set());
  const [selectedGroup, setSelectedGroup] = useState<string>('');
  const [draggedVM, setDraggedVM] = useState<VMContext | null>(null);

  const loadUngroupedVMs = async () => {
    try {
      const response = await fetch(`/api/vm-contexts/ungrouped?_t=${Date.now()}`, {
        headers: {
          'Cache-Control': 'no-cache',
        },
      });
      if (!response.ok) {
        throw new Error(`Failed to load ungrouped VMs: ${response.statusText}`);
      }
      
      const data = await response.json();
      setUngroupedVMs(data.vm_contexts || []);
    } catch (err) {
      console.error('Error loading ungrouped VMs:', err);
      setError(err instanceof Error ? err.message : 'Failed to load ungrouped VMs');
    }
  };

  const loadGroups = async () => {
    try {
      const response = await fetch(`/api/machine-groups?_t=${Date.now()}`, {
        headers: {
          'Cache-Control': 'no-cache',
        },
      });
      if (!response.ok) {
        throw new Error(`Failed to load machine groups: ${response.statusText}`);
      }
      
      const data = await response.json();
      const groupsData = data.groups || [];
      
      // Load VMs for each group
      const groupsWithVMs = await Promise.all(
        groupsData.map(async (group: MachineGroup) => {
          try {
            const vmsResponse = await fetch(`/api/machine-groups/${group.id}/vms?_t=${Date.now()}`, {
              headers: {
                'Cache-Control': 'no-cache',
              },
            });
            if (vmsResponse.ok) {
              const vmsData = await vmsResponse.json();
              return { ...group, assigned_vms: vmsData.vms || [] };
            }
          } catch (err) {
            console.error(`Error loading VMs for group ${group.id}:`, err);
          }
          return { ...group, assigned_vms: [] };
        })
      );
      
      setGroups(groupsWithVMs);
    } catch (err) {
      console.error('Error loading machine groups:', err);
      setError(err instanceof Error ? err.message : 'Failed to load machine groups');
    }
  };

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);
      await Promise.all([loadUngroupedVMs(), loadGroups()]);
    } finally {
      setLoading(false);
    }
  };

  const assignVMToGroup = async (vmContextId: string, groupId: string) => {
    try {
      setActionLoading(true);
      setError(null);
      
      const response = await fetch(`/api/machine-groups/${groupId}/vms`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          vm_context_id: vmContextId,
          enabled: true,
          priority: 0
        }),
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to assign VM to group: ${response.statusText}`);
      }
      
      await loadData(); // Reload both ungrouped VMs and groups
    } catch (err) {
      console.error('Error assigning VM to group:', err);
      setError(err instanceof Error ? err.message : 'Failed to assign VM to group');
    } finally {
      setActionLoading(false);
    }
  };

  const removeVMFromGroup = async (vmContextId: string, groupId: string) => {
    try {
      setActionLoading(true);
      setError(null);
      
      const response = await fetch(`/api/machine-groups/${groupId}/vms/${vmContextId}`, {
        method: 'DELETE',
        headers: {
          'Cache-Control': 'no-cache',
        },
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to remove VM from group: ${response.statusText}`);
      }
      
      // Reload data to reflect the removal
      await loadData();
      
    } catch (err) {
      console.error('Error removing VM from group:', err);
      setError(err instanceof Error ? err.message : 'Failed to remove VM from group');
    } finally {
      setActionLoading(false);
    }
  };

  const bulkAssignVMs = async () => {
    if (selectedVMs.size === 0 || !selectedGroup) return;
    
    try {
      setActionLoading(true);
      setError(null);
      
      // Assign all selected VMs to the selected group
      for (const vmContextId of selectedVMs) {
        await assignVMToGroup(vmContextId, selectedGroup);
      }
      
      setSelectedVMs(new Set());
      setSelectedGroup('');
    } catch (err) {
      console.error('Error bulk assigning VMs:', err);
      setError(err instanceof Error ? err.message : 'Failed to bulk assign VMs');
    } finally {
      setActionLoading(false);
    }
  };

  const toggleVMSelection = (vmContextId: string) => {
    const newSelection = new Set(selectedVMs);
    if (newSelection.has(vmContextId)) {
      newSelection.delete(vmContextId);
    } else {
      newSelection.add(vmContextId);
    }
    setSelectedVMs(newSelection);
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'discovered': return 'info';
      case 'replicating': return 'warning';
      case 'ready_for_failover': return 'success';
      case 'failed_over_test': return 'purple';
      case 'failed_over_live': return 'dark';
      case 'completed': return 'success';
      case 'failed': return 'failure';
      case 'cleanup_required': return 'warning';
      default: return 'gray';
    }
  };

  const formatMemory = (memoryMB?: number): string => {
    if (!memoryMB) return 'N/A';
    if (memoryMB >= 1024) {
      return `${(memoryMB / 1024).toFixed(1)} GB`;
    }
    return `${memoryMB} MB`;
  };

  const handleDragStart = (vm: VMContext) => {
    setDraggedVM(vm);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDrop = (e: React.DragEvent, groupId: string) => {
    e.preventDefault();
    if (draggedVM) {
      assignVMToGroup(draggedVM.context_id, groupId);
      setDraggedVM(null);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  return (
    <VMCentricLayout>
      <div className="p-6">
        {/* Header */}
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white">VM Group Assignment</h1>
            <p className="text-gray-300">Assign VMs to machine groups for organized scheduling</p>
          </div>
          <div className="flex gap-2">
            <Button color="gray" onClick={loadData} disabled={loading}>
              <HiRefresh className="mr-2 h-4 w-4" />
              Refresh
            </Button>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="mb-4 p-4 bg-red-500/20 border border-red-500/30 rounded-lg flex items-center">
            <HiExclamationCircle className="h-5 w-5 text-red-300 flex-shrink-0" />
            <span className="ml-3 text-red-300">{error}</span>
            <button
              onClick={() => setError(null)}
              className="ml-auto text-red-300 hover:text-red-100 text-xl font-bold leading-none"
            >
              ×
            </button>
          </div>
        )}

        {/* Bulk Assignment Controls */}
        {selectedVMs.size > 0 && (
          <div className="bg-cyan-500/20 border border-cyan-500/30 rounded-lg p-4 mb-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <span className="text-sm font-medium text-cyan-300">
                  {selectedVMs.size} VM{selectedVMs.size !== 1 ? 's' : ''} selected
                </span>
                <select
                  value={selectedGroup}
                  onChange={(e) => setSelectedGroup(e.target.value)}
                  className="px-3 py-1 bg-slate-700 border border-slate-600 rounded text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500"
                >
                  <option value="">Select a group...</option>
                  {groups.map((group) => (
                    <option key={group.id} value={group.id}>
                      {group.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex gap-2">
                <Button 
                  size="sm" 
                  onClick={bulkAssignVMs} 
                  disabled={!selectedGroup || actionLoading}
                >
                  <HiArrowRight className="mr-1 h-4 w-4" />
                  Assign to Group
                </Button>
                <Button 
                  size="sm" 
                  color="gray" 
                  onClick={() => setSelectedVMs(new Set())}
                >
                  Clear Selection
                </Button>
              </div>
            </div>
          </div>
        )}

        {loading ? (
          <div className="flex justify-center items-center h-64">
            <span className="text-gray-300">Loading VM assignment data...</span>
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Ungrouped VMs */}
            <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg">
              <div className="p-4 border-b border-slate-700/50">
                <h2 className="text-lg font-semibold text-white flex items-center gap-2">
                  <HiServer className="h-5 w-5 text-cyan-400" />
                  Ungrouped VMs ({ungroupedVMs.length})
                </h2>
                <p className="text-sm text-gray-300 mt-1">VMs not assigned to any machine group</p>
              </div>
              
              <div className="p-4">
                {ungroupedVMs.length === 0 ? (
                  <div className="text-center py-8 text-gray-400">
                    <HiCheck className="h-8 w-8 mx-auto mb-2 text-emerald-400" />
                    All VMs are assigned to groups
                  </div>
                ) : (
                  <div className="space-y-3">
                    {ungroupedVMs.map((vm) => (
                      <div 
                        key={vm.context_id} 
                        className={`border rounded-lg p-3 cursor-move hover:bg-slate-700/50 transition-all duration-200 ${
                          selectedVMs.has(vm.context_id) ? 'border-cyan-500 bg-cyan-500/20' : 'border-slate-600'
                        }`}
                        draggable
                        onDragStart={() => handleDragStart(vm)}
                        onClick={() => toggleVMSelection(vm.context_id)}
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-1">
                              <div className={`w-4 h-4 rounded border-2 flex items-center justify-center ${
                                selectedVMs.has(vm.context_id) ? 'border-cyan-500 bg-cyan-500' : 'border-gray-400'
                              }`}>
                                {selectedVMs.has(vm.context_id) && (
                                  <HiCheck className="h-3 w-3 text-white" />
                                )}
                              </div>
                              <h3 className="font-medium text-white">{vm.vm_name}</h3>
                              <Badge color={getStatusColor(vm.current_status)} size="sm">
                                {vm.current_status.replace('_', ' ')}
                              </Badge>
                            </div>
                            <div className="text-sm text-gray-300 space-y-1">
                              <div>Path: {vm.vm_path}</div>
                              <div className="flex items-center gap-4">
                                {vm.cpu_count && <span>CPU: {vm.cpu_count}</span>}
                                {vm.memory_mb && <span>Memory: {formatMemory(vm.memory_mb)}</span>}
                                {vm.os_type && <span>OS: {vm.os_type}</span>}
                              </div>
                              <div>Jobs: {vm.successful_jobs}/{vm.total_jobs_run} successful</div>
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Machine Groups */}
            <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg">
              <div className="p-4 border-b border-slate-700/50">
                <h2 className="text-lg font-semibold text-white flex items-center gap-2">
                  <HiCollection className="h-5 w-5 text-emerald-400" />
                  Machine Groups ({groups.length})
                </h2>
                <p className="text-sm text-gray-300 mt-1">Drop VMs here to assign them to groups</p>
              </div>
              
              <div className="p-4 space-y-4">
                {groups.length === 0 ? (
                  <div className="text-center py-8 text-gray-400">
                    <HiCollection className="h-8 w-8 mx-auto mb-2 text-gray-400" />
                    No machine groups found
                    <br />
                    <a href="/machine-groups" className="text-cyan-400 hover:text-cyan-300 hover:underline text-sm">
                      Create machine groups first
                    </a>
                  </div>
                ) : (
                  groups.map((group) => (
                    <div 
                      key={group.id}
                      className="border border-slate-600 rounded-lg p-3 hover:border-emerald-400 transition-colors bg-slate-700/30"
                      onDragOver={handleDragOver}
                      onDrop={(e) => handleDrop(e, group.id)}
                    >
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <HiCollection className="h-4 w-4 text-emerald-400" />
                          <h3 className="font-medium text-white">{group.name}</h3>
                          <Badge color="info" size="sm">
                            {group.assigned_vms?.length || 0} VMs
                          </Badge>
                        </div>
                      </div>
                      
                      {group.description && (
                        <p className="text-sm text-gray-300 mb-2">{group.description}</p>
                      )}
                      
                      <div className="text-xs text-gray-400 mb-3">
                        Max concurrent: {group.max_concurrent_vms} | Priority: {group.priority}
                      </div>
                      
                      {/* Assigned VMs */}
                      {group.assigned_vms && group.assigned_vms.length > 0 ? (
                        <div className="space-y-2">
                          {group.assigned_vms.map((vm) => (
                            <div key={vm.vm_context_id} className="bg-slate-600/50 rounded p-2 flex items-center justify-between">
                              <div>
                                <div className="font-medium text-sm text-white">{vm.vm_name}</div>
                                <div className="text-xs text-gray-300">
                                  {vm.cpu_count && `${vm.cpu_count} CPU`}
                                  {vm.cpu_count && vm.memory_mb && ' | '}
                                  {vm.memory_mb && formatMemory(vm.memory_mb)}
                                </div>
                              </div>
                              <Button 
                                size="xs" 
                                color="failure"
                                onClick={() => removeVMFromGroup(vm.vm_context_id, group.id)}
                                disabled={actionLoading}
                              >
                                <HiX className="h-3 w-3" />
                              </Button>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="text-center py-4 text-gray-400 text-sm border-2 border-dashed border-slate-600 rounded">
                          Drop VMs here or select VMs and use bulk assign
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </VMCentricLayout>
  );
}

