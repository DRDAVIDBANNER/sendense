"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Plus, Users, Calendar, Settings, MoreHorizontal, Server } from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { CreateGroupModal, VMDiscoveryModal } from "@/components/features/protection-groups";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface ProtectionGroup {
  id: string;
  name: string;
  description: string;
  vmCount: number;
  protectedVMs: number;
  schedule: string;
  lastRun: string;
  status: 'active' | 'inactive' | 'error';
  policy: 'daily' | 'weekly' | 'monthly';
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

const mockGroups: ProtectionGroup[] = [
  {
    id: '1',
    name: 'Production Web Servers',
    description: 'Critical web application servers requiring daily backups',
    vmCount: 8,
    protectedVMs: 8,
    schedule: 'Daily at 02:00',
    lastRun: '2025-10-06T02:00:00Z',
    status: 'active',
    policy: 'daily'
  },
  {
    id: '2',
    name: 'Development Environment',
    description: 'Development and testing VMs with weekly backups',
    vmCount: 12,
    protectedVMs: 10,
    schedule: 'Weekly on Sunday 03:00',
    lastRun: '2025-09-29T03:00:00Z',
    status: 'active',
    policy: 'weekly'
  },
  {
    id: '3',
    name: 'Legacy Applications',
    description: 'Older systems requiring monthly archive backups',
    vmCount: 5,
    protectedVMs: 3,
    schedule: 'Monthly on 1st at 04:00',
    lastRun: '2025-09-01T04:00:00Z',
    status: 'error',
    policy: 'monthly'
  },
  {
    id: '4',
    name: 'Database Servers',
    description: 'Critical database instances with frequent backups',
    vmCount: 3,
    protectedVMs: 3,
    schedule: 'Every 6 hours',
    lastRun: '2025-10-06T08:00:00Z',
    status: 'active',
    policy: 'daily'
  }
];

export default function ProtectionGroupsPage() {
  const [selectedGroupId, setSelectedGroupId] = useState<string>();
  const [groups, setGroups] = useState<ProtectionGroup[]>(mockGroups);

  // Modal states
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isDiscoveryModalOpen, setIsDiscoveryModalOpen] = useState(false);

  // Ungrouped VMs state
  const [ungroupedVMs, setUngroupedVMs] = useState<VMContext[]>([]);
  const [isLoadingUngroupedVMs, setIsLoadingUngroupedVMs] = useState(false);

  const handleCreateGroup = () => {
    setIsCreateModalOpen(true);
  };

  const handleAddVMs = () => {
    setIsDiscoveryModalOpen(true);
  };

  const fetchUngroupedVMs = async () => {
    setIsLoadingUngroupedVMs(true);
    try {
      const response = await fetch('/api/v1/discovery/ungrouped-vms');
      if (response.ok) {
        const vms = await response.json();
        setUngroupedVMs(vms);
      } else {
        console.error('Failed to fetch ungrouped VMs:', response.statusText);
      }
    } catch (error) {
      console.error('Failed to fetch ungrouped VMs:', error);
    } finally {
      setIsLoadingUngroupedVMs(false);
    }
  };

  // Fetch ungrouped VMs on component mount
  useEffect(() => {
    fetchUngroupedVMs();
  }, []);

  const handleCreateGroupSubmit = (groupData: {
    name: string;
    description: string;
    policy: string;
    schedule: string;
    vmIds: string[];
  }) => {
    const newGroup: ProtectionGroup = {
      id: Date.now().toString(),
      name: groupData.name,
      description: groupData.description,
      vmCount: groupData.vmIds.length,
      protectedVMs: groupData.vmIds.length,
      schedule: groupData.schedule.replace(/-/g, ' ').replace(/(\d{2}):(\d{2})/, '$1:$2'),
      lastRun: new Date().toISOString(),
      status: 'active',
      policy: groupData.policy as 'daily' | 'weekly' | 'monthly'
    };
    setGroups(prev => [...prev, newGroup]);
  };

  const handleEditGroup = (group: ProtectionGroup) => {
    // TODO: Open edit group modal
    console.log('Edit group:', group.id);
  };

  const handleDeleteGroup = (group: ProtectionGroup) => {
    // TODO: Open delete confirmation modal
    console.log('Delete group:', group.id);
  };

  const handleManageVMs = (group: ProtectionGroup) => {
    // TODO: Open VM assignment interface
    console.log('Manage VMs for group:', group.id);
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
                <div className="text-2xl font-bold">{groups.length}</div>
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
                  {groups.reduce((sum, group) => sum + group.vmCount, 0)}
                </div>
                <p className="text-xs text-muted-foreground">
                  Virtual machines managed
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
                  {groups.reduce((sum, group) => sum + group.protectedVMs, 0)}
                </div>
                <p className="text-xs text-muted-foreground">
                  VMs with active protection
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
                  {groups.filter(group => group.status === 'active').length}
                </div>
                <p className="text-xs text-muted-foreground">
                  Groups with active schedules
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Groups Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {groups.map((group) => (
              <Card
                key={group.id}
                className={`cursor-pointer transition-all hover:shadow-md ${
                  selectedGroupId === group.id ? 'ring-2 ring-primary' : ''
                }`}
                onClick={() => setSelectedGroupId(group.id)}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <CardTitle className="text-lg mb-1">{group.name}</CardTitle>
                      <p className="text-sm text-muted-foreground line-clamp-2">
                        {group.description}
                      </p>
                    </div>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => e.stopPropagation()}
                          className="h-8 w-8 p-0"
                        >
                          <MoreHorizontal className="h-4 w-4" />
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
                </CardHeader>

                <CardContent className="space-y-4">
                  {/* Status and Policy */}
                  <div className="flex items-center gap-2">
                    {getStatusBadge(group.status)}
                    {getPolicyBadge(group.policy)}
                  </div>

                  {/* VM Count */}
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">VMs Protected</span>
                      <span className="font-medium">
                        {group.protectedVMs}/{group.vmCount}
                      </span>
                    </div>
                    <Progress
                      value={(group.protectedVMs / group.vmCount) * 100}
                      className="h-2"
                    />
                  </div>

                  {/* Schedule Info */}
                  <div className="space-y-1">
                    <div className="flex items-center gap-2 text-sm">
                      <Calendar className="h-4 w-4 text-muted-foreground" />
                      <span className="text-muted-foreground">Schedule:</span>
                      <span className="font-medium">{group.schedule}</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <span className="text-muted-foreground">Last run:</span>
                      <span className="font-medium">{formatLastRun(group.lastRun)}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}

            {/* Add New Group Card */}
            <Card
              className="border-2 border-dashed border-muted-foreground/20 hover:border-primary/50 cursor-pointer transition-colors"
              onClick={handleCreateGroup}
            >
              <CardContent className="flex flex-col items-center justify-center py-12">
                <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4">
                  <Plus className="h-6 w-6 text-muted-foreground" />
                </div>
                <h3 className="text-lg font-medium text-foreground mb-2">Create New Group</h3>
                <p className="text-sm text-muted-foreground text-center">
                  Organize VMs with shared backup policies and schedules
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Ungrouped VMs Section */}
          {(ungroupedVMs.length > 0 || isLoadingUngroupedVMs) && (
            <div className="mt-8">
              <div className="mb-4">
                <h2 className="text-lg font-semibold text-foreground mb-2">
                  Discovered Virtual Machines
                </h2>
                <p className="text-muted-foreground">
                  VMs discovered from vCenter that are not yet assigned to protection groups
                </p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {isLoadingUngroupedVMs ? (
                  // Loading skeletons
                  Array.from({ length: 3 }).map((_, i) => (
                    <Card key={i} className="p-4">
                      <div className="space-y-3">
                        <div className="h-4 bg-muted rounded animate-pulse w-3/4" />
                        <div className="h-3 bg-muted rounded animate-pulse w-1/2" />
                        <div className="flex gap-2">
                          <div className="h-6 bg-muted rounded animate-pulse w-16" />
                          <div className="h-6 bg-muted rounded animate-pulse w-20" />
                        </div>
                      </div>
                    </Card>
                  ))
                ) : (
                  ungroupedVMs.map((vm) => (
                    <Card key={vm.context_id} className="p-4 hover:shadow-md transition-shadow">
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex-1">
                          <h3 className="font-medium text-sm mb-1">{vm.vm_name}</h3>
                          <p className="text-xs text-muted-foreground">{vm.datacenter}</p>
                        </div>
                        <Badge
                          variant="secondary"
                          className={`text-xs ${
                            vm.power_state === 'poweredOn'
                              ? 'bg-green-500/10 text-green-400 border-green-500/20'
                              : vm.power_state === 'poweredOff'
                              ? 'bg-gray-500/10 text-gray-400 border-gray-500/20'
                              : 'bg-yellow-500/10 text-yellow-400 border-yellow-500/20'
                          }`}
                        >
                          {vm.power_state}
                        </Badge>
                      </div>

                      <div className="text-xs text-muted-foreground mb-3">
                        vCenter: {vm.vcenter_host}
                      </div>

                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          className="flex-1 text-xs"
                          onClick={() => {
                            // TODO: Open group selection modal for this VM
                            console.log('Add to group:', vm.context_id);
                          }}
                        >
                          Add to Group
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          className="flex-1 text-xs"
                          onClick={() => {
                            // TODO: Navigate to create flow with this VM
                            console.log('Create flow:', vm.context_id);
                          }}
                        >
                          Create Flow
                        </Button>
                      </div>
                    </Card>
                  ))
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Modals */}
      <CreateGroupModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onCreate={handleCreateGroupSubmit}
      />
      <VMDiscoveryModal
        isOpen={isDiscoveryModalOpen}
        onClose={() => setIsDiscoveryModalOpen(false)}
        onDiscoveryComplete={fetchUngroupedVMs}
      />
    </div>
  );
}
