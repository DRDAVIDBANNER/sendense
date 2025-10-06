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
import { Server, CheckCircle, Plus, Loader2 } from "lucide-react";
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

interface CreateGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (groupData: {
    name: string;
    description: string;
    policy: string;
    schedule: string;
    vmIds: string[];
  }) => void;
}

export function CreateGroupModal({ isOpen, onClose, onCreate }: CreateGroupModalProps) {
  // State for available VMs loaded from API
  const [availableVMs, setAvailableVMs] = useState<VMContext[]>([]);
  const [isLoadingVMs, setIsLoadingVMs] = useState(false);
  const [currentStep, setCurrentStep] = useState(1);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    policy: 'daily',
    schedule: '',
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
          const vms = await response.json();
          setAvailableVMs(vms);
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

  const handleSubmit = () => {
    onCreate(formData);
    // Reset form
    setFormData({
      name: '',
      description: '',
      policy: 'daily',
      schedule: '',
      vmIds: []
    });
    setCurrentStep(1);
    onClose();
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
        return formData.policy && formData.schedule;
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
                <Label htmlFor="backup-policy">Backup Policy</Label>
                <Select value={formData.policy} onValueChange={(value) => handleInputChange('policy', value)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="daily">Daily Backup</SelectItem>
                    <SelectItem value="weekly">Weekly Backup</SelectItem>
                    <SelectItem value="monthly">Monthly Archive</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="backup-schedule">Schedule</Label>
                <Select value={formData.schedule} onValueChange={(value) => {
                  if (value === 'create-new') {
                    setIsCreateScheduleModalOpen(true);
                  } else {
                    handleInputChange('schedule', value);
                  }
                }}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select backup schedule" />
                  </SelectTrigger>
                  <SelectContent>
                    {formData.policy === 'daily' && (
                      <>
                        <SelectItem value="daily-02:00">Daily at 02:00 AM</SelectItem>
                        <SelectItem value="daily-22:00">Daily at 10:00 PM</SelectItem>
                        <SelectItem value="every-6h">Every 6 hours</SelectItem>
                        <SelectItem value="every-12h">Every 12 hours</SelectItem>
                      </>
                    )}
                    {formData.policy === 'weekly' && (
                      <>
                        <SelectItem value="weekly-sunday-03:00">Weekly on Sunday at 03:00 AM</SelectItem>
                        <SelectItem value="weekly-saturday-02:00">Weekly on Saturday at 02:00 AM</SelectItem>
                      </>
                    )}
                    {formData.policy === 'monthly' && (
                      <>
                        <SelectItem value="monthly-1st-04:00">Monthly on 1st at 04:00 AM</SelectItem>
                        <SelectItem value="monthly-15th-04:00">Monthly on 15th at 04:00 AM</SelectItem>
                      </>
                    )}
                    <SelectItem value="create-new" className="border-t mt-2 pt-2">
                      <div className="flex items-center gap-2 text-primary">
                        <Plus className="h-4 w-4" />
                        Create New Schedule
                      </div>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Policy Summary</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-sm space-y-1">
                    <div><strong>Policy:</strong> {formData.policy.charAt(0).toUpperCase() + formData.policy.slice(1)} Backup</div>
                    <div><strong>Schedule:</strong> {formData.schedule.replace(/-/g, ' ').replace(/(\d{2}):(\d{2})/, '$1:$2')}</div>
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
              disabled={!canProceedToNext()}
            >
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
