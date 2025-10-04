'use client';

import React, { useState, useEffect } from 'react';
import { VMCentricLayout } from '@/components/layout/VMCentricLayout';
import { Button, Alert } from 'flowbite-react';
import { HiRefresh, HiPlus, HiExclamationCircle, HiClock, HiPencil, HiTrash } from 'react-icons/hi';

interface Schedule {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
  cron_expression: string;
  timezone: string;
  max_concurrent_jobs: number;
  retry_attempts: number;
  retry_delay_minutes: number;
}

interface CreateScheduleForm {
  name: string;
  description: string;
  cron_expression: string;
  timezone: string;
  max_concurrent_jobs: number;
  retry_attempts: number;
  retry_delay_minutes: number;
  skip_if_running: boolean;
  enabled: boolean;
  // Enhanced UI helpers for schedule building
  frequency: 'interval' | 'daily' | 'weekly' | 'monthly' | 'custom';
  // Interval options (every X minutes/hours/days)
  interval_value: number;
  interval_unit: 'minutes' | 'hours' | 'days';
  // Time options
  time_hour: number;
  time_minute: number;
  weekly_days: string[];
  monthly_day: number;
}

// Helper functions for schedule formatting
const formatScheduleDescription = (cronExp: string, timezone: string): string => {
  // Parse basic cron patterns and return human-readable description
  const parts = cronExp.split(' ');
  if (parts.length !== 6) return cronExp; // Return original if not standard format
  
  const [second, minute, hour, dayOfMonth, month, dayOfWeek] = parts;
  
  // Daily pattern: "0 M H * * *"
  if (dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const h = parseInt(hour);
    const m = parseInt(minute);
    const timeStr = formatTime(h, m);
    return `Daily at ${timeStr}`;
  }
  
  // Weekly pattern: "0 M H * * DOW"
  if (dayOfMonth === '*' && month === '*' && dayOfWeek !== '*') {
    const h = parseInt(hour);
    const m = parseInt(minute);
    const timeStr = formatTime(h, m);
    const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    const dayName = dayNames[parseInt(dayOfWeek)] || `Day ${dayOfWeek}`;
    return `Weekly on ${dayName} at ${timeStr}`;
  }
  
  // Monthly pattern: "0 M H D * *"
  if (dayOfMonth !== '*' && month === '*' && dayOfWeek === '*') {
    const h = parseInt(hour);
    const m = parseInt(minute);
    const timeStr = formatTime(h, m);
    const dayNum = parseInt(dayOfMonth);
    const suffix = dayNum === 1 ? 'st' : dayNum === 2 ? 'nd' : dayNum === 3 ? 'rd' : 'th';
    return `Monthly on the ${dayNum}${suffix} at ${timeStr}`;
  }
  
  return cronExp; // Return original for complex patterns
};

const formatTime = (hour: number, minute: number): string => {
  const h12 = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
  const ampm = hour >= 12 ? 'PM' : 'AM';
  const m = minute.toString().padStart(2, '0');
  return `${h12}:${m} ${ampm}`;
};

const buildCronExpression = (formData: CreateScheduleForm): string => {
  const { frequency, time_hour, time_minute, weekly_days, monthly_day, interval_value, interval_unit } = formData;
  
  if (frequency === 'custom') {
    return formData.cron_expression;
  }
  
  const second = '0';
  const minute = time_minute.toString();
  const hour = time_hour.toString();
  
  switch (frequency) {
    case 'interval':
      // Handle "every X minutes/hours/days"
      switch (interval_unit) {
        case 'minutes':
          return `0 */${interval_value} * * * *`;
        case 'hours':
          return `0 0 */${interval_value} * * *`;
        case 'days':
          return `0 ${minute} ${hour} */${interval_value} * *`;
        default:
          return `0 */${interval_value} * * * *`;
      }
    case 'daily':
      return `${second} ${minute} ${hour} * * *`;
    case 'weekly':
      const dayOfWeek = weekly_days.length > 0 ? weekly_days[0] : '1';
      return `${second} ${minute} ${hour} * * ${dayOfWeek}`;
    case 'monthly':
      return `${second} ${minute} ${hour} ${monthly_day} * *`;
    default:
      return `${second} ${minute} ${hour} * * *`;
  }
};

export default function SchedulesPage() {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [formData, setFormData] = useState<CreateScheduleForm>({
    name: '',
    description: '',
    cron_expression: '0 0 2 * * *',
    timezone: 'Europe/London', // Fix timezone to match server
    max_concurrent_jobs: 5,
    retry_attempts: 3,
    retry_delay_minutes: 15,
    skip_if_running: true,
    enabled: true,
    frequency: 'daily',
    // Interval options
    interval_value: 30,
    interval_unit: 'minutes',
    // Time options
    time_hour: 2,
    time_minute: 0,
    weekly_days: ['1'],
    monthly_day: 1,
  });

  const loadSchedules = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await fetch('/api/schedules');
      if (!response.ok) {
        throw new Error(`Failed to load schedules: ${response.statusText}`);
      }
      
      const data = await response.json();
      setSchedules(data.schedules || []);
    } catch (err) {
      console.error('Error loading schedules:', err);
      setError(err instanceof Error ? err.message : 'Failed to load schedules');
    } finally {
      setLoading(false);
    }
  };

  const createSchedule = async () => {
    try {
      setActionLoading(true);
      setError(null);
      
      // Build cron expression from UI inputs
      const cronExpression = buildCronExpression(formData);
      
      const response = await fetch('/api/schedules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: formData.name,
          description: formData.description,
          cron_expression: cronExpression,
          timezone: formData.timezone,
          max_concurrent_jobs: formData.max_concurrent_jobs,
          retry_attempts: formData.retry_attempts,
          retry_delay_minutes: formData.retry_delay_minutes,
          skip_if_running: formData.skip_if_running,
          enabled: formData.enabled,
          created_by: 'web-gui'
        }),
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to create schedule: ${response.statusText}`);
      }
      
      await loadSchedules();
      setShowCreateModal(false);
      resetForm();
    } catch (err) {
      console.error('Error creating schedule:', err);
      setError(err instanceof Error ? err.message : 'Failed to create schedule');
    } finally {
      setActionLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      cron_expression: '0 0 2 * * *',
      timezone: 'Europe/London', // Fix timezone to match server
      max_concurrent_jobs: 5,
      retry_attempts: 3,
      retry_delay_minutes: 15,
      skip_if_running: true,
      enabled: true,
      frequency: 'daily',
      // Interval options
      interval_value: 30,
      interval_unit: 'minutes',
      // Time options
      time_hour: 2,
      time_minute: 0,
      weekly_days: ['1'],
      monthly_day: 1,
    });
  };

  const parseScheduleToForm = (schedule: Schedule): CreateScheduleForm => {
    // Parse cron expression to determine frequency and settings
    const parts = schedule.cron_expression.split(' ');
    
    // Default form data
    const formData: CreateScheduleForm = {
      name: schedule.name,
      description: schedule.description || '',
      cron_expression: schedule.cron_expression,
      timezone: schedule.timezone,
      max_concurrent_jobs: schedule.max_concurrent_jobs,
      retry_attempts: schedule.retry_attempts,
      retry_delay_minutes: schedule.retry_delay_minutes,
      skip_if_running: true, // Default, not stored in schedule
      enabled: schedule.enabled,
      frequency: 'custom', // Default to custom
      interval_value: 30,
      interval_unit: 'minutes',
      time_hour: 2,
      time_minute: 0,
      weekly_days: ['1'],
      monthly_day: 1,
    };

    if (parts.length === 6) {
      const [second, minute, hour, dayOfMonth, month, dayOfWeek] = parts;
      
      // Try to detect interval patterns
      if (minute.startsWith('*/')) {
        formData.frequency = 'interval';
        formData.interval_value = parseInt(minute.substring(2));
        formData.interval_unit = 'minutes';
      } else if (hour.startsWith('*/')) {
        formData.frequency = 'interval';
        formData.interval_value = parseInt(hour.substring(2));
        formData.interval_unit = 'hours';
      } else if (dayOfMonth.startsWith('*/')) {
        formData.frequency = 'interval';
        formData.interval_value = parseInt(dayOfMonth.substring(2));
        formData.interval_unit = 'days';
        formData.time_hour = parseInt(hour) || 2;
        formData.time_minute = parseInt(minute) || 0;
      }
      // Daily pattern: "0 M H * * *"
      else if (dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
        formData.frequency = 'daily';
        formData.time_hour = parseInt(hour) || 2;
        formData.time_minute = parseInt(minute) || 0;
      }
      // Weekly pattern: "0 M H * * D"
      else if (dayOfMonth === '*' && month === '*' && dayOfWeek !== '*') {
        formData.frequency = 'weekly';
        formData.time_hour = parseInt(hour) || 2;
        formData.time_minute = parseInt(minute) || 0;
        formData.weekly_days = [dayOfWeek];
      }
      // Monthly pattern: "0 M H D * *"
      else if (month === '*' && dayOfWeek === '*' && dayOfMonth !== '*') {
        formData.frequency = 'monthly';
        formData.time_hour = parseInt(hour) || 2;
        formData.time_minute = parseInt(minute) || 0;
        formData.monthly_day = parseInt(dayOfMonth) || 1;
      }
    }

    return formData;
  };

  const handleEditSchedule = (schedule: Schedule) => {
    // Parse the schedule data and populate the form
    const parsedData = parseScheduleToForm(schedule);
    setFormData(parsedData);
    setEditingSchedule(schedule);
    setShowCreateModal(true);
  };

  const handleDeleteSchedule = async (scheduleId: string) => {
    const schedule = schedules.find(s => s.id === scheduleId);
    const scheduleName = schedule?.name || 'this schedule';
    
    if (!confirm(`Are you sure you want to delete "${scheduleName}"?\n\nThis action cannot be undone.`)) {
      return;
    }

    try {
      setActionLoading(true);
      setError(null);
      
      const response = await fetch(`/api/schedules/${scheduleId}`, {
        method: 'DELETE',
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        
        // Debug: Log the error response
        console.log('Delete error response:', { status: response.status, errorData });
        
        // Handle specific error cases
        if (response.status === 409) {
          // Parse the conflict error to show which groups are using the schedule
          const errorMsg = errorData.error || errorData.message || 'Schedule is in use';
          console.log('409 Conflict error message:', errorMsg);
          
          if (errorMsg.includes('machine groups')) {
            throw new Error(`Cannot delete "${scheduleName}": This schedule is still assigned to machine groups. Please unassign it from all groups first, or delete the groups that use this schedule.`);
          } else {
            throw new Error(`Cannot delete "${scheduleName}": ${errorMsg}`);
          }
        }
        
        throw new Error(errorData.error || errorData.message || `Failed to delete schedule: ${response.statusText}`);
      }
      
      await loadSchedules();
    } catch (err) {
      console.error('Error deleting schedule:', err);
      setError(err instanceof Error ? err.message : 'Failed to delete schedule');
    } finally {
      setActionLoading(false);
    }
  };

  const updateSchedule = async () => {
    if (!editingSchedule) return;

    try {
      setActionLoading(true);
      setError(null);
      
      // Build cron expression from UI inputs
      const cronExpression = buildCronExpression(formData);
      
      const response = await fetch(`/api/schedules/${editingSchedule.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: formData.name,
          description: formData.description,
          cron_expression: cronExpression,
          timezone: formData.timezone,
          max_concurrent_jobs: formData.max_concurrent_jobs,
          retry_attempts: formData.retry_attempts,
          retry_delay_minutes: formData.retry_delay_minutes,
          skip_if_running: formData.skip_if_running,
          enabled: formData.enabled,
        }),
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to update schedule: ${response.statusText}`);
      }
      
      await loadSchedules();
      setShowCreateModal(false);
      setEditingSchedule(null);
      resetForm();
    } catch (err) {
      console.error('Error updating schedule:', err);
      setError(err instanceof Error ? err.message : 'Failed to update schedule');
    } finally {
      setActionLoading(false);
    }
  };

  useEffect(() => {
    loadSchedules();
  }, []);

  return (
    <VMCentricLayout>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white">Replication Schedules</h1>
            <p className="text-gray-300">Manage automated replication schedules</p>
          </div>
          <div className="flex gap-2">
            <Button color="gray" onClick={loadSchedules} disabled={loading}>
              <HiRefresh className="mr-2 h-4 w-4" />
              Refresh
            </Button>
            <Button onClick={() => setShowCreateModal(true)}>
              <HiPlus className="mr-2 h-4 w-4" />
              Create Schedule
            </Button>
          </div>
        </div>

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

        {loading ? (
          <div className="flex justify-center items-center h-64">
            <span className="text-gray-300">Loading schedules...</span>
          </div>
        ) : (
          <div className="grid gap-4">
            {schedules.length === 0 ? (
              <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
                <div className="text-center py-8">
                  <h3 className="text-lg font-medium text-white mb-2">No schedules found</h3>
                  <p className="text-gray-300 mb-4">Create your first replication schedule to get started.</p>
                  <Button onClick={() => setShowCreateModal(true)} className="mx-auto">
                    <HiPlus className="mr-2 h-4 w-4" />
                    Create Schedule
                  </Button>
                </div>
              </div>
            ) : (
              schedules.map((schedule) => (
                <div key={schedule.id} className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-800/70 transition-all duration-200">
                  <div className="flex justify-between items-start">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className="text-lg font-semibold text-white hover:text-cyan-300 cursor-pointer">
                          <a href={`/schedules/${schedule.id}`}>{schedule.name}</a>
                        </h3>
                      </div>
                      {schedule.description && (
                        <p className="text-gray-300 mb-2">{schedule.description}</p>
                      )}
                      <div className="flex items-center gap-2 text-sm text-gray-300 mb-2">
                        <HiClock className="h-4 w-4 text-cyan-400" />
                        <span className="font-medium">{formatScheduleDescription(schedule.cron_expression, schedule.timezone)}</span>
                        <span className="text-gray-400">({schedule.timezone})</span>
                      </div>
                      <div className="flex items-center gap-4 text-sm text-gray-400">
                        <span>Max jobs: {schedule.max_concurrent_jobs}</span>
                        <span>Retries: {schedule.retry_attempts}</span>
                        <span className="text-xs text-gray-500">{schedule.cron_expression}</span>
                      </div>
                      <p className="text-sm text-gray-400 mt-1">
                        Status: {schedule.enabled ? (
                          <span className="text-emerald-400 font-medium">Enabled</span>
                        ) : (
                          <span className="text-gray-500">Disabled</span>
                        )}
                      </p>
                    </div>
                    
                    {/* Action Buttons */}
                    <div className="flex items-center gap-2 ml-4">
                      <Button
                        size="sm"
                        color="gray"
                        onClick={() => handleEditSchedule(schedule)}
                        className="flex items-center gap-1"
                      >
                        <HiPencil className="h-4 w-4" />
                        Edit
                      </Button>
                      <Button
                        size="sm"
                        color="failure"
                        onClick={() => handleDeleteSchedule(schedule.id)}
                        className="flex items-center gap-1"
                      >
                        <HiTrash className="h-4 w-4" />
                        Delete
                      </Button>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {showCreateModal && (
          <div 
            className="fixed inset-0 z-50 bg-black bg-opacity-75 flex items-center justify-center p-4"
            onClick={() => setShowCreateModal(false)}
          >
            <div 
              className="bg-slate-800 border border-slate-700 rounded-lg shadow-xl max-w-md w-full max-h-90vh overflow-y-auto"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="bg-slate-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                <div className="flex justify-between items-center mb-6">
                  <h3 className="text-lg font-medium text-white">
                    {editingSchedule ? 'Edit Schedule' : 'Create New Schedule'}
                  </h3>
                  <button
                    onClick={() => setShowCreateModal(false)}
                    className="text-gray-400 hover:text-gray-200 text-2xl font-bold leading-none"
                  >
                    ×
                  </button>
                </div>
                
                <div className="space-y-6">
                  <div>
                    <label htmlFor="name" className="block text-sm font-medium text-gray-300 mb-2">
                      Schedule Name
                    </label>
                    <input
                      id="name"
                      type="text"
                      placeholder="Enter schedule name"
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
                    <input
                      id="description"
                      type="text"
                      placeholder="Enter description (optional)"
                      value={formData.description}
                      onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                      className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                    />
                  </div>
                  
                  {/* Schedule Builder */}
                  <div className="bg-slate-700/50 border border-slate-600 rounded-lg p-4">
                    <label className="block text-sm font-medium text-gray-300 mb-3">
                      Schedule Frequency
                    </label>
                    
                    {/* Frequency Selector */}
                    <div className="grid grid-cols-5 gap-2 mb-4">
                      {(['interval', 'daily', 'weekly', 'monthly', 'custom'] as const).map((freq) => (
                        <button
                          key={freq}
                          type="button"
                          onClick={() => setFormData(prev => ({ ...prev, frequency: freq }))}
                          className={`px-3 py-2 text-sm font-medium rounded-md border transition-colors ${
                            formData.frequency === freq
                              ? 'bg-cyan-600 text-white border-cyan-600'
                              : 'bg-slate-600 text-gray-300 border-slate-500 hover:bg-slate-500'
                          }`}
                        >
                          {freq === 'interval' ? 'Every X' : freq.charAt(0).toUpperCase() + freq.slice(1)}
                        </button>
                      ))}
                    </div>

                    {/* Interval Options - Show when 'interval' is selected */}
                    {formData.frequency === 'interval' && (
                      <div className="mb-4 p-3 bg-cyan-500/10 rounded-md border border-cyan-500/30">
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                          Run every:
                        </label>
                        <div className="flex items-center gap-2">
                          <input
                            type="number"
                            min="1"
                            max="999"
                            value={formData.interval_value}
                            onChange={(e) => setFormData(prev => ({ 
                              ...prev, 
                              interval_value: parseInt(e.target.value) || 1 
                            }))}
                            className="w-20 px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                          />
                          <select
                            value={formData.interval_unit}
                            onChange={(e) => setFormData(prev => ({ 
                              ...prev, 
                              interval_unit: e.target.value as 'minutes' | 'hours' | 'days'
                            }))}
                            className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                          >
                            <option value="minutes">minutes</option>
                            <option value="hours">hours</option>
                            <option value="days">days</option>
                          </select>
                        </div>
                        <p className="text-xs text-cyan-400 mt-1">
                          Next run: Every {formData.interval_value} {formData.interval_unit}
                        </p>
                      </div>
                    )}
                    
                    {/* Time Picker */}
                    {formData.frequency !== 'custom' && (
                      <div className="mb-4">
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                          Time
                        </label>
                        <div className="flex items-center gap-2">
                          <select
                            value={formData.time_hour}
                            onChange={(e) => setFormData(prev => ({ ...prev, time_hour: parseInt(e.target.value) }))}
                            className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500"
                          >
                            {Array.from({ length: 24 }, (_, i) => (
                              <option key={i} value={i}>
                                {formatTime(i, 0).split(':')[0]} {formatTime(i, 0).split(' ')[1]}
                              </option>
                            ))}
                          </select>
                          <span className="text-gray-400">:</span>
                          <select
                            value={formData.time_minute}
                            onChange={(e) => setFormData(prev => ({ ...prev, time_minute: parseInt(e.target.value) }))}
                            className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500"
                          >
                            {[0, 15, 30, 45].map((min) => (
                              <option key={min} value={min}>
                                {min.toString().padStart(2, '0')}
                              </option>
                            ))}
                          </select>
                        </div>
                      </div>
                    )}
                    
                    {/* Weekly Day Selector */}
                    {formData.frequency === 'weekly' && (
                      <div className="mb-4">
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                          Day of Week
                        </label>
                        <div className="grid grid-cols-7 gap-1">
                          {[
                            { value: '0', label: 'Sun' },
                            { value: '1', label: 'Mon' },
                            { value: '2', label: 'Tue' },
                            { value: '3', label: 'Wed' },
                            { value: '4', label: 'Thu' },
                            { value: '5', label: 'Fri' },
                            { value: '6', label: 'Sat' }
                          ].map((day) => (
                            <button
                              key={day.value}
                              type="button"
                              onClick={() => setFormData(prev => ({ ...prev, weekly_days: [day.value] }))}
                              className={`px-2 py-1 text-xs font-medium rounded border transition-colors ${
                                formData.weekly_days.includes(day.value)
                                  ? 'bg-cyan-600 text-white border-cyan-600'
                                  : 'bg-slate-600 text-gray-300 border-slate-500 hover:bg-slate-500'
                              }`}
                            >
                              {day.label}
                            </button>
                          ))}
                        </div>
                      </div>
                    )}
                    
                    {/* Monthly Day Selector */}
                    {formData.frequency === 'monthly' && (
                      <div className="mb-4">
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                          Day of Month
                        </label>
                        <select
                          value={formData.monthly_day}
                          onChange={(e) => setFormData(prev => ({ ...prev, monthly_day: parseInt(e.target.value) }))}
                          className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500"
                        >
                          {Array.from({ length: 31 }, (_, i) => i + 1).map((day) => (
                            <option key={day} value={day}>
                              {day}{day === 1 ? 'st' : day === 2 ? 'nd' : day === 3 ? 'rd' : 'th'}
                            </option>
                          ))}
                        </select>
                      </div>
                    )}
                    
                    {/* Custom Cron Input */}
                    {formData.frequency === 'custom' && (
                      <div className="mb-4">
                        <label htmlFor="custom-cron" className="block text-sm font-medium text-gray-300 mb-2">
                          Cron Expression
                        </label>
                        <input
                          id="custom-cron"
                          type="text"
                          placeholder="0 0 2 * * *"
                          value={formData.cron_expression}
                          onChange={(e) => setFormData(prev => ({ ...prev, cron_expression: e.target.value }))}
                          className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm font-mono"
                        />
                        <p className="text-xs text-gray-400 mt-1">
                          Format: second minute hour day-of-month month day-of-week
                        </p>
                      </div>
                    )}
                    
                    {/* Schedule Preview */}
                    <div className="bg-cyan-500/10 border border-cyan-500/30 rounded p-3">
                      <div className="flex items-center gap-2 text-sm text-cyan-300">
                        <HiClock className="h-4 w-4" />
                        <span className="font-medium">
                          {formData.frequency === 'custom' 
                            ? formatScheduleDescription(formData.cron_expression, formData.timezone)
                            : formatScheduleDescription(buildCronExpression(formData), formData.timezone)
                          }
                        </span>
                      </div>
                      <p className="text-xs text-cyan-400 mt-1">
                        Cron: {formData.frequency === 'custom' ? formData.cron_expression : buildCronExpression(formData)}
                      </p>
                    </div>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label htmlFor="timezone" className="block text-sm font-medium text-gray-300 mb-2">
                        Timezone
                      </label>
                      <select
                        id="timezone"
                        value={formData.timezone}
                        onChange={(e) => setFormData(prev => ({ ...prev, timezone: e.target.value }))}
                        required
                        className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                      >
                        <option value="UTC">UTC</option>
                        <option value="America/New_York">Eastern Time</option>
                        <option value="America/Chicago">Central Time</option>
                        <option value="America/Denver">Mountain Time</option>
                        <option value="America/Los_Angeles">Pacific Time</option>
                      </select>
                    </div>
                    <div>
                      <label htmlFor="max_jobs" className="block text-sm font-medium text-gray-300 mb-2">
                        Max Jobs
                      </label>
                      <input
                        id="max_jobs"
                        type="number"
                        min="1"
                        max="20"
                        value={formData.max_concurrent_jobs.toString()}
                        onChange={(e) => setFormData(prev => ({ ...prev, max_concurrent_jobs: parseInt(e.target.value) || 5 }))}
                        required
                        className="block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md shadow-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 sm:text-sm"
                      />
                    </div>
                  </div>
                  
                  <div className="flex items-center gap-6">
                    <div className="flex items-center gap-2">
                      <input
                        id="skip_if_running"
                        type="checkbox"
                        checked={formData.skip_if_running}
                        onChange={(e) => setFormData(prev => ({ ...prev, skip_if_running: e.target.checked }))}
                        className="h-4 w-4 text-cyan-500 focus:ring-cyan-500 bg-slate-700 border-slate-600 rounded"
                      />
                      <label htmlFor="skip_if_running" className="text-sm font-medium text-gray-300">
                        Skip if running
                      </label>
                    </div>
                    <div className="flex items-center gap-2">
                      <input
                        id="enabled"
                        type="checkbox"
                        checked={formData.enabled}
                        onChange={(e) => setFormData(prev => ({ ...prev, enabled: e.target.checked }))}
                        className="h-4 w-4 text-cyan-500 focus:ring-cyan-500 bg-slate-700 border-slate-600 rounded"
                      />
                      <label htmlFor="enabled" className="text-sm font-medium text-gray-300">
                        Enabled
                      </label>
                    </div>
                  </div>
                </div>
              </div>
              
              <div className="bg-slate-700/50 border-t border-slate-600 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse gap-3">
                <button
                  onClick={editingSchedule ? updateSchedule : createSchedule}
                  disabled={actionLoading || !formData.name}
                  className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-cyan-600 text-base font-medium text-white hover:bg-cyan-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {actionLoading ? (editingSchedule ? 'Updating...' : 'Creating...') : (editingSchedule ? 'Update Schedule' : 'Create Schedule')}
                </button>
                <button
                  onClick={() => { 
                    setShowCreateModal(false); 
                    setEditingSchedule(null); 
                    resetForm(); 
                  }}
                  className="mt-3 w-full inline-flex justify-center rounded-md border border-slate-600 shadow-sm px-4 py-2 bg-slate-700 text-base font-medium text-gray-300 hover:bg-slate-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm transition-colors"
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