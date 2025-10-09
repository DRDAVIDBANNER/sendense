"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Flow, FlowType } from "./types";

interface EditFlowModalProps {
  isOpen: boolean;
  onClose: () => void;
  flow: Flow | null;
  onUpdate: (flowId: string, updates: Partial<Flow>) => void;
}

export function EditFlowModal({ isOpen, onClose, flow, onUpdate }: EditFlowModalProps) {
  const [formData, setFormData] = useState({
    name: '',
    type: 'backup' as FlowType,
    source: '',
    destination: '',
    nextRun: '',
    description: ''
  });

  // Populate form when flow changes
  useEffect(() => {
    if (flow) {
      setFormData({
        name: flow.name,
        type: flow.flow_type,
        source: flow.source || '',
        destination: flow.destination || '',
        nextRun: flow.nextRun?.split('T')[0] + 'T' + flow.nextRun?.split('T')[1]?.slice(0, 5) || '', // Format for datetime-local
        description: ''
      });
    }
  }, [flow]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!flow) return;

    // Update the flow
    const updates: Partial<Flow> = {
      name: formData.name,
      flow_type: formData.type,
      source: formData.source,
      destination: formData.destination,
      nextRun: formData.nextRun
    };

    onUpdate(flow.id, updates);

    onClose();
  };

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  if (!flow) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Edit Protection Flow</DialogTitle>
          <DialogDescription>
            Modify the settings for this protection flow.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="edit-name">Flow Name</Label>
              <Input
                id="edit-name"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-type">Flow Type</Label>
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
              <Label htmlFor="edit-source">Source</Label>
              <Select value={formData.source} onValueChange={(value) => handleInputChange('source', value)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="vCenter-ESXi-01">vCenter-ESXi-01</SelectItem>
                  <SelectItem value="vCenter-ESXi-02">vCenter-ESXi-02</SelectItem>
                  <SelectItem value="vCenter-ESXi-03">vCenter-ESXi-03</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-destination">Destination</Label>
              <Select value={formData.destination} onValueChange={(value) => handleInputChange('destination', value)}>
                <SelectTrigger>
                  <SelectValue />
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
            <Label htmlFor="edit-nextRun">Next Run Schedule</Label>
            <Input
              id="edit-nextRun"
              type="datetime-local"
              value={formData.nextRun}
              onChange={(e) => handleInputChange('nextRun', e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-description">Description (Optional)</Label>
            <Textarea
              id="edit-description"
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
            Update Flow
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
