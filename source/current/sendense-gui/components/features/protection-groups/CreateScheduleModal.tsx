"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
// Using Select instead of RadioGroup for simplicity
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Clock, Calendar } from "lucide-react";

interface ScheduleRequest {
  name: string;
  description: string;
  frequency: 'daily' | 'weekly' | 'monthly' | 'custom';
  time: string;
  dayOfWeek?: number; // 0-6 for weekly
  dayOfMonth?: number; // 1-31 for monthly
  cronExpression?: string; // For custom schedules
}

interface CreateScheduleModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (schedule: ScheduleRequest) => Promise<{ id: string; name: string; displayName: string }>;
  policyType?: 'daily' | 'weekly' | 'monthly'; // Optional filter based on policy
}

export function CreateScheduleModal({ isOpen, onClose, onCreate, policyType }: CreateScheduleModalProps) {
  const [formData, setFormData] = useState<ScheduleRequest>({
    name: '',
    description: '',
    frequency: policyType || 'daily',
    time: '02:00',
    dayOfWeek: 0,
    dayOfMonth: 1
  });
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleInputChange = (field: keyof ScheduleRequest, value: string | number) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleSubmit = async () => {
    if (!formData.name.trim()) return;

    setIsSubmitting(true);
    try {
      const result = await onCreate(formData);

      // Reset form
      setFormData({
        name: '',
        description: '',
        frequency: policyType || 'daily',
        time: '02:00',
        dayOfWeek: 0,
        dayOfMonth: 1
      });

      onClose();
    } catch (error) {
      console.error('Failed to create schedule:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const getFrequencyOptions = () => {
    if (policyType) {
      return [{ value: policyType, label: policyType.charAt(0).toUpperCase() + policyType.slice(1) }];
    }

    return [
      { value: 'daily', label: 'Daily' },
      { value: 'weekly', label: 'Weekly' },
      { value: 'monthly', label: 'Monthly' },
      { value: 'custom', label: 'Custom (Advanced)' }
    ];
  };

  const getSchedulePreview = () => {
    const timeFormatted = formData.time;
    const timeDisplay = new Date(`2000-01-01T${timeFormatted}`).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit'
    });

    switch (formData.frequency) {
      case 'daily':
        return `Daily at ${timeDisplay}`;
      case 'weekly':
        const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
        return `Weekly on ${days[formData.dayOfWeek || 0]} at ${timeDisplay}`;
      case 'monthly':
        return `Monthly on day ${formData.dayOfMonth} at ${timeDisplay}`;
      case 'custom':
        return `Custom schedule: ${formData.cronExpression || 'Advanced configuration'}`;
      default:
        return 'Invalid schedule';
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            Create New Schedule
          </DialogTitle>
          <DialogDescription>
            Define a custom backup schedule for your protection policies
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Basic Information */}
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="schedule-name">Schedule Name</Label>
              <Input
                id="schedule-name"
                placeholder="e.g., Business Hours Backup"
                value={formData.name}
                onChange={(e) => handleInputChange('name', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="schedule-description">Description (Optional)</Label>
              <Input
                id="schedule-description"
                placeholder="Brief description of this schedule"
                value={formData.description}
                onChange={(e) => handleInputChange('description', e.target.value)}
              />
            </div>
          </div>

          {/* Frequency Selection */}
          <div className="space-y-2">
            <Label htmlFor="frequency-select">Frequency</Label>
            <Select
              value={formData.frequency}
              onValueChange={(value) => handleInputChange('frequency', value as ScheduleRequest['frequency'])}
            >
              <SelectTrigger id="frequency-select">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {getFrequencyOptions().map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Time Selection */}
          <div className="space-y-2">
            <Label htmlFor="schedule-time">Time</Label>
            <Input
              id="schedule-time"
              type="time"
              value={formData.time}
              onChange={(e) => handleInputChange('time', e.target.value)}
            />
          </div>

          {/* Day Selection Based on Frequency */}
          {formData.frequency === 'weekly' && (
            <div className="space-y-2">
              <Label>Day of Week</Label>
              <Select
                value={formData.dayOfWeek?.toString()}
                onValueChange={(value) => handleInputChange('dayOfWeek', parseInt(value))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="0">Sunday</SelectItem>
                  <SelectItem value="1">Monday</SelectItem>
                  <SelectItem value="2">Tuesday</SelectItem>
                  <SelectItem value="3">Wednesday</SelectItem>
                  <SelectItem value="4">Thursday</SelectItem>
                  <SelectItem value="5">Friday</SelectItem>
                  <SelectItem value="6">Saturday</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          {formData.frequency === 'monthly' && (
            <div className="space-y-2">
              <Label>Day of Month</Label>
              <Select
                value={formData.dayOfMonth?.toString()}
                onValueChange={(value) => handleInputChange('dayOfMonth', parseInt(value))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Array.from({ length: 31 }, (_, i) => i + 1).map((day) => (
                    <SelectItem key={day} value={day.toString()}>
                      {day}{day === 1 ? 'st' : day === 2 ? 'nd' : day === 3 ? 'rd' : 'th'}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {formData.frequency === 'custom' && (
            <div className="space-y-2">
              <Label htmlFor="cron-expression">Cron Expression</Label>
              <Input
                id="cron-expression"
                placeholder="e.g., 0 2 * * 1-5 (Mon-Fri at 2 AM)"
                value={formData.cronExpression || ''}
                onChange={(e) => handleInputChange('cronExpression', e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Advanced users only. Format: minute hour day month day-of-week
              </p>
            </div>
          )}

          {/* Schedule Preview */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Clock className="h-4 w-4" />
                Schedule Preview
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm font-medium">{getSchedulePreview()}</p>
              {formData.description && (
                <p className="text-xs text-muted-foreground mt-1">{formData.description}</p>
              )}
            </CardContent>
          </Card>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!formData.name.trim() || isSubmitting}
          >
            {isSubmitting ? 'Creating...' : 'Create Schedule'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
