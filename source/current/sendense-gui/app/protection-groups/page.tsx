"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Plus, Users, Calendar, Settings, MoreHorizontal, Server } from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { CreateGroupModal, EditGroupModal, ManageVMsModal, VMDiscoveryModal } from "@/components/features/protection-groups";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

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

export default function ProtectionGroupsPage() {
  // Real API integration
  const [groups, setGroups] = useState<ProtectionGroup[]>([]);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [selectedGroupId, setSelectedGroupId] = useState<string>();
  const [isLoadingGroups, setIsLoadingGroups] = useState(false);
  const [isLoadingSchedules, setIsLoadingSchedules] = useState(false);

  // Modal states
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<ProtectionGroup | null>(null);
  const [managingGroup, setManagingGroup] = useState<ProtectionGroup | null>(null);
  const [isDiscoveryModalOpen, setIsDiscoveryModalOpen] = useState(false);

  // Ungrouped VMs state
  const [ungroupedVMs, setUngroupedVMs] = useState<VMContext[]>([]);
  const [isLoadingUngroupedVMs, setIsLoadingUngroupedVMs] = useState(false);

  const handleCreateGroup = () => {
    setIsCreateModalOpen(true);
  };

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

  const handleAddVMs = () => {
    setIsDiscoveryModalOpen(true);
  };

  const fetchUngroupedVMs = async () => {
    setIsLoadingUngroupedVMs(true);
    try {
      const response = await fetch('/api/v1/discovery/ungrouped-vms');
      if (response.ok) {
        const data = await response.json();
        setUngroupedVMs(data.vms || []);
      } else {
        console.error('Failed to fetch ungrouped VMs:', response.statusText);
      }
    } catch (error) {
      console.error('Failed to fetch ungrouped VMs:', error);
    } finally {
      setIsLoadingUngroupedVMs(false);
    }
  };

  // Fetch on component mount and after operations
  useEffect(() => {
    fetchGroups();
    fetchSchedules();
    fetchUngroupedVMs();
  }, []);

  const handleCreateGroupSubmit = (groupData: {
    name: string;
    description: string;
    schedule: string;
    maxConcurrentVMs: number;
    priority: number;
    vmIds: string[];
  }) => {
    // Refresh groups after creation (API call happens in modal)
    fetchGroups();
    setIsCreateModalOpen(false);
  };

  const handleEditGroup = (group: ProtectionGroup) => {
    setEditingGroup(group);
  };

  const handleDeleteGroup = async (group: ProtectionGroup) => {
    // TODO: Add confirmation dialog
    try {
      const response = await fetch(`/api/v1/machine-groups/${group.id}`, {
        method: 'DELETE'
      });

      if (response.ok) {
        console.log('✅ Group deleted:', group.id);
        fetchGroups();
      } else {
        console.error('Failed to delete group');
      }
    } catch (error) {
      console.error('Error deleting group:', error);
    }
  };

  const handleManageVMs = (group: ProtectionGroup) => {
    setManagingGroup(group);
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return <Badge className="bg-green-500/10 text-green-400 border-green-500/20">Active</Badge>;
      case 'inactive':
        return <Badge variant="secondary">Inactive</Badge>;
      case 'error':
        return <Badge className="bg-red-500/10 text-red-400 border-red-500/20">Error</Badge>;
      default:
        return <Badge variant="secondary">Unknown</Badge>;
    }
  };

  const getPolicyBadge = (policy: string) => {
    switch (policy) {
      case 'daily':
        return <Badge variant="outline" className="text-blue-400 border-blue-400/20">Daily</Badge>;
      case 'weekly':
        return <Badge variant="outline" className="text-purple-400 border-purple-400/20">Weekly</Badge>;
      case 'monthly':
        return <Badge variant="outline" className="text-orange-400 border-orange-400/20">Monthly</Badge>;
      default:
        return <Badge variant="outline">Unknown</Badge>;
    }
  };

  const formatLastRun = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = diffMs / (1000 * 60 * 60 * 24);

    if (diffDays < 1) {
      return 'Today';
    } else if (diffDays < 7) {
      return `${Math.floor(diffDays)} days ago`;
    } else {
      return date.toLocaleDateString();
    }
  };

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Protection Groups"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Protection Groups" }
        ]}
        actions={
          <div className="flex gap-3">
            <Button onClick={handleAddVMs} variant="outline" className="gap-2">
              <Server className="h-4 w-4" />
              Add VMs
            </Button>
            <Button onClick={handleCreateGroup} className="gap-2">
              <Plus className="h-4 w-4" />
              Create Group
            </Button>
          </div>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="p-6">
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-foreground mb-2">
              VM Protection Groups
            </h2>
            <p className="text-muted-foreground">
              Organize virtual machines into groups with shared backup policies and schedules
            </p>
          </div>

          {/* Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
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

          {/* Groups Grid - COMPACT DESIGN */}
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

          {/* VM Management Section */}
          <div className="mt-8">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-semibold text-foreground mb-1">
                  Virtual Machines
                </h2>
                <p className="text-sm text-muted-foreground">
                  {isLoadingUngroupedVMs ? 'Loading...' : `${ungroupedVMs.length} ungrouped • ${groups.reduce((sum, g) => sum + g.total_vms, 0)} in groups`}
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
                    {isLoadingUngroupedVMs ? (
                      // Loading rows
                      Array.from({ length: 3 }).map((_, i) => (
                        <tr key={i}>
                          <td className="p-3"><div className="h-4 bg-muted rounded animate-pulse w-32" /></td>
                          <td className="p-3"><div className="h-4 bg-muted rounded animate-pulse w-24" /></td>
                          <td className="p-3"><div className="h-4 bg-muted rounded animate-pulse w-16" /></td>
                          <td className="p-3"><div className="h-4 bg-muted rounded animate-pulse w-20" /></td>
                          <td className="p-3"><div className="h-4 bg-muted rounded animate-pulse w-16" /></td>
                          <td className="p-3"><div className="h-4 bg-muted rounded animate-pulse w-20" /></td>
                        </tr>
                      ))
                    ) : (
                      ungroupedVMs.map((vm) => (
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
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    className="text-xs h-7"
                                  >
                                    Add to Group
                                  </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end">
                                  {groups.length === 0 ? (
                                    <div className="px-2 py-1.5 text-xs text-muted-foreground">
                                      No groups available
                                    </div>
                                  ) : (
                                    groups.map((group) => (
                                      <DropdownMenuItem
                                        key={group.id}
                                        onClick={async () => {
                                          try {
                                            const response = await fetch(`/api/v1/machine-groups/${group.id}/vms`, {
                                              method: 'POST',
                                              headers: { 'Content-Type': 'application/json' },
                                              body: JSON.stringify({
                                                vm_context_id: vm.context_id, // FIXED: singular, not array
                                                priority: 50,
                                                enabled: true,
                                              }),
                                            });
                                            if (response.ok) {
                                              console.log(`✅ Added VM to group ${group.name}`);
                                              fetchGroups();
                                              fetchUngroupedVMs();
                                            } else {
                                              const error = await response.json();
                                              console.error('Failed to add VM to group:', error);
                                              alert(`Failed to add VM: ${error.error || 'Unknown error'}`);
                                            }
                                          } catch (error) {
                                            console.error('Error adding VM to group:', error);
                                            alert(`Error: ${error}`);
                                          }
                                        }}
                                      >
                                        {group.name}
                                      </DropdownMenuItem>
                                    ))
                                  )}
                                </DropdownMenuContent>
                              </DropdownMenu>
                            </div>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>

                {!isLoadingUngroupedVMs && ungroupedVMs.length === 0 && (
                  <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                    <Server className="h-12 w-12 mb-2 opacity-50" />
                    <p className="text-sm">No ungrouped VMs</p>
                    <p className="text-xs">All discovered VMs are assigned to groups</p>
                  </div>
                )}
              </div>
            </Card>
          </div>
        </div>
      </div>

      {/* Modals */}
      <CreateGroupModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onCreate={handleCreateGroupSubmit}
        schedules={schedules}
      />
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
      <VMDiscoveryModal
        isOpen={isDiscoveryModalOpen}
        onClose={() => setIsDiscoveryModalOpen(false)}
        onDiscoveryComplete={fetchUngroupedVMs}
      />
    </div>
  );
}
