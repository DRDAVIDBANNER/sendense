"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Server, CheckCircle, Plus, Loader2, AlertCircle } from "lucide-react";
import { CreateScheduleModal } from "./CreateScheduleModal";

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

interface CreateGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (groupData: {
    name: string;
    description: string;
    schedule: string;
    maxConcurrentVMs: number;
    priority: number;
    vmIds: string[];
  }) => void;
  schedules: Schedule[];
}

export function CreateGroupModal({ isOpen, onClose, onCreate, schedules }: CreateGroupModalProps) {
  // State for available VMs loaded from API
  const [availableVMs, setAvailableVMs] = useState<VMContext[]>([]);
  const [isLoadingVMs, setIsLoadingVMs] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentStep, setCurrentStep] = useState(1);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    schedule: '',
    maxConcurrentVMs: 10,
    priority: 50,
    vmIds: [] as string[]
  });
  const [isCreateScheduleModalOpen, setIsCreateScheduleModalOpen] = useState(false);

  // Fetch available VMs when modal opens
  useEffect(() => {
    const fetchAvailableVMs = async () => {
      if (!isOpen) return; // Only fetch when modal opens

      setIsLoadingVMs(true);
      try {
        const response = await fetch('/api/v1/discovery/ungrouped-vms');
        if (response.ok) {
          const data = await response.json();
          setAvailableVMs(data.vms || []); // Extract vms array from response
        } else {
          console.error('Failed to fetch ungrouped VMs:', response.statusText);
        }
      } catch (error) {
        console.error('Failed to fetch ungrouped VMs:', error);
      } finally {
        setIsLoadingVMs(false);
      }
    };

    fetchAvailableVMs();
  }, [isOpen]);

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleVMSelection = (vmId: string, checked: boolean) => {
    setFormData(prev => ({
      ...prev,
      vmIds: checked
        ? [...prev.vmIds, vmId]
        : prev.vmIds.filter(id => id !== vmId)
    }));
  };

  const handleNext = () => {
    if (currentStep < 3) {
      setCurrentStep(currentStep + 1);
    }
  };

  const handlePrevious = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleSubmit = async () => {
    // Validate
    if (!formData.name) {
      setError('Group name is required');
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
          schedule_id: formData.schedule || null, // Schedule ID from dropdown (optional)
          max_concurrent_vms: formData.maxConcurrentVMs || 10,
          priority: formData.priority || 50,
          created_by: 'gui-user',
        }),
      });

      if (response.ok) {
        const result = await response.json();
        console.log('✅ Group created:', result.id);

        // If VMs selected, assign them to the group
        if (formData.vmIds.length > 0) {
          await assignVMsToGroup(result.id, formData.vmIds);
        }

        onCreate({
          ...formData,
          vmIds: formData.vmIds,
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

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      schedule: '',
      maxConcurrentVMs: 10,
      priority: 50,
      vmIds: []
    });
    setCurrentStep(1);
    setError(null);
  };

  const selectedVMs = availableVMs.filter(vm => formData.vmIds.includes(vm.context_id));

  const handleCreateSchedule = async (scheduleData: any) => {
    // Mock schedule creation - in real implementation this would call API
    const mockSchedule = {
      id: `schedule-${Date.now()}`,
      name: scheduleData.name,
      displayName: scheduleData.name
    };

    // For now, just set the schedule to the new one
    setFormData(prev => ({
      ...prev,
      schedule: mockSchedule.id
    }));

    return mockSchedule;
  };

  const getStatusBadge = (powerState: string) => {
    switch (powerState) {
      case 'poweredOn':
        return <Badge className="bg-green-500/10 text-green-400 border-green-500/20">Running</Badge>;
      case 'poweredOff':
        return <Badge className="bg-gray-500/10 text-gray-400 border-gray-500/20">Stopped</Badge>;
      case 'suspended':
        return <Badge className="bg-yellow-500/10 text-yellow-400 border-yellow-500/20">Suspended</Badge>;
      default:
        return <Badge variant="secondary">Unknown</Badge>;
    }
  };

  const canProceedToNext = () => {
    switch (currentStep) {
      case 1:
        return formData.name.trim() && formData.description.trim();
      case 2:
        return true; // Schedule is optional, always allow proceeding
      case 3:
        return formData.vmIds.length > 0;
      default:
        return false;
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle>Create Protection Group</DialogTitle>
          <DialogDescription>
            Set up a new protection group with backup policies and VM assignments.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <span className="text-sm text-destructive">{error}</span>
          </div>
        )}

        <div className="flex items-center justify-center mb-6">
          <div className="flex items-center space-x-4">
            {[1, 2, 3].map((step) => (
              <div key={step} className="flex items-center">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                  step <= currentStep
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted text-muted-foreground'
                }`}>
                  {step}
                </div>
                {step < 3 && (
                  <div className={`w-12 h-0.5 mx-2 ${
                    step < currentStep ? 'bg-primary' : 'bg-muted'
                  }`} />
                )}
              </div>
            ))}
          </div>
        </div>

        <div className="flex-1 overflow-auto">
          {currentStep === 1 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="group-name">Group Name</Label>
                <Input
                  id="group-name"
                  placeholder="e.g., Production Web Servers"
                  value={formData.name}
                  onChange={(e) => handleInputChange('name', e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="group-description">Description</Label>
                <Textarea
                  id="group-description"
                  placeholder="Describe the purpose and scope of this protection group..."
                  value={formData.description}
                  onChange={(e) => handleInputChange('description', e.target.value)}
                  rows={3}
                />
              </div>
            </div>
          )}

          {currentStep === 2 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="schedule">Schedule (Optional)</Label>
                <Select
                  value={formData.schedule}
                  onValueChange={(value) => setFormData(prev => ({ ...prev, schedule: value }))}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="No schedule (manual backups)" />
                  </SelectTrigger>
                  <SelectContent>
                    {schedules.length === 0 ? (
                      <div className="px-2 py-1.5 text-xs text-muted-foreground">
                        No schedules available. Create one first.
                      </div>
                    ) : (
                      schedules.map((schedule) => (
                        <SelectItem key={schedule.id} value={schedule.id}>
                          {schedule.name} - {schedule.cron_expression}
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Without a schedule, backups will be manual only
                </p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="maxConcurrentVMs">Max Concurrent VMs</Label>
                  <Input
                    id="maxConcurrentVMs"
                    type="number"
                    min="1"
                    max="100"
                    value={formData.maxConcurrentVMs}
                    onChange={(e) => setFormData(prev => ({ ...prev, maxConcurrentVMs: parseInt(e.target.value) }))}
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

              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Group Configuration</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-sm space-y-1">
                    <div><strong>Schedule:</strong> {schedules.find(s => s.id === formData.schedule)?.name || <span className="text-muted-foreground">None (manual backups only)</span>}</div>
                    <div><strong>Max Concurrent VMs:</strong> {formData.maxConcurrentVMs}</div>
                    <div><strong>Priority:</strong> {formData.priority}</div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {currentStep === 3 && (
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-medium mb-4">Select Virtual Machines</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  Choose which VMs to include in this protection group ({formData.vmIds.length} selected)
                </p>

                {isLoadingVMs ? (
                  <div className="space-y-3">
                    {Array.from({ length: 3 }).map((_, i) => (
                      <div key={i} className="flex items-center space-x-3 p-3 rounded-lg border">
                        <div className="w-4 h-4 bg-muted rounded animate-pulse" />
                        <div className="flex items-center gap-3 flex-1">
                          <div className="w-4 h-4 bg-muted rounded animate-pulse" />
                          <div className="flex-1 space-y-2">
                            <div className="h-4 bg-muted rounded animate-pulse w-3/4" />
                            <div className="h-3 bg-muted rounded animate-pulse w-1/2" />
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : availableVMs.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    <Server className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>No ungrouped VMs available.</p>
                    <p className="text-sm">Use "Add VMs" to discover VMs from vCenter first.</p>
                  </div>
                ) : (
                  <div className="space-y-3 max-h-60 overflow-y-auto">
                    {availableVMs.map((vm) => (
                      <div
                        key={vm.context_id}
                        className="flex items-center space-x-3 p-3 rounded-lg border hover:bg-muted/50"
                      >
                        <Checkbox
                          id={`vm-${vm.context_id}`}
                          checked={formData.vmIds.includes(vm.context_id)}
                          onCheckedChange={(checked) => handleVMSelection(vm.context_id, checked as boolean)}
                        />
                        <div className="flex items-center gap-3 flex-1">
                          <Server className="h-4 w-4 text-muted-foreground" />
                          <div className="flex-1">
                            <div className="flex items-center gap-2">
                              <span className="font-medium">{vm.vm_name}</span>
                              {getStatusBadge(vm.power_state)}
                            </div>
                            <div className="text-sm text-muted-foreground">
                              {vm.datacenter} • vCenter: {vm.vcenter_host}
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {selectedVMs.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm">Selected VMs ({selectedVMs.length})</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {selectedVMs.map((vm) => (
                        <div key={vm.context_id} className="flex items-center gap-2 text-sm">
                          <CheckCircle className="h-4 w-4 text-green-500" />
                          <span>{vm.vm_name}</span>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={currentStep === 1 ? onClose : handlePrevious}
          >
            {currentStep === 1 ? 'Cancel' : 'Previous'}
          </Button>

          {currentStep < 3 ? (
            <Button
              type="button"
              onClick={handleNext}
              disabled={!canProceedToNext()}
            >
              Next
            </Button>
          ) : (
            <Button
              type="button"
              onClick={handleSubmit}
              disabled={!canProceedToNext() || isSubmitting}
            >
              {isSubmitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Create Group
            </Button>
          )}
        </DialogFooter>
      </DialogContent>

      {/* Create Schedule Modal */}
      <CreateScheduleModal
        isOpen={isCreateScheduleModalOpen}
        onClose={() => setIsCreateScheduleModalOpen(false)}
        onCreate={handleCreateSchedule}
        policyType={formData.policy as 'daily' | 'weekly' | 'monthly'}
      />
    </Dialog>
  );
}
