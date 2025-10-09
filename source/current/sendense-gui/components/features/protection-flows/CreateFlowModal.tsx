"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Flow, FlowType } from "./types";

interface CreateFlowModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (flow: Omit<Flow, 'id' | 'status' | 'lastRun' | 'progress'>) => void;
}

export function CreateFlowModal({ isOpen, onClose, onCreate }: CreateFlowModalProps) {
  const [formData, setFormData] = useState({
    name: '',
    type: 'backup' as FlowType,
    source: '',
    destination: '',
    nextRun: '',
    description: ''
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    // Create a new flow object
    const newFlow = {
      name: formData.name,
      flow_type: formData.type as 'backup' | 'replication',
      target_type: 'vm' as const,
      target_id: formData.source || '',
      repository_id: formData.destination || '',
      enabled: true,
    };

    onCreate(newFlow as any);

    // Reset form and close modal
    setFormData({
      name: '',
      type: 'backup',
      source: '',
      destination: '',
      nextRun: '',
      description: ''
    });
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
              <Select value={formData.source} onValueChange={(value) => handleInputChange('source', value)}>
                <SelectTrigger>
                  <SelectValue placeholder="Select source" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="vCenter-ESXi-01">vCenter-ESXi-01</SelectItem>
                  <SelectItem value="vCenter-ESXi-02">vCenter-ESXi-02</SelectItem>
                  <SelectItem value="vCenter-ESXi-03">vCenter-ESXi-03</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="destination">Destination</Label>
              <Select value={formData.destination} onValueChange={(value) => handleInputChange('destination', value)}>
                <SelectTrigger>
                  <SelectValue placeholder="Select destination" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="CloudStack-Primary">CloudStack-Primary</SelectItem>
                  <SelectItem value="CloudStack-DR">CloudStack-DR</SelectItem>
                  <SelectItem value="CloudStack-Archive">CloudStack-Archive</SelectItem>
                  <SelectItem value="CloudStack-Dev">CloudStack-Dev</SelectItem>
                </SelectContent>
              </Select>
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
