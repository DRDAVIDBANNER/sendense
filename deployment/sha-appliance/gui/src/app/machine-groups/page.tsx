'use client';

import React, { useState, useEffect } from 'react';
import { VMCentricLayout } from '@/components/layout/VMCentricLayout';
import { Button, Alert, Badge } from 'flowbite-react';
import { 
  HiRefresh, 
  HiPlus, 
  HiExclamationCircle,
  HiCollection,
  HiServer,
  HiClock,
  HiPencil,
  HiTrash,
  HiUsers
} from 'react-icons/hi';

interface VMContext {
  context_id: string;
  vm_name: string;
  vmware_vm_id: string;
  vm_path: string;
  current_status: string;
  cpu_count?: number;
  memory_mb?: number;
  os_type?: string;
  total_jobs_run: number;
  successful_jobs: number;
  failed_jobs: number;
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
  // Computed fields from API
  vm_count?: number;
  schedule_name?: string;
  assigned_vms?: VMContext[];
}

interface Schedule {
  id: string;
  name: string;
  cron_expression: string;
  timezone: string;
  enabled: boolean;
}

interface CreateGroupForm {
  name: string;
  description: string;
  schedule_id: string;
  max_concurrent_vms: number;
  priority: number;
}

export default function MachineGroupsPage() {
  const [groups, setGroups] = useState<MachineGroup[]>([]);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingGroup, setEditingGroup] = useState<MachineGroup | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [formData, setFormData] = useState<CreateGroupForm>({
    name: '',
    description: '',
    schedule_id: '',
    max_concurrent_vms: 5,
    priority: 0,
  });

  const loadGroups = async () => {
    try {
      setLoading(true);
      setError(null);
      
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
      
      // Load VMs for each group to get accurate VM counts
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
              return { ...group, vm_count: vmsData.vms?.length || 0, assigned_vms: vmsData.vms || [] };
            }
          } catch (err) {
            console.error(`Error loading VMs for group ${group.id}:`, err);
          }
          return { ...group, vm_count: 0, assigned_vms: [] };
        })
      );
      
      setGroups(groupsWithVMs);
    } catch (err) {
      console.error('Error loading machine groups:', err);
      setError(err instanceof Error ? err.message : 'Failed to load machine groups');
    } finally {
      setLoading(false);
    }
  };

  const loadSchedules = async () => {
    try {
      const response = await fetch('/api/schedules');
      if (!response.ok) {
        throw new Error(`Failed to load schedules: ${response.statusText}`);
      }
      
      const data = await response.json();
      setSchedules(data.schedules || []);
    } catch (err) {
      console.error('Error loading schedules:', err);
      // Don't set error for schedules load failure, just log it
    }
  };

  const createGroup = async () => {
    try {
      setActionLoading(true);
      setError(null);
      
      const payload = {
        ...formData,
        schedule_id: formData.schedule_id || null, // Convert empty string to null
      };
      
      const response = await fetch('/api/machine-groups', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to create group: ${response.statusText}`);
      }
      
      await loadGroups();
      setShowCreateModal(false);
      resetForm();
    } catch (err) {
      console.error('Error creating group:', err);
      setError(err instanceof Error ? err.message : 'Failed to create group');
    } finally {
      setActionLoading(false);
    }
  };

  const updateGroup = async () => {
    if (!editingGroup) return;
    
    try {
      setActionLoading(true);
      setError(null);
      
      const payload = {
        ...formData,
        schedule_id: formData.schedule_id || null,
      };
      
      const response = await fetch(`/api/machine-groups/${editingGroup.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to update group: ${response.statusText}`);
      }
      
      await loadGroups();
      setEditingGroup(null);
      resetForm();
    } catch (err) {
      console.error('Error updating group:', err);
      setError(err instanceof Error ? err.message : 'Failed to update group');
    } finally {
      setActionLoading(false);
    }
  };

  const deleteGroup = async (groupId: string, groupName: string) => {
    if (!confirm(`Are you sure you want to delete the group "${groupName}"? This action cannot be undone.`)) {
      return;
    }
    
    try {
      setActionLoading(true);
      setError(null);
      
      const response = await fetch(`/api/machine-groups/${groupId}`, {
        method: 'DELETE',
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to delete group: ${response.statusText}`);
      }
      
      await loadGroups();
    } catch (err) {
      console.error('Error deleting group:', err);
      setError(err instanceof Error ? err.message : 'Failed to delete group');
    } finally {
      setActionLoading(false);
    }
  };

  const startEdit = (group: MachineGroup) => {
    setEditingGroup(group);
    setFormData({
      name: group.name,
      description: group.description || '',
      schedule_id: group.schedule_id || '',
      max_concurrent_vms: group.max_concurrent_vms,
      priority: group.priority,
    });
  };

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      schedule_id: '',
      max_concurrent_vms: 5,
      priority: 0,
    });
    setEditingGroup(null);
  };

  const formatScheduleDescription = (cronExp: string, timezone: string): string => {
    const parts = cronExp.split(' ');
    if (parts.length !== 6) return cronExp;
    
    const [second, minute, hour, dayOfMonth, month, dayOfWeek] = parts;
    
    if (dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
      const h = parseInt(hour);
      const m = parseInt(minute);
      const timeStr = formatTime(h, m);
      return `Daily at ${timeStr}`;
    }
    
    return cronExp; // Simplified for now
  };

  const formatTime = (hour: number, minute: number): string => {
    const h12 = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
    const ampm = hour >= 12 ? 'PM' : 'AM';
    const m = minute.toString().padStart(2, '0');
    return `${h12}:${m} ${ampm}`;
  };

  const getScheduleInfo = (scheduleId?: string) => {
    if (!scheduleId) return null;
    return schedules.find(s => s.id === scheduleId);
  };

  useEffect(() => {
    loadGroups();
    loadSchedules();
  }, []);

  return (
    <VMCentricLayout>
      <div className="p-6">
        {/* Header */}
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white">Machine Groups</h1>
            <p className="text-gray-300">Organize VMs into groups for scheduled operations</p>
          </div>
          <div className="flex gap-2">
            <Button color="gray" onClick={loadGroups} disabled={loading}>
              <HiRefresh className="mr-2 h-4 w-4" />
              Refresh
            </Button>
            <Button onClick={() => setShowCreateModal(true)}>
              <HiPlus className="mr-2 h-4 w-4" />
              Create Group
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

        {/* Loading */}
        {loading ? (
          <div className="flex justify-center items-center h-64">
            <span className="text-gray-300">Loading machine groups...</span>
          </div>
        ) : (
          /* Groups List */
          <div className="grid gap-4">
            {groups.length === 0 ? (
              <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
                <div className="text-center py-8">
                  <HiCollection className="h-12 w-12 mx-auto mb-4 text-gray-400" />
                  <h3 className="text-lg font-medium text-white mb-2">No machine groups found</h3>
                  <p className="text-gray-300 mb-4">Create your first machine group to organize VMs for scheduled operations.</p>
                  <Button onClick={() => setShowCreateModal(true)} className="mx-auto">
                    <HiPlus className="mr-2 h-4 w-4" />
                    Create Group
                  </Button>
                </div>
              </div>
            ) : (
              groups.map((group) => {
                const scheduleInfo = getScheduleInfo(group.schedule_id);
                return (
                  <div key={group.id} className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-800/70 transition-all duration-200">
                    <div className="flex justify-between items-start">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <HiCollection className="h-5 w-5 text-cyan-400" />
                          <h3 className="text-lg font-semibold text-white">{group.name}</h3>
                          {group.priority > 0 && (
                            <Badge color="info" size="sm">Priority {group.priority}</Badge>
                          )}
                        </div>
                        
                        {group.description && (
                          <p className="text-gray-300 mb-3">{group.description}</p>
                        )}
                        
                        <div className="space-y-2">
                          <div className="flex items-center gap-4 text-sm">
                            <div className="flex items-center gap-1">
                              <HiUsers className="h-4 w-4 text-gray-400" />
                              <span className="text-gray-300">
                                {group.vm_count || 0} VM{(group.vm_count || 0) !== 1 ? 's' : ''}
                              </span>
                            </div>
                            <div className="flex items-center gap-1">
                              <HiServer className="h-4 w-4 text-gray-400" />
                              <span className="text-gray-300">
                                Max {group.max_concurrent_vms} concurrent
                              </span>
                            </div>
                          </div>
                          
                          {scheduleInfo ? (
                            <div className="flex items-center gap-2 text-sm">
                              <HiClock className="h-4 w-4 text-emerald-400" />
                              <span className="text-gray-300">
                                Schedule: <span className="font-medium text-emerald-300">{scheduleInfo.name}</span>
                              </span>
                              <span className="text-gray-400">
                                ({formatScheduleDescription(scheduleInfo.cron_expression, scheduleInfo.timezone)})
                              </span>
                              <Badge color={scheduleInfo.enabled ? 'success' : 'gray'} size="sm">
                                {scheduleInfo.enabled ? 'Active' : 'Disabled'}
                              </Badge>
                            </div>
                          ) : (
                            <div className="flex items-center gap-2 text-sm text-gray-400">
                              <HiClock className="h-4 w-4" />
                              <span>No schedule assigned</span>
                            </div>
                          )}
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-2 ml-4">
                        <Button 
                          size="sm" 
                          color="gray"
                          onClick={() => startEdit(group)}
                          disabled={actionLoading}
                        >
                          <HiPencil className="h-4 w-4" />
                        </Button>
                        <Button 
                          size="sm" 
                          color="failure"
                          onClick={() => deleteGroup(group.id, group.name)}
                          disabled={actionLoading}
                        >
                          <HiTrash className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        )}

        {/* Create/Edit Group Modal */}
        {(showCreateModal || editingGroup) && (
          <div 
            className="fixed inset-0 z-50 bg-black bg-opacity-50 flex items-center justify-center p-4"
            onClick={() => {
              setShowCreateModal(false);
              setEditingGroup(null);
              resetForm();
            }}
          >
            <div 
              className="bg-slate-800 rounded-lg shadow-xl max-w-md w-full max-h-90vh overflow-y-auto border border-slate-700"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="bg-slate-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                <div className="flex justify-between items-center mb-6">
                  <h3 className="text-lg font-medium text-white">
                    {editingGroup ? 'Edit Machine Group' : 'Create New Machine Group'}
                  </h3>
                  <button
                    onClick={() => {
                      setShowCreateModal(false);
                      setEditingGroup(null);
                      resetForm();
                    }}
                    className="text-gray-400 hover:text-gray-200 text-2xl font-bold leading-none"
                  >
                    ×
                  </button>
                </div>
                
                <div className="space-y-4">
                  <div>
                    <label htmlFor="name" className="block text-sm font-medium text-gray-300 mb-2">
                      Group Name
                    </label>
                    <input
                      id="name"
                      type="text"
                      placeholder="Enter group name"
                      value={formData.name}
                      onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                      required
                      className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                    />
                  </div>
                  
                  <div>
                    <label htmlFor="description" className="block text-sm font-medium text-gray-300 mb-2">
                      Description
                    </label>
                    <textarea
                      id="description"
                      placeholder="Enter description (optional)"
                      value={formData.description}
                      onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                      rows={3}
                      className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                    />
                  </div>
                  
                  <div>
                    <label htmlFor="schedule" className="block text-sm font-medium text-gray-300 mb-2">
                      Assigned Schedule
                    </label>
                    <select
                      id="schedule"
                      value={formData.schedule_id}
                      onChange={(e) => setFormData(prev => ({ ...prev, schedule_id: e.target.value }))}
                      className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                    >
                      <option value="">No schedule (manual operation)</option>
                      {schedules.map((schedule) => (
                        <option key={schedule.id} value={schedule.id}>
                          {schedule.name} ({schedule.enabled ? 'Active' : 'Disabled'})
                        </option>
                      ))}
                    </select>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label htmlFor="max_concurrent" className="block text-sm font-medium text-gray-300 mb-2">
                        Max Concurrent VMs
                      </label>
                      <input
                        id="max_concurrent"
                        type="number"
                        min="1"
                        max="50"
                        value={formData.max_concurrent_vms}
                        onChange={(e) => setFormData(prev => ({ ...prev, max_concurrent_vms: parseInt(e.target.value) || 5 }))}
                        className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                      />
                    </div>
                    <div>
                      <label htmlFor="priority" className="block text-sm font-medium text-gray-300 mb-2">
                        Priority
                      </label>
                      <input
                        id="priority"
                        type="number"
                        min="0"
                        max="100"
                        value={formData.priority}
                        onChange={(e) => setFormData(prev => ({ ...prev, priority: parseInt(e.target.value) || 0 }))}
                        className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                      />
                      <p className="text-xs text-gray-400 mt-1">Higher values = higher priority</p>
                    </div>
                  </div>
                </div>
              </div>
              
              <div className="bg-slate-700 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse gap-3">
                <button
                  onClick={editingGroup ? updateGroup : createGroup}
                  disabled={actionLoading || !formData.name}
                  className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-cyan-600 text-base font-medium text-white hover:bg-cyan-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {actionLoading ? (editingGroup ? 'Updating...' : 'Creating...') : (editingGroup ? 'Update Group' : 'Create Group')}
                </button>
                <button
                  onClick={() => {
                    setShowCreateModal(false);
                    setEditingGroup(null);
                    resetForm();
                  }}
                  className="mt-3 w-full inline-flex justify-center rounded-md border border-slate-600 shadow-sm px-4 py-2 bg-slate-600 text-base font-medium text-gray-300 hover:bg-slate-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </VMCentricLayout>
  );
}