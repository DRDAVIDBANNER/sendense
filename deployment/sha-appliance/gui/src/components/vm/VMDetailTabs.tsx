'use client';

import React from 'react';
import { Card, Button, Spinner, Alert, Badge, Tabs } from 'flowbite-react';
import { HiArrowLeft, HiExclamationCircle, HiRefresh } from 'react-icons/hi';
import { ClientIcon } from '../ClientIcon';
import { useVMContext } from '../../hooks/useVMContext';
import { useVMJobProgress } from '../../hooks/useJobProgress';

export interface VMDetailTabsProps {
  vmName: string;
  onBack: () => void;
}

export const VMDetailTabs = React.memo(({ vmName, onBack }: VMDetailTabsProps) => {
  const { data: vmContext, isLoading, error, refetch } = useVMContext(vmName);
  const { data: jobProgress, isLoading: jobLoading } = useVMJobProgress(vmName);

  const handleBack = React.useCallback(() => {
    onBack();
  }, [onBack]);

  const handleRefresh = React.useCallback(() => {
    refetch();
  }, [refetch]);

  const formatBytes = React.useCallback((bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }, []);

  const formatJobStatus = React.useCallback((status: string | null | undefined) => {
    if (!status) {
      return { color: 'gray' as const };
    }
    switch (status.toLowerCase()) {
      case 'completed':
        return { color: 'success' as const };
      case 'failed':
        return { color: 'failure' as const };
      case 'replicating':
      case 'running':
        return { color: 'warning' as const };
      default:
        return { color: 'gray' as const };
    }
  }, []);

  if (isLoading) {
    return (
      <div className="h-full p-6">
        <Card className="h-full">
          <div className="flex items-center justify-center h-64">
            <Spinner size="lg" />
            <span className="ml-3 text-gray-600 dark:text-gray-400">
              Loading VM details...
            </span>
          </div>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className="h-full p-6">
        <Alert color="failure" className="mb-4">
          <ClientIcon className="w-5 h-5 mr-2">
            <HiExclamationCircle />
          </ClientIcon>
          <span>Failed to load VM details: {error.message}</span>
        </Alert>
        <Card>
          <div className="text-center p-8">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
              Unable to Load VM Details
            </h3>
            <div className="space-x-2">
              <Button color="gray" onClick={handleBack}>
                <ClientIcon className="w-4 h-4 mr-2">
                  <HiArrowLeft />
                </ClientIcon>
                Back
              </Button>
              <Button color="blue" onClick={handleRefresh}>
                <ClientIcon className="w-4 h-4 mr-2">
                  <HiRefresh />
                </ClientIcon>
                Try Again
              </Button>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  if (!vmContext) {
    return (
      <div className="h-full p-6">
        <Card>
          <div className="text-center p-8">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
              VM Not Found
            </h3>
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              The virtual machine &quot;{vmName}&quot; was not found.
            </p>
            <Button color="gray" onClick={handleBack}>
              <ClientIcon className="w-4 h-4 mr-2">
                <HiArrowLeft />
              </ClientIcon>
              Back to VM List
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="h-full p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-4">
          <Button color="gray" size="sm" onClick={handleBack}>
            <ClientIcon className="w-4 h-4 mr-2">
              <HiArrowLeft />
            </ClientIcon>
            Back
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {vmName}
            </h1>
            <p className="text-gray-600 dark:text-gray-400">
              Virtual Machine Details
            </p>
          </div>
        </div>
        <Button color="blue" size="sm" onClick={handleRefresh}>
          <ClientIcon className="w-4 h-4 mr-2">
            <HiRefresh />
          </ClientIcon>
          Refresh
        </Button>
      </div>

      {/* VM Detail Tabs */}
      <Card className="h-full">
        <Tabs aria-label="VM Details" variant="underline">
          {/* Overview Tab */}
          <Tabs.Item active title="Overview">
            <div className="space-y-6">
              {/* Current Job Status */}
              {(jobProgress || vmContext.current_job) && (
                <Card>
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                    Current Job
                  </h3>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                        Status
                      </label>
                      <Badge color={formatJobStatus(jobProgress?.status || vmContext.current_job?.status).color} className="mt-1">
                        {jobProgress?.status || vmContext.current_job?.status}
                      </Badge>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                        Operation
                      </label>
                      <p className="mt-1 text-sm text-gray-900 dark:text-white">
                        {jobProgress?.current_operation || vmContext.current_job?.current_operation || 'N/A'}
                      </p>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                        Progress
                      </label>
                      <div className="mt-1">
                        <div className="flex items-center space-x-2">
                          <div className="flex-1 bg-gray-200 rounded-full h-2 dark:bg-gray-700">
                            <div 
                              className="bg-blue-600 h-2 rounded-full transition-all duration-300" 
                              style={{ 
                                width: `${Math.max(0, Math.min(100, jobProgress?.progress_percent || vmContext.current_job?.progress_percentage || 0))}%` 
                              }}
                            />
                          </div>
                          <span className="text-sm font-medium">
                            {Math.round(jobProgress?.progress_percent || vmContext.current_job?.progress_percentage || 0)}%
                          </span>
                        </div>
                        {(jobProgress?.vma_eta_seconds || vmContext.current_job?.vma_eta_seconds) && (
                          <p className="text-xs text-gray-500 mt-1">
                            ETA: {Math.floor((jobProgress?.vma_eta_seconds || vmContext.current_job?.vma_eta_seconds || 0) / 60)} minutes
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                </Card>
              )}

              {/* VM Specifications */}
              <Card>
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                  Virtual Machine Specifications
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      CPU Count
                    </label>
                    <p className="mt-1 text-sm text-gray-900 dark:text-white">
                      {vmContext.context.cpu_count ? `${vmContext.context.cpu_count}` : 'N/A'} vCPUs
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Memory
                    </label>
                    <p className="mt-1 text-sm text-gray-900 dark:text-white">
                      {vmContext.context.memory_mb 
                        ? `${(vmContext.context.memory_mb / 1024).toFixed(1)} GB`
                        : 'N/A'
                      }
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Guest OS
                    </label>
                    <p className="mt-1 text-sm text-gray-900 dark:text-white">
                      {vmContext.context.guest_os || vmContext.context.os_type || 'N/A'}
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Power State
                    </label>
                    <Badge color={vmContext.context.power_state === 'poweredOn' ? 'success' : 'gray'} className="mt-1">
                      {vmContext.context.power_state || 'Unknown'}
                    </Badge>
                  </div>
                </div>
              </Card>
            </div>
          </Tabs.Item>

          {/* Jobs Tab */}
          <Tabs.Item title="Jobs">
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                Job History
              </h3>
              
              {vmContext.job_history && vmContext.job_history.length > 0 ? (
                <div className="overflow-auto">
                  <table className="w-full text-sm text-left text-gray-500 dark:text-gray-400">
                    <thead className="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400">
                      <tr>
                        <th scope="col" className="px-4 py-3">Job ID</th>
                        <th scope="col" className="px-4 py-3">Type</th>
                        <th scope="col" className="px-4 py-3">Status</th>
                        <th scope="col" className="px-4 py-3">Progress</th>
                        <th scope="col" className="px-4 py-3">Created</th>
                        <th scope="col" className="px-4 py-3">Duration</th>
                      </tr>
                    </thead>
                    <tbody>
                      {vmContext.job_history.map((job, index) => (
                        <tr key={`${job.id}-${index}`} className="bg-white border-b dark:bg-gray-800 dark:border-gray-700">
                          <td className="px-4 py-3 font-mono text-xs">
                            {job.id.substring(0, 16)}...
                          </td>
                          <td className="px-4 py-3">
                            <Badge color="info" size="xs">
                              {job.replication_type || 'Migration'}
                            </Badge>
                          </td>
                          <td className="px-4 py-3">
                            <Badge color={formatJobStatus(job.status).color} size="xs">
                              {job.status}
                            </Badge>
                          </td>
                          <td className="px-4 py-3">
                            {Math.round(job.progress_percentage || 0)}%
                          </td>
                          <td className="px-4 py-3 text-xs">
                            {job.created_at ? new Date(job.created_at).toLocaleDateString() : 'N/A'}
                          </td>
                          <td className="px-4 py-3 text-xs">
                            {job.started_at && job.completed_at 
                              ? `${Math.round((new Date(job.completed_at).getTime() - new Date(job.started_at).getTime()) / 60000)}m`
                              : 'N/A'
                            }
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="text-center py-8">
                  <p className="text-gray-600 dark:text-gray-400">
                    No job history available for this VM.
                  </p>
                </div>
              )}
            </div>
          </Tabs.Item>

          {/* Details Tab */}
          <Tabs.Item title="Details">
            <div className="space-y-6">
              {/* VM Information */}
              <Card>
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                  VM Information
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      VM Path
                    </label>
                    <p className="mt-1 text-sm text-gray-900 dark:text-white font-mono">
                      {vmContext.context.source_vm_path}
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Datacenter
                    </label>
                    <p className="mt-1 text-sm text-gray-900 dark:text-white">
                      {vmContext.context.datacenter}
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Created
                    </label>
                    <p className="mt-1 text-sm text-gray-900 dark:text-white">
                      {new Date(vmContext.context.created_at).toLocaleString()}
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      Last Updated
                    </label>
                    <p className="mt-1 text-sm text-gray-900 dark:text-white">
                      {new Date(vmContext.context.updated_at).toLocaleString()}
                    </p>
                  </div>
                </div>
              </Card>

              {/* Disk Information */}
              {vmContext.disks && vmContext.disks.length > 0 && (
                <Card>
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                    Disk Configuration
                  </h3>
                  <div className="overflow-auto">
                    <table className="w-full text-sm text-left text-gray-500 dark:text-gray-400">
                      <thead className="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400">
                        <tr>
                          <th scope="col" className="px-4 py-3">Label</th>
                          <th scope="col" className="px-4 py-3">Size</th>
                          <th scope="col" className="px-4 py-3">Provisioning</th>
                          <th scope="col" className="px-4 py-3">Datastore</th>
                          <th scope="col" className="px-4 py-3">Unit #</th>
                        </tr>
                      </thead>
                      <tbody>
                        {vmContext.disks.map((disk, index) => (
                          <tr key={`${disk.id}-${index}`} className="bg-white border-b dark:bg-gray-800 dark:border-gray-700">
                            <td className="px-4 py-3 font-medium">
                              {disk.disk_label}
                            </td>
                            <td className="px-4 py-3">
                              {disk.size_gb}GB ({formatBytes(disk.capacity_bytes)})
                            </td>
                            <td className="px-4 py-3">
                              <Badge color="info" size="xs">
                                {disk.provisioning_type}
                              </Badge>
                            </td>
                            <td className="px-4 py-3">
                              {disk.datastore}
                            </td>
                            <td className="px-4 py-3">
                              {disk.unit_number}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </Card>
              )}
            </div>
          </Tabs.Item>

          {/* CBT Tab */}
          <Tabs.Item title="CBT History">
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                Change Block Tracking History
              </h3>
              
              {vmContext.cbt_history && vmContext.cbt_history.length > 0 ? (
                <div className="overflow-auto">
                  <table className="w-full text-sm text-left text-gray-500 dark:text-gray-400">
                    <thead className="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400">
                      <tr>
                        <th scope="col" className="px-4 py-3">Change ID</th>
                        <th scope="col" className="px-4 py-3">Operation</th>
                        <th scope="col" className="px-4 py-3">Bytes Changed</th>
                        <th scope="col" className="px-4 py-3">Captured</th>
                      </tr>
                    </thead>
                    <tbody>
                      {vmContext.cbt_history.map((cbt, index) => (
                        <tr key={`${cbt.id}-${index}`} className="bg-white border-b dark:bg-gray-800 dark:border-gray-700">
                          <td className="px-4 py-3 font-mono text-xs">
                            {cbt.change_id.substring(0, 20)}...
                          </td>
                          <td className="px-4 py-3">
                            <Badge color="purple" size="xs">
                              {cbt.operation_type}
                            </Badge>
                          </td>
                          <td className="px-4 py-3">
                            {formatBytes(cbt.bytes_changed)}
                          </td>
                          <td className="px-4 py-3 text-xs">
                            {new Date(cbt.captured_at).toLocaleString()}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="text-center py-8">
                  <p className="text-gray-600 dark:text-gray-400">
                    No CBT history available for this VM.
                  </p>
                </div>
              )}
            </div>
          </Tabs.Item>
        </Tabs>
      </Card>
    </div>
  );
});

VMDetailTabs.displayName = 'VMDetailTabs';
