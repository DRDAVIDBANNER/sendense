import { useState, useEffect } from 'react';

export interface FailoverJob {
  job_id: string;
  vm_name: string;
  vm_id: string;
  job_type: 'live' | 'test';
  status: string;
  progress_percent: number;
  current_phase: string;
  created_at: string;
  updated_at: string;
  error_message?: string;
}

export interface FailoverProgress {
  jobs: FailoverJob[];
  totalJobs: number;
  activeJobs: number;
  completedJobs: number;
  failedJobs: number;
}

export const useFailoverProgress = () => {
  const [data, setData] = useState<FailoverProgress | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchFailoverProgress = async () => {
    try {
      // Workaround: Check job tracking for active failover operations
      const response = await fetch('/api/vm-contexts');
      if (!response.ok) {
        throw new Error('Failed to fetch VM contexts');
      }
      
      const vmContexts = await response.json();
      
      // Extract failover jobs from VM context data
      const jobs: FailoverJob[] = [];
      
      if (vmContexts && Array.isArray(vmContexts)) {
        for (const vm of vmContexts) {
          // Check if VM has active failover operation
          if (vm.current_status && vm.current_status.includes('failover') || vm.current_status.includes('failed_over')) {
            jobs.push({
              job_id: vm.current_job_id || `failover-${vm.vm_name}-${Date.now()}`,
              vm_name: vm.vm_name,
              vm_id: vm.vmware_vm_id,
              job_type: vm.current_status.includes('live') ? 'live' : 'test',
              status: vm.current_status.includes('failed_over') ? 'completed' : 'running',
              progress_percent: 0, // Will be updated by individual status calls
              current_phase: vm.current_status,
              created_at: vm.updated_at || new Date().toISOString(),
              updated_at: vm.updated_at || new Date().toISOString(),
            });
          }
        }
      }
      
      // Process jobs data
      const failoverProgress: FailoverProgress = {
        jobs: jobs || [],
        totalJobs: jobs?.length || 0,
        activeJobs: jobs?.filter((job: FailoverJob) => 
          ['pending', 'running', 'executing'].includes(job.status)
        ).length || 0,
        completedJobs: jobs?.filter((job: FailoverJob) => 
          job.status === 'completed'
        ).length || 0,
        failedJobs: jobs?.filter((job: FailoverJob) => 
          job.status === 'failed'
        ).length || 0,
      };
      
      setData(failoverProgress);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setData(null);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchFailoverProgress();
    
    // Poll for updates every 2 seconds during active operations
    const interval = setInterval(() => {
      if (data?.activeJobs && data.activeJobs > 0) {
        fetchFailoverProgress();
      }
    }, 2000);

    return () => clearInterval(interval);
  }, [data?.activeJobs]);

  return {
    data,
    isLoading,
    error,
    refetch: fetchFailoverProgress,
  };
};

export const getFailoverJobForVM = (failoverJobs: FailoverJob[] | undefined, vmName: string): FailoverJob | null => {
  if (!failoverJobs) return null;
  
  // Find the most recent failover job for this VM
  return failoverJobs
    .filter(job => job.vm_name === vmName)
    .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())[0] || null;
};
