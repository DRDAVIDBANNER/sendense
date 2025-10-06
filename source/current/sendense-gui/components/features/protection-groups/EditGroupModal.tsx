"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Loader2, AlertCircle } from "lucide-react";

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

interface EditGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUpdate: () => void;
  group: ProtectionGroup | null;
  schedules: Schedule[];
}

export function EditGroupModal({ isOpen, onClose, onUpdate, group, schedules }: EditGroupModalProps) {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    schedule_id: '',
    max_concurrent_vms: 10,
    priority: 50,
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Populate form when group changes
  useEffect(() => {
    if (group) {
      setFormData({
        name: group.name,
        description: group.description || '',
        schedule_id: group.schedule_id || '',
        max_concurrent_vms: group.max_concurrent_vms,
        priority: group.priority,
      });
    }
  }, [group]);

  const handleSubmit = async () => {
    if (!group) return;

    if (!formData.name) {
      setError('Group name is required');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const response = await fetch(`/api/v1/machine-groups/${group.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: formData.name,
          description: formData.description || null,
          schedule_id: formData.schedule_id || null,
          max_concurrent_vms: formData.max_concurrent_vms,
          priority: formData.priority,
        }),
      });

      if (response.ok) {
        console.log('✅ Group updated:', group.id);
        onUpdate();
        onClose();
      } else {
        const errorResult = await response.json();
        setError(errorResult.error || 'Failed to update group');
      }
    } catch (err) {
      setError('Failed to update group');
      console.error('Error updating group:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Edit Protection Group</DialogTitle>
          <DialogDescription>
            Update group settings, schedule, and priority
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <span className="text-sm text-destructive">{error}</span>
          </div>
        )}

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="name">Group Name</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
              placeholder="Production Web Servers"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              placeholder="Group description..."
              rows={3}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="schedule">Schedule</Label>
            <Select
              value={formData.schedule_id}
              onValueChange={(value) => setFormData(prev => ({ ...prev, schedule_id: value }))}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select schedule (optional)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">No schedule</SelectItem>
                {schedules.map((schedule) => (
                  <SelectItem key={schedule.id} value={schedule.id}>
                    {schedule.name} - {schedule.cron_expression}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="maxConcurrentVMs">Max Concurrent VMs</Label>
              <Input
                id="maxConcurrentVMs"
                type="number"
                min="1"
                max="100"
                value={formData.max_concurrent_vms}
                onChange={(e) => setFormData(prev => ({ ...prev, max_concurrent_vms: parseInt(e.target.value) }))}
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
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            Update Group
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
