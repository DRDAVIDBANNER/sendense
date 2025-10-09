"use client";

import { useState, useMemo } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Flow, FlowType } from "./types";
import { useProtectionGroups, useVMContexts, useRepositories } from "@/src/features/protection-flows/hooks/useFlowSources";

interface CreateFlowModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (flow: Omit<Flow, 'id' | 'status' | 'lastRun' | 'progress'>) => void;
}

// Helper function for formatting bytes
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
};

export function CreateFlowModal({ isOpen, onClose, onCreate }: CreateFlowModalProps) {
  // API data hooks
  const { data: groupsData, isLoading: loadingGroups } = useProtectionGroups();
  const { data: vmsData, isLoading: loadingVMs } = useVMContexts();
  const { data: reposData, isLoading: loadingRepos } = useRepositories();

  const groups = groupsData?.groups || [];
  const vms = vmsData?.vm_contexts || [];
  const repos = reposData?.repositories?.filter(r => r.enabled) || [];

  const [formData, setFormData] = useState<{
    name: string;
    type: FlowType;
    source: string;  // Will store "group:GROUP_ID" or "vm:CONTEXT_ID"
    sourceType: 'group' | 'vm' | '';  // Track selection type
    destination: string;  // Will store REPOSITORY_ID
    nextRun: string;
    description: string;
  }>({
    name: '',
    type: 'backup' as FlowType,
    source: '',  // Will store "group:GROUP_ID" or "vm:CONTEXT_ID"
    sourceType: '',  // Track selection type
    destination: '',  // Will store REPOSITORY_ID
    nextRun: '',
    description: ''
  });

  // Search state for large lists
  const [sourceSearch, setSourceSearch] = useState('');
  const [repoSearch, setRepoSearch] = useState('');

  // Filter logic for search
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

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Create Protection Flow</DialogTitle>
          <DialogDescription>
            Set up a new backup or replication flow for your virtual machines.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="name">Flow Name</Label>
              <Input
                id="name"
                placeholder="e.g., Daily VM Backup - Production"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="type">Flow Type</Label>
              <Select value={formData.type} onValueChange={(value: FlowType) => handleInputChange('type', value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="backup">Backup</SelectItem>
                  <SelectItem value="replication">Replication</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
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
          </div>

          <div className="space-y-2">
            <Label htmlFor="nextRun">Next Run Schedule</Label>
            <Input
              id="nextRun"
              type="datetime-local"
              value={formData.nextRun}
              onChange={(e) => handleInputChange('nextRun', e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description (Optional)</Label>
            <Textarea
              id="description"
              placeholder="Additional notes about this protection flow..."
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              rows={3}
            />
          </div>
        </form>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            onClick={handleSubmit}
            disabled={!formData.name || !formData.source || !formData.destination}
          >
            Create Flow
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
