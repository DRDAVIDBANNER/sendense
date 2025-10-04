'use client';

import { useQuery } from '@tanstack/react-query';

export interface JobProgress {
  id: string;
  vm_name: string;
  status: string;
  progress_percent: number;
  current_operation: string;
  bytes_transferred: number;
  total_bytes: number;
  transfer_speed_bps: number;
  vma_throughput_mbps: number;
  vma_eta_seconds: number | null;
  replication_type: string;
  created_at: string;
  started_at: string | null;
  completed_at: string | null;
  updated_at: string;
  error_message: string | null;
}

// Grace period for new jobs (15 seconds) to avoid false error notifications
const JOB_TRANSITION_GRACE_PERIOD_MS = 15000;

// Helper function to check if a job is in grace period
function isJobInGracePeriod(createdAt: string): boolean {
  const jobCreatedTime = new Date(createdAt).getTime();
  const currentTime = Date.now();
  const timeSinceCreation = currentTime - jobCreatedTime;
  
  return timeSinceCreation < JOB_TRANSITION_GRACE_PERIOD_MS;
}

// Helper function to check if any recent jobs are in grace period
function hasJobsInGracePeriod(jobs: JobProgress[]): boolean {
  return jobs.some(job => isJobInGracePeriod(job.created_at));
}

export function useJobProgress() {
  return useQuery<JobProgress[]>({
    queryKey: ['jobProgress'],
    queryFn: async () => {
      const response = await fetch('/api/migrations');
      
      if (!response.ok) {
        throw new Error(`Failed to fetch job progress: ${response.statusText}`);
      }
      
      return response.json();
    },
    refetchInterval: 2000, // Poll every 2 seconds for real-time updates
    staleTime: 1000, // Consider data stale after 1 second
    retry: (failureCount, error) => {
      // Enhanced retry logic with grace period consideration
      
      // Always retry network errors during grace period
      if (failureCount < 3) {
        console.log(`🔄 Job progress fetch failed (attempt ${failureCount + 1}/3), retrying...`);
        return true;
      }
      
      // After max retries, check if we should suppress the error
      console.warn(`❌ Job progress fetch failed after ${failureCount} attempts:`, error);
      return false;
    },
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 5000),
    // Suppress query errors during grace period to avoid false error notifications
    throwOnError: (error, query) => {
      // Get previous data to check for grace period jobs
      const previousData = query.state.data as JobProgress[] | undefined;
      
      if (previousData && hasJobsInGracePeriod(previousData)) {
        console.log('🕐 Suppressing job progress error during grace period for new jobs');
        return false; // Don't throw error during grace period
      }
      
      // After grace period, allow errors to surface normally
      console.error('❌ Job progress fetch error after grace period:', error);
      return true;
    },
  });
}

export function useVMJobProgress(vmName: string | null) {
  const { data: allJobs, ...rest } = useJobProgress();
  
  // Find the most recent job for this VM
  const vmJob = vmName && allJobs 
    ? allJobs
        .filter(job => job.vm_name === vmName)
        .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())[0]
    : null;
  
  return {
    data: vmJob,
    ...rest
  };
}

// Helper functions for progress display
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function formatSpeed(bytesPerSecond: number): string {
  if (bytesPerSecond === 0) return '0 B/s';
  const k = 1024;
  const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
  const i = Math.floor(Math.log(bytesPerSecond) / Math.log(k));
  return parseFloat((bytesPerSecond / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function formatETA(seconds: number | null): string {
  if (!seconds || seconds <= 0) return 'Unknown';
  
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remainingSeconds = seconds % 60;
  
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else if (minutes > 0) {
    return `${minutes}m ${remainingSeconds}s`;
  } else {
    return `${remainingSeconds}s`;
  }
}

export function getProgressColor(status: string): string {
  switch (status.toLowerCase()) {
    case 'completed':
      return 'bg-green-600';
    case 'replicating':
    case 'running':
    case 'syncing':
      return 'bg-blue-600';
    case 'failed':
    case 'error':
      return 'bg-red-600';
    case 'pending':
    case 'initializing':
      return 'bg-yellow-600';
    default:
      return 'bg-gray-600';
  }
}

export function formatDuration(startTime: string | null, endTime?: string | null): string {
  if (!startTime) return 'Not started';
  
  const start = new Date(startTime);
  const end = endTime ? new Date(endTime) : new Date();
  const diffMs = end.getTime() - start.getTime();
  
  if (diffMs < 0) return 'Invalid duration';
  
  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  
  if (days > 0) {
    return `${days}d ${hours % 24}h ${minutes % 60}m`;
  } else if (hours > 0) {
    return `${hours}h ${minutes % 60}m`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  } else {
    return `${seconds}s`;
  }
}

export function formatStartTime(startTime: string | null): string {
  if (!startTime) return 'Not started';
  
  const start = new Date(startTime);
  const now = new Date();
  const diffMs = now.getTime() - start.getTime();
  
  // If less than 24 hours ago, show time only
  if (diffMs < 24 * 60 * 60 * 1000) {
    return start.toLocaleTimeString();
  }
  
  // If more than 24 hours ago, show date and time
  return start.toLocaleString();
}
