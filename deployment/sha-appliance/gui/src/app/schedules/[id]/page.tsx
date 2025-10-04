'use client';

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { VMCentricLayout } from '@/components/layout/VMCentricLayout';
import { Button, Alert, Badge } from 'flowbite-react';
import { 
  HiArrowLeft, 
  HiRefresh, 
  HiPlay, 
  HiStop, 
  HiClock, 
  HiExclamationCircle,
  HiCheckCircle,
  HiXCircle,
  HiInformationCircle
} from 'react-icons/hi';

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

interface ScheduleExecution {
  id: string;
  schedule_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  started_at?: string;
  completed_at?: string;
  execution_duration_seconds?: number;
  jobs_created: number;
  jobs_completed: number;
  jobs_failed: number;
  error_message?: string;
  summary?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

interface ScheduleExecutionListResponse {
  executions: ScheduleExecution[];
  total_count: number;
  page: number;
  page_size: number;
  has_more: boolean;
  retrieved_at: string;
}

// Helper functions
const formatScheduleDescription = (cronExp: string, timezone: string): string => {
  const parts = cronExp.split(' ');
  if (parts.length !== 6) return cronExp;
  
  const [second, minute, hour, dayOfMonth, month, dayOfWeek] = parts;
  
  if (dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const h = parseInt(hour);
    const m = parseInt(minute);
    const timeStr = formatTime(h, m);
    return `Daily at ${timeStr}`;
  }
  
  if (dayOfMonth === '*' && month === '*' && dayOfWeek !== '*') {
    const h = parseInt(hour);
    const m = parseInt(minute);
    const timeStr = formatTime(h, m);
    const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    const dayName = dayNames[parseInt(dayOfWeek)] || `Day ${dayOfWeek}`;
    return `Weekly on ${dayName} at ${timeStr}`;
  }
  
  if (dayOfMonth !== '*' && month === '*' && dayOfWeek === '*') {
    const h = parseInt(hour);
    const m = parseInt(minute);
    const timeStr = formatTime(h, m);
    const dayNum = parseInt(dayOfMonth);
    const suffix = dayNum === 1 ? 'st' : dayNum === 2 ? 'nd' : dayNum === 3 ? 'rd' : 'th';
    return `Monthly on the ${dayNum}${suffix} at ${timeStr}`;
  }
  
  return cronExp;
};

const formatTime = (hour: number, minute: number): string => {
  const h12 = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
  const ampm = hour >= 12 ? 'PM' : 'AM';
  const m = minute.toString().padStart(2, '0');
  return `${h12}:${m} ${ampm}`;
};

const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${minutes}m`;
};

const formatRelativeTime = (dateStr: string): string => {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  
  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
  return `${Math.floor(diffMins / 1440)}d ago`;
};

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'completed': return <HiCheckCircle className="h-5 w-5 text-green-500" />;
    case 'failed': return <HiXCircle className="h-5 w-5 text-red-500" />;
    case 'running': return <HiClock className="h-5 w-5 text-blue-500 animate-spin" />;
    case 'pending': return <HiClock className="h-5 w-5 text-yellow-500" />;
    case 'cancelled': return <HiXCircle className="h-5 w-5 text-gray-500" />;
    default: return <HiInformationCircle className="h-5 w-5 text-gray-500" />;
  }
};

const getStatusColor = (status: string): string => {
  switch (status) {
    case 'completed': return 'success';
    case 'failed': return 'failure';
    case 'running': return 'info';
    case 'pending': return 'warning';
    case 'cancelled': return 'gray';
    default: return 'gray';
  }
};

export default function ScheduleDetailPage() {
  const params = useParams();
  const router = useRouter();
  const scheduleId = params.id as string;
  
  const [schedule, setSchedule] = useState<Schedule | null>(null);
  const [executions, setExecutions] = useState<ScheduleExecution[]>([]);
  const [loading, setLoading] = useState(true);
  const [executionsLoading, setExecutionsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [totalCount, setTotalCount] = useState(0);

  const loadSchedule = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await fetch(`/api/schedules/${scheduleId}`);
      if (!response.ok) {
        throw new Error(`Failed to load schedule: ${response.statusText}`);
      }
      
      const data = await response.json();
      setSchedule(data.schedule);
    } catch (err) {
      console.error('Error loading schedule:', err);
      setError(err instanceof Error ? err.message : 'Failed to load schedule');
    } finally {
      setLoading(false);
    }
  };

  const loadExecutions = async (pageNum: number = 1, append: boolean = false) => {
    try {
      setExecutionsLoading(true);
      
      const response = await fetch(`/api/schedules/${scheduleId}/executions?page=${pageNum}&limit=20`);
      if (!response.ok) {
        throw new Error(`Failed to load executions: ${response.statusText}`);
      }
      
      const data: ScheduleExecutionListResponse = await response.json();
      
      if (append) {
        setExecutions(prev => [...prev, ...data.executions]);
      } else {
        setExecutions(data.executions);
      }
      
      setTotalCount(data.total_count);
      setHasMore(data.has_more);
      setPage(pageNum);
    } catch (err) {
      console.error('Error loading executions:', err);
      setError(err instanceof Error ? err.message : 'Failed to load executions');
    } finally {
      setExecutionsLoading(false);
    }
  };

  const triggerSchedule = async () => {
    try {
      setActionLoading(true);
      setError(null);
      
      const response = await fetch(`/api/schedules/${scheduleId}/trigger`, {
        method: 'POST',
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to trigger schedule: ${response.statusText}`);
      }
      
      // Reload executions to show the new one
      await loadExecutions();
    } catch (err) {
      console.error('Error triggering schedule:', err);
      setError(err instanceof Error ? err.message : 'Failed to trigger schedule');
    } finally {
      setActionLoading(false);
    }
  };

  const toggleSchedule = async () => {
    if (!schedule) return;
    
    try {
      setActionLoading(true);
      setError(null);
      
      const response = await fetch(`/api/schedules/${scheduleId}/enable`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !schedule.enabled }),
      });
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `Failed to update schedule: ${response.statusText}`);
      }
      
      // Reload schedule to get updated status
      await loadSchedule();
    } catch (err) {
      console.error('Error updating schedule:', err);
      setError(err instanceof Error ? err.message : 'Failed to update schedule');
    } finally {
      setActionLoading(false);
    }
  };

  const loadMoreExecutions = () => {
    if (hasMore && !executionsLoading) {
      loadExecutions(page + 1, true);
    }
  };

  useEffect(() => {
    if (scheduleId) {
      loadSchedule();
      loadExecutions();
    }
  }, [scheduleId]);

  // Auto-refresh executions every 30 seconds for real-time updates
  useEffect(() => {
    const interval = setInterval(() => {
      if (scheduleId && !executionsLoading) {
        loadExecutions(1, false);
      }
    }, 30000);

    return () => clearInterval(interval);
  }, [scheduleId, executionsLoading]);

  if (loading) {
    return (
      <VMCentricLayout>
        <div className="flex justify-center items-center h-64">
          <span>Loading schedule...</span>
        </div>
      </VMCentricLayout>
    );
  }

  if (!schedule) {
    return (
      <VMCentricLayout>
        <div className="p-6">
          <div className="text-center py-8">
            <h3 className="text-lg font-medium text-gray-900 mb-2">Schedule not found</h3>
            <Button onClick={() => router.push('/schedules')}>
              <HiArrowLeft className="mr-2 h-4 w-4" />
              Back to Schedules
            </Button>
          </div>
        </div>
      </VMCentricLayout>
    );
  }

  return (
    <VMCentricLayout>
      <div className="p-6">
        {/* Header */}
        <div className="flex justify-between items-start mb-6">
          <div className="flex items-center gap-4">
            <Button color="gray" onClick={() => router.push('/schedules')}>
              <HiArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{schedule.name}</h1>
              <div className="flex items-center gap-2 mt-1">
                <HiClock className="h-4 w-4 text-gray-500" />
                <span className="text-gray-600">
                  {formatScheduleDescription(schedule.cron_expression, schedule.timezone)}
                </span>
                <span className="text-gray-400">({schedule.timezone})</span>
              </div>
              {schedule.description && (
                <p className="text-gray-600 mt-1">{schedule.description}</p>
              )}
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <Badge color={schedule.enabled ? 'success' : 'gray'} size="lg">
              {schedule.enabled ? 'Enabled' : 'Disabled'}
            </Badge>
            <Button 
              color={schedule.enabled ? 'failure' : 'success'}
              onClick={toggleSchedule}
              disabled={actionLoading}
            >
              {schedule.enabled ? <HiStop className="mr-2 h-4 w-4" /> : <HiPlay className="mr-2 h-4 w-4" />}
              {schedule.enabled ? 'Disable' : 'Enable'}
            </Button>
            <Button onClick={triggerSchedule} disabled={actionLoading || !schedule.enabled}>
              <HiPlay className="mr-2 h-4 w-4" />
              Trigger Now
            </Button>
            <Button color="gray" onClick={() => loadExecutions()}>
              <HiRefresh className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <Alert color="failure" onDismiss={() => setError(null)} className="mb-4">
            <HiExclamationCircle className="h-4 w-4" />
            <span className="ml-2">{error}</span>
          </Alert>
        )}

        {/* Schedule Configuration */}
        <div className="bg-white border border-gray-200 rounded-lg p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Configuration</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-gray-500">Max Concurrent Jobs:</span>
              <div className="font-medium">{schedule.max_concurrent_jobs}</div>
            </div>
            <div>
              <span className="text-gray-500">Retry Attempts:</span>
              <div className="font-medium">{schedule.retry_attempts}</div>
            </div>
            <div>
              <span className="text-gray-500">Retry Delay:</span>
              <div className="font-medium">{schedule.retry_delay_minutes}m</div>
            </div>
            <div>
              <span className="text-gray-500">Cron Expression:</span>
              <div className="font-mono text-xs bg-gray-100 px-2 py-1 rounded">{schedule.cron_expression}</div>
            </div>
          </div>
        </div>

        {/* Execution History */}
        <div className="bg-white border border-gray-200 rounded-lg">
          <div className="p-6 border-b border-gray-200">
            <div className="flex justify-between items-center">
              <h2 className="text-lg font-semibold text-gray-900">Execution History</h2>
              <span className="text-sm text-gray-500">
                {totalCount} total execution{totalCount !== 1 ? 's' : ''}
              </span>
            </div>
          </div>
          
          <div className="divide-y divide-gray-200">
            {executions.length === 0 ? (
              <div className="p-8 text-center text-gray-500">
                <HiInformationCircle className="h-8 w-8 mx-auto mb-2 text-gray-400" />
                No executions found
              </div>
            ) : (
              executions.map((execution) => (
                <div key={execution.id} className="p-6">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                      {getStatusIcon(execution.status)}
                      <div>
                        <div className="flex items-center gap-2 mb-1">
                          <Badge color={getStatusColor(execution.status)}>
                            {execution.status.toUpperCase()}
                          </Badge>
                          <span className="text-sm text-gray-500">
                            {formatRelativeTime(execution.created_at)}
                          </span>
                        </div>
                        
                        <div className="text-sm text-gray-600 space-y-1">
                          {execution.started_at && (
                            <div>Started: {new Date(execution.started_at).toLocaleString()}</div>
                          )}
                          {execution.completed_at && (
                            <div>Completed: {new Date(execution.completed_at).toLocaleString()}</div>
                          )}
                          {execution.execution_duration_seconds && (
                            <div>Duration: {formatDuration(execution.execution_duration_seconds)}</div>
                          )}
                        </div>
                        
                        {execution.error_message && (
                          <div className="mt-2 p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">
                            <strong>Error:</strong> {execution.error_message}
                          </div>
                        )}
                      </div>
                    </div>
                    
                    <div className="text-right text-sm">
                      <div className="text-gray-500">Jobs</div>
                      <div className="space-y-1">
                        <div>Created: <span className="font-medium">{execution.jobs_created}</span></div>
                        <div>Completed: <span className="font-medium text-green-600">{execution.jobs_completed}</span></div>
                        <div>Failed: <span className="font-medium text-red-600">{execution.jobs_failed}</span></div>
                      </div>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
          
          {hasMore && (
            <div className="p-4 border-t border-gray-200 text-center">
              <Button 
                color="gray" 
                onClick={loadMoreExecutions}
                disabled={executionsLoading}
              >
                {executionsLoading ? 'Loading...' : 'Load More'}
              </Button>
            </div>
          )}
        </div>
      </div>
    </VMCentricLayout>
  );
}

