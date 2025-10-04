'use client';

import React from 'react';
import { Button, Badge, Spinner, Alert, Card } from 'flowbite-react';
import { HiRefresh, HiPlay, HiExclamationCircle } from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';
import { useVMContexts, VMContextListItem } from '../../hooks/useVMContext';
import { useJobProgress, formatBytes, getProgressColor } from '../../hooks/useJobProgress';
import { useFailoverProgress, getFailoverJobForVM } from '../../hooks/useFailoverProgress';

export interface VMTableProps {
  onVMSelect: (vmName: string) => void;
}

const VMTable = React.memo(({ onVMSelect }: VMTableProps) => {
  const { data: vmContexts, isLoading, error, refetch } = useVMContexts();
  const { data: allJobs } = useJobProgress();
  const { data: failoverJobs } = useFailoverProgress();

  const handleVMSelect = React.useCallback((vmName: string) => {
    onVMSelect(vmName);
  }, [onVMSelect]);

  const handleRefresh = React.useCallback(() => {
    refetch();
  }, [refetch]);

  const formatJobStatus = React.useCallback((status: string | null | undefined) => {
    if (!status) {
      return { color: 'gray' as const, text: 'Unknown' };
    }
    switch (status.toLowerCase()) {
      case 'completed':
        return { color: 'success' as const, text: 'Completed' };
      case 'failed':
        return { color: 'failure' as const, text: 'Failed' };
      case 'replicating':
        return { color: 'warning' as const, text: 'Replicating' };
      case 'running':
        return { color: 'info' as const, text: 'Running' };
      case 'pending':
        return { color: 'gray' as const, text: 'Pending' };
      default:
        return { color: 'gray' as const, text: status };
    }
  }, []);

  const formatProgress = React.useCallback((job?: VMContextListItem['current_job']) => {
    if (!job || !job.progress_percentage) {
      return null;
    }
    return (
      <div className="w-full bg-gray-200 rounded-full h-2.5 dark:bg-gray-700">
        <div 
          className="bg-blue-600 h-2.5 rounded-full transition-all duration-300" 
          style={{ width: `${Math.min(100, Math.max(0, job.progress_percentage))}%` }}
        ></div>
        <div className="text-xs mt-1 text-gray-600 dark:text-gray-400">
          {job.progress_percentage.toFixed(1)}%
          {job.vma_eta_seconds > 0 && (
            <span className="ml-2">
              ETA: {Math.round(job.vma_eta_seconds / 60)}m
            </span>
          )}
        </div>
      </div>
    );
  }, []);

  const formatLastActivity = React.useCallback((dateString: string) => {
    try {
      const date = new Date(dateString);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      const diffHours = Math.floor(diffMins / 60);
      const diffDays = Math.floor(diffHours / 24);

      if (diffDays > 0) {
        return `${diffDays}d ago`;
      } else if (diffHours > 0) {
        return `${diffHours}h ago`;
      } else if (diffMins > 0) {
        return `${diffMins}m ago`;
      } else {
        return 'Just now';
      }
    } catch {
      return 'Unknown';
    }
  }, []);

  if (isLoading) {
    return (
      <Card className="h-full">
        <div className="flex items-center justify-center h-96">
          <div className="text-center">
            <Spinner size="xl" />
            <p className="mt-4 text-gray-600 dark:text-gray-400">Loading virtual machines...</p>
          </div>
        </div>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="h-full">
        <div className="flex items-center justify-center h-96">
          <Alert color="failure" icon={HiExclamationCircle}>
            <div className="text-center">
              <h3 className="text-lg font-medium mb-2">Failed to Load VMs</h3>
              <p className="mb-4">{error instanceof Error ? error.message : 'Unknown error occurred'}</p>
              <Button onClick={handleRefresh} size="sm">
                <HiRefresh className="mr-2 h-4 w-4" />
                Retry
              </Button>
            </div>
          </Alert>
        </div>
      </Card>
    );
  }

  if (!vmContexts || vmContexts.length === 0) {
    return (
      <Card className="h-full">
        <div className="flex items-center justify-center h-96">
          <div className="text-center">
            <ClientIcon className="mx-auto h-12 w-12 text-gray-400 mb-4">
              <HiExclamationCircle />
            </ClientIcon>
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              No Virtual Machines Found
            </h3>
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              No VMs are currently available for migration.
            </p>
            <Button onClick={handleRefresh} size="sm">
              <HiRefresh className="mr-2 h-4 w-4" />
              Refresh
            </Button>
          </div>
        </div>
      </Card>
    );
  }

  return (
    <Card className="h-full">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            Virtual Machines
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            {vmContexts.length} VM{vmContexts.length !== 1 ? 's' : ''} available for migration
          </p>
        </div>
        <Button onClick={handleRefresh} size="sm" color="gray">
          <HiRefresh className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      <div className="overflow-x-auto">
        <div className="min-w-full">
          <div className="bg-white dark:bg-gray-800 shadow-sm rounded-lg overflow-hidden">
            {/* Table Header */}
            <div className="bg-gray-50 dark:bg-gray-700 px-6 py-3">
              <div className="grid grid-cols-6 gap-4 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                <div>VM Name</div>
                <div>Status</div>
                <div>Progress</div>
                <div>Jobs</div>
                <div>Last Activity</div>
                <div>Actions</div>
              </div>
            </div>
            
            {/* Table Body */}
            <div className="divide-y divide-gray-200 dark:divide-gray-700">
              {vmContexts.map((vm) => {
                const statusInfo = formatJobStatus(vm.current_job?.status || vm.status);
                // Find the most recent replication job for this VM
                const vmJob = allJobs 
                  ? allJobs
                      .filter(job => job.vm_name === vm.vm_name)
                      .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())[0]
                  : null;
                
                // Find active failover job for this VM
                const failoverJob = getFailoverJobForVM(failoverJobs?.jobs, vm.vm_name);
                
                return (
                  <div 
                    key={vm.vm_name}
                    className="px-6 py-4 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer transition-colors"
                    onClick={() => handleVMSelect(vm.vm_name)}
                  >
                    <div className="grid grid-cols-6 gap-4 items-center">
                      <div className="font-medium text-gray-900 dark:text-white">
                        <div className="text-sm">{vm.vm_name}</div>
                      </div>
                      <div className="space-y-1">
                        {/* Primary status badge - prioritize failover over replication */}
                        <Badge 
                          color={
                            failoverJob && ['pending', 'running', 'executing'].includes(failoverJob.status) 
                              ? 'purple' 
                              : failoverJob && failoverJob.status === 'completed'
                              ? 'success'
                              : failoverJob && failoverJob.status === 'failed'
                              ? 'failure'
                              : vmJob 
                              ? getProgressColor(vmJob.status).includes('green') ? 'success' : vmJob.status === 'replicating' ? 'warning' : vmJob.status === 'failed' ? 'failure' : 'gray' 
                              : statusInfo.color
                          } 
                          size="sm"
                        >
                          {failoverJob && ['pending', 'running', 'executing'].includes(failoverJob.status)
                            ? `${failoverJob.job_type} failover - ${failoverJob.current_phase || failoverJob.status}`
                            : failoverJob && failoverJob.status === 'completed'
                            ? `${failoverJob.job_type} failover completed`
                            : failoverJob && failoverJob.status === 'failed'
                            ? `${failoverJob.job_type} failover failed`
                            : vmJob 
                            ? vmJob.current_operation || vmJob.status 
                            : statusInfo.text
                          }
                        </Badge>
                        
                        {/* Secondary badge for replication when failover is active */}
                        {failoverJob && ['pending', 'running', 'executing'].includes(failoverJob.status) && vmJob && (
                          <Badge color="gray" size="xs">
                            Replication: {vmJob.status}
                          </Badge>
                        )}
                      </div>
                      <div>
                        {/* Show failover progress if active, otherwise show replication progress */}
                        {failoverJob && ['pending', 'running', 'executing'].includes(failoverJob.status) ? (
                          <div className="w-full space-y-1">
                            <div className="flex items-center justify-between text-xs">
                              <span className="text-purple-600 dark:text-purple-400 font-medium">
                                {failoverJob.progress_percent.toFixed(1)}% - {failoverJob.current_phase}
                              </span>
                            </div>
                            <div className="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
                              <div 
                                className="bg-purple-500 h-2 rounded-full transition-all duration-300"
                                style={{ width: `${Math.min(100, Math.max(0, failoverJob.progress_percent))}%` }}
                              />
                            </div>
                          </div>
                        ) : vmJob && (vmJob.status === 'replicating' || vmJob.status === 'running') ? (
                          <div className="w-full space-y-1">
                            <div className="flex items-center justify-between text-xs">
                              <span className="text-gray-600 dark:text-gray-400">
                                {vmJob.progress_percent.toFixed(1)}%
                              </span>
                              {vmJob.total_bytes > 0 && (
                                <span className="text-gray-500">
                                  {formatBytes(vmJob.bytes_transferred)} / {formatBytes(vmJob.total_bytes)}
                                </span>
                              )}
                            </div>
                            <div className="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
                              <div 
                                className={`h-2 rounded-full transition-all duration-500 ${getProgressColor(vmJob.status)}`}
                                style={{ width: `${Math.min(vmJob.progress_percent, 100)}%` }}
                              />
                            </div>
                          </div>
                        ) : vmJob && vmJob.status === 'completed' ? (
                          <div className="text-center">
                            <span className="text-sm text-green-600 dark:text-green-400 font-medium">
                              ✅ Completed
                            </span>
                          </div>
                        ) : vmJob && vmJob.status === 'failed' ? (
                          <div className="text-center">
                            <span className="text-sm text-red-600 dark:text-red-400 font-medium">
                              ❌ Failed
                            </span>
                          </div>
                        ) : (
                          <span className="text-sm text-gray-500 dark:text-gray-400">
                            No active job
                          </span>
                        )}
                      </div>
                      <div>
                        <span className="text-sm text-gray-900 dark:text-white">
                          {vm.job_count}
                        </span>
                      </div>
                      <div>
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          {formatLastActivity(vm.last_activity)}
                        </span>
                      </div>
                      <div>
                        <Button 
                          size="xs" 
                          color="blue"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleVMSelect(vm.vm_name);
                          }}
                        >
                          <HiPlay className="mr-1 h-3 w-3" />
                          Manage
                        </Button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      <div className="mt-6 text-sm text-gray-600 dark:text-gray-400 text-center">
        Click on any VM to view details and manage migration operations
      </div>
    </Card>
  );
});

VMTable.displayName = 'VMTable';

export default VMTable;
